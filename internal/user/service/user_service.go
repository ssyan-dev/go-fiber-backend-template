package service

import (
	"context"
	"errors"

	"github.com/ssyan-dev/go-fiber-backend-template/internal/models"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/user/repository"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserAlreadyExists                = errors.New("user already exists")
	ErrInvalidCurrentPassword           = errors.New("invalid current password")
	ErrInvalidLoginTypeToChangePassword = errors.New("cannot change password: account created via oauth without password")
	ErrPasswordRequired                 = errors.New("both passwords are required")
	ErrCannotDeleteSelf                 = errors.New("admin cannot delete their own account")
	ErrCannotDemoteSelf                 = errors.New("admin cannot demote their own account")
	ErrCannotBanSelf                    = errors.New("admin cannot ban their own account")
	ErrUserNotFound                     = errors.New("user not found")
)

type SessionRevoker interface {
	DeleteAllByUserID(ctx context.Context, userID string) error
}

type AdminUpdateUserInput struct {
	Email           *string
	Role            *models.UserRole
	Password        *string
	AvatarURL       *string
	IsBanned        *bool
	IsEmailVerified *bool
}

type UserService interface {
	Create(ctx context.Context, email, password string) (*models.User, error)
	GetByID(ctx context.Context, id string) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	Update(ctx context.Context, id string, email, curPassword, newPassword, avatarURL *string) error
	Delete(ctx context.Context, id string) error
	SetEmailVerified(ctx context.Context, id string) error
	ResetPassword(ctx context.Context, id, newPassword string) error
	CreateOAuthUser(ctx context.Context, email string, avatarURL *string) (*models.User, error)
	LinkOAuthProvider(ctx context.Context, userID, provider, providerUserID, email string) error

	List(ctx context.Context, filter models.ListUsersFilter) ([]models.User, int64, error)
	AdminUpdate(ctx context.Context, targetUserID, actingAdminID string, input AdminUpdateUserInput) (*models.User, error)
	AdminDelete(ctx context.Context, targetUserID, actingAdminID string) error
}

type userSvc struct {
	repo       repository.UserRepository
	redisRepo  repository.UserRedisRepository
	sessionSvc SessionRevoker
}

func NewUserService(repo repository.UserRepository, redisRepo repository.UserRedisRepository, sessionSvc SessionRevoker) UserService {
	return &userSvc{
		repo:       repo,
		redisRepo:  redisRepo,
		sessionSvc: sessionSvc,
	}
}

func (s *userSvc) Create(ctx context.Context, email, password string) (*models.User, error) {
	existing, _ := s.repo.GetByEmail(ctx, email)
	if existing != nil {
		return nil, ErrUserAlreadyExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	hashStr := string(hash)
	user := &models.User{
		Email:        email,
		PasswordHash: &hashStr,
		Role:         models.RoleDefault,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *userSvc) GetByID(ctx context.Context, id string) (*models.User, error) {
	user, err := s.redisRepo.GetUser(ctx, id)
	if err == nil && user != nil {
		return user, nil
	}

	user, err = s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	_ = s.redisRepo.SetUser(ctx, user)

	return user, nil
}

func (s *userSvc) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	return s.repo.GetByEmail(ctx, email)
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

		if user.PasswordHash == nil {
			return ErrInvalidLoginTypeToChangePassword
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

	if err := s.repo.Update(ctx, user); err != nil {
		return err
	}

	return s.redisRepo.DeleteUser(ctx, id)
}

func (s *userSvc) Delete(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	return s.redisRepo.DeleteUser(ctx, id)
}

func (s *userSvc) SetEmailVerified(ctx context.Context, id string) error {
	if err := s.repo.SetEmailVerified(ctx, id); err != nil {
		return err
	}
	return s.redisRepo.DeleteUser(ctx, id)
}

func (s *userSvc) ResetPassword(ctx context.Context, id, newPassword string) error {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	newHash := string(hash)
	user.PasswordHash = &newHash

	if err := s.repo.Update(ctx, user); err != nil {
		return err
	}

	return s.redisRepo.DeleteUser(ctx, id)
}

func (s *userSvc) CreateOAuthUser(ctx context.Context, email string, avatarURL *string) (*models.User, error) {
	user := &models.User{
		Email:           email,
		AvatarURL:       avatarURL,
		Role:            models.RoleDefault,
		IsEmailVerified: true,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *userSvc) LinkOAuthProvider(ctx context.Context, userID, provider, providerUserID, email string) error {
	return s.repo.LinkOAuthProvider(ctx, userID, provider, providerUserID, email)
}

func (s *userSvc) List(ctx context.Context, filter models.ListUsersFilter) ([]models.User, int64, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.Limit <= 0 {
		filter.Limit = 10
	} else if filter.Limit > 100 {
		filter.Limit = 100
	}

	return s.repo.List(ctx, filter)
}

func (s *userSvc) AdminUpdate(ctx context.Context, targetUserID, actingAdminID string, input AdminUpdateUserInput) (*models.User, error) {
	if targetUserID == actingAdminID {
		if input.Role != nil && *input.Role != models.RoleAdmin {
			return nil, ErrCannotDemoteSelf
		}
		if input.IsBanned != nil && *input.IsBanned {
			return nil, ErrCannotBanSelf
		}
	}

	user, err := s.repo.GetByID(ctx, targetUserID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	revokeSessions := false

	if input.Email != nil && *input.Email != user.Email {
		existing, _ := s.repo.GetByEmail(ctx, *input.Email)
		if existing != nil && existing.ID.String() != targetUserID {
			return nil, ErrUserAlreadyExists
		}
		user.Email = *input.Email
		revokeSessions = true
	}

	if input.Role != nil && *input.Role != user.Role {
		user.Role = *input.Role
		revokeSessions = true
	}

	if input.Password != nil && *input.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(*input.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		newHash := string(hash)
		user.PasswordHash = &newHash
		revokeSessions = true
	}

	if input.AvatarURL != nil {
		user.AvatarURL = input.AvatarURL
	}

	if input.IsBanned != nil && *input.IsBanned != user.IsBanned {
		user.IsBanned = *input.IsBanned
		if *input.IsBanned {
			revokeSessions = true
		}
	}

	if input.IsEmailVerified != nil {
		user.IsEmailVerified = *input.IsEmailVerified
	}

	if err := s.repo.Update(ctx, user); err != nil {
		return nil, err
	}

	_ = s.redisRepo.DeleteUser(ctx, targetUserID)

	if revokeSessions && s.sessionSvc != nil {
		_ = s.sessionSvc.DeleteAllByUserID(ctx, targetUserID)
	}

	return user, nil
}

func (s *userSvc) AdminDelete(ctx context.Context, targetUserID, actingAdminID string) error {
	if targetUserID == actingAdminID {
		return ErrCannotDeleteSelf
	}

	_, err := s.repo.GetByID(ctx, targetUserID)
	if err != nil {
		return ErrUserNotFound
	}

	_ = s.redisRepo.DeleteUser(ctx, targetUserID)

	if s.sessionSvc != nil {
		_ = s.sessionSvc.DeleteAllByUserID(ctx, targetUserID)
	}

	return s.repo.Delete(ctx, targetUserID)
}
