package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/models"
)

type VerificationCodeRepository interface {
	Create(ctx context.Context, vc *models.VerificationCode) error
	GetByCode(ctx context.Context, code string) (*models.VerificationCode, error)
	DeleteByUserIDAndType(ctx context.Context, userID string, t models.VerificationType) error
}

type verificationCodeRepo struct {
	db *pgxpool.Pool
}

func NewVerificationCodeRepository(db *pgxpool.Pool) VerificationCodeRepository {
	return &verificationCodeRepo{db: db}
}

func (r *verificationCodeRepo) Create(ctx context.Context, vc *models.VerificationCode) error {
	query := `INSERT INTO verification_codes (user_id, code, type, expires_at)
	          VALUES ($1, $2, $3, $4)
	          RETURNING id, created_at`

	return r.db.QueryRow(ctx, query, vc.UserID, vc.Code, vc.Type, vc.ExpiresAt).
		Scan(&vc.ID, &vc.CreatedAt)
}

func (r *verificationCodeRepo) GetByCode(ctx context.Context, code string) (*models.VerificationCode, error) {
	query := `SELECT id, user_id, code, type, expires_at, created_at
	          FROM verification_codes WHERE code = $1`

	var vc models.VerificationCode
	err := r.db.QueryRow(ctx, query, code).Scan(
		&vc.ID, &vc.UserID, &vc.Code, &vc.Type, &vc.ExpiresAt, &vc.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &vc, nil
}

func (r *verificationCodeRepo) DeleteByUserIDAndType(ctx context.Context, userID string, t models.VerificationType) error {
	query := `DELETE FROM verification_codes WHERE user_id = $1 AND type = $2`
	_, err := r.db.Exec(ctx, query, userID, t)
	return err
}
