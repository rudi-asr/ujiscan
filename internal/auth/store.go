// Package auth provides user store implementations
package auth

import (
	"fmt"
	"sync"
	"time"
)

// MemoryUserStore implements UserStore using in-memory storage
type MemoryUserStore struct {
	mu    sync.RWMutex
	users map[string]*User
}

// NewMemoryUserStore creates a new in-memory user store
func NewMemoryUserStore() *MemoryUserStore {
	store := &MemoryUserStore{
		users: make(map[string]*User),
	}

	// Create default admin user (admin@ujiscan.local / admin123)
	adminHash := PasswordHash("admin123")
	store.users["admin"] = &User{
		ID:           "admin",
		Email:        "admin@ujiscan.local",
		Name:         "Administrator",
		Role:         RoleAdmin,
		PasswordHash: adminHash,
		Enabled:      true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	return store
}

// CreateUser creates a new user
func (s *MemoryUserStore) CreateUser(email, name, passwordHash string, role Role) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if user already exists
	for _, u := range s.users {
		if u.Email == email {
			return nil, fmt.Errorf("user with email %s already exists", email)
		}
	}

	id := fmt.Sprintf("user_%d", time.Now().UnixNano())
	user := &User{
		ID:           id,
		Email:        email,
		Name:         name,
		Role:         role,
		PasswordHash: passwordHash,
		Enabled:      true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	s.users[id] = user
	return user, nil
}

// GetUser gets a user by ID
func (s *MemoryUserStore) GetUser(id string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.users[id]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}

	return user, nil
}

// GetUserByEmail gets a user by email
func (s *MemoryUserStore) GetUserByEmail(email string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, u := range s.users {
		if u.Email == email {
			return u, nil
		}
	}

	return nil, fmt.Errorf("user not found")
}

// UpdateUser updates a user
func (s *MemoryUserStore) UpdateUser(user *User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[user.ID]; !exists {
		return fmt.Errorf("user not found")
	}

	user.UpdatedAt = time.Now()
	s.users[user.ID] = user
	return nil
}

// DeleteUser deletes a user
func (s *MemoryUserStore) DeleteUser(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[id]; !exists {
		return fmt.Errorf("user not found")
	}

	delete(s.users, id)
	return nil
}

// ListUsers lists all users
func (s *MemoryUserStore) ListUsers() ([]*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	users := make([]*User, 0, len(s.users))
	for _, u := range s.users {
		users = append(users, u)
	}

	return users, nil
}

// UpdateLastLogin updates last login time
func (s *MemoryUserStore) UpdateLastLogin(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, exists := s.users[id]
	if !exists {
		return fmt.Errorf("user not found")
	}

	now := time.Now()
	user.LastLogin = &now
	user.UpdatedAt = now
	s.users[id] = user

	return nil
}

// MemorySessionStore implements SessionStore using in-memory storage
type MemorySessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

// NewMemorySessionStore creates a new in-memory session store
func NewMemorySessionStore() *MemorySessionStore {
	return &MemorySessionStore{
		sessions: make(map[string]*Session),
	}
}

// CreateSession creates a new session
func (s *MemorySessionStore) CreateSession(userID, token, ipAddress, userAgent string, expiresAt time.Time) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := fmt.Sprintf("sess_%d", time.Now().UnixNano())
	session := &Session{
		ID:        id,
		UserID:    userID,
		Token:     token,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
		IPAddress: ipAddress,
		UserAgent: userAgent,
	}

	s.sessions[id] = session
	return session, nil
}

// GetSession gets a session by ID
func (s *MemorySessionStore) GetSession(sessionID string) (*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}

	return session, nil
}

// ValidateSession validates a session
func (s *MemorySessionStore) ValidateSession(sessionID string) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}

	if time.Now().After(session.ExpiresAt) {
		delete(s.sessions, sessionID)
		return nil, fmt.Errorf("session expired")
	}

	// Update last used
	session.LastUsed = time.Now()
	s.sessions[sessionID] = session

	return session, nil
}

// InvalidateSession invalidates a session
func (s *MemorySessionStore) InvalidateSession(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.sessions[sessionID]; !exists {
		return fmt.Errorf("session not found")
	}

	delete(s.sessions, sessionID)
	return nil
}

// InvalidateAllUserSessions invalidates all sessions for a user
func (s *MemorySessionStore) InvalidateAllUserSessions(userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for sessionID, session := range s.sessions {
		if session.UserID == userID {
			delete(s.sessions, sessionID)
		}
	}

	return nil
}

// ListUserSessions lists all sessions for a user
func (s *MemorySessionStore) ListUserSessions(userID string) ([]*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sessions := make([]*Session, 0)
	for _, session := range s.sessions {
		if session.UserID == userID {
			sessions = append(sessions, session)
		}
	}

	return sessions, nil
}
