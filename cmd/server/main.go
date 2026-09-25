package server

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"

	"github.com/rudi-asr/ujiscan/internal/api"
	"github.com/rudi-asr/ujiscan/internal/audit"
	"github.com/rudi-asr/ujiscan/internal/auth"
	"github.com/rudi-asr/ujiscan/internal/compression"
	"github.com/rudi-asr/ujiscan/internal/dashboard"
	"github.com/rudi-asr/ujiscan/internal/db"
	"github.com/rudi-asr/ujiscan/internal/engagement"
	"github.com/rudi-asr/ujiscan/internal/executor"
	"github.com/rudi-asr/ujiscan/internal/metrics"
	"github.com/rudi-asr/ujiscan/internal/persistence"
	"github.com/rudi-asr/ujiscan/internal/playbook"
	"github.com/rudi-asr/ujiscan/internal/registry"
	"github.com/rudi-asr/ujiscan/internal/store"
	"github.com/rudi-asr/ujiscan/internal/tools"
)

const (
	Port = ":8081"
)

func Run() error {
	// Get project root from current working directory
	projectRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Initialize SQLite database (DATABASE_PATH env overrides default — used by Docker)
	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = filepath.Join(projectRoot, "ujiscan.db")
	}
	sqliteDB, err := db.InitDB(dbPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer sqliteDB.Close()
	log.Printf("✅ SQLite database initialized: %s", dbPath)

	// Create connection pool with optimizations
	poolConfig := db.PoolConfig{
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 300,
	}
	dbPool := db.NewPool(sqliteDB, poolConfig)

	// Apply SQLite optimizations
	if err := dbPool.Optimize(); err != nil {
		log.Printf("Warning: Failed to optimize database: %v", err)
	}

	// Add performance indexes
	if err := dbPool.AddIndexes(); err != nil {
		log.Printf("Warning: Failed to add indexes: %v", err)
	}

	log.Printf("✅ Database pool configured (maxOpen=%d, maxIdle=%d)", poolConfig.MaxOpenConns, poolConfig.MaxIdleConns)

	// Initialize persistence manager
	pm := persistence.NewPersistenceManager(sqliteDB)

	// Load registry from tools.yaml
	toolsYAML := filepath.Join(projectRoot, "tools.yaml")
	reg, err := registry.LoadRegistry(toolsYAML)
	if err != nil {
		log.Printf("Warning: Failed to load tools registry: %v", err)
		log.Printf("Using legacy tool executor instead")
	}

	// Initialize stores and executors
	scanStore := store.NewScanStore()
	toolExecutor := tools.NewExecutor()
	toolExecutor.InitializeDefaultTools()
	scanExecutor := executor.NewScanExecutor(scanStore, toolExecutor)

	// Create registry executor if registry loaded
	var regExecutor *tools.RegistryExecutor
	if reg != nil {
		regExecutor = tools.NewRegistryExecutor(reg)
		if err := regExecutor.ValidateRegistry(); err != nil {
			log.Printf("Warning: Registry validation failed: %v", err)
		} else {
			log.Printf("✅ Tool registry loaded successfully (%d tools, %d modes)", len(reg.Tools), len(reg.Modes))
		}
	}

	// Initialize playbook engine
	playbookLoader := playbook.NewPlaybookLoader(filepath.Join(projectRoot, "playbooks"))
	playbookEngine := playbook.NewEngine(playbookLoader, scanExecutor, scanStore)

	// Initialize authentication
	userStore := auth.NewMemoryUserStore()
	sessionStore := auth.NewMemorySessionStore()
	tokenManager := auth.NewSimpleTokenManager("ujiscan-secret-key")
	authService := auth.NewAuthService(userStore, sessionStore, tokenManager)
	authHandler := auth.NewHandler(authService, userStore)

	// Initialize Phase 7 services (engagement, audit, dashboard)
	engagementStore := engagement.NewMemoryEngagementStore()
	findingStore := engagement.NewMemoryFindingStore()
	commentStore := engagement.NewMemoryCommentStore()
	engagementService := engagement.NewEngagementService(engagementStore, findingStore, commentStore)
	engagementHandler := engagement.NewHandler(engagementService, engagementStore, findingStore, commentStore)

	// Load engagement state from database (if exists)
	_ = engagement.LoadState(pm, engagementStore, findingStore, commentStore)

	auditStore := audit.NewMemoryAuditStore()
	auditService := audit.NewAuditService(auditStore)
	auditHandler := audit.NewHandler(auditService)

	dashboardService := dashboard.NewMemoryDashboardService()
	notificationService := dashboard.NewMemoryNotificationService()
	dashboardHandler := dashboard.NewHandler(dashboardService, notificationService)

	// Initialize metrics collector
	metricsCollector := metrics.New(25) // max connections from pool config
	metricsHandler := metrics.NewHandler(metricsCollector)
	metricsHandler.SetDatabaseStatsCallback(func() (int, int, error) {
		stats := dbPool.Stats()
		return stats.OpenConnections, stats.Idle, nil
	})

	// Create API handler
	apiHandler := api.NewHandler(scanStore, toolExecutor, scanExecutor, playbookEngine)

	// Attach persistence hooks so every engagement mutation is saved
	engagementHandler.SetPersistence(pm)

	// Routes
	mux := http.NewServeMux()

	// Static files
	webDir := filepath.Join(projectRoot, "web")
	log.Printf("Web directory: %s", webDir)

	// Check if web directory exists
	if _, err := os.Stat(webDir); err != nil {
		return fmt.Errorf("web directory not found: %s", webDir)
	}

	mux.Handle("/", http.FileServer(http.Dir(webDir)))

	// API routes
	mux.HandleFunc("/api/status", handleStatus)

	// Auth routes (no authentication required for login)
	mux.HandleFunc("/auth/login", authHandler.HandleLogin)
	mux.HandleFunc("/auth/logout", authHandler.HandleLogout)
	mux.HandleFunc("/auth/me", authHandler.HandleMe)
	mux.HandleFunc("/auth/change-password", authHandler.HandleChangePassword)

	// User management routes (admin only)
	adminOnly := auth.RequireRoles(auth.RoleAdmin)
	mux.Handle("/api/users", adminOnly(http.HandlerFunc(requireAuth(authHandler.HandleListUsers))))
	mux.Handle("/api/users/create", adminOnly(http.HandlerFunc(requireAuth(authHandler.HandleCreateUser))))
	mux.HandleFunc("/api/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if auth.GetRole(r) != auth.RoleAdmin {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		authHandler.HandleDeleteUser(w, r, r.PathValue("id"))
	})

	// Phase 7 Engagement routes
	mux.HandleFunc("/api/engagements", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			engagementHandler.HandleListEngagements(w, r)
		case http.MethodPost:
			engagementHandler.HandleCreateEngagement(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))
	mux.HandleFunc("/api/engagements/{id}", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		engID := r.PathValue("id")
		switch r.Method {
		case http.MethodGet:
			engagementHandler.HandleGetEngagement(w, r, engID)
		case http.MethodPut:
			engagementHandler.HandleUpdateEngagement(w, r, engID)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))
	mux.HandleFunc("POST /api/engagements/{id}/activate", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		engagementHandler.HandleActivateEngagement(w, r, r.PathValue("id"))
	}))
	mux.HandleFunc("POST /api/engagements/{id}/complete", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		engagementHandler.HandleCompleteEngagement(w, r, r.PathValue("id"))
	}))
	mux.HandleFunc("POST /api/engagements/{id}/signoff", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		engagementHandler.HandleSignOffEngagement(w, r, r.PathValue("id"))
	}))
	mux.HandleFunc("POST /api/engagements/{id}/assign", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		engagementHandler.HandleAssignPentester(w, r, r.PathValue("id"))
	}))
	mux.HandleFunc("GET /api/engagements/{id}/findings", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		engagementHandler.HandleListFindings(w, r, r.PathValue("id"))
	}))

	// Phase 7 Finding routes
	mux.HandleFunc("GET /api/findings/{id}", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		engagementHandler.HandleGetFinding(w, r, r.PathValue("id"))
	}))
	mux.HandleFunc("PUT /api/findings/{id}/verify", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		engagementHandler.HandleMarkFindingAsVerified(w, r, r.PathValue("id"))
	}))
	mux.HandleFunc("PUT /api/findings/{id}/false-positive", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		engagementHandler.HandleMarkFindingAsFalsePositive(w, r, r.PathValue("id"))
	}))
	mux.HandleFunc("PUT /api/findings/{id}/approve", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		engagementHandler.HandleApproveFinding(w, r, r.PathValue("id"))
	}))
	mux.HandleFunc("/api/findings/{id}/comments", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		findingID := r.PathValue("id")
		switch r.Method {
		case http.MethodGet:
			engagementHandler.HandleGetFindingComments(w, r, findingID)
		case http.MethodPost:
			engagementHandler.HandleAddComment(w, r, findingID)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	// Phase 7 Audit routes (admin + auditor only)
	auditRoles := auth.RequireRoles(auth.RoleAdmin, auth.RoleAuditor)
	mux.Handle("/api/audit/logs", auditRoles(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Support date-range query: /api/audit/logs?start=...&end=...
		if r.URL.Query().Get("start") != "" && r.URL.Query().Get("end") != "" {
			auditHandler.HandleListLogsByDateRange(w, r)
			return
		}
		auditHandler.HandleListLogs(w, r)
	})))
	mux.Handle("/api/audit/user/{userID}", auditRoles(http.HandlerFunc(auditHandler.HandleListLogsByUser)))
	mux.Handle("/api/audit/action/{action}", auditRoles(http.HandlerFunc(auditHandler.HandleListLogsByAction)))
	mux.Handle("/api/audit/resource/{resource}", auditRoles(http.HandlerFunc(auditHandler.HandleListLogsByResource)))
	mux.Handle("/api/audit/export", auditRoles(http.HandlerFunc(auditHandler.HandleExportLogs)))
	mux.Handle("/api/audit/stats", auditRoles(http.HandlerFunc(auditHandler.HandleStats)))

	// Phase 7 Dashboard routes (role-scoped)
	mux.Handle("/api/dashboard/team",
		auth.RequireRoles(auth.RoleAdmin, auth.RolePentester)(http.HandlerFunc(dashboardHandler.HandleTeamDashboard)))
	mux.Handle("/api/dashboard/client/{engagementID}",
		auth.RequireRoles(auth.RoleAdmin, auth.RoleAuditor, auth.RoleClient)(http.HandlerFunc(dashboardHandler.HandleClientDashboard)))
	mux.Handle("/api/dashboard/admin",
		auth.RequireRoles(auth.RoleAdmin)(http.HandlerFunc(dashboardHandler.HandleAdminDashboard)))

	// Metrics endpoints (admin + auditor only for detailed stats)
	metricsRoles := auth.RequireRoles(auth.RoleAdmin, auth.RoleAuditor)
	mux.Handle("/api/metrics", metricsRoles(http.HandlerFunc(metricsHandler.HandleMetrics)))
	mux.Handle("/api/metrics/health", metricsRoles(http.HandlerFunc(metricsHandler.HandleHealth)))
	mux.Handle("/api/metrics/database", metricsRoles(http.HandlerFunc(metricsHandler.HandleDatabaseStats)))
	mux.Handle("/api/metrics/cache", metricsRoles(http.HandlerFunc(metricsHandler.HandleCacheStats)))
	mux.Handle("/api/metrics/requests", metricsRoles(http.HandlerFunc(metricsHandler.HandleRequestStats)))
	mux.Handle("/api/metrics/export", metricsRoles(http.HandlerFunc(metricsHandler.HandleExport)))

	// Phase 7 Notification routes
	mux.HandleFunc("/api/notifications", requireAuth(dashboardHandler.HandleGetNotifications))
	mux.HandleFunc("PUT /api/notifications/{id}/read", requireAuth(dashboardHandler.HandleMarkNotificationAsRead))
	mux.HandleFunc("DELETE /api/notifications/{id}", requireAuth(dashboardHandler.HandleDeleteNotification))

	mux.HandleFunc("/api/tools", apiHandler.HandleListTools)
	mux.HandleFunc("/api/stats", apiHandler.HandleStats)
	mux.HandleFunc("/api/playbooks", apiHandler.HandleListPlaybooks)

	// Scan API routes with custom handler
	mux.HandleFunc("/api/scan", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			apiHandler.HandleStartScan(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Playbook scan endpoint (supports both regular and agentic via query param or path)
	mux.HandleFunc("/api/scan/playbook", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			// Check if this is agentic mode (by query param or path ending with /agentic)
			isAgentic := r.URL.Query().Get("agentic") == "true" || strings.HasSuffix(r.URL.Path, "/agentic")
			if isAgentic {
				apiHandler.HandleAgenticPlaybookScan(w, r)
			} else {
				apiHandler.HandlePlaybookScan(w, r)
			}
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Alternative agentic endpoint for convenience
	mux.HandleFunc("/api/agentic", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			apiHandler.HandleAgenticPlaybookScan(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Dynamic scan routes: /api/scan/{id}
	// Must come AFTER specific routes like /api/scan/playbook
	mux.HandleFunc("/api/scan/", func(w http.ResponseWriter, r *http.Request) {
		// Extract scan ID from path: /api/scan/{id}
		// Skip if this is a playbook-specific route
		if strings.Contains(r.URL.Path, "/playbook") {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/scan/"), "/")
		if len(parts) == 0 || parts[0] == "" {
			http.Error(w, "Scan ID required", http.StatusBadRequest)
			return
		}

		scanID := parts[0]

		// Check if this is a report request
		if len(parts) > 1 && parts[1] == "report" {
			// /api/scan/{id}/report or /api/scan/{id}/report/{format}
			if len(parts) > 2 {
				// Export in specific format: /api/scan/{id}/report/{format}
				format := parts[2]
				apiHandler.HandleExportReport(w, r, scanID, format)
			} else {
				// Generate report: /api/scan/{id}/report
				apiHandler.HandleGenerateReport(w, r, scanID)
			}
			return
		}

		switch r.Method {
		case http.MethodGet:
			apiHandler.HandleGetScan(w, r, scanID)
		case http.MethodDelete:
			apiHandler.HandleDeleteScan(w, r, scanID)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Start server
	log.Printf("ujiscan server starting on http://localhost%s", Port)
	log.Printf("API endpoints:")
	log.Printf("  POST /auth/login         - User login")
	log.Printf("  POST /auth/logout        - User logout")
	log.Printf("  GET  /auth/me            - Get current user info")
	log.Printf("  POST /auth/change-password - Change password")
	log.Printf("  GET  /api/status         - Server health")
	log.Printf("  GET  /api/tools          - List tools")
	log.Printf("  GET  /api/stats          - Scan statistics")
	log.Printf("  GET  /api/playbooks      - List playbooks")
	log.Printf("  POST /api/scan           - Start scan")
	log.Printf("  POST /api/scan/playbook  - Start playbook scan")
	log.Printf("  GET  /api/scan/{id}      - Get scan details")
	log.Printf("  DEL  /api/scan/{id}      - Delete scan")
	log.Printf("  POST /api/scan/{id}/report           - Generate report (JSON)")
	log.Printf("  GET  /api/scan/{id}/report/{format}  - Export report (html/md/json)")
	log.Printf("  GET  /api/users          - List users (admin only)")
	log.Printf("  POST /api/users/create   - Create user (admin only)")
	log.Printf("CORS enabled for: http://localhost:8081, https://rudi-asr.github.io")
	log.Printf("")
	log.Printf("DEFAULT CREDENTIALS (CHANGE IN PRODUCTION):")
	log.Printf("  Email: admin@ujiscan.local")
	log.Printf("  Password: admin123")

	// Wrap mux with CORS and auth middleware
	corsHandler := corsMiddleware(mux)
	authMiddleware := auth.AuthMiddleware(tokenManager)
	return http.ListenAndServe(Port, compression.Middleware(authMiddleware(corsHandler)))
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"ok","service":"ujiscan"}`)
}

// requireAuth rejects requests without a valid authenticated user context
func requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if auth.GetUserID(r) == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

// corsMiddleware adds CORS headers to allow requests from GitHub Pages & localhost
func corsMiddleware(next http.Handler) http.Handler {
	// Default allowed origins (localhost dev + GitHub Pages dashboard)
	allowedOrigins := map[string]bool{
		"http://localhost:8081":      true,
		"http://localhost:3000":      true,
		"https://rudi-asr.github.io": true,
	}
	// Extend with CORS_ALLOWED_ORIGINS env (comma-separated) — used by Docker
	for _, o := range strings.Split(os.Getenv("CORS_ALLOWED_ORIGINS"), ",") {
		if o = strings.TrimSpace(o); o != "" {
			allowedOrigins[o] = true
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow requests from GitHub Pages and localhost
		origin := r.Header.Get("Origin")
		if origin != "" && (allowedOrigins[origin] || strings.HasPrefix(origin, "http://127.0.0.1")) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Max-Age", "3600")

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
