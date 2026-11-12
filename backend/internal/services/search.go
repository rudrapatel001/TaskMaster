package services

import (
	"context"
	"fmt"

	"github.com/rudrapatel001/taskmaster/backend/internal/models"
)

// SearchStore is what SearchService needs from the data layer.
type SearchStore interface {
	Tasks(ctx context.Context, workspaceID, q string, limit int) ([]models.TaskSearchResult, error)
}

type SearchService struct {
	store SearchStore
}

func NewSearchService(store SearchStore) *SearchService {
	return &SearchService{store: store}
}

// Tasks searches a workspace's tasks. The query is already validated.
func (s *SearchService) Tasks(ctx context.Context, workspaceID, q string, limit int) ([]models.TaskSearchResult, error) {
	results, err := s.store.Tasks(ctx, workspaceID, q, limit)
	if err != nil {
		return nil, fmt.Errorf("searching tasks: %w", err)
	}
	return results, nil
}
