package service

import (
	"fmt"

	"service/internal/entity"
	"service/internal/repository"
)

type Service interface {
	CreateTeam(teamName string, members []entity.User) (*entity.Team, error)
	GetTeam(teamName string) (*entity.Team, []entity.User, error)
	SetUserActive(userID string, isActive bool) (*entity.User, error)
	GetUserReviewPRs(userID string) ([]entity.PullRequest, error)
	CreatePR(prID, title, authorID string) (*entity.PullRequest, error)
	MergePR(prID string) (*entity.PullRequest, error)
	ReassignReviewer(prID, oldUserID string) (*entity.PullRequest, string, error)
	GetPR(prID string) (*entity.PullRequest, error)
	GetStats() (*entity.Stats, error)
}

type ServiceImpl struct {
	repo repository.Repository
}

func NewService(repo repository.Repository) Service {  
	return &ServiceImpl{repo: repo}
}

func (s *ServiceImpl) CreateTeam(teamName string, members []entity.User) (*entity.Team, error) {
	team := &entity.Team{Name: teamName}
	err := s.repo.CreateTeam(team, members)
	if err != nil {
		return nil, err
	}
	return team, nil
}

func (s *ServiceImpl) GetTeam(teamName string) (*entity.Team, []entity.User, error) {
	return s.repo.GetTeam(teamName)
}

func (s *ServiceImpl) SetUserActive(userID string, isActive bool) (*entity.User, error) {
	return s.repo.SetUserActive(userID, isActive)
}

func (s *ServiceImpl) GetUserReviewPRs(userID string) ([]entity.PullRequest, error) {
	return s.repo.GetUserReviewPRs(userID)
}

func (s *ServiceImpl) CreatePR(prID, title, authorID string) (*entity.PullRequest, error) {
	author, err := s.repo.SetUserActive(authorID, true)
	if err != nil {
		return nil, fmt.Errorf("author not found: %w", entity.ErrNotFound)
	}
	if !author.IsActive {
		return nil, fmt.Errorf("author is inactive")
	}
	candidateIDs, err := s.repo.GetCandidateReviewers(authorID, 2)
	if err != nil {
		return nil, fmt.Errorf("failed to get candidate reviewers: %w", err)
	}
	if len(candidateIDs) == 0 {
		return nil, entity.ErrNoCandidate
	}
	pr := &entity.PullRequest{
		ID:       prID,
		Title:    title,
		AuthorID: authorID,
		Status:   "OPEN",
	}
	err = s.repo.CreatePR(pr, candidateIDs)
	if err != nil {
		return nil, err
	}
	return s.repo.GetPR(prID)
}

func (s *ServiceImpl) MergePR(prID string) (*entity.PullRequest, error) {
	pr, err := s.repo.MergePR(prID)
	if err != nil {
		return nil, err
	}
	return pr, nil
}

func (s *ServiceImpl) ReassignReviewer(prID, oldUserID string) (*entity.PullRequest, string, error) {
	pr, err := s.repo.GetPR(prID)
	if err != nil {
		return nil, "", err
	}

	if pr.Status != "OPEN" {
		return nil, "", entity.ErrPRMerged
	}
	isAssigned := false
	for _, reviewer := range pr.AssignedReviewers {
		if reviewer.ID == oldUserID {
			isAssigned = true
			break
		}
	}
	if !isAssigned {
		return nil, "", entity.ErrNotAssigned
	}
	newUserID, err := s.repo.ReassignReviewer(prID, oldUserID)
	if err != nil {
		return nil, "", err
	}
	updatedPR, err := s.repo.GetPR(prID)
	if err != nil {
		return nil, "", err
	}
	return updatedPR, newUserID, nil
}

func (s *ServiceImpl) GetPR(prID string) (*entity.PullRequest, error) {
	return s.repo.GetPR(prID)
}

func (s *ServiceImpl) GetStats() (*entity.Stats, error) {
    return s.repo.GetStats()
}