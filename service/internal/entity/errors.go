package entity

import (
	"errors"
)

var (
	ErrTeamExists    = errors.New("team already exists")
	ErrPRExists      = errors.New("pull request already exists")
	ErrPRMerged      = errors.New("pull request is merged")
	ErrNotAssigned   = errors.New("reviewer is not assigned")
	ErrNoCandidate   = errors.New("no active replacement candidate")
	ErrNotFound      = errors.New("resource not found")
)