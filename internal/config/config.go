package config

import (
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	App      AppConfig
	Limiter  LimiterConfig
	Queue    QueueConfig
	JWT      JWTConfig
	SMTP     SMTPConfig
	Auth     AuthConfig
	OAuth    OAuthConfig
	Postgres PostgresConfig
	Redis    RedisConfig
}

type AppConfig struct {
	Env           string `env:"APP_ENV" envDefault:"development"`
	Port          int    `env:"APP_PORT" envDefault:"8080"`
	URL           string `env:"APP_URL" envDefault:"localhost"`
	GlobalPrefix  string `env:"APP_GLOBAL_PREFIX" envDefault:"/api/v1"`
	AllowedOrigin string `env:"APP_ALLOWED_ORIGIN" envDefault:"http://localhost:3000"`
}

type LimiterConfig struct {
	AuthMax          int           `env:"RATE_LIMIT_AUTH_MAX" envDefault:"10"`
	AuthExpiration   time.Duration `env:"RATE_LIMIT_AUTH_EXPIRATION" envDefault:"1m"`
	GlobalMax        int           `env:"RATE_LIMIT_GLOBAL_MAX" envDefault:"120"`
	GlobalExpiration time.Duration `env:"RATE_LIMIT_GLOBAL_EXPIRATION" envDefault:"1m"`
}

type QueueConfig struct {
	InProcess          bool          `env:"QUEUE_IN_PROCESS" envDefault:"true"`
	Concurrency        int           `env:"QUEUE_CONCURRENCY" envDefault:"10"`
	CriticalWeight     int           `env:"QUEUE_CRITICAL_WEIGHT" envDefault:"6"`
	DefaultWeight      int           `env:"QUEUE_DEFAULT_WEIGHT" envDefault:"3"`
	LowWeight          int           `env:"QUEUE_LOW_WEIGHT" envDefault:"1"`
	MaxRetry           int           `env:"QUEUE_MAX_RETRY" envDefault:"3"`
	TaskTimeout        time.Duration `env:"QUEUE_TASK_TIMEOUT" envDefault:"20s"`
	ShutdownTimeout    time.Duration `env:"QUEUE_SHUTDOWN_TIMEOUT" envDefault:"10s"`
	FallbackSyncOnFail bool          `env:"QUEUE_FALLBACK_SYNC_ON_FAIL" envDefault:"true"`
}

type JWTConfig struct {
	SecretKey       string        `env:"JWT_SECRET_KEY,required"`
	AccessTokenTTL  time.Duration `env:"JWT_ACCESS_TOKEN_TTL" envDefault:"30m"`
	RefreshTokenTTL time.Duration `env:"JWT_REFRESH_TOKEN_TTL" envDefault:"336h"`
}

type SMTPConfig struct {
	Enabled  bool   `env:"SMTP_ENABLED" envDefault:"false"`
	Host     string `env:"SMTP_HOST"`
	Port     int    `env:"SMTP_PORT" envDefault:"587"`
	Email    string `env:"SMTP_EMAIL"`
	Password string `env:"SMTP_PASSWORD"`
	FromName string `env:"SMTP_FROM_NAME" envDefault:"Backend"`
}

type AuthConfig struct {
	VerifyEmail bool `env:"AUTH_VERIFY_EMAIL" envDefault:"false"`
}

type OAuthConfig struct {
	Google GoogleOAuthConfig
	GitHub GitHubOAuthConfig
	Yandex YandexOAuthConfig
}

type GoogleOAuthConfig struct {
	Enabled      bool   `env:"GOOGLE_OAUTH_ENABLED" envDefault:"false"`
	ClientID     string `env:"GOOGLE_CLIENT_ID"`
	ClientSecret string `env:"GOOGLE_CLIENT_SECRET"`
}

type GitHubOAuthConfig struct {
	Enabled      bool   `env:"GITHUB_OAUTH_ENABLED" envDefault:"false"`
	ClientID     string `env:"GITHUB_CLIENT_ID"`
	ClientSecret string `env:"GITHUB_CLIENT_SECRET"`
}

type YandexOAuthConfig struct {
	Enabled      bool   `env:"YANDEX_OAUTH_ENABLED" envDefault:"false"`
	ClientID     string `env:"YANDEX_CLIENT_ID"`
	ClientSecret string `env:"YANDEX_CLIENT_SECRET"`
}

type PostgresConfig struct {
	Host     string `env:"POSTGRES_HOST" envDefault:"localhost"`
	Port     int    `env:"POSTGRES_PORT" envDefault:"5432"`
	User     string `env:"POSTGRES_USER,required"`
	Password string `env:"POSTGRES_PASSWORD,required"`
	DBName   string `env:"POSTGRES_DB,required"`

	MaxConns          int32         `env:"POSTGRES_MAX_CONNS" envDefault:"25"`
	MinConns          int32         `env:"POSTGRES_MIN_CONNS" envDefault:"5"`
	MaxConnLifetime   time.Duration `env:"POSTGRES_MAX_CONN_LIFETIME" envDefault:"1h"`
	MaxConnIdleTime   time.Duration `env:"POSTGRES_MAX_CONN_IDLE_TIME" envDefault:"15m"`
	HealthCheckPeriod time.Duration `env:"POSTGRES_HEALTH_CHECK_PERIOD" envDefault:"1m"`
}

type RedisConfig struct {
	Host     string `env:"REDIS_HOST" envDefault:"localhost"`
	Port     int    `env:"REDIS_PORT" envDefault:"6379"`
	Password string `env:"REDIS_PASSWORD,required"`
}

func InitConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		return nil, err
	}

	var cfg Config
	err := env.Parse(&cfg)

	return &cfg, err
}
