package config

import (
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	App      AppConfig
	JWT      JWTConfig
	OAuth    OAuthConfig
	Postgres PostgresConfig
	Redis    RedisConfig
}

type AppConfig struct {
	Env           string `env:"APP_ENV" envDefault:"development"`
	Port          int    `env:"APP_PORT" envDefault:"8080"`
	GlobalPrefix  string `env:"APP_GLOBAL_PREFIX" envDefault:"/api/v1"`
	AllowedOrigin string `env:"APP_ALLOWED_ORIGIN" envDefault:"http://localhost:3000"`
}

type JWTConfig struct {
	SecretKey       string        `env:"JWT_SECRET_KEY,required"`
	AccessTokenTTL  time.Duration `env:"JWT_ACCESS_TOKEN_TTL" envDefault:"30m"`
	RefreshTokenTTL time.Duration `env:"JWT_REFRESH_TOKEN_TTL" envDefault:"336h"`
}

type OAuthConfig struct {
	Google GoogleOAuthConfig
	GitHub GitHubOAuthConfig
	Yandex YandexOAuthConfig
}

type GoogleOAuthConfig struct {
	Enabled      bool   `env:"GOOGLE_OAUTH_ENABLED" envDefault:"false"`
	ClientID     string `env:"GOOGLE_CLIENT_ID,required"`
	ClientSecret string `env:"GOOGLE_CLIENT_SECRET,required"`
	RedirectURL  string `env:"GOOGLE_REDIRECT_URL,required"`
}

type GitHubOAuthConfig struct {
	Enabled      bool   `env:"GITHUB_OAUTH_ENABLED" envDefault:"false"`
	ClientID     string `env:"GITHUB_CLIENT_ID,required"`
	ClientSecret string `env:"GITHUB_CLIENT_SECRET,required"`
	RedirectURL  string `env:"GITHUB_REDIRECT_URL,required"`
}

type YandexOAuthConfig struct {
	Enabled      bool   `env:"YANDEX_OAUTH_ENABLED" envDefault:"false"`
	ClientID     string `env:"YANDEX_CLIENT_ID,required"`
	ClientSecret string `env:"YANDEX_CLIENT_SECRET,required"`
	RedirectURL  string `env:"YANDEX_REDIRECT_URL,required"`
}

type PostgresConfig struct {
	Host     string `env:"POSTGRES_HOST" envDefault:"localhost"`
	Port     int    `env:"POSTGRES_PORT" envDefault:"5432"`
	User     string `env:"POSTGRES_USER,required"`
	Password string `env:"POSTGRES_PASSWORD,required"`
	DBName   string `env:"POSTGRES_DB,required"`
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
