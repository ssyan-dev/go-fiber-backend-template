package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/models"
)

type AuthRepository interface {
	CreateUser(ctx context.Context, user *models.User) error
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	GetByID(ctx context.Context, id string) (*models.User, error)
}

type authRepo struct {
	db *pgxpool.Pool
}

func NewAuthRepository(db *pgxpool.Pool) AuthRepository {
	return &authRepo{db: db}
}

func (r *authRepo) CreateUser(ctx context.Context, user *models.User) error {
	query := `INSERT INTO users (email, password_hash, role) VALUES ($1, $2, $3) RETURNING id, created_at, updated_at`

	err := r.db.QueryRow(ctx, query, user.Email, user.PasswordHash, user.Role).
		Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	return err
}

func (r *authRepo) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `SELECT id, email, password_hash, role, avatar_url, is_banned, is_email_verified, created_at, updated_at FROM users WHERE email = $1`

	var user models.User
	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Role, &user.AvatarURL,
		&user.IsBanned, &user.IsEmailVerified, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *authRepo) GetByID(ctx context.Context, id string) (*models.User, error) {
	query := `SELECT id, email, password_hash, role, avatar_url, is_banned, is_email_verified, created_at, updated_at FROM users WHERE id = $1`

	var user models.User
	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Role, &user.AvatarURL,
		&user.IsBanned, &user.IsEmailVerified, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}
