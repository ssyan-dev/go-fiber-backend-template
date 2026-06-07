package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/auth/repository"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/config"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/models"
	sessionService "github.com/ssyan-dev/go-fiber-backend-template/internal/sessions/service"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/yandex"
)

var (
	ErrProviderNotSupported = errors.New("provider not supported")
	ErrProviderDisabled     = errors.New("provider is disabled")
	ErrFailedToFetchUser    = errors.New("failed to fetch user info from provider")
)

type OAuthService interface {
	GetAuthURL(provider string) (string, error)
	HandleCallback(ctx context.Context, provider, code, ip, userAgent string) (string, string, error)
}

type oAuthSvc struct {
	repo       repository.AuthRepository
	sessionSvc sessionService.SessionService
	cfg        *config.OAuthConfig
	jwtCfg     *config.JWTConfig
	l          *zap.Logger
}

func NewOAuthService(
	repo repository.AuthRepository,
	sessionSvc sessionService.SessionService,
	jwtCfg *config.JWTConfig,
	oauthCfg *config.OAuthConfig,
	l *zap.Logger,
) OAuthService {
	return &oAuthSvc{
		repo:       repo,
		sessionSvc: sessionSvc,
		jwtCfg:     jwtCfg,
		cfg:        oauthCfg,
		l:          l,
	}
}

func (s *oAuthSvc) GetAuthURL(provider string) (string, error) {
	conf, err := s.getProviderConfig(provider)
	if err != nil {
		return "", err
	}

	return conf.AuthCodeURL("state"), nil
}

func (s *oAuthSvc) HandleCallback(ctx context.Context, provider, code, ip, userAgent string) (string, string, error) {
	conf, err := s.getProviderConfig(provider)
	if err != nil {
		return "", "", err
	}

	token, err := conf.Exchange(ctx, code)
	if err != nil {
		return "", "", err
	}

	providerUserID, email, avatarURL, err := s.fetchUserInfo(ctx, provider, conf, token)
	if err != nil {
		return "", "", err
	}

	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		user = &models.User{
			Email:           email,
			AvatarURL:       &avatarURL,
			Role:            models.RoleDefault,
			IsEmailVerified: true,
		}
		if err := s.repo.CreateUser(ctx, user); err != nil {
			s.l.Error("failed to create oauth user", zap.Error(err))
			return "", "", err
		}
	}

	err = s.repo.LinkOAuthProvider(ctx, user.ID.String(), provider, providerUserID, email)
	if err != nil {
		s.l.Error("failed to link oauth provider", zap.Error(err))
	}

	accessToken, err := s.generateJWTToken(user, s.jwtCfg.AccessTokenTTL)
	if err != nil {
		return "", "", err
	}

	refreshToken, err := s.generateJWTToken(user, s.jwtCfg.RefreshTokenTTL)
	if err != nil {
		return "", "", err
	}

	session := &models.Session{
		UserID:       user.ID,
		RefreshToken: refreshToken,
		AccessToken:  accessToken,
		IP:           ip,
		UserAgent:    userAgent,
		ExpiresAt:    time.Now().Add(s.jwtCfg.RefreshTokenTTL),
	}
	if err := s.sessionSvc.Create(ctx, session); err != nil {
		s.l.Error("failed to create session", zap.Error(err))
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func (s *oAuthSvc) getProviderConfig(provider string) (*oauth2.Config, error) {
	switch provider {
	case "google":
		if !s.cfg.Google.Enabled {
			return nil, ErrProviderDisabled
		}
		return &oauth2.Config{
			ClientID:     s.cfg.Google.ClientID,
			ClientSecret: s.cfg.Google.ClientSecret,
			RedirectURL:  s.cfg.Google.RedirectURL,
			Endpoint:     google.Endpoint,
			Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"},
		}, nil
	case "github":
		if !s.cfg.GitHub.Enabled {
			return nil, ErrProviderDisabled
		}
		return &oauth2.Config{
			ClientID:     s.cfg.GitHub.ClientID,
			ClientSecret: s.cfg.GitHub.ClientSecret,
			RedirectURL:  s.cfg.GitHub.RedirectURL,
			Endpoint:     github.Endpoint,
			Scopes:       []string{"user:email"},
		}, nil
	case "yandex":
		if !s.cfg.Yandex.Enabled {
			return nil, ErrProviderDisabled
		}
		return &oauth2.Config{
			ClientID:     s.cfg.Yandex.ClientID,
			ClientSecret: s.cfg.Yandex.ClientSecret,
			RedirectURL:  s.cfg.Yandex.RedirectURL,
			Endpoint:     yandex.Endpoint,
		}, nil
	default:
		return nil, ErrProviderNotSupported
	}
}

func (s *oAuthSvc) fetchUserInfo(ctx context.Context, provider string, conf *oauth2.Config, token *oauth2.Token) (string, string, string, error) {
	client := conf.Client(ctx, token)
	var url string
	switch provider {
	case "google":
		url = "https://www.googleapis.com/oauth2/v2/userinfo"
	case "github":
		url = "https://api.github.com/user"
	case "yandex":
		url = "https://login.yandex.ru/info"
	}

	resp, err := client.Get(url)
	if err != nil {
		return "", "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", "", ErrFailedToFetchUser
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", "", err
	}

	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return "", "", "", err
	}

	providerUserID := ""
	if id, ok := data["id"]; ok {
		providerUserID = fmt.Sprintf("%v", id)
	}

	email := ""
	if e, ok := data["email"].(string); ok && e != "" {
		email = e
	} else if de, ok := data["default_email"].(string); ok && de != "" {
		email = de
	}

	if email == "" && provider == "github" {
		emailsResp, err := client.Get("https://api.github.com/user/emails")
		if err == nil {
			defer emailsResp.Body.Close()
			var emails []struct {
				Email   string `json:"email"`
				Primary bool   `json:"primary"`
			}
			json.NewDecoder(emailsResp.Body).Decode(&emails)
			for _, e := range emails {
				if e.Primary {
					email = e.Email
					break
				}
			}
		}
	}

	avatar := ""
	if av, ok := data["avatar_url"].(string); ok && av != "" {
		avatar = av
	} else if pic, ok := data["picture"].(string); ok && pic != "" {
		avatar = pic
	} else if daID, ok := data["default_avatar_id"].(string); ok && daID != "" {
		avatar = fmt.Sprintf("https://avatars.yandex.net/get-yapic/%s/islands-200", daID)
	}

	return providerUserID, email, avatar, nil
}

func (s *oAuthSvc) generateJWTToken(user *models.User, ttl time.Duration) (string, error) {
	claims := jwt.MapClaims{
		"sub":  user.ID.String(),
		"role": user.Role,
		"exp":  time.Now().Add(ttl).Unix(),
		"iat":  time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtCfg.SecretKey))
}
