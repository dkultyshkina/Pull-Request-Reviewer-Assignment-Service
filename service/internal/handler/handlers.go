package handlers

import (
    "encoding/json"
    "net/http"

    "service/internal/service"
	"service/internal/entity"
)

type ErrorResponse struct {
    Error struct {
        Code    string `json:"code"`
        Message string `json:"message"`
    } `json:"error"`
}

type Handlers struct {
    service service.Service  
}

func NewHandlers(service service.Service) *Handlers {  
    return &Handlers{service: service}
}

func (h *Handlers) writeError(w http.ResponseWriter, code int, errorCode, message string) {
    w.WriteHeader(code)
    json.NewEncoder(w).Encode(ErrorResponse{
        Error: struct {
            Code    string `json:"code"`
            Message string `json:"message"`
        }{
            Code:    errorCode,
            Message: message,
        },
    })
}

func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "OK",
	})
}

func (h *Handlers) AddTeam(w http.ResponseWriter, r *http.Request) {
    var request struct {
        TeamName string            `json:"team_name"`
        Members  []entity.User `json:"members"`
    }
    if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
        h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
        return
    }
    team, err := h.service.CreateTeam(request.TeamName, request.Members)
    if err != nil {
        switch err {
        case entity.ErrTeamExists:
            h.writeError(w, http.StatusBadRequest, "TEAM_EXISTS", "team already exists")
        default:
            h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
        }
        return
    }
	type TeamResponse struct {
		TeamName string        `json:"team_name"`
		Members  []entity.User `json:"members"`
	}
	type AddTeamResponse struct {
		Team TeamResponse `json:"team"`
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(AddTeamResponse{
		Team: TeamResponse{
			TeamName: team.Name,
			Members:  request.Members,
		},
	})
}

func (h *Handlers) GetTeam(w http.ResponseWriter, r *http.Request) {
    teamName := r.URL.Query().Get("team_name")
    if teamName == "" {
        h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "team_name is required")
        return
    }
    team, members, err := h.service.GetTeam(teamName)
    if err != nil {
        if err == entity.ErrNotFound {
            h.writeError(w, http.StatusNotFound, "NOT_FOUND", "team not found")
        } else {
            h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
        }
        return
    }
	type TeamResponse struct {
		TeamName string        `json:"team_name"`
		Members  []entity.User `json:"members"`
	}
	response := TeamResponse{
		TeamName: team.Name,
		Members:  members,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *Handlers) SetUserActive(w http.ResponseWriter, r *http.Request) {
    var request struct {
        UserID   string `json:"user_id"`
        IsActive *bool   `json:"is_active"`
    }
    if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
        h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
        return
    }
    if request.UserID == "" {
        h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "user_id is required")
        return
    }
    user, err := h.service.SetUserActive(request.UserID, *request.IsActive)
    if err != nil {
        if err == entity.ErrNotFound {
            h.writeError(w, http.StatusNotFound, "NOT_FOUND", "user not found")
        } else {
            h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
        }
        return
    }
	type UserResponse struct {
		UserID   string `json:"user_id"`
		Username string `json:"username"`
		TeamName string `json:"team_name"`
		IsActive bool   `json:"is_active"`
	}
	type SetUserActiveResponse struct {
		User UserResponse `json:"user"`
	}
	json.NewEncoder(w).Encode(SetUserActiveResponse{
		User: UserResponse{
			UserID:   user.ID,
			Username: user.Username,
			TeamName: user.TeamName,
			IsActive: user.IsActive,
		},
	})
}

func (h *Handlers) CreatePR(w http.ResponseWriter, r *http.Request) {
    var request struct {
        PRID     string `json:"pull_request_id"`
        PRName   string `json:"pull_request_name"`
        AuthorID string `json:"author_id"`
    }
    if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
        h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
        return
    }
    pr, err := h.service.CreatePR(request.PRID, request.PRName, request.AuthorID)
    if err != nil {
        switch err {
        case entity.ErrPRExists:
            h.writeError(w, http.StatusConflict, "PR_EXISTS", "pull request already exists")
        case entity.ErrNotFound:
            h.writeError(w, http.StatusNotFound, "NOT_FOUND", "author or team not found")
        case entity.ErrNoCandidate:
            h.writeError(w, http.StatusNotFound, "NO_CANDIDATE", "no active reviewers available in team")
        default:
            h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
        }
        return
    }
	type PRResponse struct {
		PullRequestID    string   `json:"pull_request_id"`
		PullRequestName  string   `json:"pull_request_name"`
		AuthorID         string   `json:"author_id"`
		Status           string   `json:"status"`
		AssignedReviewers []string `json:"assigned_reviewers"`
	}
	type CreatePRResponse struct {
		PR PRResponse `json:"pr"`
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(CreatePRResponse{
		PR: PRResponse{
			PullRequestID:    pr.ID,
			PullRequestName:  pr.Title,
			AuthorID:         pr.AuthorID,
			Status:           pr.Status,
			AssignedReviewers: getReviewerIDs(pr.AssignedReviewers),
		},
	})
}

func (h *Handlers) MergePR(w http.ResponseWriter, r *http.Request) {
    var request struct {
        PRID string `json:"pull_request_id"`
    }
    if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
        h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
        return
    }
    pr, err := h.service.MergePR(request.PRID)
    if err != nil {
        if err == entity.ErrNotFound {
            h.writeError(w, http.StatusNotFound, "NOT_FOUND", "pull request not found")
        } else {
            h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
        }
        return
    }
	json.NewEncoder(w).Encode(struct {
		PR struct {
			PullRequestID    string   `json:"pull_request_id"`
			PullRequestName  string   `json:"pull_request_name"`
			AuthorID         string   `json:"author_id"`
			Status           string   `json:"status"`
			AssignedReviewers []string `json:"assigned_reviewers"`
			MergedAt         interface{} `json:"mergedAt"`
		} `json:"pr"`
	}{
		PR: struct {
			PullRequestID    string   `json:"pull_request_id"`
			PullRequestName  string   `json:"pull_request_name"`
			AuthorID         string   `json:"author_id"`
			Status           string   `json:"status"`
			AssignedReviewers []string `json:"assigned_reviewers"`
			MergedAt         interface{} `json:"mergedAt"`
		}{
			PullRequestID:    pr.ID,
			PullRequestName:  pr.Title,
			AuthorID:         pr.AuthorID,
			Status:           pr.Status,
			AssignedReviewers: getReviewerIDs(pr.AssignedReviewers),
			MergedAt:         pr.MergedAt,
		},
	})
}

func (h *Handlers) ReassignReviewer(w http.ResponseWriter, r *http.Request) {
    var request struct {
        PRID      string `json:"pull_request_id"`
        OldUserID string `json:"old_user_id"`
    }
    if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
        h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
        return
    }
    pr, newUserID, err := h.service.ReassignReviewer(request.PRID, request.OldUserID)
    if err != nil {
        switch err {
        case entity.ErrNotFound:
            h.writeError(w, http.StatusNotFound, "NOT_FOUND", "pull request or user not found")
        case entity.ErrPRMerged:
            h.writeError(w, http.StatusConflict, "PR_MERGED", "cannot reassign on merged PR")
        case entity.ErrNotAssigned:
            h.writeError(w, http.StatusConflict, "NOT_ASSIGNED", "reviewer is not assigned to this PR")
        case entity.ErrNoCandidate:
            h.writeError(w, http.StatusConflict, "NO_CANDIDATE", "no active replacement candidate in team")
        default:
            h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
        }
        return
    }
	type PRResponse struct {
		PullRequestID    string   `json:"pull_request_id"`
		PullRequestName  string   `json:"pull_request_name"`
		AuthorID         string   `json:"author_id"`
		Status           string   `json:"status"`
		AssignedReviewers []string `json:"assigned_reviewers"`
	}
	type ReassignReviewerResponse struct {
		PR         PRResponse `json:"pr"`
		ReplacedBy string     `json:"replaced_by"`
	}
	json.NewEncoder(w).Encode(ReassignReviewerResponse{
		PR: PRResponse{
			PullRequestID:    pr.ID,
			PullRequestName:  pr.Title,
			AuthorID:         pr.AuthorID,
			Status:           pr.Status,
			AssignedReviewers: getReviewerIDs(pr.AssignedReviewers),
		},
		ReplacedBy: newUserID,
	})
}

func (h *Handlers) GetUserReviewPRs(w http.ResponseWriter, r *http.Request) {
    userID := r.URL.Query().Get("user_id")
    if userID == "" {
        h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "user_id is required")
        return
    }
    prs, err := h.service.GetUserReviewPRs(userID)
    if err != nil {
        h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
        return
    }
	type PullRequestShort struct {
		PullRequestID   string `json:"pull_request_id"`
		PullRequestName string `json:"pull_request_name"`
		AuthorID        string `json:"author_id"`
		Status          string `json:"status"`
	}
	type UserReviewResponse struct {
		UserID       string             `json:"user_id"`
		PullRequests []PullRequestShort `json:"pull_requests"`
	}
	shortPRs := make([]PullRequestShort, len(prs))
	for i, pr := range prs {
		shortPRs[i] = PullRequestShort{
			PullRequestID:   pr.ID,
			PullRequestName: pr.Title,
			AuthorID:        pr.AuthorID,
			Status:          pr.Status,
		}
	}
	response := UserReviewResponse{
        UserID:       userID,
        PullRequests: shortPRs,
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func getReviewerIDs(reviewers []entity.User) []string {
    ids := make([]string, len(reviewers))
    for i, reviewer := range reviewers {
        ids[i] = reviewer.ID
    }
    return ids
}

func (h *Handlers) GetStats(w http.ResponseWriter, r *http.Request) {
    stats, err := h.service.GetStats()
    if err != nil {
        h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
        return
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "stats": stats,
    })
}