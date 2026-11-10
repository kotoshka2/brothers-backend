package application

import (
	"context"
	"errors"
	"testing"
	"time"
)

type memorySessions struct {
	sessions map[string]string
}

func newMemorySessions() *memorySessions {
	return &memorySessions{sessions: make(map[string]string)}
}

func (m *memorySessions) CreateRefreshSession(_ context.Context, hash, username string, _ time.Time) error {
	m.sessions[hash] = username
	return nil
}

func (m *memorySessions) RotateRefreshSession(_ context.Context, oldHash, newHash, username string, _ time.Time) error {
	if m.sessions[oldHash] != username {
		return ErrUnauthorized
	}
	delete(m.sessions, oldHash)
	m.sessions[newHash] = username
	return nil
}

func (m *memorySessions) RevokeRefreshSession(_ context.Context, hash, username string) error {
	if m.sessions[hash] != username {
		return ErrUnauthorized
	}
	delete(m.sessions, hash)
	return nil
}

func TestAuthRefreshRotationAndLogout(t *testing.T) {
	passwordHash, err := HashPassword("test-password-long-enough")
	if err != nil {
		t.Fatal(err)
	}
	sessions := newMemorySessions()
	service := NewAuthService("admin", passwordHash, []byte("01234567890123456789012345678901"), sessions)

	tokens, err := service.Login(context.Background(), "admin", "test-password-long-enough")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if err := service.ValidateAccessToken(tokens.AccessToken); err != nil {
		t.Fatalf("validate access token: %v", err)
	}

	rotated, err := service.Refresh(context.Background(), tokens.RefreshToken)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if _, err := service.Refresh(context.Background(), tokens.RefreshToken); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected old refresh token to be rejected, got %v", err)
	}
	if err := service.Logout(context.Background(), rotated.RefreshToken); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if _, err := service.Refresh(context.Background(), rotated.RefreshToken); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected logged-out refresh token to be rejected, got %v", err)
	}
}

func TestAuthRejectsInvalidCredentialsAndTokenType(t *testing.T) {
	passwordHash, err := HashPassword("test-password-long-enough")
	if err != nil {
		t.Fatal(err)
	}
	service := NewAuthService("admin", passwordHash, []byte("01234567890123456789012345678901"), newMemorySessions())
	if _, err := service.Login(context.Background(), "admin", "wrong-password"); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected invalid credentials to be rejected, got %v", err)
	}
	tokens, err := service.Login(context.Background(), "admin", "test-password-long-enough")
	if err != nil {
		t.Fatal(err)
	}
	if err := service.ValidateAccessToken(tokens.RefreshToken); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected refresh token to be rejected as access token, got %v", err)
	}
}
