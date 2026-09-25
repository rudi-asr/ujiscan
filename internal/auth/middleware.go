// Package auth provides RBAC middleware
package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// ContextKey type for storing values in request context
type ContextKey string

const (
	ContextKeyUserID   ContextKey = "user_id"
	ContextKeyEmail    ContextKey = "email"
	ContextKeyRole     ContextKey = "role"
	ContextKeyClaims   ContextKey = "claims"
)

// AuthMiddleware checks JWT token and adds user info to context
func AuthMiddleware(tokenManager TokenManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p := r.URL.Path

			// CORS preflight must always pass through to corsMiddleware
			if r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			// Public endpoints: login, health check, and static assets
			if strings.HasSuffix(p, "/auth/login") ||
				p == "/api/status" ||
				(r.Method == http.MethodGet && !strings.HasPrefix(p, "/api/") && !strings.HasPrefix(p, "/auth/")) {
				next.ServeHTTP(w, r)
				return
			}

			// Get token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "missing authorization header", http.StatusUnauthorized)
				return
			}

			// Extract token from "Bearer <token>"
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, "invalid authorization header", http.StatusUnauthorized)
				return
			}

			token := parts[1]

			// Validate token
			claims, err := tokenManager.ValidateToken(token)
			if err != nil {
				http.Error(w, fmt.Sprintf("invalid token: %v", err), http.StatusUnauthorized)
				return
			}

			// Add user info to context
			ctx := context.WithValue(r.Context(), ContextKeyUserID, claims.UserID)
			ctx = context.WithValue(ctx, ContextKeyEmail, claims.Email)
			ctx = context.WithValue(ctx, ContextKeyRole, claims.Role)
			ctx = context.WithValue(ctx, ContextKeyClaims, claims)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole middleware checks if user has required role
func RequireRole(requiredRole Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get role from context
			role, ok := r.Context().Value(ContextKeyRole).(Role)
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			// Check if role matches (Admin can access everything)
			if role != RoleAdmin && role != requiredRole {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequirePermission middleware checks if user has required permission
func RequirePermission(requiredPerm Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get role from context
			role, ok := r.Context().Value(ContextKeyRole).(Role)
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			// Check if role has permission
			if !HasPermission(role, requiredPerm) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireRoles middleware checks if user has one of the required roles
func RequireRoles(allowedRoles ...Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get role from context
			role, ok := r.Context().Value(ContextKeyRole).(Role)
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			// Check if role is in allowed list
			found := false
			for _, allowedRole := range allowedRoles {
				if role == allowedRole || role == RoleAdmin {
					found = true
					break
				}
			}

			if !found {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GetUserID extracts user ID from request context
func GetUserID(r *http.Request) string {
	userID, ok := r.Context().Value(ContextKeyUserID).(string)
	if !ok {
		return ""
	}
	return userID
}

// GetRole extracts role from request context
func GetRole(r *http.Request) Role {
	role, ok := r.Context().Value(ContextKeyRole).(Role)
	if !ok {
		return ""
	}
	return role
}

// GetClaims extracts JWT claims from request context
func GetClaims(r *http.Request) *Claims {
	claims, ok := r.Context().Value(ContextKeyClaims).(*Claims)
	if !ok {
		return nil
	}
	return claims
}

// GetEmail extracts email from request context
func GetEmail(r *http.Request) string {
	email, ok := r.Context().Value(ContextKeyEmail).(string)
	if !ok {
		return ""
	}
	return email
}
