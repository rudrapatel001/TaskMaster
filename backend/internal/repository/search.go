package repository

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rudrapatel001/taskmaster/backend/internal/models"
)

// A task identifier as people type it: a project key, a dash, a number ("web-12").
var identifierPattern = regexp.MustCompile(`^([A-Za-z][A-Za-z0-9]{1,9})-([0-9]{1,9})$`)

// parseIdentifier recognises a task identifier such as "web-12" and returns
// its project key (in capitals) and number.
func parseIdentifier(q string) (key string, number int, ok bool) {
	m := identifierPattern.FindStringSubmatch(q)
	if m == nil {
		return "", 0, false
	}
	n, err := strconv.Atoi(m[2])
	if err != nil {
		return "", 0, false
	}
	return strings.ToUpper(m[1]), n, true
}

type SearchRepository struct {
	db *pgxpool.Pool
}

func NewSearchRepository(db *pgxpool.Pool) *SearchRepository {
	return &SearchRepository{db: db}
}

// Tasks finds a workspace's tasks whose title contains q (ignoring capitals),
// or whose identifier is exactly q ("web-12"). An exact identifier comes first,
// then the most recently changed. Deleted tasks are never found.
func (r *SearchRepository) Tasks(ctx context.Context, workspaceID, q string, limit int) ([]models.TaskSearchResult, error) {
	// For a non-identifier these stay nil, which the query treats as "no match".
	var key *string
	var number *int
	if k, n, ok := parseIdentifier(q); ok {
		key, number = &k, &n
	}

	rows, err := r.db.Query(ctx, `
		SELECT p.key || '-' || t.number, t.number, t.title, p.key, p.name
		FROM tasks t
		JOIN projects p ON p.id = t.project_id
		WHERE p.workspace_id = $1 AND t.deleted_at IS NULL
		  AND (t.title ILIKE $2 OR (p.key = $3::text AND t.number = $4::int))
		ORDER BY COALESCE(p.key = $3::text AND t.number = $4::int, false) DESC, t.updated_at DESC, t.id
		LIMIT $5`,
		workspaceID, "%"+likeEscaper.Replace(q)+"%", key, number, limit)
	if err != nil {
		return nil, fmt.Errorf("searching tasks: %w", err)
	}
	defer rows.Close()

	results := []models.TaskSearchResult{}
	for rows.Next() {
		var t models.TaskSearchResult
		if err := rows.Scan(&t.Identifier, &t.Number, &t.Title, &t.ProjectKey, &t.ProjectName); err != nil {
			return nil, fmt.Errorf("reading search row: %w", err)
		}
		results = append(results, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading search rows: %w", err)
	}
	return results, nil
}
