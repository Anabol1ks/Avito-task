package service

import (
	"context"
	"errors"
	"math/rand"
	"reviewer_pr/internal/models"
	"reviewer_pr/internal/repository"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type PRService interface {
	CreateWithAutoAssign(ctx context.Context, in CreatePRInput) (*CreatePROutput, error)
	Merge(ctx context.Context, prID string) (*models.PullRequest, error)
	ReassignReviewer(ctx context.Context, in ReassignInput) (*ReassignOutput, error)
	GetReviewsByUser(ctx context.Context, reviewerID string) ([]models.PullRequest, error)
}

type prService struct {
	repo *repository.Repository
	log  *zap.Logger
	rnd  *rand.Rand
}

func NewPRService(repo *repository.Repository, log *zap.Logger) PRService {
	return &prService{
		repo: repo,
		log:  log,
		rnd:  rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

type CreatePRInput struct {
	ID       string
	Name     string
	AuthorID string
}

type CreatePROutput struct {
	PR        *models.PullRequest
	Reviewers []models.User
}

func (s *prService) CreateWithAutoAssign(ctx context.Context, in CreatePRInput) (*CreatePROutput, error) {
	var out *CreatePROutput

	err := s.repo.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if existing, err := s.repo.PRs.GetPullRequestByID(ctx, in.ID); err != nil && existing != nil {
			return NewErr(ErrorCodePRExists, "pull request already exists")
		} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		author, err := s.repo.Users.GetUserByID(ctx, in.AuthorID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return NewErr(ErrorCodeNotFound, "author not found")
			}
			return err
		}

		candidates, err := s.repo.Users.GetActiveTeamMembersExcept(ctx, author.TeamName, author.ID)
		if err != nil {
			return err
		}

		reviewers := pickReviewers(s.rnd, candidates, 2)
		pr := &models.PullRequest{
			ID:       in.ID,
			Name:     in.Name,
			AuthorID: author.ID,
			Status:   models.PRStatusOpen,
		}

		if err := s.repo.PRs.Create(ctx, pr); err != nil {
			return err
		}

		reviewerIDs := make([]string, 0, len(reviewers))
		for _, u := range reviewers {
			reviewerIDs = append(reviewerIDs, u.ID)
		}

		if err := s.repo.PRs.AddReviewers(ctx, pr.ID, reviewerIDs); err != nil {
			return err
		}

		out = &CreatePROutput{
			PR:        pr,
			Reviewers: reviewers,
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return out, nil
}

func pickReviewers(r *rand.Rand, users []models.User, max int) []models.User {
	if len(users) == 0 || max <= 0 {
		return nil
	}
	cpy := make([]models.User, len(users))
	copy(cpy, users)

	r.Shuffle(len(cpy), func(i, j int) {
		cpy[i], cpy[j] = cpy[j], cpy[i]
	})

	if len(cpy) > max {
		cpy = cpy[:max]
	}
	return cpy
}

func (s *prService) Merge(ctx context.Context, prID string) (*models.PullRequest, error) {
	pr, err := s.repo.PRs.GetPullRequestByID(ctx, prID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, NewErr(ErrorCodeNotFound, "pull request not found")
		}
		return nil, err
	}

	if pr.Status == models.PRStatusMerged {
		return pr, nil
	}

	_, err = s.repo.PRs.SetPullRequestMerged(ctx, prID, time.Now().UTC())
	if err != nil {
		return nil, err
	}

	return s.repo.PRs.GetPullRequestByID(ctx, prID)
}

type ReassignInput struct {
	PRID          string
	OldReviewerID string
}

type ReassignOutput struct {
	PR           *models.PullRequest
	ReplacedByID string
}

func (s *prService) ReassignReviewer(ctx context.Context, in ReassignInput) (*ReassignOutput, error) {
	var out *ReassignOutput

	err := s.repo.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		pr, err := s.repo.PRs.GetPullRequestByID(ctx, in.PRID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return NewErr(ErrorCodeNotFound, "pull request not found")
			}
			return err
		}

		if pr.Status == models.PRStatusMerged {
			return NewErr(ErrorCodePRMerged, "cannot reassing reviewer for merged pull request")
		}

		reviewers, err := s.repo.PRs.GetReviewersForPR(ctx, in.PRID)
		if err != nil {
			return err
		}

		assigned := false
		var otherReviewerID string
		for _, r := range reviewers {
			if r.ReviewerID == in.OldReviewerID {
				assigned = true
			} else {
				otherReviewerID = r.ReviewerID
			}
		}
		if !assigned {
			return NewErr(ErrorCodeNotAssigned, "user is not assigned as reviewer for this PR")
		}
		oldUser, err := s.repo.Users.GetUserByID(ctx, in.OldReviewerID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return NewErr(ErrorCodeNotFound, "old reviewer not found")
			}
			return err
		}

		candidates, err := s.repo.Users.GetActiveTeamMembersExcept(ctx, oldUser.TeamName, oldUser.ID)
		if err != nil {
			return err
		}

		if otherReviewerID != "" {
			filtered := candidates[:0]
			for _, c := range candidates {
				if c.ID != otherReviewerID {
					filtered = append(filtered, c)
				}
			}
			candidates = filtered
		}

		if len(candidates) == 0 {
			return NewErr(ErrorCodeNoCandidate, "no active candidate in reviewer team")
		}

		newReviewer := pickReviewers(s.rnd, candidates, 1)[0]

		if err := s.repo.PRs.ReplaceReviewer(ctx, in.PRID, in.OldReviewerID, newReviewer.ID); err != nil {
			return err
		}

		upd, err := s.repo.PRs.GetPullRequestByID(ctx, in.PRID)
		if err != nil {
			return err
		}

		out = &ReassignOutput{
			PR:           upd,
			ReplacedByID: newReviewer.ID,
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return out, nil
}

func (s *prService) GetReviewsByUser(ctx context.Context, reviewerID string) ([]models.PullRequest, error) {
	prs, err := s.repo.PRs.GetPullRequestsByReviewer(ctx, reviewerID)
	if err != nil {
		return nil, err
	}
	return prs, nil
}
