package auth

import (
	"sync"
	"testing"
	"time"
)

func TestSimpleTokenManagerConcurrentAccess(t *testing.T) {
	manager := NewSimpleTokenManager("test-secret")
	user := &User{ID: "concurrent-user", Email: "test@example.com", Name: "Tester", Role: RoleAdmin}

	const workers = 16
	const tokensPerWorker = 100
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(workers)

	for range workers {
		go func() {
			defer wg.Done()
			<-start
			for range tokensPerWorker {
				token, err := manager.GenerateToken(user, time.Hour)
				if err != nil {
					t.Errorf("GenerateToken() error = %v", err)
					return
				}
				if _, err := manager.ValidateToken(token); err != nil {
					t.Errorf("ValidateToken() error = %v", err)
					return
				}
			}
		}()
	}

	close(start)
	wg.Wait()
}
