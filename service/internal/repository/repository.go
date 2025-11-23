package repository

import (
	"database/sql"

	"service/internal/entity"
)

type Repository interface {
	CreateTeam(team *entity.Team, members []entity.User) error
	GetTeam(teamName string) (*entity.Team, []entity.User, error)
	SetUserActive(userID string, isActive bool) (*entity.User, error)
	GetUserReviewPRs(userID string) ([]entity.PullRequest, error)
	CreatePR(pr *entity.PullRequest, reviewerIDs []string) error
	MergePR(prID string) (*entity.PullRequest, error)
	GetPR(prID string) (*entity.PullRequest, error)
	GetPRReviewers(prID string) ([]entity.User, error)
	ReassignReviewer(prID, oldUserID string) (string, error)
	GetCandidateReviewers(authorID string, limit int) ([]string, error)
	GetStats() (*entity.Stats, error)
}

type RepositoryImpl struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &RepositoryImpl{db: db}
}

func (r *RepositoryImpl) CreateTeam(team *entity.Team, members []entity.User) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var existingTeamID string
	err = tx.QueryRow("SELECT team_id FROM teams WHERE LOWER(team_name) = LOWER($1)", team.Name).Scan(&existingTeamID)
	if err == nil {
		return entity.ErrTeamExists
	} else if err != sql.ErrNoRows {
		return err
	}
	err = tx.QueryRow(
		"INSERT INTO teams (team_name) VALUES ($1) RETURNING team_id",
		team.Name,
	).Scan(&team.ID)
	if err != nil {
		return err
	}
	for _, member := range members {
		_, err = tx.Exec(`
			INSERT INTO users (user_id, username, is_active) 
			VALUES ($1, $2, $3)
			ON CONFLICT (user_id) DO UPDATE SET 
				username = EXCLUDED.username,
				is_active = EXCLUDED.is_active
		`, member.ID, member.Username, member.IsActive)
		if err != nil {
			return err
		}
		_, err = tx.Exec(
			"INSERT INTO team_members (team_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING",
			team.ID, member.ID,
		)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *RepositoryImpl) GetTeam(teamName string) (*entity.Team, []entity.User, error) {
	var team entity.Team
	err := r.db.QueryRow(
		"SELECT team_id, team_name FROM teams WHERE LOWER(team_name) = LOWER($1)",
		teamName,
	).Scan(&team.ID, &team.Name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, entity.ErrNotFound
		}
		return nil, nil, err
	}
	rows, err := r.db.Query(`
		SELECT u.user_id, u.username, u.is_active 
		FROM users u
		JOIN team_members tm ON u.user_id = tm.user_id
		WHERE tm.team_id = $1
	`, team.ID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var members []entity.User
	for rows.Next() {
		var member entity.User
		err := rows.Scan(&member.ID, &member.Username, &member.IsActive)
		if err != nil {
			return nil, nil, err
		}
		members = append(members, member)
	}
	return &team, members, nil
}

func (r *RepositoryImpl) SetUserActive(userID string, isActive bool) (*entity.User, error) {
	var user entity.User
	err := r.db.QueryRow(`
		UPDATE users SET is_active = $1 
		WHERE user_id = $2 
		RETURNING user_id, username, is_active
	`, isActive, userID).Scan(&user.ID, &user.Username, &user.IsActive)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, entity.ErrNotFound
		}
		return nil, err
	}
	err = r.db.QueryRow(`
		SELECT t.team_name 
		FROM teams t
		JOIN team_members tm ON t.team_id = tm.team_id
		WHERE tm.user_id = $1
	`, userID).Scan(&user.TeamName)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	return &user, nil
}

func (r *RepositoryImpl) GetUserReviewPRs(userID string) ([]entity.PullRequest, error) {
	rows, err := r.db.Query(`
		SELECT pr.pull_request_id, pr.pull_request_name, pr.author_id, pr.status
		FROM pull_requests pr
		JOIN reviewers r ON pr.pull_request_id = r.pull_request_id
		WHERE r.user_id = $1 AND r.is_active = true
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var prs []entity.PullRequest
	for rows.Next() {
		var pr entity.PullRequest
		err := rows.Scan(&pr.ID, &pr.Title, &pr.AuthorID, &pr.Status)
		if err != nil {
			return nil, err
		}
		prs = append(prs, pr)
	}
	return prs, nil
}

func (r *RepositoryImpl) CreatePR(pr *entity.PullRequest, reviewerIDs []string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var existingPRID string
	err = tx.QueryRow("SELECT pull_request_id FROM pull_requests WHERE pull_request_id = $1", pr.ID).Scan(&existingPRID)
	if err == nil {
		return entity.ErrPRExists
	} else if err != sql.ErrNoRows {
		return err
	}
	_, err = tx.Exec(`
		INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status)
		VALUES ($1, $2, $3, $4)
	`, pr.ID, pr.Title, pr.AuthorID, "OPEN")
	if err != nil {
		return err
	}
	for _, reviewerID := range reviewerIDs {
		_, err = tx.Exec(`
			INSERT INTO reviewers (pull_request_id, user_id, is_active)
			VALUES ($1, $2, true)
		`, pr.ID, reviewerID)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *RepositoryImpl) MergePR(prID string) (*entity.PullRequest, error) {
	var pr entity.PullRequest
	err := r.db.QueryRow(`
		UPDATE pull_requests 
		SET status = 'MERGED', merged_at = CURRENT_TIMESTAMP
		WHERE pull_request_id = $1 AND status != 'MERGED'
		RETURNING pull_request_id, pull_request_name, author_id, status, created_at, merged_at
	`, prID).Scan(&pr.ID, &pr.Title, &pr.AuthorID, &pr.Status, &pr.CreatedAt, &pr.MergedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			var status string
			err = r.db.QueryRow("SELECT status FROM pull_requests WHERE pull_request_id = $1", prID).Scan(&status)
			if err == nil && status == "MERGED" {
				return r.GetPR(prID)
			}
			return nil, entity.ErrNotFound
		}
		return nil, err
	}
	return &pr, nil
}

func (r *RepositoryImpl) GetPR(prID string) (*entity.PullRequest, error) {
	var pr entity.PullRequest
	err := r.db.QueryRow(`
		SELECT pull_request_id, pull_request_name, author_id, status, created_at, merged_at
		FROM pull_requests 
		WHERE pull_request_id = $1
	`, prID).Scan(&pr.ID, &pr.Title, &pr.AuthorID, &pr.Status, &pr.CreatedAt, &pr.MergedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, entity.ErrNotFound
		}
		return nil, err
	}
	reviewers, err := r.GetPRReviewers(prID)
	if err != nil {
		return nil, err
	}
	pr.AssignedReviewers = reviewers
	return &pr, nil
}

func (r *RepositoryImpl) GetPRReviewers(prID string) ([]entity.User, error) {
	rows, err := r.db.Query(`
		SELECT u.user_id, u.username, u.is_active
		FROM users u
		JOIN reviewers r ON u.user_id = r.user_id
		WHERE r.pull_request_id = $1 AND r.is_active = true
	`, prID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var reviewers []entity.User
	for rows.Next() {
		var user entity.User
		err := rows.Scan(&user.ID, &user.Username, &user.IsActive)
		if err != nil {
			return nil, err
		}
		reviewers = append(reviewers, user)
	}
	return reviewers, nil
}

func (r *RepositoryImpl) ReassignReviewer(prID, oldUserID string) (string, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	var status string
	err = tx.QueryRow("SELECT status FROM pull_requests WHERE pull_request_id = $1", prID).Scan(&status)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", entity.ErrNotFound
		}
		return "", err
	}
	if status == "MERGED" {
		return "", entity.ErrPRMerged
	}
	var isAssigned bool
	err = tx.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM reviewers 
			WHERE pull_request_id = $1 AND user_id = $2 AND is_active = true
		)
	`, prID, oldUserID).Scan(&isAssigned)
	if err != nil {
		return "", err
	}
	if !isAssigned {
		return "", entity.ErrNotAssigned
	}
	var authorID string
	var teamID string
	err = tx.QueryRow(`
		SELECT pr.author_id, t.team_id
		FROM pull_requests pr
		JOIN team_members tm ON pr.author_id = tm.user_id
		JOIN teams t ON tm.team_id = t.team_id
		WHERE pr.pull_request_id = $1
	`, prID).Scan(&authorID, &teamID)
	if err != nil {
		return "", err
	}
	var newUserID string
	err = tx.QueryRow(`
		SELECT u.user_id 
		FROM users u
		JOIN team_members tm ON u.user_id = tm.user_id
		WHERE tm.team_id = $1 
		AND u.user_id != $2 
		AND u.user_id != $3
		AND u.is_active = true
		AND u.user_id NOT IN (
			SELECT user_id FROM reviewers 
			WHERE pull_request_id = $4 AND is_active = true
		)
		LIMIT 1
	`, teamID, authorID, oldUserID, prID).Scan(&newUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", entity.ErrNoCandidate
		}
		return "", err
	}
	_, err = tx.Exec(`
		UPDATE reviewers SET is_active = false 
		WHERE pull_request_id = $1 AND user_id = $2
	`, prID, oldUserID)
	if err != nil {
		return "", err
	}
	_, err = tx.Exec(`
		INSERT INTO reviewers (pull_request_id, user_id, is_active)
		VALUES ($1, $2, true)
	`, prID, newUserID)
	if err != nil {
		return "", err
	}
	return newUserID, tx.Commit()
}
func (r *RepositoryImpl) GetCandidateReviewers(authorID string, limit int) ([]string, error) {
    rows, err := r.db.Query(`
        SELECT 
            u.user_id,
            COUNT(r.user_id) as current_assignments
        FROM users u
        JOIN team_members tm ON u.user_id = tm.user_id
        JOIN team_members tm_author ON tm.team_id = tm_author.team_id
        LEFT JOIN reviewers r ON u.user_id = r.user_id AND r.is_active = true
        LEFT JOIN pull_requests pr ON r.pull_request_id = pr.pull_request_id AND pr.status = 'OPEN'
        WHERE tm_author.user_id = $1 
            AND u.user_id != $1
            AND u.is_active = true
        GROUP BY u.user_id
        ORDER BY current_assignments ASC, u.user_id
        LIMIT $2
    `, authorID, limit)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var userIDs []string
    for rows.Next() {
        var userID string
        var currentAssignments int
        err := rows.Scan(&userID, &currentAssignments)
        if err != nil {
            return nil, err
        }
        userIDs = append(userIDs, userID)
    }
    return userIDs, nil
}

func (r *RepositoryImpl) GetStats() (*entity.Stats, error) {
    stats := &entity.Stats{}
    userRows, err := r.db.Query(`
        SELECT u.user_id, u.username, COUNT(r.user_id) as assignment_count
        FROM users u
        LEFT JOIN reviewers r ON u.user_id = r.user_id AND r.is_active = true
        GROUP BY u.user_id, u.username
        ORDER BY assignment_count DESC
    `)
    if err != nil {
        return nil, err
    }
    defer userRows.Close()
    for userRows.Next() {
        var userStat entity.UserAssignmentCount
        err := userRows.Scan(&userStat.UserID, &userStat.Username, &userStat.Count)
        if err != nil {
            return nil, err
        }
        stats.UserAssignmentCounts = append(stats.UserAssignmentCounts, userStat)
        stats.TotalAssignments += userStat.Count
    }
    prRows, err := r.db.Query(`
        SELECT pr.pull_request_id, pr.pull_request_name, COUNT(r.user_id) as assignment_count
        FROM pull_requests pr
        LEFT JOIN reviewers r ON pr.pull_request_id = r.pull_request_id AND r.is_active = true
        GROUP BY pr.pull_request_id, pr.pull_request_name
        ORDER BY assignment_count DESC
    `)
    if err != nil {
        return nil, err
    }
    defer prRows.Close()
    for prRows.Next() {
        var prStat entity.PRAssignmentCount
        err := prRows.Scan(&prStat.PRID, &prStat.Title, &prStat.Count)
        if err != nil {
            return nil, err
        }
        stats.PRAssignmentCounts = append(stats.PRAssignmentCounts, prStat)
    }
    return stats, nil
}

