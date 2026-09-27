package models

import (
	"time"

	"github.com/google/uuid"
)

type UserRole string

const (
	RoleDefault UserRole = "default"
	RoleAdmin   UserRole = "admin"
)

type User struct {
	ID              uuid.UUID `json:"id" db:"id"`
	Email           string    `json:"email" db:"email"`
	Role            UserRole  `json:"role" db:"role"`
	AvatarURL       *string   `json:"avatar_url" db:"avatar_url"`
	PasswordHash    *string   `json:"-" db:"password_hash"`
	IsBanned        bool      `json:"is_banned" db:"is_banned"`
	IsEmailVerified bool      `json:"is_email_verified" db:"is_email_verified"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

type ListUsersFilter struct {
	Search   *string
	Role     *UserRole
	IsBanned *bool
	Page     int
	Limit    int
}

type PaginationMeta struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

type PaginatedUsersResponse struct {
	Users []User         `json:"users"`
	Meta  PaginationMeta `json:"meta"`
}
