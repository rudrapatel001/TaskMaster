package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/rudrapatel001/taskmaster/backend/internal/models"
	"github.com/rudrapatel001/taskmaster/backend/internal/repository"
)

// ErrUserNotFound is returned when the user does not exist (any more).
var ErrUserNotFound = errors.New("user not found")

// ProfileStore is what UserService needs from the data layer.
type ProfileStore interface {
	GetByID(ctx context.Context, id string) (*models.User, error)
	UpdateName(ctx context.Context, id, name string) (*models.User, error)
}

type UserService struct {
	users ProfileStore
}

func NewUserService(users ProfileStore) *UserService {
	return &UserService{users: users}
}

// Get returns one user's profile.
func (s *UserService) Get(ctx context.Context, id string) (*models.User, error) {
	user, err := s.users.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("getting user: %w", err)
	}
	return user, nil
}

// UpdateName changes a user's display name. The caller has already validated it.
func (s *UserService) UpdateName(ctx context.Context, id, name string) (*models.User, error) {
	user, err := s.users.UpdateName(ctx, id, name)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("updating user: %w", err)
	}
	return user, nil
}
