package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID           string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

type Session struct {
	ID        string
	UserID    string
	Token     string
	ExpiresAt time.Time
	CreatedAt time.Time
}

func (s *Store) CreateUser(ctx context.Context, email, password string) (*User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || len(password) < 6 {
		return nil, fmt.Errorf("email and password (min 6 chars) required")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	user := &User{
		ID:           uuid.NewString(),
		Email:        email,
		PasswordHash: string(hash),
		CreatedAt:    now,
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO users (id, email, password_hash, created_at) VALUES (?, ?, ?, ?)`,
		user.ID, user.Email, user.PasswordHash, now.Format(time.RFC3339),
	)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return nil, fmt.Errorf("bu e-posta zaten kayıtlı")
		}
		return nil, err
	}
	return user, nil
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	row := s.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash, created_at FROM users WHERE email = ?`, email,
	)
	var u User
	var created string
	if err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &created); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	u.CreatedAt, _ = time.Parse(time.RFC3339, created)
	return &u, nil
}

func (s *Store) AuthenticateUser(ctx context.Context, email, password string) (*User, error) {
	u, err := s.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, fmt.Errorf("e-posta veya şifre hatalı")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("e-posta veya şifre hatalı")
	}
	return u, nil
}

func (s *Store) CreateSession(ctx context.Context, userID string, ttl time.Duration) (*Session, error) {
	if ttl <= 0 {
		ttl = 30 * 24 * time.Hour
	}
	now := time.Now().UTC()
	sess := &Session{
		ID:        uuid.NewString(),
		UserID:    userID,
		Token:     "ugs_" + uuid.NewString(),
		ExpiresAt: now.Add(ttl),
		CreatedAt: now,
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO user_sessions (id, user_id, token, expires_at, created_at) VALUES (?, ?, ?, ?, ?)`,
		sess.ID, sess.UserID, sess.Token, sess.ExpiresAt.Format(time.RFC3339), sess.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return nil, err
	}
	return sess, nil
}

func (s *Store) GetUserBySessionToken(ctx context.Context, token string) (*User, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, nil
	}
	row := s.db.QueryRowContext(ctx, `
		SELECT u.id, u.email, u.password_hash, u.created_at, s.expires_at
		FROM user_sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token = ?`, token)
	var u User
	var created, expires string
	if err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &created, &expires); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	exp, _ := time.Parse(time.RFC3339, expires)
	if time.Now().UTC().After(exp) {
		_, _ = s.db.ExecContext(ctx, `DELETE FROM user_sessions WHERE token = ?`, token)
		return nil, nil
	}
	u.CreatedAt, _ = time.Parse(time.RFC3339, created)
	return &u, nil
}

func (s *Store) DeleteSession(ctx context.Context, token string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM user_sessions WHERE token = ?`, token)
	return err
}

func (s *Store) ValidateAuthToken(ctx context.Context, token string) bool {
	token = strings.TrimSpace(token)
	if token == "" {
		return false
	}
	if s.ValidateToken(ctx, token) {
		return true
	}
	u, err := s.GetUserBySessionToken(ctx, token)
	return err == nil && u != nil
}
