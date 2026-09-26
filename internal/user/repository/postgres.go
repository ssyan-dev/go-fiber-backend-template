package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/models"
)

type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	GetByID(ctx context.Context, id string) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	Update(ctx context.Context, user *models.User) error
	Delete(ctx context.Context, id string) error
	SetEmailVerified(ctx context.Context, id string) error
	LinkOAuthProvider(ctx context.Context, userID, provider, providerUserID, email string) error
}

type userRepo struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) UserRepository {
	return &userRepo{db: db}
}

func (r *userRepo) Create(ctx context.Context, user *models.User) error {
	query := `INSERT INTO users (email, password_hash, role, avatar_url, is_email_verified)
	          VALUES ($1, $2, $3, $4, $5)
	          RETURNING id, created_at, updated_at`

	return r.db.QueryRow(ctx, query, user.Email, user.PasswordHash, user.Role, user.AvatarURL, user.IsEmailVerified).
		Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
}

func (r *userRepo) GetByID(ctx context.Context, id string) (*models.User, error) {
	query := `SELECT id, email, role, avatar_url, password_hash, is_banned, is_email_verified, created_at, updated_at FROM users WHERE id = $1`

	var user models.User
	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.Role, &user.AvatarURL, &user.PasswordHash,
		&user.IsBanned, &user.IsEmailVerified, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *userRepo) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `SELECT id, email, role, avatar_url, password_hash, is_banned, is_email_verified, created_at, updated_at FROM users WHERE email = $1`

	var user models.User
	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.Role, &user.AvatarURL, &user.PasswordHash,
		&user.IsBanned, &user.IsEmailVerified, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *userRepo) Update(ctx context.Context, user *models.User) error {
	query := `UPDATE users SET email = $1, password_hash = $2, avatar_url = $3, is_email_verified = $4, updated_at = NOW() WHERE id = $5`

	_, err := r.db.Exec(ctx, query, user.Email, user.PasswordHash, user.AvatarURL, user.IsEmailVerified, user.ID.String())
	return err
}

func (r *userRepo) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM users WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

func (r *userRepo) SetEmailVerified(ctx context.Context, id string) error {
	query := `UPDATE users SET is_email_verified = TRUE, updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

func (r *userRepo) LinkOAuthProvider(ctx context.Context, userID, provider, providerUserID, email string) error {
	query := `INSERT INTO user_oauth_providers (user_id, provider, provider_user_id, email)
	          VALUES ($1, $2, $3, $4)
	          ON CONFLICT (provider, provider_user_id) DO UPDATE SET email = EXCLUDED.email`
	_, err := r.db.Exec(ctx, query, userID, provider, providerUserID, email)
	return err
}
