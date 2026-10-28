package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/rudrapatel001/taskmaster/backend/internal/models"
	"github.com/rudrapatel001/taskmaster/backend/internal/permissions"
	"github.com/rudrapatel001/taskmaster/backend/internal/repository"
)

// ErrCommentNotFound means no such comment exists on the task.
var ErrCommentNotFound = errors.New("comment not found")

// CommentStore is what CommentService needs from the data layer.
type CommentStore interface {
	List(ctx context.Context, workspaceID, projectID string, number int) ([]models.Comment, error)
	Create(ctx context.Context, workspaceID, projectID string, number int, authorID, body string) (*models.Comment, error)
	Get(ctx context.Context, workspaceID, projectID string, number int, commentID string) (*models.Comment, error)
	Update(ctx context.Context, workspaceID, projectID string, number int, commentID, body string) (*models.Comment, error)
	Delete(ctx context.Context, workspaceID, projectID string, number int, commentID string) error
}

type CommentService struct {
	store CommentStore
}

func NewCommentService(store CommentStore) *CommentService {
	return &CommentService{store: store}
}

func (s *CommentService) List(ctx context.Context, workspaceID, projectID string, number int) ([]models.Comment, error) {
	comments, err := s.store.List(ctx, workspaceID, projectID, number)
	if err != nil {
		return nil, commentError(err)
	}
	return comments, nil
}

// Create adds a comment written by the actor. The body is already validated.
func (s *CommentService) Create(ctx context.Context, actor Actor, projectID string, number int, body string) (*models.Comment, error) {
	c, err := s.store.Create(ctx, actor.WorkspaceID, projectID, number, actor.UserID, body)
	if err != nil {
		return nil, commentError(err)
	}
	return c, nil
}

// Update changes a comment, if the actor may: only the author, only the words.
func (s *CommentService) Update(ctx context.Context, actor Actor, projectID string, number int, commentID, body string) (*models.Comment, error) {
	existing, err := s.store.Get(ctx, actor.WorkspaceID, projectID, number, commentID)
	if err != nil {
		return nil, commentError(err)
	}
	if !permissions.CanEditComment(actor.Role, isAuthor(existing, actor.UserID)) {
		return nil, ErrForbidden
	}

	updated, err := s.store.Update(ctx, actor.WorkspaceID, projectID, number, commentID, body)
	if err != nil {
		return nil, commentError(err)
	}
	return updated, nil
}

// Delete removes a comment, if the actor may: its author, or an admin or owner.
func (s *CommentService) Delete(ctx context.Context, actor Actor, projectID string, number int, commentID string) error {
	existing, err := s.store.Get(ctx, actor.WorkspaceID, projectID, number, commentID)
	if err != nil {
		return commentError(err)
	}
	if !permissions.CanDeleteComment(actor.Role, isAuthor(existing, actor.UserID)) {
		return ErrForbidden
	}

	return commentError(s.store.Delete(ctx, actor.WorkspaceID, projectID, number, commentID))
}

func isAuthor(c *models.Comment, userID string) bool {
	return c.AuthorID != nil && strings.EqualFold(*c.AuthorID, userID)
}

// commentError turns repository errors into the service's own errors. It
// returns nil for a nil error.
func commentError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, repository.ErrNoSuchTask):
		return ErrTaskNotFound
	case errors.Is(err, repository.ErrNotFound):
		return ErrCommentNotFound
	default:
		return fmt.Errorf("comment operation failed: %w", err)
	}
}
