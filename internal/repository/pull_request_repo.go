package repository

import (
	"context"
	"reviewer_pr/internal/models"
	"time"

	"gorm.io/gorm"
)

type PRRepo interface {
	Create(ctx context.Context, pr *models.PullRequest) error
	GetPullRequestByID(ctx context.Context, id string) (*models.PullRequest, error)
	GetPullRequestWithReviewers(ctx context.Context, id string) (*models.PullRequest, []models.PRReviewer, error)
	SetPullRequestMerged(ctx context.Context, id string, mergedAt time.Time) (bool, error)
	AddReviewers(ctx context.Context, prID string, reviewerIDs []string) error
	ReplaceReviewer(ctx context.Context, prID, oldID, newID string) error
	GetPullRequestsByReviewer(ctx context.Context, reviewerID string) ([]models.PullRequest, error)
	GetReviewersForPR(ctx context.Context, prID string) ([]models.PRReviewer, error)
}

type prRepo struct {
	db *gorm.DB
}

func NewPRRepo(db *gorm.DB) PRRepo {
	return &prRepo{db: db}
}

func (r *prRepo) Create(ctx context.Context, pr *models.PullRequest) error {
	return r.db.WithContext(ctx).Create(&pr).Error
}

func (r *prRepo) GetPullRequestByID(ctx context.Context, id string) (*models.PullRequest, error) {
	var pr models.PullRequest
	err := r.db.WithContext(ctx).Where("pull_request_id = ?", id).First(&pr).Error
	if err != nil {
		return nil, err
	}
	return &pr, nil
}

func (r *prRepo) GetPullRequestWithReviewers(ctx context.Context, id string) (*models.PullRequest, []models.PRReviewer, error) {
	var pr models.PullRequest
	err := r.db.WithContext(ctx).
		Where("pull_request_id = ?", id).
		First(&pr).Error
	if err != nil {
		return nil, nil, err
	}

	var reviewers []models.PRReviewer
	err = r.db.WithContext(ctx).
		Where("pull_request_id = ?", id).
		Find(&reviewers).Error
	if err != nil {
		return nil, nil, err
	}

	return &pr, reviewers, nil
}

func (r *prRepo) SetPullRequestMerged(ctx context.Context, id string, mergedAt time.Time) (bool, error) {
	res := r.db.WithContext(ctx).Model(&models.PullRequest{}).Where("pull_request_id = ? AND status = ", id, models.PRStatusOpen).Updates(map[string]any{
		"status":    models.PRStatusMerged,
		"merged_at": mergedAt,
	})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

func (r *prRepo) AddReviewers(ctx context.Context, prID string, reviewerIDs []string) error {
	if len(reviewerIDs) == 0 {
		return nil
	}

	reviewers := make([]models.PRReviewer, 0, len(reviewerIDs))
	now := time.Now().UTC()
	for _, id := range reviewerIDs {
		reviewers = append(reviewers, models.PRReviewer{
			PullRequestID: prID,
			ReviewerID:    id,
			AssignedAt:    now,
		})
	}

	return r.db.WithContext(ctx).Create(&reviewers).Error
}

func (r *prRepo) ReplaceReviewer(ctx context.Context, prID, oldID, newID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.WithContext(ctx).Where("pull_request_id = ? AND reviewer_id", prID, oldID).Delete(&models.PRReviewer{}).Error; err != nil {
			return err
		}
		reviewer := models.PRReviewer{
			PullRequestID: prID,
			ReviewerID:    newID,
			AssignedAt:    time.Now().UTC(),
		}
		if err := tx.WithContext(ctx).Create(&reviewer).Error; err != nil {
			return err
		}
		return nil
	})

}

func (r *prRepo) GetPullRequestsByReviewer(ctx context.Context, reviewerID string) ([]models.PullRequest, error) {
	var prs []models.PullRequest

	err := r.db.WithContext(ctx).Model(&models.PullRequest{}).Joins("JOIN pr_reviewers ON pr_reviewers.pull_request_id = pull_requests.pull_request_id").Where("pr_reviewers.reviewer_id = ?", reviewerID).Order("pull_requests.created_at DESC").Find(&prs).Error
	if err != nil {
		return nil, err
	}
	return prs, nil
}

func (r *prRepo) GetReviewersForPR(ctx context.Context, prID string) ([]models.PRReviewer, error) {
	var reviewers []models.PRReviewer
	err := r.db.WithContext(ctx).Where("pull_request_id = ?", prID).Find(&reviewers).Error
	if err != nil {
		return nil, err
	}
	return reviewers, nil
}
