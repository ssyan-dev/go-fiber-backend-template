package main

import (
	"context"
	"log"
	"time"

	"github.com/ssyan-dev/go-fiber-backend-template/internal/auth/repository"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/config"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/database"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/models"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cfg, err := config.InitConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	pg, err := database.NewPostgres(ctx, &cfg.Postgres)
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}
	defer pg.Close()

	authRepo := repository.NewAuthRepository(pg)
	seedUsers(ctx, authRepo)

	log.Println("seeding completed successfully!")
}

func seedUsers(ctx context.Context, repo repository.AuthRepository) {
	users := []struct {
		email           string
		password        string
		role            models.UserRole
		isEmailVerified bool
	}{
		{"unverified@backend.com", "user123", models.RoleDefault, false},
		{"verified@backend.com", "user123", models.RoleDefault, true},
		{"admin@backend.com", "admin123", models.RoleAdmin, true},
	}

	for _, u := range users {
		_, err := repo.GetByEmail(ctx, u.email)
		if err == nil {
			log.Printf("user %s already exists. skip..", u.email)
			continue
		}

		hash, _ := bcrypt.GenerateFromPassword([]byte(u.password), bcrypt.DefaultCost)
		hashStr := string(hash)

		user := &models.User{
			Email:           u.email,
			PasswordHash:    &hashStr,
			Role:            u.role,
			IsEmailVerified: true,
		}

		if err := repo.CreateUser(ctx, user); err != nil {
			log.Printf("failed to create user with email '%s' (%s): %v", u.email, u.role, err)
		} else {
			log.Printf("user with email '%s' (%s) created!", u.email, u.role)
		}
	}
}
