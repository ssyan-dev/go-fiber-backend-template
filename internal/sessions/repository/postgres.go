package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/models"
)

type SessionRepository interface {
	Create(ctx context.Context, session *models.Session) error
	GetByUserID(ctx context.Context, userID string) ([]models.Session, error)
	GetByAccessToken(ctx context.Context, token string) (*models.Session, error)
	GetByRefreshToken(ctx context.Context, token string) (*models.Session, error)
	GetByIP(ctx context.Context, ip string) (*models.Session, error)
	Block(ctx context.Context, id string, userID string, isBlocked bool) error
	DeleteByID(ctx context.Context, id string, userID string) error
	DeleteAllByUserID(ctx context.Context, userID string) error
	DeleteByRefreshToken(ctx context.Context, token string) error
	DeleteByAccessToken(ctx context.Context, token string) error
}

type sessionRepo struct {
	db *pgxpool.Pool
}

func NewSessionRepository(db *pgxpool.Pool) SessionRepository {
	return &sessionRepo{db: db}
}

func (r *sessionRepo) Create(ctx context.Context, s *models.Session) error {
	query := `INSERT INTO sessions (user_id, refresh_token, access_token, user_agent, ip, expires_at) 
	          VALUES ($1, $2, $3, $4, $5, $6) 
	          RETURNING id, created_at`
	return r.db.QueryRow(ctx, query, s.UserID, s.RefreshToken, s.AccessToken, s.UserAgent, s.IP, s.ExpiresAt).
		Scan(&s.ID, &s.CreatedAt)
}

func (r *sessionRepo) GetByUserID(ctx context.Context, userID string) ([]models.Session, error) {
	query := `SELECT id, user_id, user_agent, ip, is_blocked, expires_at, created_at 
	          FROM sessions WHERE user_id = $1 AND expires_at > NOW() AND is_blocked = FALSE 
	          ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []models.Session
	for rows.Next() {
		var s models.Session
		err := rows.Scan(&s.ID, &s.UserID, &s.UserAgent, &s.IP, &s.IsBlocked, &s.ExpiresAt, &s.CreatedAt)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, nil
}

func (r *sessionRepo) GetByRefreshToken(ctx context.Context, token string) (*models.Session, error) {
	query := `SELECT id, user_id, refresh_token, access_token, user_agent, ip, is_blocked, expires_at, created_at 
	          FROM sessions WHERE refresh_token = $1`

	var s models.Session
	err := r.db.QueryRow(ctx, query, token).Scan(
		&s.ID, &s.UserID, &s.RefreshToken, &s.AccessToken, &s.UserAgent, &s.IP, &s.IsBlocked, &s.ExpiresAt, &s.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *sessionRepo) GetByAccessToken(ctx context.Context, token string) (*models.Session, error) {
	query := `SELECT id, user_id, refresh_token, access_token, user_agent, ip, is_blocked, expires_at, created_at 
	          FROM sessions WHERE access_token = $1`

	var s models.Session
	err := r.db.QueryRow(ctx, query, token).Scan(
		&s.ID, &s.UserID, &s.RefreshToken, &s.AccessToken, &s.UserAgent, &s.IP, &s.IsBlocked, &s.ExpiresAt, &s.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *sessionRepo) GetByIP(ctx context.Context, ip string) (*models.Session, error) {
	query := `SELECT id, user_id, refresh_token, access_token, user_agent, ip, is_blocked, expires_at, created_at 
	          FROM sessions WHERE ip = $1`

	var s models.Session
	err := r.db.QueryRow(ctx, query, ip).Scan(
		&s.ID, &s.UserID, &s.RefreshToken, &s.AccessToken, &s.UserAgent, &s.IP, &s.IsBlocked, &s.ExpiresAt, &s.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *sessionRepo) Block(ctx context.Context, id string, userID string, isBlocked bool) error {
	query := `UPDATE sessions SET is_blocked = $1 WHERE id = $2 AND user_id = $3`
	_, err := r.db.Exec(ctx, query, isBlocked, id, userID)
	return err
}

func (r *sessionRepo) DeleteByID(ctx context.Context, id string, userID string) error {
	query := `DELETE FROM sessions WHERE id = $1 AND user_id = $2`
	_, err := r.db.Exec(ctx, query, id, userID)
	return err
}

func (r *sessionRepo) DeleteAllByUserID(ctx context.Context, userID string) error {
	query := `DELETE FROM sessions WHERE user_id = $1`
	_, err := r.db.Exec(ctx, query, userID)
	return err
}

func (r *sessionRepo) DeleteByRefreshToken(ctx context.Context, token string) error {
	query := `DELETE FROM sessions WHERE refresh_token = $1`
	_, err := r.db.Exec(ctx, query, token)
	return err
}

func (r *sessionRepo) DeleteByAccessToken(ctx context.Context, token string) error {
	query := `DELETE FROM sessions WHERE access_token = $1`
	_, err := r.db.Exec(ctx, query, token)
	return err
}
