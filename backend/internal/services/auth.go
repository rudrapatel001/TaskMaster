package services

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"

	"github.com/rudrapatel001/taskmaster/backend/internal/models"
	"github.com/rudrapatel001/taskmaster/backend/internal/repository"
)

// bcryptCost controls how slow hashing is. Each +1 doubles the time.
// 12 takes roughly a quarter of a second on a modern machine.
const bcryptCost = 12

// ErrEmailTaken is returned by Register when the email is already in use.
var ErrEmailTaken = errors.New("email already in use")

// UserStore is what AuthService needs from the data layer. It is defined here,
// where it is used, so tests can pass in a fake instead of a real database.
type UserStore interface {
	Create(ctx context.Context, email, name, passwordHash string) (*models.User, error)
}

type AuthService struct {
	users UserStore
}

func NewAuthService(users UserStore) *AuthService {
	return &AuthService{users: users}
}

type RegisterInput struct {
	Email    string // already trimmed and lowercased by the caller
	Name     string
	Password string
}

// Register hashes the password and creates the account.
func (s *AuthService) Register(ctx context.Context, in RegisterInput) (*models.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	user, err := s.users.Create(ctx, in.Email, in.Name, string(hash))
	if err != nil {
		if errors.Is(err, repository.ErrEmailTaken) {
			return nil, ErrEmailTaken
		}
		return nil, fmt.Errorf("creating user: %w", err)
	}

	return user, nil
}
