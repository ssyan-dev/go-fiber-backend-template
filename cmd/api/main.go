package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	swaggo "github.com/gofiber/contrib/v3/swaggo"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/helmet"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	fiberRedis "github.com/gofiber/storage/redis/v3"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	authHandler "github.com/ssyan-dev/go-fiber-backend-template/internal/auth/handler"
	authService "github.com/ssyan-dev/go-fiber-backend-template/internal/auth/service"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/config"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/database"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/infra/mailer"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/infra/queue"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/logger"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/middleware"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/models"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/pkg/response"
	sessionHandler "github.com/ssyan-dev/go-fiber-backend-template/internal/sessions/handler"
	sessionRepo "github.com/ssyan-dev/go-fiber-backend-template/internal/sessions/repository"
	sessionService "github.com/ssyan-dev/go-fiber-backend-template/internal/sessions/service"
	userHandler "github.com/ssyan-dev/go-fiber-backend-template/internal/user/handler"
	userRepo "github.com/ssyan-dev/go-fiber-backend-template/internal/user/repository"
	userService "github.com/ssyan-dev/go-fiber-backend-template/internal/user/service"
	vcRepo "github.com/ssyan-dev/go-fiber-backend-template/internal/verification_codes/repository"
	vcService "github.com/ssyan-dev/go-fiber-backend-template/internal/verification_codes/service"
	"go.uber.org/zap"

	_ "github.com/ssyan-dev/go-fiber-backend-template/docs"
)

// @title			Backend API
// @version		1.0
// @description	Backend API

// @contact.name	Stanislav Simakhin
// @contact.url	https://ssyan.ru

// @host			localhost:8080
// @BasePath		/api/v1

// @securityDefinitions.apikey	BearerAuth
// @in							header
// @name						Authorization
// @description			Bearer [JWT]
func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cfg, err := config.InitConfig()
	if err != nil {
		panic(err.Error())
	}
	config.IsProduction = cfg.App.IsProduction()

	l := logger.NewLogger(cfg.App.IsProduction())
	defer l.Sync()

	l.Info("starting backend",
		zap.String("env", cfg.App.Env),
		zap.Int("port", cfg.App.Port),
	)

	pg, err := database.NewPostgres(ctx, &cfg.Postgres)
	if err != nil {
		l.Fatal("failed to make postgres connection", zap.Error(err))
	}
	defer pg.Close()
	l.Info("postgres connected!")

	rdb, err := database.NewRedis(ctx, &cfg.Redis)
	if err != nil {
		l.Fatal("failed to make redis connection", zap.Error(err))
	}
	defer rdb.Close()
	l.Info("redis connected!")

	app := fiber.New(fiber.Config{
		ServerHeader: "backend-template",
		ErrorHandler: response.ErrorHandler,
	})

	baseURL := cfg.App.BaseURL()

	l.Info("fiber initialized!",
		zap.String("base_url", baseURL),
		zap.String("allowed_origin", cfg.App.AllowedOrigin),
	)

	app.Use(requestid.New())

	helmetConfig := helmet.Config{
		CrossOriginResourcePolicy: "cross-origin",
	}
	if cfg.App.IsProduction() {
		helmetConfig.HSTSMaxAge = 31536000
		helmetConfig.HSTSPreloadEnabled = true
	}
	app.Use(helmet.New(helmetConfig))

	app.Use(recover.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     []string{cfg.App.AllowedOrigin},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"},
		ExposeHeaders:    []string{"X-Request-ID"},
		AllowCredentials: true,
	}))
	app.Use(middleware.NewLogger(l))

	redisStorage := fiberRedis.NewFromConnection(rdb)

	authLimiter := middleware.AuthLimiter(&cfg.Limiter, redisStorage)
	globalLimiter := middleware.GlobalLimiter(&cfg.Limiter, redisStorage)

	api := app.Group(cfg.App.GlobalPrefix, globalLimiter)

	sr := sessionRepo.NewSessionRepository(pg)
	srr := sessionRepo.NewSessionRedisRepository(rdb)
	ss := sessionService.NewSessionService(sr, srr)

	ms, err := mailer.NewMailerService(&cfg.SMTP, l, mailer.DefaultTemplatesFS)
	if err != nil {
		l.Fatal("mailer err", zap.Error(err))
	}

	distributor := queue.NewRedisTaskDistributor(&cfg.Redis, &cfg.Queue, ms, l)

	var processor queue.TaskProcessor
	if cfg.Queue.InProcess {
		processor = queue.NewRedisTaskProcessor(&cfg.Redis, &cfg.Queue, ms, l)
		if err := processor.Start(); err != nil {
			l.Fatal("failed to start queue processor", zap.Error(err))
		}
	}

	ur := userRepo.NewUserRepository(pg)
	urr := userRepo.NewUserRedisRepository(rdb)
	us := userService.NewUserService(ur, urr, ss)
	uh := userHandler.NewUserHandler(us)

	vcr := vcRepo.NewVerificationCodeRepository(pg)
	vcs := vcService.NewVerificationCodeService(vcr, l)

	as := authService.NewAuthService(us, ss, vcs, distributor, &cfg.JWT, &cfg.Auth, baseURL, l)
	ah := authHandler.NewAuthHandler(as, authLimiter)
	ah.RegisterRoutes(api)

	if cfg.OAuth.IsEnabled() {
		oas := authService.NewOAuthService(us, ss, &cfg.JWT, &cfg.OAuth, baseURL, l)
		oah := authHandler.NewOAuthHandler(oas, as)
		oah.RegisterRoutes(api)
	}

	api.Get("/docs/*", swaggo.HandlerDefault)
	api.Get("/health", healthCheck(cfg.App.Env, pg, rdb))

	sh := sessionHandler.NewSessionHandler(ss)

	protected := api.Group("/", middleware.AuthMiddleware(&cfg.JWT))
	uh.RegisterRoutes(protected)
	sh.RegisterRoutes(protected)

	admin := api.Group("/admin", middleware.AuthMiddleware(&cfg.JWT), middleware.RolesMiddleware(string(models.RoleAdmin)))
	auh := userHandler.NewAdminUserHandler(us)
	auh.RegisterRoutes(admin)

	addr := fmt.Sprintf(":%d", cfg.App.Port)
	go func() {
		if err := app.Listen(addr); err != nil {
			l.Fatal("failed to run http server: %v", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	l.Info("shutting down...")

	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		l.Error("shutdown failed", zap.Error(err))
	}

	if processor != nil {
		processor.Shutdown()
	}

	if err := distributor.Close(); err != nil {
		l.Error("failed to close task distributor", zap.Error(err))
	}

	l.Info("server stopped!")
}

// healthCheck godoc
// @Summary		Health check
// @Description	Check if the server, postgresql and redis is healthy
// @Tags			system
// @Produce		json
// @Success		200	{object}	response.HealthResponse
// @Failure		503	{object}	response.HealthResponse
// @Router			/health [get]
func healthCheck(env string, pg *pgxpool.Pool, rdb *redis.Client) fiber.Handler {
	return func(c fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.Context(), 2*time.Second)
		defer cancel()

		services := make(map[string]string)
		isHealthy := true

		if err := pg.Ping(ctx); err != nil {
			services["postgres"] = "down: " + err.Error()
			isHealthy = false
		} else {
			services["postgres"] = "up"
		}

		if err := rdb.Ping(ctx).Err(); err != nil {
			services["redis"] = "down: " + err.Error()
			isHealthy = false
		} else {
			services["redis"] = "up"
		}

		healthData := response.HealthData{
			Env:       env,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Services:  services,
		}

		if !isHealthy {
			healthData.Status = "not ok"
			return response.Error(c, fiber.StatusServiceUnavailable, "server is not healthy", healthData)
		}

		healthData.Status = "ok"
		return response.Success(c, fiber.StatusOK, "server is healthy", healthData)
	}
}
