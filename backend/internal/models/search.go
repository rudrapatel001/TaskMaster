package models

// TaskSearchResult is one task found by a search.
type TaskSearchResult struct {
	Identifier  string `json:"identifier"` // "WEB-12"
	Number      int    `json:"number"`
	Title       string `json:"title"`
	ProjectKey  string `json:"project_key"`
	ProjectName string `json:"project_name"`
}
