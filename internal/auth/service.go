// Package auth provides JWT token management
package auth

import (
	"fmt"
	"time"
)

// SimpleTokenManager implements TokenManager using JWT-like tokens
// (Note: For production, use a real JWT library like github.com/golang-jwt/jwt)
type SimpleTokenManager struct {
	secretKey string
	tokens    map[string]*Claims // In-memory token storage
}

// NewSimpleTokenManager creates a new token manager
func NewSimpleTokenManager(secretKey string) *SimpleTokenManager {
	if secretKey == "" {
		secretKey = "ujiscan-default-secret-key-change-in-production"
	}

	return &SimpleTokenManager{
		secretKey: secretKey,
		tokens:    make(map[string]*Claims),
	}
}

// GenerateToken generates a new JWT token
func (tm *SimpleTokenManager) GenerateToken(user *User, duration time.Duration) (string, error) {
	now := time.Now()
	expiresAt := now.Add(duration)

	claims := &Claims{
		UserID:    user.ID,
		Email:     user.Email,
		Name:      user.Name,
		Role:      user.Role,
		ExpiresAt: expiresAt,
		IssuedAt:  now,
	}

	// For simplicity, use a token ID (in production, encode claims in JWT)
	tokenID := fmt.Sprintf("token_%d_%s", now.UnixNano(), user.ID)

	// Store claims
	tm.tokens[tokenID] = claims

	return tokenID, nil
}

// ValidateToken validates a JWT token
func (tm *SimpleTokenManager) ValidateToken(token string) (*Claims, error) {
	claims, exists := tm.tokens[token]
	if !exists {
		return nil, fmt.Errorf("invalid token")
	}

	if claims.IsExpired() {
		delete(tm.tokens, token)
		return nil, fmt.Errorf("token expired")
	}

	return claims, nil
}

// RefreshToken refreshes a JWT token
func (tm *SimpleTokenManager) RefreshToken(token string) (string, error) {
	claims, err := tm.ValidateToken(token)
	if err != nil {
		return "", err
	}

	// Create new token with same user info
	newTokenID := fmt.Sprintf("token_%d_%s", time.Now().UnixNano(), claims.UserID)

	// Extend expiration by 24 hours
	expiresAt := time.Now().Add(24 * time.Hour)
	newClaims := &Claims{
		UserID:    claims.UserID,
		Email:     claims.Email,
		Name:      claims.Name,
		Role:      claims.Role,
		ExpiresAt: expiresAt,
		IssuedAt:  time.Now(),
	}

	tm.tokens[newTokenID] = newClaims

	// Invalidate old token
	delete(tm.tokens, token)

	return newTokenID, nil
}

// InvalidateToken invalidates a token
func (tm *SimpleTokenManager) InvalidateToken(token string) error {
	delete(tm.tokens, token)
	return nil
}

// AuthService provides authentication operations
type AuthService struct {
	userStore       UserStore
	sessionStore    SessionStore
	tokenManager    TokenManager
	tokenDuration   time.Duration
}

// NewAuthService creates a new auth service
func NewAuthService(userStore UserStore, sessionStore SessionStore, tokenManager TokenManager) *AuthService {
	return &AuthService{
		userStore:     userStore,
		sessionStore:  sessionStore,
		tokenManager:  tokenManager,
		tokenDuration: 24 * time.Hour,
	}
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	Token     string `json:"token"`
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	Role      string `json:"role"`
	ExpiresAt int64  `json:"expires_at"`
}

// Login authenticates a user
func (as *AuthService) Login(email, password, ipAddress, userAgent string) (*LoginResponse, error) {
	// Get user by email
	user, err := as.userStore.GetUserByEmail(email)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Check if user is enabled
	if !user.Enabled {
		return nil, fmt.Errorf("user is disabled")
	}

	// Verify password
	if !VerifyPassword(password, user.PasswordHash) {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Generate token
	token, err := as.tokenManager.GenerateToken(user, as.tokenDuration)
	if err != nil {
		return nil, err
	}

	// Create session
	expiresAt := time.Now().Add(as.tokenDuration)
	_, err = as.sessionStore.CreateSession(user.ID, token, ipAddress, userAgent, expiresAt)
	if err != nil {
		return nil, err
	}

	// Update last login
	as.userStore.UpdateLastLogin(user.ID)

	return &LoginResponse{
		Token:     token,
		UserID:    user.ID,
		Email:     user.Email,
		Name:      user.Name,
		Role:      string(user.Role),
		ExpiresAt: expiresAt.Unix(),
	}, nil
}

// Logout logs out a user
func (as *AuthService) Logout(token string) error {
	claims, err := as.tokenManager.ValidateToken(token)
	if err != nil {
		return err
	}

	// Invalidate all sessions for user
	return as.sessionStore.InvalidateAllUserSessions(claims.UserID)
}

// ValidateToken validates a token and returns user info
func (as *AuthService) ValidateToken(token string) (*Claims, error) {
	return as.tokenManager.ValidateToken(token)
}

// ChangePassword changes a user's password
func (as *AuthService) ChangePassword(userID, currentPassword, newPassword string) error {
	user, err := as.userStore.GetUser(userID)
	if err != nil {
		return err
	}

	// Verify current password
	if !VerifyPassword(currentPassword, user.PasswordHash) {
		return fmt.Errorf("current password is incorrect")
	}

	// Hash new password
	user.PasswordHash = PasswordHash(newPassword)

	// Update user
	return as.userStore.UpdateUser(user)
}
