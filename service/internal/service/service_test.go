package service

import (
	"errors"
	"testing"

	"service/internal/entity"
)

type mockRepo struct {
    createTeamFunc        func(team *entity.Team, members []entity.User) error
    getTeamFunc           func(teamName string) (*entity.Team, []entity.User, error)
    setUserActiveFunc     func(userID string, isActive bool) (*entity.User, error)
    getUserReviewPRsFunc  func(userID string) ([]entity.PullRequest, error)
    createPRFunc          func(pr *entity.PullRequest, reviewerIDs []string) error
    mergePRFunc           func(prID string) (*entity.PullRequest, error)
    getPRFunc             func(prID string) (*entity.PullRequest, error)
    reassignReviewerFunc  func(prID, oldUserID string) (string, error)
    getCandidateReviewersFunc func(authorID string, limit int) ([]string, error)
    getStatsFunc          func() (*entity.Stats, error) 
}

func (m *mockRepo) CreateTeam(team *entity.Team, members []entity.User) error {
    if m.createTeamFunc != nil {
        return m.createTeamFunc(team, members)
    }
    return nil
}

func (m *mockRepo) GetTeam(teamName string) (*entity.Team, []entity.User, error) {
    if m.getTeamFunc != nil {
        return m.getTeamFunc(teamName)
    }
    return &entity.Team{Name: teamName}, []entity.User{}, nil
}

func (m *mockRepo) SetUserActive(userID string, isActive bool) (*entity.User, error) {
    if m.setUserActiveFunc != nil {
        return m.setUserActiveFunc(userID, isActive)
    }
    return &entity.User{ID: userID, IsActive: isActive}, nil
}

func (m *mockRepo) GetUserReviewPRs(userID string) ([]entity.PullRequest, error) {
    if m.getUserReviewPRsFunc != nil {
        return m.getUserReviewPRsFunc(userID)
    }
    return []entity.PullRequest{}, nil
}

func (m *mockRepo) CreatePR(pr *entity.PullRequest, reviewerIDs []string) error {
    if m.createPRFunc != nil {
        return m.createPRFunc(pr, reviewerIDs)
    }
    return nil
}

func (m *mockRepo) MergePR(prID string) (*entity.PullRequest, error) {
    if m.mergePRFunc != nil {
        return m.mergePRFunc(prID)
    }
    return &entity.PullRequest{ID: prID, Status: "MERGED"}, nil
}

func (m *mockRepo) GetPR(prID string) (*entity.PullRequest, error) {
    if m.getPRFunc != nil {
        return m.getPRFunc(prID)
    }
    return &entity.PullRequest{ID: prID}, nil
}

func (m *mockRepo) ReassignReviewer(prID, oldUserID string) (string, error) {
    if m.reassignReviewerFunc != nil {
        return m.reassignReviewerFunc(prID, oldUserID)
    }
    return "new-user", nil
}

func (m *mockRepo) GetCandidateReviewers(authorID string, limit int) ([]string, error) {
    if m.getCandidateReviewersFunc != nil {
        return m.getCandidateReviewersFunc(authorID, limit)
    }
    return []string{"reviewer1", "reviewer2"}, nil
}

func (m *mockRepo) GetPRReviewers(prID string) ([]entity.User, error) {
    return []entity.User{}, nil
}

func (m *mockRepo) GetStats() (*entity.Stats, error) {
    if m.getStatsFunc != nil {
        return m.getStatsFunc()
    }
    return &entity.Stats{
        UserAssignmentCounts: []entity.UserAssignmentCount{},
        PRAssignmentCounts:   []entity.PRAssignmentCount{},
        TotalAssignments:     0,
    }, nil
}

func TestService_CreateTeam_Success(t *testing.T) {
    mockRepo := &mockRepo{
        createTeamFunc: func(team *entity.Team, members []entity.User) error {
            return nil
        },
    }
    service := NewService(mockRepo)
    members := []entity.User{
        {ID: "u1", Username: "Alice", IsActive: true},
        {ID: "u2", Username: "Bob", IsActive: true},
    }
    team, err := service.CreateTeam("backend", members)
    if err != nil {
        t.Fatalf("Expected no error, got %v", err)
    }
    if team.Name != "backend" {
        t.Errorf("Expected team name 'backend', got %s", team.Name)
    }
}

func TestService_CreateTeam_RepositoryError(t *testing.T) {
    mockRepo := &mockRepo{
        createTeamFunc: func(team *entity.Team, members []entity.User) error {
            return entity.ErrTeamExists
        },
    }
    service := NewService(mockRepo)
    _, err := service.CreateTeam("backend", []entity.User{})
    if !errors.Is(err, entity.ErrTeamExists) {
        t.Errorf("Expected ErrTeamExists, got %v", err)
    }
}

func TestService_SetUserActive_Success(t *testing.T) {
    mockRepo := &mockRepo{
        setUserActiveFunc: func(userID string, isActive bool) (*entity.User, error) {
            return &entity.User{ID: userID, Username: "testuser", IsActive: isActive}, nil
        },
    }
    service := NewService(mockRepo)
    user, err := service.SetUserActive("u1", true)
    if err != nil {
        t.Fatalf("Expected no error, got %v", err)
    }

    if user.ID != "u1" {
        t.Errorf("Expected user ID 'u1', got %s", user.ID)
    }
    if !user.IsActive {
        t.Error("Expected user to be active")
    }
}

func TestService_CreatePR_Success(t *testing.T) {
    mockRepo := &mockRepo{
        setUserActiveFunc: func(userID string, isActive bool) (*entity.User, error) {
            return &entity.User{ID: userID, Username: "author", IsActive: true}, nil
        },
        getCandidateReviewersFunc: func(authorID string, limit int) ([]string, error) {
            return []string{"reviewer1", "reviewer2"}, nil
        },
        createPRFunc: func(pr *entity.PullRequest, reviewerIDs []string) error {
            return nil
        },
        getPRFunc: func(prID string) (*entity.PullRequest, error) {
            return &entity.PullRequest{
                ID:       prID,
                Title:    "Test PR",
                AuthorID: "author1",
                Status:   "OPEN",
                AssignedReviewers: []entity.User{
                    {ID: "reviewer1", Username: "Reviewer1", IsActive: true},
                    {ID: "reviewer2", Username: "Reviewer2", IsActive: true},
                },
            }, nil
        },
    }
    service := NewService(mockRepo)
    pr, err := service.CreatePR("pr-1", "Test PR", "author1")
    if err != nil {
        t.Fatalf("Expected no error, got %v", err)
    }
    if pr.ID != "pr-1" {
        t.Errorf("Expected PR ID 'pr-1', got %s", pr.ID)
    }
    if pr.Status != "OPEN" {
        t.Errorf("Expected status 'OPEN', got %s", pr.Status)
    }
    if len(pr.AssignedReviewers) != 2 {
        t.Errorf("Expected 2 assigned reviewers, got %d", len(pr.AssignedReviewers))
    }
}

func TestService_CreatePR_AuthorNotFound(t *testing.T) {
    mockRepo := &mockRepo{
        setUserActiveFunc: func(userID string, isActive bool) (*entity.User, error) {
            return nil, entity.ErrNotFound
        },
    }
    service := NewService(mockRepo)
    _, err := service.CreatePR("pr-1", "Test PR", "nonexistent")
    if !errors.Is(err, entity.ErrNotFound) {
        t.Errorf("Expected ErrNotFound, got %v", err)
    }
}

func TestService_CreatePR_AuthorInactive(t *testing.T) {
    mockRepo := &mockRepo{
        setUserActiveFunc: func(userID string, isActive bool) (*entity.User, error) {
            return &entity.User{ID: userID, Username: "author", IsActive: false}, nil
        },
    }
    service := NewService(mockRepo)
    _, err := service.CreatePR("pr-1", "Test PR", "inactive-author")
    if err == nil {
        t.Error("Expected error for inactive author")
    }
}

func TestService_CreatePR_NoCandidateReviewers(t *testing.T) {
    mockRepo := &mockRepo{
        setUserActiveFunc: func(userID string, isActive bool) (*entity.User, error) {
            return &entity.User{ID: userID, Username: "author", IsActive: true}, nil
        },
        getCandidateReviewersFunc: func(authorID string, limit int) ([]string, error) {
            return []string{}, nil
        },
    }
    service := NewService(mockRepo)
    _, err := service.CreatePR("pr-1", "Test PR", "author1")
    if !errors.Is(err, entity.ErrNoCandidate) {
        t.Errorf("Expected ErrNoCandidate, got %v", err)
    }
}

func TestService_CreatePR_CandidateReviewersError(t *testing.T) {
    mockRepo := &mockRepo{
        setUserActiveFunc: func(userID string, isActive bool) (*entity.User, error) {
            return &entity.User{ID: userID, Username: "author", IsActive: true}, nil
        },
        getCandidateReviewersFunc: func(authorID string, limit int) ([]string, error) {
            return nil, errors.New("database error")
        },
    }
    service := NewService(mockRepo)
    _, err := service.CreatePR("pr-1", "Test PR", "author1")
    if err == nil {
        t.Error("Expected error from candidate reviewers")
    }
}

func TestService_MergePR_Success(t *testing.T) {
    mockRepo := &mockRepo{
        mergePRFunc: func(prID string) (*entity.PullRequest, error) {
            return &entity.PullRequest{ID: prID, Status: "MERGED"}, nil
        },
    }
    service := NewService(mockRepo)
    pr, err := service.MergePR("pr-1")
    if err != nil {
        t.Fatalf("Expected no error, got %v", err)
    }

    if pr.Status != "MERGED" {
        t.Errorf("Expected status 'MERGED', got %s", pr.Status)
    }
}

func TestService_ReassignReviewer_Success(t *testing.T) {
    mockRepo := &mockRepo{
        getPRFunc: func(prID string) (*entity.PullRequest, error) {
            return &entity.PullRequest{
                ID:     prID,
                Status: "OPEN",
                AssignedReviewers: []entity.User{
                    {ID: "old-reviewer", Username: "Old Reviewer", IsActive: true},
                    {ID: "other-reviewer", Username: "Other Reviewer", IsActive: true},
                },
            }, nil
        },
        reassignReviewerFunc: func(prID, oldUserID string) (string, error) {
            return "new-reviewer", nil
        },
    }
    service := NewService(mockRepo)
    updatedPR, newUserID, err := service.ReassignReviewer("pr-1", "old-reviewer")
    if err != nil {
        t.Fatalf("Expected no error, got %v", err)
    }
    if newUserID != "new-reviewer" {
        t.Errorf("Expected new reviewer 'new-reviewer', got %s", newUserID)
    }
    if updatedPR == nil {
        t.Error("Expected updated PR to be returned")
    }
}

func TestService_ReassignReviewer_PRNotFound(t *testing.T) {
    mockRepo := &mockRepo{
        getPRFunc: func(prID string) (*entity.PullRequest, error) {
            return nil, entity.ErrNotFound
        },
    }
    service := NewService(mockRepo)
    _, _, err := service.ReassignReviewer("nonexistent-pr", "reviewer1")
    if !errors.Is(err, entity.ErrNotFound) {
        t.Errorf("Expected ErrNotFound, got %v", err)
    }
}

func TestService_ReassignReviewer_PRAlreadyMerged(t *testing.T) {
    mockRepo := &mockRepo{
        getPRFunc: func(prID string) (*entity.PullRequest, error) {
            return &entity.PullRequest{
                ID:     prID,
                Status: "MERGED",
                AssignedReviewers: []entity.User{
                    {ID: "reviewer1", Username: "Reviewer1", IsActive: true},
                },
            }, nil
        },
    }
    service := NewService(mockRepo)
    _, _, err := service.ReassignReviewer("pr-1", "reviewer1")
    if !errors.Is(err, entity.ErrPRMerged) {
        t.Errorf("Expected ErrPRMerged, got %v", err)
    }
}

func TestService_ReassignReviewer_ReviewerNotAssigned(t *testing.T) {
    mockRepo := &mockRepo{
        getPRFunc: func(prID string) (*entity.PullRequest, error) {
            return &entity.PullRequest{
                ID:     prID,
                Status: "OPEN",
                AssignedReviewers: []entity.User{
                    {ID: "reviewer1", Username: "Reviewer1", IsActive: true},
                },
            }, nil
        },
    }
    service := NewService(mockRepo)
    _, _, err := service.ReassignReviewer("pr-1", "not-assigned-reviewer")
    if !errors.Is(err, entity.ErrNotAssigned) {
        t.Errorf("Expected ErrNotAssigned, got %v", err)
    }
}

func TestService_ReassignReviewer_ReassignmentError(t *testing.T) {
    mockRepo := &mockRepo{
        getPRFunc: func(prID string) (*entity.PullRequest, error) {
            return &entity.PullRequest{
                ID:     prID,
                Status: "OPEN",
                AssignedReviewers: []entity.User{
                    {ID: "reviewer1", Username: "Reviewer1", IsActive: true},
                },
            }, nil
        },
        reassignReviewerFunc: func(prID, oldUserID string) (string, error) {
            return "", entity.ErrNoCandidate
        },
    }
    service := NewService(mockRepo)
    _, _, err := service.ReassignReviewer("pr-1", "reviewer1")
    if !errors.Is(err, entity.ErrNoCandidate) {
        t.Errorf("Expected ErrNoCandidate, got %v", err)
    }
}

func TestService_GetPR_Success(t *testing.T) {
    mockRepo := &mockRepo{
        getPRFunc: func(prID string) (*entity.PullRequest, error) {
            return &entity.PullRequest{
                ID:       prID,
                Title:    "Test PR",
                AuthorID: "author1",
                Status:   "OPEN",
            }, nil
        },
    }
    service := NewService(mockRepo)
    pr, err := service.GetPR("pr-1")
    if err != nil {
        t.Fatalf("Expected no error, got %v", err)
    }
    if pr.ID != "pr-1" {
        t.Errorf("Expected PR ID 'pr-1', got %s", pr.ID)
    }
}

func TestService_GetPR_NotFound(t *testing.T) {
    mockRepo := &mockRepo{
        getPRFunc: func(prID string) (*entity.PullRequest, error) {
            return nil, entity.ErrNotFound
        },
    }
    service := NewService(mockRepo)
    _, err := service.GetPR("nonexistent-pr")
    if !errors.Is(err, entity.ErrNotFound) {
        t.Errorf("Expected ErrNotFound, got %v", err)
    }
}

func TestService_GetTeam_Success(t *testing.T) {
    expectedTeam := &entity.Team{Name: "backend"}
    expectedMembers := []entity.User{
        {ID: "u1", Username: "Alice", IsActive: true},
        {ID: "u2", Username: "Bob", IsActive: true},
    }

    mockRepo := &mockRepo{
        getTeamFunc: func(teamName string) (*entity.Team, []entity.User, error) {
            return expectedTeam, expectedMembers, nil
        },
    }

    service := NewService(mockRepo)
    team, members, err := service.GetTeam("backend")
    if err != nil {
        t.Fatalf("Expected no error, got %v", err)
    }

    if team.Name != "backend" {
        t.Errorf("Expected team name 'backend', got %s", team.Name)
    }
    if len(members) != 2 {
        t.Errorf("Expected 2 members, got %d", len(members))
    }
}

func TestService_GetTeam_NotFound(t *testing.T) {
    mockRepo := &mockRepo{
        getTeamFunc: func(teamName string) (*entity.Team, []entity.User, error) {
            return nil, nil, entity.ErrNotFound
        },
    }

    service := NewService(mockRepo)
    _, _, err := service.GetTeam("nonexistent")
    if !errors.Is(err, entity.ErrNotFound) {
        t.Errorf("Expected ErrNotFound, got %v", err)
    }
}

func TestService_GetUserReviewPRs_Success(t *testing.T) {
    expectedPRs := []entity.PullRequest{
        {
            ID:       "pr-1",
            Title:    "Feature A",
            AuthorID: "author1",
            Status:   "OPEN",
        },
        {
            ID:       "pr-2",
            Title:    "Feature B",
            AuthorID: "author2",
            Status:   "OPEN",
        },
    }

    mockRepo := &mockRepo{
        getUserReviewPRsFunc: func(userID string) ([]entity.PullRequest, error) {
            return expectedPRs, nil
        },
    }

    service := NewService(mockRepo)
    prs, err := service.GetUserReviewPRs("reviewer1")
    if err != nil {
        t.Fatalf("Expected no error, got %v", err)
    }

    if len(prs) != 2 {
        t.Errorf("Expected 2 PRs, got %d", len(prs))
    }
}

func TestService_GetUserReviewPRs_Empty(t *testing.T) {
    mockRepo := &mockRepo{
        getUserReviewPRsFunc: func(userID string) ([]entity.PullRequest, error) {
            return []entity.PullRequest{}, nil
        },
    }

    service := NewService(mockRepo)
    prs, err := service.GetUserReviewPRs("new-reviewer")
    if err != nil {
        t.Fatalf("Expected no error, got %v", err)
    }

    if len(prs) != 0 {
        t.Errorf("Expected 0 PRs for new reviewer, got %d", len(prs))
    }
}

func TestService_GetUserReviewPRs_RepositoryError(t *testing.T) {
    mockRepo := &mockRepo{
        getUserReviewPRsFunc: func(userID string) ([]entity.PullRequest, error) {
            return nil, errors.New("database error")
        },
    }

    service := NewService(mockRepo)
    _, err := service.GetUserReviewPRs("reviewer1")
    if err == nil {
        t.Error("Expected error from repository")
    }
}

func TestService_CreatePR_DuplicatePR(t *testing.T) {
    mockRepo := &mockRepo{
        setUserActiveFunc: func(userID string, isActive bool) (*entity.User, error) {
            return &entity.User{ID: userID, Username: "author", IsActive: true}, nil
        },
        getCandidateReviewersFunc: func(authorID string, limit int) ([]string, error) {
            return []string{"reviewer1", "reviewer2"}, nil
        },
        createPRFunc: func(pr *entity.PullRequest, reviewerIDs []string) error {
            return entity.ErrPRExists
        },
    }

    service := NewService(mockRepo)
    _, err := service.CreatePR("pr-1", "Test PR", "author1")
    if !errors.Is(err, entity.ErrPRExists) {
        t.Errorf("Expected ErrPRExists, got %v", err)
    }
}

func TestService_CreatePR_CreateError(t *testing.T) {
    mockRepo := &mockRepo{
        setUserActiveFunc: func(userID string, isActive bool) (*entity.User, error) {
            return &entity.User{ID: userID, Username: "author", IsActive: true}, nil
        },
        getCandidateReviewersFunc: func(authorID string, limit int) ([]string, error) {
            return []string{"reviewer1", "reviewer2"}, nil
        },
        createPRFunc: func(pr *entity.PullRequest, reviewerIDs []string) error {
            return errors.New("create failed")
        },
    }

    service := NewService(mockRepo)
    _, err := service.CreatePR("pr-1", "Test PR", "author1")
    if err == nil {
        t.Error("Expected error from PR creation")
    }
}

func TestService_MergePR_NotFound(t *testing.T) {
    mockRepo := &mockRepo{
        mergePRFunc: func(prID string) (*entity.PullRequest, error) {
            return nil, entity.ErrNotFound
        },
    }

    service := NewService(mockRepo)
    _, err := service.MergePR("nonexistent-pr")
    if !errors.Is(err, entity.ErrNotFound) {
        t.Errorf("Expected ErrNotFound, got %v", err)
    }
}

func TestService_MergePR_AlreadyMerged(t *testing.T) {
    mockRepo := &mockRepo{
        mergePRFunc: func(prID string) (*entity.PullRequest, error) {
            return &entity.PullRequest{ID: prID, Status: "MERGED"}, nil
        },
    }

    service := NewService(mockRepo)
    pr, err := service.MergePR("already-merged-pr")
    if err != nil {
        t.Fatalf("Should handle already merged PR gracefully, got error: %v", err)
    }
    if pr.Status != "MERGED" {
        t.Errorf("Expected status MERGED, got %s", pr.Status)
    }
}

func TestService_SetUserActive_NotFound(t *testing.T) {
    mockRepo := &mockRepo{
        setUserActiveFunc: func(userID string, isActive bool) (*entity.User, error) {
            return nil, entity.ErrNotFound
        },
    }

    service := NewService(mockRepo)
    _, err := service.SetUserActive("nonexistent", true)
    if !errors.Is(err, entity.ErrNotFound) {
        t.Errorf("Expected ErrNotFound, got %v", err)
    }
}

func TestService_SetUserActive_RepositoryError(t *testing.T) {
    mockRepo := &mockRepo{
        setUserActiveFunc: func(userID string, isActive bool) (*entity.User, error) {
            return nil, errors.New("database error")
        },
    }

    service := NewService(mockRepo)
    _, err := service.SetUserActive("user1", true)
    if err == nil {
        t.Error("Expected error from repository")
    }
}

func TestService_GetStats_Success(t *testing.T) {
    expectedStats := &entity.Stats{
        UserAssignmentCounts: []entity.UserAssignmentCount{
            {UserID: "u1", Username: "Alice", Count: 10},
            {UserID: "u2", Username: "Bob", Count: 8},
        },
        PRAssignmentCounts: []entity.PRAssignmentCount{
            {PRID: "pr-1", Title: "Feature A", Count: 3},
            {PRID: "pr-2", Title: "Feature B", Count: 2},
        },
        TotalAssignments: 18,
    }

    mockRepo := &mockRepo{
        getStatsFunc: func() (*entity.Stats, error) {
            return expectedStats, nil
        },
    }

    service := NewService(mockRepo)
    stats, err := service.GetStats()
    if err != nil {
        t.Fatalf("Expected no error, got %v", err)
    }

    if stats.TotalAssignments != 18 {
        t.Errorf("Expected total assignments 18, got %d", stats.TotalAssignments)
    }
    if len(stats.UserAssignmentCounts) != 2 {
        t.Errorf("Expected 2 user assignment counts, got %d", len(stats.UserAssignmentCounts))
    }
    if len(stats.PRAssignmentCounts) != 2 {
        t.Errorf("Expected 2 PR assignment counts, got %d", len(stats.PRAssignmentCounts))
    }
}

func TestService_GetStats_Empty(t *testing.T) {
    mockRepo := &mockRepo{
        getStatsFunc: func() (*entity.Stats, error) {
            return &entity.Stats{
                UserAssignmentCounts: []entity.UserAssignmentCount{},
                PRAssignmentCounts:   []entity.PRAssignmentCount{},
                TotalAssignments:     0,
            }, nil
        },
    }
    service := NewService(mockRepo)
    stats, err := service.GetStats()
    if err != nil {
        t.Fatalf("Expected no error, got %v", err)
    }
    if stats.TotalAssignments != 0 {
        t.Errorf("Expected 0 total assignments, got %d", stats.TotalAssignments)
    }
    if len(stats.UserAssignmentCounts) != 0 {
        t.Errorf("Expected 0 user assignment counts, got %d", len(stats.UserAssignmentCounts))
    }
}

func TestService_GetStats_RepositoryError(t *testing.T) {
    mockRepo := &mockRepo{
        getStatsFunc: func() (*entity.Stats, error) {
            return nil, errors.New("stats query failed")
        },
    }
    service := NewService(mockRepo)
    _, err := service.GetStats()
    if err == nil {
        t.Error("Expected error from repository")
    }
}

