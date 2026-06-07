package handler

import (
	"github.com/gofiber/fiber/v3"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/auth/service"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/pkg/cookie"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/pkg/response"
)

type OAuthHandler struct {
	svc     service.OAuthService
	authSvc service.AuthService
}

func NewOAuthHandler(svc service.OAuthService, authSvc service.AuthService) *OAuthHandler {
	return &OAuthHandler{
		svc:     svc,
		authSvc: authSvc,
	}
}

func (h *OAuthHandler) RegisterRoutes(api fiber.Router) {
	g := api.Group("/auth/oauth")
	g.Get("/:provider", h.getAuthURL)
	g.Get("/:provider/callback", h.handleCallback)
}

// getAuthURL godoc
// @Summary		Get OAuth URL
// @Description	Get redirect URL for OAuth provider
// @Tags			auth
// @Param			provider	path	string	true	"Provider (google/yandex/github)"
// @Success		302
// @Router			/auth/oauth/{provider} [get]
func (h *OAuthHandler) getAuthURL(c fiber.Ctx) error {
	provider := c.Params("provider")
	url, err := h.svc.GetAuthURL(provider)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	}

	return c.Redirect().To(url)
}

// handleCallback godoc
// @Summary		OAuth Callback
// @Description	Handle OAuth provider callback
// @Tags			auth
// @Param			provider	path	string	true	"Provider (google, yandex, github)"
// @Param			code		query	string	true	"OAuth code"
// @Success		200
// @Router			/auth/oauth/{provider}/callback [get]
func (h *OAuthHandler) handleCallback(c fiber.Ctx) error {
	provider := c.Params("provider")
	code := c.Query("code")
	if code == "" {
		return response.Error(c, fiber.StatusBadRequest, "code is required", nil)
	}

	accessToken, refreshToken, err := h.svc.HandleCallback(c.Context(), provider, code, c.IP(), c.Get("User-Agent"))
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	cookie.SetCookie(c, cookie.RefreshToken, refreshToken, h.authSvc.GetRefreshTokenTTL())

	return response.Success(c, fiber.StatusOK, "oauth login successful", fiber.Map{
		"access_token": accessToken,
	})
}
