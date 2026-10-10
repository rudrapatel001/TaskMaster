package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/rudrapatel001/taskmaster/backend/internal/auth"
	"github.com/rudrapatel001/taskmaster/backend/internal/models"
	"github.com/rudrapatel001/taskmaster/backend/internal/repository"
)

const (
	// bcryptCost controls how slow hashing is. Each +1 doubles the time.
	// 12 takes roughly a quarter of a second on a modern machine.
	bcryptCost = 12

	// bcrypt only uses the first 72 bytes of a password.
	maxPasswordBytes = 72

	// How long a refresh token (and so a login session) lasts.
	refreshTokenTTL = 30 * 24 * time.Hour
)

var (
	// ErrEmailTaken is returned by Register when the email is already in use.
	ErrEmailTaken = errors.New("email already in use")

	// ErrInvalidCredentials is returned by Login for a wrong email OR a wrong
	// password. It is deliberately the same error for both.
	ErrInvalidCredentials = errors.New("invalid email or password")
)

// dummyHash is compared against when a login uses an unknown email, so that
// unknown emails take as long to reject as wrong passwords do.
var dummyHash = func() []byte {
	h, err := bcrypt.GenerateFromPassword([]byte("not-a-real-password"), bcryptCost)
	if err != nil {
		panic(err) // only possible if the cost or password is invalid: a programming error
	}
	return h
}()

// UserStore is what AuthService needs from the user data. It is defined here,
// where it is used, so tests can pass in a fake instead of a real database.
type UserStore interface {
	Create(ctx context.Context, email, name, passwordHash string) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
}

// RefreshTokenStore is what AuthService needs to keep refresh tokens.
type RefreshTokenStore interface {
	Create(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error
}

type AuthService struct {
	users         UserStore
	refreshTokens RefreshTokenStore
	tokens        *auth.TokenManager
}

func NewAuthService(users UserStore, refreshTokens RefreshTokenStore, tokens *auth.TokenManager) *AuthService {
	return &AuthService{users: users, refreshTokens: refreshTokens, tokens: tokens}
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

type LoginInput struct {
	Email    string // already trimmed and lowercased by the caller
	Password string
}

// Session is everything a successful login produces.
type Session struct {
	User                  *models.User
	AccessToken           string
	AccessTokenExpiresAt  time.Time
	RefreshToken          string // the plain token: goes to the client, never stored
	RefreshTokenExpiresAt time.Time
}

// Login checks the credentials and starts a new session.
func (s *AuthService) Login(ctx context.Context, in LoginInput) (*Session, error) {
	// No real password can be longer than this, so don't even try.
	if len(in.Password) > maxPasswordBytes {
		return nil, ErrInvalidCredentials
	}

	user, err := s.users.GetByEmail(ctx, in.Email)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			// Do the same slow work as a real check, so timing doesn't reveal
			// which emails have accounts.
			_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(in.Password))
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("looking up user: %w", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(in.Password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("comparing password: %w", err)
	}

	return s.newSession(ctx, user)
}

// newSession issues an access token and a refresh token for user.
func (s *AuthService) newSession(ctx context.Context, user *models.User) (*Session, error) {
	accessToken, accessExpiresAt, err := s.tokens.NewAccessToken(user.ID)
	if err != nil {
		return nil, err
	}

	refreshToken, refreshHash, err := auth.NewRefreshToken()
	if err != nil {
		return nil, err
	}
	refreshExpiresAt := time.Now().Add(refreshTokenTTL)

	if err := s.refreshTokens.Create(ctx, user.ID, refreshHash, refreshExpiresAt); err != nil {
		return nil, fmt.Errorf("storing refresh token: %w", err)
	}

	return &Session{
		User:                  user,
		AccessToken:           accessToken,
		AccessTokenExpiresAt:  accessExpiresAt,
		RefreshToken:          refreshToken,
		RefreshTokenExpiresAt: refreshExpiresAt,
	}, nil
}
