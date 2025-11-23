package repository_test

import (
	"database/sql"
	"testing"
	"errors"

	_ "github.com/lib/pq"

	"service/internal/repository"
	"service/internal/entity"
)

func setupTestDB(t *testing.T) *sql.DB {
	connStr := "postgres://reviewer_user:password@test-db:5432/reviewer?sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Skipf("Skipping test - cannot connect to test DB: %v", err)
	}
	_, err = db.Exec(`
		DROP TABLE IF EXISTS reviewers, team_members, pull_requests, users, teams CASCADE;
		
		CREATE TABLE teams (
			team_id SERIAL PRIMARY KEY,
			team_name VARCHAR(100) UNIQUE NOT NULL
		);

		CREATE TABLE users (
			user_id TEXT PRIMARY KEY,
			username VARCHAR(100) NOT NULL,
			is_active BOOLEAN NOT NULL DEFAULT true,
			created_at TIMESTAMP DEFAULT NOW()
		);

		CREATE TABLE team_members (
			team_id INT REFERENCES teams(team_id) ON DELETE CASCADE,
			user_id TEXT REFERENCES users(user_id) ON DELETE CASCADE,
			PRIMARY KEY (team_id, user_id)
		);

		CREATE TABLE pull_requests (
			pull_request_id TEXT PRIMARY KEY,
			pull_request_name VARCHAR(200) NOT NULL,
			author_id TEXT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
			status VARCHAR(20) NOT NULL DEFAULT 'OPEN' CHECK (status IN ('OPEN', 'MERGED')),
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			merged_at TIMESTAMP WITH TIME ZONE NULL
		);

		CREATE TABLE reviewers (
			pull_request_id TEXT REFERENCES pull_requests(pull_request_id) ON DELETE CASCADE,
			user_id TEXT REFERENCES users(user_id) ON DELETE CASCADE,
			is_active BOOLEAN NOT NULL DEFAULT true,
			PRIMARY KEY (pull_request_id, user_id)
		);
	`)
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	return db
}

func TestRepository_CreateTeam(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := repository.NewRepository(db)
	t.Run("create team successfully", func(t *testing.T) {
		team := &entity.Team{Name: "backend"}
		members := []entity.User{
			{ID: "u1", Username: "Alice", IsActive: true},
			{ID: "u2", Username: "Bob", IsActive: true},
		}
		err := repo.CreateTeam(team, members)
		if err != nil {
			t.Errorf("CreateTeam failed: %v", err)
		}
		if team.ID == "" {
			t.Error("Team ID should be set")
		}
	})
	t.Run("create duplicate team", func(t *testing.T) {
		team := &entity.Team{Name: "backend"}
		members := []entity.User{{ID: "u3", Username: "Charlie", IsActive: true}}
		err := repo.CreateTeam(team, members)
		if err != entity.ErrTeamExists {
			t.Errorf("Expected ErrTeamExists, got %v", err)
		}
	})
}

func TestRepository_GetTeam(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := repository.NewRepository(db)
	team := &entity.Team{Name: "frontend"}
	members := []entity.User{
		{ID: "u1", Username: "Alice", IsActive: true},
	}
	repo.CreateTeam(team, members)
	t.Run("get existing team", func(t *testing.T) {
		team, members, err := repo.GetTeam("frontend")
		if err != nil {
			t.Errorf("GetTeam failed: %v", err)
		}
		if team.Name != "frontend" {
			t.Errorf("Expected team name 'frontend', got '%s'", team.Name)
		}
		if len(members) == 0 {
			t.Error("Expected at least one team member")
		}
	})
	t.Run("get non-existent team", func(t *testing.T) {
		_, _, err := repo.GetTeam("nonexistent")
		if err != entity.ErrNotFound {
			t.Errorf("Expected ErrNotFound, got %v", err)
		}
	})
}

func TestRepository_CreateTeam_EmptyTeam(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := repository.NewRepository(db)
    team := &entity.Team{Name: "empty_team"}
    members := []entity.User{} 
    err := repo.CreateTeam(team, members)
    if err != nil {
        t.Errorf("Should create team with no members, got error: %v", err)
    }
    retrievedTeam, retrievedMembers, err := repo.GetTeam("empty_team")
    if err != nil {
        t.Errorf("Should retrieve created team: %v", err)
    }
    if retrievedTeam.Name != "empty_team" {
        t.Errorf("Expected team name 'empty_team', got %s", retrievedTeam.Name)
    }
    if len(retrievedMembers) != 0 {
        t.Errorf("Expected 0 members, got %d", len(retrievedMembers))
    }
}

func TestRepository_CreateTeam_CaseInsensitive(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := repository.NewRepository(db)
    team1 := &entity.Team{Name: "Backend"}
    err := repo.CreateTeam(team1, []entity.User{})
    if err != nil {
        t.Fatalf("Failed to create first team: %v", err)
    }
    team2 := &entity.Team{Name: "BACKEND"}
    err = repo.CreateTeam(team2, []entity.User{})
    if !errors.Is(err, entity.ErrTeamExists) {
        t.Errorf("Expected ErrTeamExists for case-insensitive duplicate, got: %v", err)
    }
}

func TestRepository_SetUserActive_UserNotExists(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := repository.NewRepository(db)
    _, err := repo.SetUserActive("nonexistent-user", true)
    if !errors.Is(err, entity.ErrNotFound) {
        t.Errorf("Expected ErrNotFound for non-existent user, got: %v", err)
    }
}

func TestRepository_SetUserActive_UserWithoutTeam(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := repository.NewRepository(db)
    _, err := db.Exec("INSERT INTO users (user_id, username, is_active) VALUES ($1, $2, $3)", 
        "lonely_user", "Lonely", true)
    if err != nil {
        t.Fatalf("Failed to setup test: %v", err)
    }
    user, err := repo.SetUserActive("lonely_user", false)
    if err != nil {
        t.Errorf("Should deactivate user without team: %v", err)
    }
    if user.TeamName != "" {
        t.Errorf("Expected empty team name for user without team, got: %s", user.TeamName)
    }
}

func TestRepository_GetPR_NotExists(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := repository.NewRepository(db)
    _, err := repo.GetPR("nonexistent-pr")
    if !errors.Is(err, entity.ErrNotFound) {
        t.Errorf("Expected ErrNotFound for non-existent PR, got: %v", err)
    }
}

func TestRepository_MergePR_AlreadyMerged(t *testing.T) {
    db := setupTestDB(t)
	defer db.Close()
	repo := repository.NewRepository(db)
    team := &entity.Team{Name: "merge-test-team"}
    members := []entity.User{
        {ID: "author1", Username: "Author1", IsActive: true},
        {ID: "reviewer1", Username: "Reviewer1", IsActive: true},
    }
    err := repo.CreateTeam(team, members)
    if err != nil {
        t.Fatalf("Failed to create team: %v", err)
    }
    pr := &entity.PullRequest{
        ID:       "pr-to-merge-twice", 
        Title:    "Test PR",
        AuthorID: "author1",
    }
    err = repo.CreatePR(pr, []string{"reviewer1"})
    if err != nil {
        t.Fatalf("Failed to create PR: %v", err)
    }
    mergedPR1, err := repo.MergePR("pr-to-merge-twice")
    if err != nil {
        t.Fatalf("Failed first merge: %v", err)
    }
    if mergedPR1.Status != "MERGED" {
        t.Errorf("First merge should set status to MERGED, got: %s", mergedPR1.Status)
    }
    mergedPR2, err := repo.MergePR("pr-to-merge-twice")
    if err != nil {
        t.Errorf("Second merge should be idempotent, got error: %v", err)
    }
    if mergedPR2.Status != "MERGED" {
        t.Errorf("Second merge should keep status MERGED, got: %s", mergedPR2.Status)
    }
}

func TestRepository_GetUserReviewPRs_MultipleReviewers(t *testing.T) {
    db := setupTestDB(t)
	defer db.Close()
	repo := repository.NewRepository(db)
    team := &entity.Team{Name: "review-test-team"}
    members := []entity.User{
        {ID: "author1", Username: "Author1", IsActive: true},
        {ID: "author2", Username: "Author2", IsActive: true},
        {ID: "reviewer1", Username: "Reviewer1", IsActive: true},
        {ID: "reviewer2", Username: "Reviewer2", IsActive: true},
        {ID: "reviewer3", Username: "Reviewer3", IsActive: true},
    }
    err := repo.CreateTeam(team, members)
    if err != nil {
        t.Fatalf("Failed to create team: %v", err)
    }
    pr1 := &entity.PullRequest{
        ID:       "pr-multi-1",
        Title:    "PR 1", 
        AuthorID: "author1",
    }
    err = repo.CreatePR(pr1, []string{"reviewer1", "reviewer2"})
    if err != nil {
        t.Fatalf("Failed to create PR1: %v", err)
    }
    pr2 := &entity.PullRequest{
        ID:       "pr-multi-2",
        Title:    "PR 2",
        AuthorID: "author2", 
    }
    err = repo.CreatePR(pr2, []string{"reviewer1", "reviewer3"})
    if err != nil {
        t.Fatalf("Failed to create PR2: %v", err)
    }
    prs, err := repo.GetUserReviewPRs("reviewer1")
    if err != nil {
        t.Errorf("Failed to get user review PRs: %v", err)
    }
    if len(prs) != 2 {
        t.Errorf("Expected 2 PRs for reviewer1, got %d", len(prs))
    }
}

func TestRepository_ReassignReviewer_ComplexScenario(t *testing.T) {
    db := setupTestDB(t)
	defer db.Close()
	repo := repository.NewRepository(db)
    team := &entity.Team{Name: "dev-team"}
    members := []entity.User{
        {ID: "author1", Username: "Author1", IsActive: true},
        {ID: "reviewer1", Username: "Reviewer1", IsActive: true},
        {ID: "reviewer2", Username: "Reviewer2", IsActive: true},
        {ID: "reviewer3", Username: "Reviewer3", IsActive: true},
    }
    err := repo.CreateTeam(team, members)
    if err != nil {
        t.Fatalf("Failed to create team: %v", err)
    }
    pr := &entity.PullRequest{
        ID:       "pr-reassign",
        Title:    "Test PR",
        AuthorID: "author1",
    }
    err = repo.CreatePR(pr, []string{"reviewer1", "reviewer2"})
    if err != nil {
        t.Fatalf("Failed to create PR: %v", err)
    }
    newReviewer, err := repo.ReassignReviewer("pr-reassign", "reviewer1")
    if err != nil {
        t.Errorf("Failed to reassign reviewer: %v", err)
    }
    if newReviewer != "reviewer3" {
        t.Errorf("Expected new reviewer to be reviewer3, got: %s", newReviewer)
    }
    updatedPR, err := repo.GetPR("pr-reassign")
    if err != nil {
        t.Errorf("Failed to get updated PR: %v", err)
    }
    reviewerIDs := make([]string, len(updatedPR.AssignedReviewers))
    for i, reviewer := range updatedPR.AssignedReviewers {
        reviewerIDs[i] = reviewer.ID
    }
    if len(reviewerIDs) != 2 {
        t.Errorf("Expected 2 reviewers, got %d", len(reviewerIDs))
    }
    if !contains(reviewerIDs, "reviewer2") || !contains(reviewerIDs, "reviewer3") {
        t.Errorf("Expected reviewers [reviewer2, reviewer3], got %v", reviewerIDs)
    }
}

func TestRepository_ReassignReviewer_Errors(t *testing.T) {
    db := setupTestDB(t)
	defer db.Close()
	repo := repository.NewRepository(db)
    team := &entity.Team{Name: "error-test-team"}
    members := []entity.User{
        {ID: "author1", Username: "Author1", IsActive: true},
        {ID: "reviewer1", Username: "Reviewer1", IsActive: true},
        {ID: "not-assigned-user", Username: "NotAssigned", IsActive: true},
    }
    err := repo.CreateTeam(team, members)
    if err != nil {
        t.Fatalf("Failed to create team: %v", err)
    }
    t.Run("PRNotExists", func(t *testing.T) {
        _, err := repo.ReassignReviewer("nonexistent-pr", "reviewer1")
        if !errors.Is(err, entity.ErrNotFound) {
            t.Errorf("Expected ErrNotFound for non-existent PR, got: %v", err)
        }
    })
    t.Run("ReviewerNotAssigned", func(t *testing.T) {
        pr := &entity.PullRequest{
            ID:       "pr-error-test",
            Title:    "Test PR",
            AuthorID: "author1",
        }
        err := repo.CreatePR(pr, []string{"reviewer1"})
        if err != nil {
            t.Fatalf("Failed to create PR: %v", err)
        }
        _, err = repo.ReassignReviewer("pr-error-test", "not-assigned-user")
        if !errors.Is(err, entity.ErrNotAssigned) {
            t.Errorf("Expected ErrNotAssigned for not assigned reviewer, got: %v", err)
        }
    })
}


func contains(slice []string, item string) bool {
    for _, s := range slice {
        if s == item {
            return true
        }
    }
    return false
}

func TestRepository_CreateTeam_DuplicateMembers(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()
    repo := repository.NewRepository(db)
    team := &entity.Team{Name: "duplicate-team"}
    members := []entity.User{
        {ID: "user1", Username: "User1", IsActive: true},
        {ID: "user1", Username: "User1", IsActive: true},
        {ID: "user2", Username: "User2", IsActive: true},
    }
    err := repo.CreateTeam(team, members)
    if err != nil {
        t.Errorf("Should handle duplicate members gracefully, got error: %v", err)
    }
    _, retrievedMembers, err := repo.GetTeam("duplicate-team")
    if err != nil {
        t.Errorf("Should retrieve team: %v", err)
    }
    uniqueUsers := make(map[string]bool)
    for _, member := range retrievedMembers {
        if uniqueUsers[member.ID] {
            t.Errorf("Found duplicate user in team: %s", member.ID)
        }
        uniqueUsers[member.ID] = true
    }
}

func TestRepository_CreatePR_TransactionRollbackOnInvalidReviewer(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()
    repo := repository.NewRepository(db)
    team := &entity.Team{Name: "transaction-team"}
    members := []entity.User{
        {ID: "author1", Username: "Author1", IsActive: true},
        {ID: "reviewer1", Username: "Reviewer1", IsActive: true},
    }
    err := repo.CreateTeam(team, members)
    if err != nil {
        t.Fatalf("Failed to create team: %v", err)
    }
    pr1 := &entity.PullRequest{
        ID:       "pr-success",
        Title:    "Success PR",
        AuthorID: "author1",
    }
    err = repo.CreatePR(pr1, []string{"reviewer1"})
    if err != nil {
        t.Fatalf("Failed to create first PR: %v", err)
    }
    pr2 := &entity.PullRequest{
        ID:       "pr-fail",
        Title:    "Fail PR", 
        AuthorID: "author1",
    }
    err = repo.CreatePR(pr2, []string{"nonexistent-reviewer"})
    if err == nil {
        t.Error("Should fail when reviewer doesn't exist")
    }
    _, err = repo.GetPR("pr-fail")
    if !errors.Is(err, entity.ErrNotFound) {
        t.Errorf("Failed PR should not be created, got: %v", err)
    }
    existingPR, err := repo.GetPR("pr-success")
    if err != nil {
        t.Errorf("First PR should still exist: %v", err)
    }
    if existingPR.ID != "pr-success" {
        t.Errorf("First PR was affected by second PR's failure")
    }
}

func TestRepository_ReassignReviewer_PRAlreadyMerged(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()
    repo := repository.NewRepository(db)
    team := &entity.Team{Name: "merged-pr-team"}
    members := []entity.User{
        {ID: "author1", Username: "Author1", IsActive: true},
        {ID: "reviewer1", Username: "Reviewer1", IsActive: true},
        {ID: "reviewer2", Username: "Reviewer2", IsActive: true},
    }
    err := repo.CreateTeam(team, members)
    if err != nil {
        t.Fatalf("Failed to create team: %v", err)
    }
    pr := &entity.PullRequest{
        ID:       "pr-merged",
        Title:    "Test PR",
        AuthorID: "author1",
    }
    err = repo.CreatePR(pr, []string{"reviewer1"})
    if err != nil {
        t.Fatalf("Failed to create PR: %v", err)
    }
    _, err = repo.MergePR("pr-merged")
    if err != nil {
        t.Fatalf("Failed to merge PR: %v", err)
    }
    _, err = repo.ReassignReviewer("pr-merged", "reviewer1")
    if !errors.Is(err, entity.ErrPRMerged) {
        t.Errorf("Expected ErrPRMerged for merged PR, got: %v", err)
    }
}

func TestRepository_ReassignReviewer_PRStillOpen(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()
    repo := repository.NewRepository(db)
    team := &entity.Team{Name: "open-pr-team"}
    members := []entity.User{
        {ID: "author1", Username: "Author1", IsActive: true},
        {ID: "reviewer1", Username: "Reviewer1", IsActive: true},
        {ID: "reviewer2", Username: "Reviewer2", IsActive: true},
    }
    err := repo.CreateTeam(team, members)
    if err != nil {
        t.Fatalf("Failed to create team: %v", err)
    }
    pr := &entity.PullRequest{
        ID:       "pr-open",
        Title:    "Test PR",
        AuthorID: "author1",
    }
    err = repo.CreatePR(pr, []string{"reviewer1"})
    if err != nil {
        t.Fatalf("Failed to create PR: %v", err)
    }
    currentPR, err := repo.GetPR("pr-open")
    if err != nil {
        t.Fatalf("Failed to get PR: %v", err)
    }
    if currentPR.Status != "OPEN" {
        t.Errorf("PR should be OPEN before reassignment, got: %s", currentPR.Status)
    }
    newReviewer, err := repo.ReassignReviewer("pr-open", "reviewer1")
    if errors.Is(err, entity.ErrPRMerged) {
        t.Error("Should not get ErrPRMerged for open PR")
    }
    if err == nil {
        if newReviewer == "" {
            t.Error("Should get new reviewer ID")
        }
    }
}

func TestRepository_ReassignReviewer_NoCandidatesInTeam(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()
    repo := repository.NewRepository(db)
    team := &entity.Team{Name: "no-candidates-team"}
    members := []entity.User{
        {ID: "author1", Username: "Author1", IsActive: true},
        {ID: "reviewer1", Username: "Reviewer1", IsActive: true},
    }
    err := repo.CreateTeam(team, members)
    if err != nil {
        t.Fatalf("Failed to create team: %v", err)
    }
    pr := &entity.PullRequest{
        ID:       "pr-no-candidates",
        Title:    "Test PR",
        AuthorID: "author1",
    }
    err = repo.CreatePR(pr, []string{"reviewer1"})
    if err != nil {
        t.Fatalf("Failed to create PR: %v", err)
    }
    _, err = repo.ReassignReviewer("pr-no-candidates", "reviewer1")
    if !errors.Is(err, entity.ErrNoCandidate) {
        t.Errorf("Expected ErrNoCandidate when no candidates available, got: %v", err)
    }
}

func TestRepository_ReassignReviewer_AllPotentialCandidatesAlreadyReviewers(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()
    repo := repository.NewRepository(db)
    team := &entity.Team{Name: "all-reviewers-team"}
    members := []entity.User{
        {ID: "author1", Username: "Author1", IsActive: true},
        {ID: "reviewer1", Username: "Reviewer1", IsActive: true},
        {ID: "reviewer2", Username: "Reviewer2", IsActive: true},
        {ID: "reviewer3", Username: "Reviewer3", IsActive: true},
    }
    err := repo.CreateTeam(team, members)
    if err != nil {
        t.Fatalf("Failed to create team: %v", err)
    }
    pr := &entity.PullRequest{
        ID:       "pr-all-reviewers",
        Title:    "Test PR",
        AuthorID: "author1",
    }
    err = repo.CreatePR(pr, []string{"reviewer1", "reviewer2", "reviewer3"})
    if err != nil {
        t.Fatalf("Failed to create PR: %v", err)
    }
    _, err = repo.ReassignReviewer("pr-all-reviewers", "reviewer1")
    if !errors.Is(err, entity.ErrNoCandidate) {
        t.Errorf("Expected ErrNoCandidate when all candidates are already reviewers, got: %v", err)
    }
}

func TestRepository_GetStats_ComplexScenario(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()
    repo := repository.NewRepository(db)
    teams := []struct {
        name    string
        members []entity.User
    }{
        {
            name: "team-a",
            members: []entity.User{
                {ID: "author-a", Username: "AuthorA", IsActive: true},
                {ID: "reviewer-a1", Username: "ReviewerA1", IsActive: true},
                {ID: "reviewer-a2", Username: "ReviewerA2", IsActive: true},
            },
        },
        {
            name: "team-b", 
            members: []entity.User{
                {ID: "author-b", Username: "AuthorB", IsActive: true},
                {ID: "reviewer-b1", Username: "ReviewerB1", IsActive: true},
                {ID: "reviewer-b2", Username: "ReviewerB2", IsActive: true},
            },
        },
    }
    for _, team := range teams {
        err := repo.CreateTeam(&entity.Team{Name: team.name}, team.members)
        if err != nil {
            t.Fatalf("Failed to create team %s: %v", team.name, err)
        }
    }
    testPRs := []struct {
        id       string
        title    string
        author   string
        reviewers []string
    }{
        {"pr-a-1", "Feature A1", "author-a", []string{"reviewer-a1", "reviewer-a2"}},
        {"pr-a-2", "Feature A2", "author-a", []string{"reviewer-a1"}},
        {"pr-a-3", "Feature A3", "author-a", []string{"reviewer-a2"}},
        {"pr-b-1", "Feature B1", "author-b", []string{"reviewer-b1"}},
        {"pr-b-2", "Feature B2", "author-b", []string{"reviewer-b1", "reviewer-b2"}},
    }
    for _, prData := range testPRs {
        pr := &entity.PullRequest{
            ID:       prData.id,
            Title:    prData.title,
            AuthorID: prData.author,
        }
        err := repo.CreatePR(pr, prData.reviewers)
        if err != nil {
            t.Fatalf("Failed to create PR %s: %v", prData.id, err)
        }
    }
    stats, err := repo.GetStats()
    if err != nil {
        t.Fatalf("GetStats failed: %v", err)
    }
    expectedTotal := 2 + 1 + 1 + 1 + 2
    if stats.TotalAssignments != expectedTotal {
        t.Errorf("Expected %d total assignments, got %d", expectedTotal, stats.TotalAssignments)
    }
    userAssignments := make(map[string]int)
    for _, uac := range stats.UserAssignmentCounts {
        userAssignments[uac.UserID] = uac.Count
    }
    expectedUserAssignments := map[string]int{
        "reviewer-a1": 2,
        "reviewer-a2": 2, 
        "reviewer-b1": 2,
        "reviewer-b2": 1, 
    }
    for userID, expectedCount := range expectedUserAssignments {
        if userAssignments[userID] != expectedCount {
            t.Errorf("User %s should have %d assignments, got %d", userID, expectedCount, userAssignments[userID])
        }
    }
    prAssignments := make(map[string]int)
    for _, prac := range stats.PRAssignmentCounts {
        prAssignments[prac.PRID] = prac.Count
    }
    expectedPRAssignments := map[string]int{
        "pr-a-1": 2,
        "pr-a-2": 1, 
        "pr-a-3": 1,
        "pr-b-1": 1,
        "pr-b-2": 2,
    }
    for prID, expectedCount := range expectedPRAssignments {
        if prAssignments[prID] != expectedCount {
            t.Errorf("PR %s should have %d assignments, got %d", prID, expectedCount, prAssignments[prID])
        }
    }
}

func TestRepository_GetStats_AfterReassignment(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()
    repo := repository.NewRepository(db)
    team := &entity.Team{Name: "reassign-stats-team"}
    members := []entity.User{
        {ID: "author1", Username: "Author1", IsActive: true},
        {ID: "reviewer1", Username: "Reviewer1", IsActive: true},
        {ID: "reviewer2", Username: "Reviewer2", IsActive: true},
        {ID: "reviewer3", Username: "Reviewer3", IsActive: true},
    }
    err := repo.CreateTeam(team, members)
    if err != nil {
        t.Fatalf("Failed to create team: %v", err)
    }
    pr := &entity.PullRequest{
        ID:       "pr-reassign-stats",
        Title:    "Test PR",
        AuthorID: "author1",
    }
    err = repo.CreatePR(pr, []string{"reviewer1", "reviewer2"})
    if err != nil {
        t.Fatalf("Failed to create PR: %v", err)
    }
    statsBefore, err := repo.GetStats()
    if err != nil {
        t.Fatalf("GetStats before reassignment failed: %v", err)
    }
    _, err = repo.ReassignReviewer("pr-reassign-stats", "reviewer1")
    if err != nil {
        t.Fatalf("ReassignReviewer failed: %v", err)
    }
    statsAfter, err := repo.GetStats()
    if err != nil {
        t.Fatalf("GetStats after reassignment failed: %v", err)
    }
    if statsBefore.TotalAssignments != statsAfter.TotalAssignments {
        t.Errorf("Total assignments should remain the same after reassignment, was %d, now %d", 
            statsBefore.TotalAssignments, statsAfter.TotalAssignments)
    }
    var reviewer1Before, reviewer1After int
    for _, uac := range statsBefore.UserAssignmentCounts {
        if uac.UserID == "reviewer1" {
            reviewer1Before = uac.Count
        }
    }
    for _, uac := range statsAfter.UserAssignmentCounts {
        if uac.UserID == "reviewer1" {
            reviewer1After = uac.Count
        }
    }
    if reviewer1After >= reviewer1Before {
        t.Errorf("Reviewer1 assignments should decrease after reassignment, was %d, now %d", 
            reviewer1Before, reviewer1After)
    }
}

func TestRepository_GetStats_WithMergedPRs(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()
    repo := repository.NewRepository(db)
    team := &entity.Team{Name: "merged-stats-team"}
    members := []entity.User{
        {ID: "author1", Username: "Author1", IsActive: true},
        {ID: "reviewer1", Username: "Reviewer1", IsActive: true},
        {ID: "reviewer2", Username: "Reviewer2", IsActive: true},
    }
    err := repo.CreateTeam(team, members)
    if err != nil {
        t.Fatalf("Failed to create team: %v", err)
    }
    pr1 := &entity.PullRequest{
        ID:       "pr-merged-1",
        Title:    "Merged PR",
        AuthorID: "author1",
    }
    err = repo.CreatePR(pr1, []string{"reviewer1", "reviewer2"})
    if err != nil {
        t.Fatalf("Failed to create PR1: %v", err)
    }
    pr2 := &entity.PullRequest{
        ID:       "pr-open-1", 
        Title:    "Open PR",
        AuthorID: "author1",
    }
    err = repo.CreatePR(pr2, []string{"reviewer1"})
    if err != nil {
        t.Fatalf("Failed to create PR2: %v", err)
    }
    _, err = repo.MergePR("pr-merged-1")
    if err != nil {
        t.Fatalf("Failed to merge PR: %v", err)
    }
    stats, err := repo.GetStats()
    if err != nil {
        t.Fatalf("GetStats failed: %v", err)
    }
    if stats.TotalAssignments != 3 { 
        t.Errorf("Expected 3 total assignments including merged PRs, got %d", stats.TotalAssignments)
    }
    var foundMergedPR, foundOpenPR bool
    for _, prac := range stats.PRAssignmentCounts {
        if prac.PRID == "pr-merged-1" {
            foundMergedPR = true
            if prac.Count != 2 {
                t.Errorf("Merged PR should have 2 assignments, got %d", prac.Count)
            }
        }
        if prac.PRID == "pr-open-1" {
            foundOpenPR = true
            if prac.Count != 1 {
                t.Errorf("Open PR should have 1 assignment, got %d", prac.Count)
            }
        }
    }
    if !foundMergedPR {
        t.Error("Merged PR should be included in stats")
    }
    if !foundOpenPR {
        t.Error("Open PR should be included in stats")
    }
}

func TestRepository_GetStats_UserWithoutAssignments(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()
    repo := repository.NewRepository(db)
    
    team := &entity.Team{Name: "no-assignments-team"}
    members := []entity.User{
        {ID: "author1", Username: "Author1", IsActive: true},
        {ID: "reviewer-no-assignments", Username: "ReviewerNoAssign", IsActive: true},
        {ID: "reviewer-with-assignments", Username: "ReviewerWithAssign", IsActive: true},
    }
    err := repo.CreateTeam(team, members)
    if err != nil {
        t.Fatalf("Failed to create team: %v", err)
    }
    pr := &entity.PullRequest{
        ID:       "pr-single-reviewer",
        Title:    "Test PR", 
        AuthorID: "author1",
    }
    err = repo.CreatePR(pr, []string{"reviewer-with-assignments"})
    if err != nil {
        t.Fatalf("Failed to create PR: %v", err)
    }
    stats, err := repo.GetStats()
    if err != nil {
        t.Fatalf("GetStats failed: %v", err)
    }
    var foundUserWithAssignments, foundUserWithoutAssignments bool
    for _, uac := range stats.UserAssignmentCounts {
        if uac.UserID == "reviewer-with-assignments" {
            foundUserWithAssignments = true
            if uac.Count != 1 {
                t.Errorf("User with assignments should have count 1, got %d", uac.Count)
            }
        }
        if uac.UserID == "reviewer-no-assignments" {
            foundUserWithoutAssignments = true
            if uac.Count != 0 {
                t.Errorf("User without assignments should have count 0, got %d", uac.Count)
            }
        }
    }
    if !foundUserWithAssignments {
        t.Error("User with assignments should be in stats")
    }
    if !foundUserWithoutAssignments {
        t.Error("User without assignments should be in stats with count 0")
    }
}

func TestRepository_GetCandidateReviewers_Simple(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()
    repo := repository.NewRepository(db)
    team := &entity.Team{Name: "simple-team"}
    members := []entity.User{
        {ID: "s1", Username: "Simple1", IsActive: true},
        {ID: "s2", Username: "Simple2", IsActive: true},
        {ID: "s3", Username: "Simple3", IsActive: true},
    }
    err := repo.CreateTeam(team, members)
    if err != nil {
        t.Fatalf("Failed to create team: %v", err)
    }
    t.Run("basic assignment", func(t *testing.T) {
        candidates, err := repo.GetCandidateReviewers("s1", 2)
        if err != nil {
            t.Fatalf("GetCandidateReviewers failed: %v", err)
        }
        if len(candidates) != 2 {
            t.Errorf("Expected 2 candidates, got %d", len(candidates))
        }
        expected := []string{"s2", "s3"}
        for _, candidate := range candidates {
            if !contains(expected, candidate) {
                t.Errorf("Unexpected candidate: %s, expected one of %v", candidate, expected)
            }
        }
        t.Logf("Basic assignment result: %v", candidates)
    })

    t.Run("after creating PR", func(t *testing.T) {
        pr := &entity.PullRequest{ID: "pr-simple-1", Title: "Simple PR", AuthorID: "s2"}
        err := repo.CreatePR(pr, []string{"s1", "s3"})
        if err != nil {
            t.Fatalf("Failed to create PR: %v", err)
        }
        candidates, err := repo.GetCandidateReviewers("s1", 2)
        if err != nil {
            t.Fatalf("GetCandidateReviewers failed: %v", err)
        }
        t.Logf("Assignment after PR creation: %v", candidates)
        foundS2 := false
        for _, candidate := range candidates {
            if candidate == "s2" {
                foundS2 = true
                break
            }
        }
        if !foundS2 {
            t.Error("s2 should be selected due to zero load")
        }
    })
}