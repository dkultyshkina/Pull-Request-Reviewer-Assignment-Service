package entity

type User struct {
    ID       string `db:"user_id" json:"user_id"`
    Username string `db:"username" json:"username"`
    IsActive bool   `db:"is_active" json:"is_active"`
    TeamName string `db:"team_name,omitempty" json:"team_name,omitempty"`
}

type Team struct {
	ID   string `db:"team_id"`
	Name string `db:"team_name"`
}

type PullRequest struct {
	ID                string  `db:"pull_request_id"`
	Title             string  `db:"pull_request_name"`
	AuthorID          string  `db:"author_id"`
	Status            string  `db:"status"`
	AssignedReviewers []User  `db:"-"`
	CreatedAt         *string `db:"created_at,omitempty"`
	MergedAt          *string `db:"merged_at,omitempty"`
}

type Stats struct {
    UserAssignmentCounts []UserAssignmentCount `json:"user_assignment_counts"`
    PRAssignmentCounts   []PRAssignmentCount   `json:"pr_assignment_counts"`
    TotalAssignments     int                   `json:"total_assignments"`
}

type UserAssignmentCount struct {
    UserID  string `json:"user_id" db:"user_id"`
    Username string `json:"username" db:"username"`
    Count   int    `json:"count" db:"assignment_count"`
}

type PRAssignmentCount struct {
    PRID   string `json:"pull_request_id" db:"pull_request_id"`
    Title  string `json:"pull_request_name" db:"pull_request_name"`
    Count  int    `json:"count" db:"assignment_count"`
}
