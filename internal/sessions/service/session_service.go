package service

import (
	"context"

	"github.com/ssyan-dev/go-fiber-backend-template/internal/models"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/sessions/repository"
)

type SessionService interface {
	Create(ctx context.Context, session *models.Session) error
	GetByUserID(ctx context.Context, userID string) ([]models.Session, error)
	GetByAccessToken(ctx context.Context, token string) (*models.Session, error)
	GetByRefreshToken(ctx context.Context, token string) (*models.Session, error)
	GetByIP(ctx context.Context, ip string) (*models.Session, error)
	Block(ctx context.Context, id string, userID string) error
	DeleteByID(ctx context.Context, id string, userID string) error
	DeleteAllByUserID(ctx context.Context, userID string) error
	DeleteByRefreshToken(ctx context.Context, token string) error
	DeleteByAccessToken(ctx context.Context, token string) error
}

type sessionService struct {
	repo      repository.SessionRepository
	redisRepo repository.SessionRedisRepository
}

func NewSessionService(repo repository.SessionRepository, redisRepo repository.SessionRedisRepository) SessionService {
	return &sessionService{
		repo:      repo,
		redisRepo: redisRepo,
	}
}

func (s *sessionService) Create(ctx context.Context, session *models.Session) error {
	if err := s.repo.Create(ctx, session); err != nil {
		return err
	}
	return s.redisRepo.SetSession(ctx, session.RefreshToken, session.UserID.String(), session.ExpiresAt.Sub(session.CreatedAt))
}

func (s *sessionService) GetByUserID(ctx context.Context, userID string) ([]models.Session, error) {
	return s.repo.GetByUserID(ctx, userID)
}

func (s *sessionService) GetByAccessToken(ctx context.Context, token string) (*models.Session, error) {
	return s.repo.GetByAccessToken(ctx, token)
}

func (s *sessionService) GetByRefreshToken(ctx context.Context, token string) (*models.Session, error) {
	return s.repo.GetByRefreshToken(ctx, token)
}

func (s *sessionService) GetByIP(ctx context.Context, ip string) (*models.Session, error) {
	return s.repo.GetByIP(ctx, ip)
}

func (s *sessionService) Block(ctx context.Context, id string, userID string) error {
	sessions, err := s.repo.GetByUserID(ctx, userID)
	if err == nil {
		for _, sess := range sessions {
			if sess.ID.String() == id {
				_ = s.redisRepo.DeleteSession(ctx, sess.RefreshToken)
				break
			}
		}
	}
	return s.repo.Block(ctx, id, userID, true)
}

func (s *sessionService) DeleteByID(ctx context.Context, id string, userID string) error {
	sessions, err := s.repo.GetByUserID(ctx, userID)
	if err == nil {
		for _, sess := range sessions {
			if sess.ID.String() == id {
				_ = s.redisRepo.DeleteSession(ctx, sess.RefreshToken)
				break
			}
		}
	}
	return s.repo.DeleteByID(ctx, id, userID)
}

func (s *sessionService) DeleteAllByUserID(ctx context.Context, userID string) error {
	sessions, err := s.repo.GetByUserID(ctx, userID)
	if err == nil {
		for _, sess := range sessions {
			_ = s.redisRepo.DeleteSession(ctx, sess.RefreshToken)
		}
	}
	return s.repo.DeleteAllByUserID(ctx, userID)
}

func (s *sessionService) DeleteByRefreshToken(ctx context.Context, token string) error {
	_ = s.redisRepo.DeleteSession(ctx, token)
	return s.repo.DeleteByRefreshToken(ctx, token)
}

func (s *sessionService) DeleteByAccessToken(ctx context.Context, token string) error {
	return s.repo.DeleteByAccessToken(ctx, token)
}
