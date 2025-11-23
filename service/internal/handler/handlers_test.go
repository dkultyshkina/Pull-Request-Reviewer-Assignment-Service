package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
    "fmt"

    "service/internal/entity"
)

type mockService struct {
    createTeamFunc        func(teamName string, members []entity.User) (*entity.Team, error)
    getTeamFunc           func(teamName string) (*entity.Team, []entity.User, error)
    setUserActiveFunc     func(userID string, isActive bool) (*entity.User, error)
    getUserReviewPRsFunc  func(userID string) ([]entity.PullRequest, error)
    createPRFunc          func(prID, title, authorID string) (*entity.PullRequest, error)
    mergePRFunc           func(prID string) (*entity.PullRequest, error)
    reassignReviewerFunc  func(prID, oldUserID string) (*entity.PullRequest, string, error)
    getPRFunc             func(prID string) (*entity.PullRequest, error)
    getStatsFunc          func() (*entity.Stats, error)
}

func (m *mockService) CreateTeam(teamName string, members []entity.User) (*entity.Team, error) {
    return m.createTeamFunc(teamName, members)
}

func (m *mockService) GetTeam(teamName string) (*entity.Team, []entity.User, error) {
    return m.getTeamFunc(teamName)
}

func (m *mockService) SetUserActive(userID string, isActive bool) (*entity.User, error) {
    return m.setUserActiveFunc(userID, isActive)
}

func (m *mockService) GetUserReviewPRs(userID string) ([]entity.PullRequest, error) {
    return []entity.PullRequest{}, nil
}

func (m *mockService) CreatePR(prID, title, authorID string) (*entity.PullRequest, error) {
    return m.createPRFunc(prID, title, authorID)
}

func (m *mockService) MergePR(prID string) (*entity.PullRequest, error) {
    return m.mergePRFunc(prID)
}

func (m *mockService) ReassignReviewer(prID, oldUserID string) (*entity.PullRequest, string, error) {
    return m.reassignReviewerFunc(prID, oldUserID)
}

func (m *mockService) GetPR(prID string) (*entity.PullRequest, error) {
    return &entity.PullRequest{}, nil
}

func (m *mockService) GetStats() (*entity.Stats, error) {
    if m.getStatsFunc != nil {
        return m.getStatsFunc()
    }
    return &entity.Stats{}, nil
}

func TestHandlers_AddTeam_Success_WithMembers(t *testing.T) {
    var capturedMembers []entity.User
    mock := &mockService{
        createTeamFunc: func(teamName string, members []entity.User) (*entity.Team, error) {
            capturedMembers = members 
            return &entity.Team{Name: teamName}, nil
        },
    }
    handler := NewHandlers(mock)
    requestBody := map[string]interface{}{
        "team_name": "payments",
        "members": []map[string]interface{}{
            {"user_id": "u1", "username": "Alice", "is_active": true},
            {"user_id": "u2", "username": "Bob", "is_active": true},
        },
    }
    body, _ := json.Marshal(requestBody)
    req := httptest.NewRequest("POST", "/teams", bytes.NewReader(body))
    w := httptest.NewRecorder()
    handler.AddTeam(w, req)
    if w.Code != http.StatusCreated {
        t.Errorf("Expected status 201, got %d", w.Code)
        return
    }
    var response map[string]interface{}
    err := json.Unmarshal(w.Body.Bytes(), &response)
    if err != nil {
        t.Fatalf("Failed to parse response: %v", err)
    }
    team, exists := response["team"].(map[string]interface{})
    if !exists {
        t.Fatal("Response must contain 'team' field")
    }
    if team["team_name"] != "payments" {
        t.Errorf("Expected team_name 'payments', got %v", team["team_name"])
    }
    membersData, exists := team["members"].([]interface{})
    if !exists {
        t.Fatal("Team must contain 'members' field")
    }
    if len(membersData) != 2 {
        t.Errorf("Expected 2 members, got %d", len(membersData))
        return
    }
    if len(capturedMembers) != 2 {
        t.Errorf("Mock should have received 2 members, got %d", len(capturedMembers))
    }
    if len(capturedMembers) >= 2 {
        if capturedMembers[0].ID != "u1" {
            t.Errorf("Expected first captured member ID 'u1', got %s", capturedMembers[0].ID)
        }
        if capturedMembers[0].Username != "Alice" {
            t.Errorf("Expected first captured member Username 'Alice', got %s", capturedMembers[0].Username)
        }
        if capturedMembers[0].IsActive != true {
            t.Errorf("Expected first captured member IsActive true, got %t", capturedMembers[0].IsActive)
        }

        if capturedMembers[1].ID != "u2" {
            t.Errorf("Expected second captured member ID 'u2', got %s", capturedMembers[1].ID)
        }
        if capturedMembers[1].Username != "Bob" {
            t.Errorf("Expected second captured member Username 'Bob', got %s", capturedMembers[1].Username)
        }
        if capturedMembers[1].IsActive != true {
            t.Errorf("Expected second captured member IsActive true, got %t", capturedMembers[1].IsActive)
        }
    }
    t.Logf("Team created successfully with %d members", len(membersData))
    t.Logf("Response: %s", w.Body.String())
}


func TestHandlers_AddTeam_TeamAlreadyExists(t *testing.T) {
    mock := &mockService{
        createTeamFunc: func(teamName string, members []entity.User) (*entity.Team, error) {
            return nil, entity.ErrTeamExists
        },
    }
    handler := NewHandlers(mock)
    requestBody := map[string]interface{}{
        "team_name": "payments",
        "members": []map[string]interface{}{
            {"user_id": "u1", "username": "Alice", "is_active": true},
            {"user_id": "u2", "username": "Bob", "is_active": true},
        },
    }
    body, _ := json.Marshal(requestBody)
    req := httptest.NewRequest("POST", "/teams", bytes.NewReader(body))
    w := httptest.NewRecorder()
    handler.AddTeam(w, req)
    if w.Code != http.StatusBadRequest {
        t.Errorf("Expected status 400, got %d", w.Code)
        t.Logf("Response: %s", w.Body.String())
        return
    }
    var response map[string]interface{}
    err := json.Unmarshal(w.Body.Bytes(), &response)
    if err != nil {
        t.Fatalf("Failed to parse error response: %v", err)
    }
    errorData, exists := response["error"].(map[string]interface{})
    if !exists {
        t.Fatal("Error response must contain 'error' field")
    }
    errorCode, exists := errorData["code"].(string)
    if !exists {
        t.Fatal("Error must contain 'code' field")
    }
    if errorCode != "TEAM_EXISTS" {
        t.Errorf("Expected error code 'TEAM_EXISTS', got %v", errorCode)
    }
    errorMessage, exists := errorData["message"].(string)
    if !exists {
        t.Fatal("Error must contain 'message' field")
    }
    if errorMessage != "team already exists" {
        t.Errorf("Expected error message 'team already exists', got %v", errorMessage)
    }
}

func TestHandlers_AddTeam_InvalidJSON(t *testing.T) {
    mock := &mockService{}
    handler := NewHandlers(mock)
    req := httptest.NewRequest("POST", "/teams", bytes.NewReader([]byte("invalid json")))
    w := httptest.NewRecorder()
    handler.AddTeam(w, req)
    if w.Code != http.StatusBadRequest {
        t.Errorf("Expected status 400, got %d", w.Code)
    }
}

func TestHandlers_GetTeam_Success(t *testing.T) {
    mock := &mockService{
        getTeamFunc: func(teamName string) (*entity.Team, []entity.User, error) {
            team := &entity.Team{Name: teamName}
            members := []entity.User{
                {ID: "u1", Username: "Alice", IsActive: true},
                {ID: "u2", Username: "Bob", IsActive: false},
            }
            return team, members, nil
        },
    }
    handler := NewHandlers(mock)
    req := httptest.NewRequest("GET", "/team/get?team_name=backend", nil)
    w := httptest.NewRecorder()
    handler.GetTeam(w, req)
    if w.Code != http.StatusOK {
        t.Errorf("Expected status 200, got %d", w.Code)
        return
    }
    var response map[string]interface{}
    err := json.Unmarshal(w.Body.Bytes(), &response)
    if err != nil {
        t.Fatalf("Failed to parse response: %v", err)
    }
    if response["team_name"] != "backend" {
        t.Errorf("Expected team_name 'backend', got %v", response["team_name"])
    }
    membersData, exists := response["members"].([]interface{})
    if !exists {
        t.Fatal("Response must contain 'members' field")
    }
    if len(membersData) != 2 {
        t.Errorf("Expected 2 members, got %d", len(membersData))
    }
    member1, ok := membersData[0].(map[string]interface{})
    if !ok {
        t.Fatal("First member should be an object")
    }
    if member1["user_id"] != "u1" {
        t.Errorf("Expected first member user_id 'u1', got %v", member1["user_id"])
    }
    if member1["username"] != "Alice" {
        t.Errorf("Expected first member username 'Alice', got %v", member1["username"])
    }
    if member1["is_active"] != true {
        t.Errorf("Expected first member is_active true, got %v", member1["is_active"])
    }
    t.Logf("Team retrieved successfully: %s", w.Body.String())
}

func TestHandlers_GetTeam_NotFound(t *testing.T) {
    mock := &mockService{
        getTeamFunc: func(teamName string) (*entity.Team, []entity.User, error) {
            return nil, nil, entity.ErrNotFound
        },
    }
    handler := NewHandlers(mock)
    req := httptest.NewRequest("GET", "/team/get?team_name=nonexistent", nil)
    w := httptest.NewRecorder()
    handler.GetTeam(w, req)
    if w.Code != http.StatusNotFound {
        t.Errorf("Expected status 404, got %d", w.Code)
        return
    }
    var response map[string]interface{}
    err := json.Unmarshal(w.Body.Bytes(), &response)
    if err != nil {
        t.Fatalf("Failed to parse error response: %v", err)
    }
    errorData, exists := response["error"].(map[string]interface{})
    if !exists {
        t.Fatal("Error response must contain 'error' field")
    }
    errorCode, exists := errorData["code"].(string)
    if !exists {
        t.Fatal("Error must contain 'code' field")
    }
    if errorCode != "NOT_FOUND" {
        t.Errorf("Expected error code 'NOT_FOUND', got %v", errorCode)
    }
    t.Logf("Team not found error handled correctly: %s", w.Body.String())
}


func TestHandlers_SetUserActive_Success(t *testing.T) {
    mock := &mockService{
        setUserActiveFunc: func(userID string, isActive bool) (*entity.User, error) {
            return &entity.User{
                ID:       userID,
                Username: "Bob",
                TeamName: "backend",
                IsActive: isActive,
            }, nil
        },
    }
    handler := NewHandlers(mock)
    requestBody := map[string]interface{}{
        "user_id":   "u2",
        "is_active": false,
    }
    body, _ := json.Marshal(requestBody)
    req := httptest.NewRequest("POST", "/users/setIsActive", bytes.NewReader(body))
    w := httptest.NewRecorder()
    handler.SetUserActive(w, req)
    if w.Code != http.StatusOK {
        t.Errorf("Expected status 200, got %d", w.Code)
        return
    }
    var response map[string]interface{}
    err := json.Unmarshal(w.Body.Bytes(), &response)
    if err != nil {
        t.Fatalf("Failed to parse response: %v", err)
    }
    userData, exists := response["user"].(map[string]interface{})
    if !exists {
        t.Fatal("Response must contain 'user' field")
    }
    if userData["user_id"] != "u2" {
        t.Errorf("Expected user_id 'u2', got %v", userData["user_id"])
    }
    if userData["username"] != "Bob" {
        t.Errorf("Expected username 'Bob', got %v", userData["username"])
    }
    if userData["team_name"] != "backend" {
        t.Errorf("Expected team_name 'backend', got %v", userData["team_name"])
    }
    if userData["is_active"] != false {
        t.Errorf("Expected is_active false, got %v", userData["is_active"])
    }
    t.Logf("User active status updated successfully: %s", w.Body.String())
}

func TestHandlers_SetUserActive_UserNotFound(t *testing.T) {
    mock := &mockService{
        setUserActiveFunc: func(userID string, isActive bool) (*entity.User, error) {
            return nil, entity.ErrNotFound
        },
    }
    handler := NewHandlers(mock)
    requestBody := map[string]interface{}{
        "user_id":   "nonexistent",
        "is_active": true,
    }
    body, _ := json.Marshal(requestBody)
    req := httptest.NewRequest("POST", "/users/setIsActive", bytes.NewReader(body))
    w := httptest.NewRecorder()
    handler.SetUserActive(w, req)
    if w.Code != http.StatusNotFound {
        t.Errorf("Expected status 404, got %d", w.Code)
        return
    }
    var response map[string]interface{}
    err := json.Unmarshal(w.Body.Bytes(), &response)
    if err != nil {
        t.Fatalf("Failed to parse error response: %v", err)
    }
    errorData, exists := response["error"].(map[string]interface{})
    if !exists {
        t.Fatal("Error response must contain 'error' field")
    }
    errorCode, exists := errorData["code"].(string)
    if !exists {
        t.Fatal("Error must contain 'code' field")
    }
    if errorCode != "NOT_FOUND" {
        t.Errorf("Expected error code 'NOT_FOUND', got %v", errorCode)
    }
    t.Logf("User not found error handled correctly")
}

func TestHandlers_SetUserActive_InvalidJSON(t *testing.T) {
    mock := &mockService{}
    handler := NewHandlers(mock)
    req := httptest.NewRequest("POST", "/users/setIsActive", bytes.NewReader([]byte("invalid json")))
    w := httptest.NewRecorder()
    handler.SetUserActive(w, req)
    if w.Code != http.StatusBadRequest {
        t.Errorf("Expected status 400, got %d", w.Code)
        return
    }
    var response map[string]interface{}
    json.Unmarshal(w.Body.Bytes(), &response)
    errorData := response["error"].(map[string]interface{})
    if errorData["code"] != "INVALID_REQUEST" {
        t.Errorf("Expected error code 'INVALID_REQUEST', got %v", errorData["code"])
    }
    t.Logf("Invalid JSON handled correctly")
}

func TestHandlers_CreatePR_Success(t *testing.T) {
    mock := &mockService{
        createPRFunc: func(prID, title, authorID string) (*entity.PullRequest, error) {
            return &entity.PullRequest{
                ID:       prID,
                Title:    title,
                AuthorID: authorID,
                Status:   "OPEN",
                AssignedReviewers: []entity.User{
                    {ID: "u2", Username: "Bob", IsActive: true},
                    {ID: "u3", Username: "Charlie", IsActive: true},
                },
            }, nil
        },
    }
    handler := NewHandlers(mock)
    requestBody := map[string]interface{}{
        "pull_request_id":   "pr-1001",
        "pull_request_name": "Add search",
        "author_id":         "u1",
    }
    body, _ := json.Marshal(requestBody)
    req := httptest.NewRequest("POST", "/pullRequest/create", bytes.NewReader(body))
    w := httptest.NewRecorder()
    handler.CreatePR(w, req)
    if w.Code != http.StatusCreated {
        t.Errorf("Expected status 201, got %d", w.Code)
        t.Logf("Response: %s", w.Body.String())
        return
    }
    var response map[string]interface{}
    err := json.Unmarshal(w.Body.Bytes(), &response)
    if err != nil {
        t.Fatalf("Failed to parse response: %v", err)
    }
    prData, exists := response["pr"].(map[string]interface{})
    if !exists {
        t.Fatal("Response must contain 'pr' field")
    }
    if prData["pull_request_id"] != "pr-1001" {
        t.Errorf("Expected pull_request_id 'pr-1001', got %v", prData["pull_request_id"])
    }
    if prData["pull_request_name"] != "Add search" {
        t.Errorf("Expected pull_request_name 'Add search', got %v", prData["pull_request_name"])
    }
    if prData["author_id"] != "u1" {
        t.Errorf("Expected author_id 'u1', got %v", prData["author_id"])
    }
    if prData["status"] != "OPEN" {
        t.Errorf("Expected status 'OPEN', got %v", prData["status"])
    }
    reviewers, exists := prData["assigned_reviewers"].([]interface{})
    if !exists {
        t.Fatal("PR must contain 'assigned_reviewers' field")
    }
    if len(reviewers) != 2 {
        t.Errorf("Expected 2 assigned reviewers, got %d", len(reviewers))
    }
    t.Logf("PR created successfully: %s", w.Body.String())
}

func TestHandlers_CreatePR_AlreadyExists(t *testing.T) {
    mock := &mockService{
        createPRFunc: func(prID, title, authorID string) (*entity.PullRequest, error) {
            return nil, entity.ErrPRExists
        },
    }
    handler := NewHandlers(mock)
    requestBody := map[string]interface{}{
        "pull_request_id":   "pr-1001",
        "pull_request_name": "Add search",
        "author_id":         "u1",
    }
    body, _ := json.Marshal(requestBody)
    req := httptest.NewRequest("POST", "/pullRequest/create", bytes.NewReader(body))
    w := httptest.NewRecorder()
    handler.CreatePR(w, req)
    if w.Code != http.StatusConflict {
        t.Errorf("Expected status 409, got %d", w.Code)
        return
    }
    var response map[string]interface{}
    err := json.Unmarshal(w.Body.Bytes(), &response)
    if err != nil {
        t.Fatalf("Failed to parse error response: %v", err)
    }
    errorData, exists := response["error"].(map[string]interface{})
    if !exists {
        t.Fatal("Error response must contain 'error' field")
    }
    if errorData["code"] != "PR_EXISTS" {
        t.Errorf("Expected error code 'PR_EXISTS', got %v", errorData["code"])
    }
    t.Logf("PR already exists error handled correctly")
}

func TestHandlers_CreatePR_AuthorNotFound(t *testing.T) {
    mock := &mockService{
        createPRFunc: func(prID, title, authorID string) (*entity.PullRequest, error) {
            return nil, entity.ErrNotFound
        },
    }
    handler := NewHandlers(mock)
    requestBody := map[string]interface{}{
        "pull_request_id":   "pr-1001",
        "pull_request_name": "Add search",
        "author_id":         "nonexistent",
    }
    body, _ := json.Marshal(requestBody)
    req := httptest.NewRequest("POST", "/pullRequest/create", bytes.NewReader(body))
    w := httptest.NewRecorder()
    handler.CreatePR(w, req)
    if w.Code != http.StatusNotFound {
        t.Errorf("Expected status 404, got %d", w.Code)
        return
    }
    var response map[string]interface{}
    json.Unmarshal(w.Body.Bytes(), &response)
    errorData := response["error"].(map[string]interface{})
    if errorData["code"] != "NOT_FOUND" {
        t.Errorf("Expected error code 'NOT_FOUND', got %v", errorData["code"])
    }
    t.Logf("Author not found error handled correctly")
}

func TestHandlers_CreatePR_NoCandidateReviewers(t *testing.T) {
    mock := &mockService{
        createPRFunc: func(prID, title, authorID string) (*entity.PullRequest, error) {
            return nil, entity.ErrNoCandidate
        },
    }
    handler := NewHandlers(mock)
    requestBody := map[string]interface{}{
        "pull_request_id":   "pr-1001",
        "pull_request_name": "Add search",
        "author_id":         "u1",
    }
    body, _ := json.Marshal(requestBody)
    req := httptest.NewRequest("POST", "/pullRequest/create", bytes.NewReader(body))
    w := httptest.NewRecorder()
    handler.CreatePR(w, req)
    if w.Code != http.StatusNotFound {
        t.Errorf("Expected status 404, got %d", w.Code)
        return
    }
    var response map[string]interface{}
    json.Unmarshal(w.Body.Bytes(), &response)
    errorData := response["error"].(map[string]interface{})
    if errorData["code"] != "NO_CANDIDATE" {
        t.Errorf("Expected error code 'NO_CANDIDATE', got %v", errorData["code"])
    }
    t.Logf("No candidate reviewers error handled correctly")
}

func TestHandlers_MergePR_Success(t *testing.T) {
    mock := &mockService{
        mergePRFunc: func(prID string) (*entity.PullRequest, error) {
            mergedAt := "2025-10-24T12:34:56Z"
            return &entity.PullRequest{
                ID:       prID,
                Title:    "Add search",
                AuthorID: "u1",
                Status:   "MERGED",
                AssignedReviewers: []entity.User{
                    {ID: "u2", Username: "Bob", IsActive: true},
                    {ID: "u3", Username: "Charlie", IsActive: true},
                },
                MergedAt: &mergedAt,
            }, nil
        },
    }
    handler := NewHandlers(mock)
    requestBody := map[string]interface{}{
        "pull_request_id": "pr-1001",
    }
    body, _ := json.Marshal(requestBody)
    req := httptest.NewRequest("POST", "/pullRequest/merge", bytes.NewReader(body))
    w := httptest.NewRecorder()
    handler.MergePR(w, req)
    if w.Code != http.StatusOK {
        t.Errorf("Expected status 200, got %d", w.Code)
        t.Logf("Response: %s", w.Body.String())
        return
    }
    var response map[string]interface{}
    err := json.Unmarshal(w.Body.Bytes(), &response)
    if err != nil {
        t.Fatalf("Failed to parse response: %v", err)
    }
    prData, exists := response["pr"].(map[string]interface{})
    if !exists {
        t.Fatal("Response must contain 'pr' field")
    }
    if prData["pull_request_id"] != "pr-1001" {
        t.Errorf("Expected pull_request_id 'pr-1001', got %v", prData["pull_request_id"])
    }
    if prData["status"] != "MERGED" {
        t.Errorf("Expected status 'MERGED', got %v", prData["status"])
    }
    if prData["mergedAt"] == nil {
        t.Error("Merged PR should have 'mergedAt' field")
    }
    t.Logf("PR merged successfully: %s", w.Body.String())
}

func TestHandlers_MergePR_NotFound(t *testing.T) {
    mock := &mockService{
        mergePRFunc: func(prID string) (*entity.PullRequest, error) {
            return nil, entity.ErrNotFound
        },
    }
    handler := NewHandlers(mock)
    requestBody := map[string]interface{}{
        "pull_request_id": "nonexistent-pr",
    }
    body, _ := json.Marshal(requestBody)
    req := httptest.NewRequest("POST", "/pullRequest/merge", bytes.NewReader(body))
    w := httptest.NewRecorder()
    handler.MergePR(w, req)
    if w.Code != http.StatusNotFound {
        t.Errorf("Expected status 404, got %d", w.Code)
        return
    }
    var response map[string]interface{}
    json.Unmarshal(w.Body.Bytes(), &response)
    errorData := response["error"].(map[string]interface{})
    if errorData["code"] != "NOT_FOUND" {
        t.Errorf("Expected error code 'NOT_FOUND', got %v", errorData["code"])
    }
    t.Logf("PR not found error handled correctly")
}

func TestHandlers_ReassignReviewer_Success(t *testing.T) {
    mock := &mockService{
        reassignReviewerFunc: func(prID, oldUserID string) (*entity.PullRequest, string, error) {
            return &entity.PullRequest{
                ID:       prID,
                Title:    "Add search",
                AuthorID: "u1",
                Status:   "OPEN",
                AssignedReviewers: []entity.User{
                    {ID: "u3", Username: "Charlie", IsActive: true},
                    {ID: "u5", Username: "Eve", IsActive: true},
                },
            }, "u5", nil
        },
    }
    handler := NewHandlers(mock)
    requestBody := map[string]interface{}{
        "pull_request_id": "pr-1001",
        "old_user_id":     "u2",
    }
    body, _ := json.Marshal(requestBody)
    req := httptest.NewRequest("POST", "/pullRequest/reassign", bytes.NewReader(body))
    w := httptest.NewRecorder()
    handler.ReassignReviewer(w, req)
    if w.Code != http.StatusOK {
        t.Errorf("Expected status 200, got %d", w.Code)
        t.Logf("Response: %s", w.Body.String())
        return
    }
    var response map[string]interface{}
    err := json.Unmarshal(w.Body.Bytes(), &response)
    if err != nil {
        t.Fatalf("Failed to parse response: %v", err)
    }
    if response["replaced_by"] != "u5" {
        t.Errorf("Expected replaced_by 'u5', got %v", response["replaced_by"])
    }
    prData, exists := response["pr"].(map[string]interface{})
    if !exists {
        t.Fatal("Response must contain 'pr' field")
    }
    if prData["pull_request_id"] != "pr-1001" {
        t.Errorf("Expected pull_request_id 'pr-1001', got %v", prData["pull_request_id"])
    }
    if prData["status"] != "OPEN" {
        t.Errorf("Expected status 'OPEN', got %v", prData["status"])
    }
    t.Logf("Reviewer reassigned successfully: %s", w.Body.String())
}

func TestHandlers_ReassignReviewer_PRNotFound(t *testing.T) {
    mock := &mockService{
        reassignReviewerFunc: func(prID, oldUserID string) (*entity.PullRequest, string, error) {
            return nil, "", entity.ErrNotFound
        },
    }
    handler := NewHandlers(mock)
    requestBody := map[string]interface{}{
        "pull_request_id": "nonexistent-pr",
        "old_user_id":     "u2",
    }
    body, _ := json.Marshal(requestBody)
    req := httptest.NewRequest("POST", "/pullRequest/reassign", bytes.NewReader(body))
    w := httptest.NewRecorder()
    handler.ReassignReviewer(w, req)
    if w.Code != http.StatusNotFound {
        t.Errorf("Expected status 404, got %d", w.Code)
        return
    }
    var response map[string]interface{}
    json.Unmarshal(w.Body.Bytes(), &response)
    errorData := response["error"].(map[string]interface{})
    if errorData["code"] != "NOT_FOUND" {
        t.Errorf("Expected error code 'NOT_FOUND', got %v", errorData["code"])
    }
    t.Logf("PR not found error handled correctly")
}

func TestHandlers_ReassignReviewer_PRAlreadyMerged(t *testing.T) {
    mock := &mockService{
        reassignReviewerFunc: func(prID, oldUserID string) (*entity.PullRequest, string, error) {
            return nil, "", entity.ErrPRMerged
        },
    }
    handler := NewHandlers(mock)
    requestBody := map[string]interface{}{
        "pull_request_id": "pr-1001",
        "old_user_id":     "u2",
    }
    body, _ := json.Marshal(requestBody)
    req := httptest.NewRequest("POST", "/pullRequest/reassign", bytes.NewReader(body))
    w := httptest.NewRecorder()
    handler.ReassignReviewer(w, req)
    if w.Code != http.StatusConflict {
        t.Errorf("Expected status 409, got %d", w.Code)
        return
    }
    var response map[string]interface{}
    json.Unmarshal(w.Body.Bytes(), &response)
    errorData := response["error"].(map[string]interface{})
    if errorData["code"] != "PR_MERGED" {
        t.Errorf("Expected error code 'PR_MERGED', got %v", errorData["code"])
    }
    t.Logf("PR merged error handled correctly")
}

func TestHandlers_ReassignReviewer_ReviewerNotAssigned(t *testing.T) {
    mock := &mockService{
        reassignReviewerFunc: func(prID, oldUserID string) (*entity.PullRequest, string, error) {
            return nil, "", entity.ErrNotAssigned
        },
    }
    handler := NewHandlers(mock)
    requestBody := map[string]interface{}{
        "pull_request_id": "pr-1001",
        "old_user_id":     "u9",
    }
    body, _ := json.Marshal(requestBody)
    req := httptest.NewRequest("POST", "/pullRequest/reassign", bytes.NewReader(body))
    w := httptest.NewRecorder()
    handler.ReassignReviewer(w, req)
    if w.Code != http.StatusConflict {
        t.Errorf("Expected status 409, got %d", w.Code)
        return
    }
    var response map[string]interface{}
    json.Unmarshal(w.Body.Bytes(), &response)
    errorData := response["error"].(map[string]interface{})
    if errorData["code"] != "NOT_ASSIGNED" {
        t.Errorf("Expected error code 'NOT_ASSIGNED', got %v", errorData["code"])
    }
    t.Logf("Reviewer not assigned error handled correctly")
}

func TestHandlers_ReassignReviewer_NoCandidate(t *testing.T) {
    mock := &mockService{
        reassignReviewerFunc: func(prID, oldUserID string) (*entity.PullRequest, string, error) {
            return nil, "", entity.ErrNoCandidate
        },
    }
    handler := NewHandlers(mock)
    requestBody := map[string]interface{}{
        "pull_request_id": "pr-1001",
        "old_user_id":     "u2",
    }
    body, _ := json.Marshal(requestBody)
    req := httptest.NewRequest("POST", "/pullRequest/reassign", bytes.NewReader(body))
    w := httptest.NewRecorder()
    handler.ReassignReviewer(w, req)
    if w.Code != http.StatusConflict {
        t.Errorf("Expected status 409, got %d", w.Code)
        return
    }
    var response map[string]interface{}
    json.Unmarshal(w.Body.Bytes(), &response)
    errorData := response["error"].(map[string]interface{})
    if errorData["code"] != "NO_CANDIDATE" {
        t.Errorf("Expected error code 'NO_CANDIDATE', got %v", errorData["code"])
    }
    t.Logf("No candidate error handled correctly")
}

func TestHandlers_GetUserReviewPRs_Success(t *testing.T) {
    mock := &mockService{
        getUserReviewPRsFunc: func(userID string) ([]entity.PullRequest, error) {
            return []entity.PullRequest{}, nil
        },
    }
    handler := NewHandlers(mock)
    req := httptest.NewRequest("GET", "/users/getReview?user_id=u2", nil)
    w := httptest.NewRecorder()
    handler.GetUserReviewPRs(w, req)
    if w.Code != http.StatusOK {
        t.Errorf("Expected status 200, got %d", w.Code)
        t.Logf("Response: %s", w.Body.String())
        return
    }
    var response map[string]interface{}
    err := json.Unmarshal(w.Body.Bytes(), &response)
    if err != nil {
        t.Fatalf("Failed to parse response: %v", err)
    }
    if response["user_id"] != "u2" {
        t.Errorf("Expected user_id 'u2', got %v", response["user_id"])
    }
    prsData, exists := response["pull_requests"].([]interface{})
    if !exists {
        t.Fatal("Response must contain 'pull_requests' field")
    }
    if len(prsData) != 0 {
        t.Errorf("Expected 0 pull requests for new user, got %d", len(prsData))
    }
    t.Logf("User u2 has no PRs for review - correct behavior")
    t.Logf("Response: %s", w.Body.String())
}

func TestHandlers_GetStats_Success(t *testing.T) {
    mockStats := &entity.Stats{
        TotalAssignments: 150,
        UserAssignmentCounts: []entity.UserAssignmentCount{
            {
                UserID:   "u123",
                Username: "alice",
                Count:    45,
            },
            {
                UserID:   "u456",
                Username: "bob",
                Count:    38,
            },
            {
                UserID:   "u789",
                Username: "charlie",
                Count:    27,
            },
        },
        PRAssignmentCounts: []entity.PRAssignmentCount{
            {
                PRID:  "pr-1001",
                Title: "Add payment feature",
                Count: 8,
            },
            {
                PRID:  "pr-1002",
                Title: "Fix authentication bug",
                Count: 6,
            },
            {
                PRID:  "pr-1003",
                Title: "Update database schema",
                Count: 5,
            },
        },
    }
    mock := &mockService{
        getStatsFunc: func() (*entity.Stats, error) {
            return mockStats, nil
        },
    }
    handler := NewHandlers(mock)
    req := httptest.NewRequest("GET", "/stats", nil)
    w := httptest.NewRecorder()
    handler.GetStats(w, req)
    if w.Code != http.StatusOK {
        t.Errorf("Expected status 200, got %d", w.Code)
        t.Logf("Response: %s", w.Body.String())
        return
    }
    var response map[string]interface{}
    err := json.Unmarshal(w.Body.Bytes(), &response)
    if err != nil {
        t.Fatalf("Failed to parse response: %v", err)
    }
    statsData, exists := response["stats"].(map[string]interface{})
    if !exists {
        t.Fatal("Response must contain 'stats' field")
    }
    if statsData["total_assignments"] != float64(150) {
        t.Errorf("Expected total_assignments 150, got %v", statsData["total_assignments"])
    }
    usersData, exists := statsData["user_assignment_counts"].([]interface{})
    if !exists {
        t.Fatal("Stats must contain 'user_assignment_counts' field")
    }
    if len(usersData) != 3 {
        t.Errorf("Expected 3 user assignment counts, got %d", len(usersData))
    }
    if len(usersData) > 0 {
        user1 := usersData[0].(map[string]interface{})
        if user1["user_id"] != "u123" {
            t.Errorf("Expected first user_id 'u123', got %v", user1["user_id"])
        }
        if user1["username"] != "alice" {
            t.Errorf("Expected first username 'alice', got %v", user1["username"])
        }
        if user1["count"] != float64(45) {
            t.Errorf("Expected first user count 45, got %v", user1["count"])
        }
    }
    prsData, exists := statsData["pr_assignment_counts"].([]interface{})
    if !exists {
        t.Fatal("Stats must contain 'pr_assignment_counts' field")
    }

    if len(prsData) != 3 {
        t.Errorf("Expected 3 PR assignment counts, got %d", len(prsData))
    }
    if len(prsData) > 0 {
        pr1 := prsData[0].(map[string]interface{})
        if pr1["pull_request_id"] != "pr-1001" {
            t.Errorf("Expected first PR ID 'pr-1001', got %v", pr1["pull_request_id"])
        }
        if pr1["pull_request_name"] != "Add payment feature" {
            t.Errorf("Expected first PR title 'Add payment feature', got %v", pr1["pull_request_name"])
        }
        if pr1["count"] != float64(8) {
            t.Errorf("Expected first PR count 8, got %v", pr1["count"])
        }
    }
    t.Logf("Stats retrieved successfully: %s", w.Body.String())
}

func TestHandlers_GetStats_EmptyData(t *testing.T) {
    mockStats := &entity.Stats{
        TotalAssignments:     0,
        UserAssignmentCounts: []entity.UserAssignmentCount{},
        PRAssignmentCounts:   []entity.PRAssignmentCount{},
    }
    mock := &mockService{
        getStatsFunc: func() (*entity.Stats, error) {
            return mockStats, nil
        },
    }
    handler := NewHandlers(mock)
    req := httptest.NewRequest("GET", "/stats", nil)
    w := httptest.NewRecorder()
    handler.GetStats(w, req)
    if w.Code != http.StatusOK {
        t.Errorf("Expected status 200, got %d", w.Code)
        return
    }
    var response map[string]interface{}
    err := json.Unmarshal(w.Body.Bytes(), &response)
    if err != nil {
        t.Fatalf("Failed to parse response: %v", err)
    }
    statsData, exists := response["stats"].(map[string]interface{})
    if !exists {
        t.Fatal("Response must contain 'stats' field")
    }
    if statsData["total_assignments"] != float64(0) {
        t.Errorf("Expected total_assignments 0, got %v", statsData["total_assignments"])
    }
    usersData, exists := statsData["user_assignment_counts"].([]interface{})
    if !exists {
        t.Fatal("Stats must contain 'user_assignment_counts' field")
    }
    if len(usersData) != 0 {
        t.Errorf("Expected 0 user assignment counts, got %d", len(usersData))
    }
    prsData, exists := statsData["pr_assignment_counts"].([]interface{})
    if !exists {
        t.Fatal("Stats must contain 'pr_assignment_counts' field")
    }
    if len(prsData) != 0 {
        t.Errorf("Expected 0 PR assignment counts, got %d", len(prsData))
    }
    t.Logf("Empty stats handled correctly: %s", w.Body.String())
}

func TestHandlers_GetStats_ServiceError(t *testing.T) {
    mock := &mockService{
        getStatsFunc: func() (*entity.Stats, error) {
            return nil, entity.ErrNotFound
        },
    }
    handler := NewHandlers(mock)
    req := httptest.NewRequest("GET", "/stats", nil)
    w := httptest.NewRecorder()
    handler.GetStats(w, req)
    if w.Code != http.StatusInternalServerError {
        t.Errorf("Expected status 500, got %d", w.Code)
        return
    }
    var response map[string]interface{}
    err := json.Unmarshal(w.Body.Bytes(), &response)
    if err != nil {
        t.Fatalf("Failed to parse error response: %v", err)
    }
    errorData, exists := response["error"].(map[string]interface{})
    if !exists {
        t.Fatal("Error response must contain 'error' field")
    }
    errorCode, exists := errorData["code"].(string)
    if !exists {
        t.Fatal("Error must contain 'code' field")
    }
    if errorCode != "INTERNAL_ERROR" {
        t.Errorf("Expected error code 'INTERNAL_ERROR', got %v", errorCode)
    }
    t.Logf("Service error handled correctly: %s", w.Body.String())
}

func TestHandlers_GetStats_SingleUserAndPR(t *testing.T) {
    mockStats := &entity.Stats{
        TotalAssignments: 15,
        UserAssignmentCounts: []entity.UserAssignmentCount{
            {
                UserID:   "u999",
                Username: "sole_reviewer",
                Count:    15,
            },
        },
        PRAssignmentCounts: []entity.PRAssignmentCount{
            {
                PRID:  "pr-5001",
                Title: "Initial commit",
                Count: 3,
            },
        },
    }
    mock := &mockService{
        getStatsFunc: func() (*entity.Stats, error) {
            return mockStats, nil
        },
    }
    handler := NewHandlers(mock)
    req := httptest.NewRequest("GET", "/stats", nil)
    w := httptest.NewRecorder()
    handler.GetStats(w, req)
    if w.Code != http.StatusOK {
        t.Errorf("Expected status 200, got %d", w.Code)
        return
    }
    var response map[string]interface{}
    err := json.Unmarshal(w.Body.Bytes(), &response)
    if err != nil {
        t.Fatalf("Failed to parse response: %v", err)
    }
    statsData := response["stats"].(map[string]interface{})
    usersData := statsData["user_assignment_counts"].([]interface{})
    if len(usersData) != 1 {
        t.Errorf("Expected 1 user assignment count, got %d", len(usersData))
    }
    user := usersData[0].(map[string]interface{})
    if user["count"] != float64(15) {
        t.Errorf("Expected user count 15, got %v", user["count"])
    }
    prsData := statsData["pr_assignment_counts"].([]interface{})
    if len(prsData) != 1 {
        t.Errorf("Expected 1 PR assignment count, got %d", len(prsData))
    }
    t.Logf("Single user/PR stats retrieved successfully: %s", w.Body.String())
}

func TestHandlers_GetStats_LargeDataset(t *testing.T) {
    userCounts := make([]entity.UserAssignmentCount, 50)
    prCounts := make([]entity.PRAssignmentCount, 100)
    for i := 0; i < 50; i++ {
        userCounts[i] = entity.UserAssignmentCount{
            UserID:   fmt.Sprintf("u%d", i+1),
            Username: fmt.Sprintf("user%d", i+1),
            Count:    i + 1,
        }
    }
    for i := 0; i < 100; i++ {
        prCounts[i] = entity.PRAssignmentCount{
            PRID:  fmt.Sprintf("pr-%d", i+1),
            Title: fmt.Sprintf("Feature %d", i+1),
            Count: (i % 10) + 1,
        }
    }
    mockStats := &entity.Stats{
        TotalAssignments:     1275,
        UserAssignmentCounts: userCounts,
        PRAssignmentCounts:   prCounts,
    }
    mock := &mockService{
        getStatsFunc: func() (*entity.Stats, error) {
            return mockStats, nil
        },
    }
    handler := NewHandlers(mock)
    req := httptest.NewRequest("GET", "/stats", nil)
    w := httptest.NewRecorder()
    handler.GetStats(w, req)
    if w.Code != http.StatusOK {
        t.Errorf("Expected status 200, got %d", w.Code)
        return
    }
    var response map[string]interface{}
    err := json.Unmarshal(w.Body.Bytes(), &response)
    if err != nil {
        t.Fatalf("Failed to parse response: %v", err)
    }
    statsData := response["stats"].(map[string]interface{})
    usersData := statsData["user_assignment_counts"].([]interface{})
    if len(usersData) != 50 {
        t.Errorf("Expected 50 user assignment counts, got %d", len(usersData))
    }
    prsData := statsData["pr_assignment_counts"].([]interface{})
    if len(prsData) != 100 {
        t.Errorf("Expected 100 PR assignment counts, got %d", len(prsData))
    }
    t.Logf("Large dataset handled successfully: %d users, %d PRs", len(usersData), len(prsData))
}

func TestHandlers_MethodNotAllowed(t *testing.T) {
    mock := &mockService{}
    handler := NewHandlers(mock)
    testCases := []struct {
        method string
        path   string
    }{
        {"PUT", "/teams"},
        {"DELETE", "/teams"},
        {"PATCH", "/teams"},
        {"PUT", "/users/setIsActive"},
        {"GET", "/users/setIsActive"},
        {"PUT", "/pullRequest/create"},
        {"GET", "/pullRequest/create"},
    }
    for _, tc := range testCases {
        t.Run(tc.method+tc.path, func(t *testing.T) {
            req := httptest.NewRequest(tc.method, tc.path, nil)
            w := httptest.NewRecorder()
            switch tc.path {
            case "/teams":
                handler.AddTeam(w, req)
            case "/users/setIsActive":
                handler.SetUserActive(w, req)
            case "/pullRequest/create":
                handler.CreatePR(w, req)
            }
            if w.Code >= 200 && w.Code < 300 {
                t.Errorf("Expected error status for %s %s, got %d", tc.method, tc.path, w.Code)
            }
        })
    }
}