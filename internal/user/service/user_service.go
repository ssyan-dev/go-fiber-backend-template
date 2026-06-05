package service

import (
	"context"
	"errors"

	"github.com/ssyan-dev/go-fiber-backend-template/internal/models"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/user/repository"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCurrentPassword = errors.New("invalid current password")
	ErrPasswordRequired       = errors.New("both passwords are required")
)

type UserService interface {
	GetByID(ctx context.Context, id string) (*models.User, error)
	Update(ctx context.Context, id string, email, curPassword, newPassword, avatarURL *string) error
	Delete(ctx context.Context, id string) error
}

type userSvc struct {
	repo repository.UserRepository
}

func NewUserService(repo repository.UserRepository) UserService {
	return &userSvc{repo: repo}
}

func (s *userSvc) GetByID(ctx context.Context, id string) (*models.User, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *userSvc) Update(ctx context.Context, id string, email, curPassword, newPassword, avatarURL *string) error {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if email != nil && *email != user.Email {
		user.Email = *email
		user.IsEmailVerified = false
	}

	if curPassword != nil || newPassword != nil {
		if curPassword == nil || newPassword == nil {
			return ErrPasswordRequired
		}

		if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(*curPassword)); err != nil {
			return ErrInvalidCurrentPassword
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(*newPassword), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		newHash := string(hash)
		user.PasswordHash = &newHash
	}

	if avatarURL != nil {
		user.AvatarURL = avatarURL
	}

	return s.repo.Update(ctx, user)
}

func (s *userSvc) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
