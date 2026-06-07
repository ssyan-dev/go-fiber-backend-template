package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/ssyan-dev/go-fiber-backend-template/internal/auth/repository"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/config"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/database"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/models"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	email := flag.String("email", "", "Admin email")
	password := flag.String("password", "", "Admin password")
	flag.Parse()

	if *email == "" || *password == "" {
		log.Fatal("email and password is required. usage: go run cmd/admin/main.go -email=admin@example.com -password=supercoolpassword")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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

	_, err = authRepo.GetByEmail(ctx, *email)
	if err == nil {
		log.Fatalf("user with email %s already exists", *email)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("failed to hash password: %v", err)
	}
	hashStr := string(hash)

	user := &models.User{
		Email:           *email,
		PasswordHash:    &hashStr,
		Role:            models.RoleAdmin,
		IsEmailVerified: true,
	}

	if err := authRepo.CreateUser(ctx, user); err != nil {
		log.Fatalf("failed to create admin: %v", err)
	}

	log.Printf("admin user created created: %s\n", *email)
}
