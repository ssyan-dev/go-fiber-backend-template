package handler

import (
	"github.com/gofiber/fiber/v3"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/auth/service"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/middleware"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/pkg/cookie"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/pkg/response"
)

type AuthHandler struct {
	svc service.AuthService
}

func NewAuthHandler(svc service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

func (h *AuthHandler) RegisterRoutes(api fiber.Router) {
	g := api.Group("/auth")
	g.Post("/register", middleware.Validate[registerReq](), h.register)
	g.Post("/login", middleware.Validate[loginReq](), h.login)
	g.Post("/logout", h.logout)
	g.Post("/refresh", h.refresh)
}

type registerReq struct {
	Email           string `json:"email" validate:"required,email"`
	Password        string `json:"password" validate:"required,min=6"`
	PasswordConfirm string `json:"passwordConfirm" validate:"required,min=6"`
}

type loginReq struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// register godoc
// @Summary		Register new user
// @Description	Create a new user
// @Tags			auth
// @Accept	  json
// @Produce		json
// @Param			request	body		registerReq	true	"Register dto"
// @Router			/auth/register [post]
func (h *AuthHandler) register(c fiber.Ctx) error {
	req := c.Locals("body").(registerReq)

	if req.Password != req.PasswordConfirm {
		return response.Error(c, fiber.StatusBadRequest, "password didn't match", nil)
	}

	user, err := h.svc.Register(c.Context(), req.Email, req.Password)
	if err != nil {
		if err == service.ErrUserAlreadyExists {
			return response.Error(c, fiber.StatusConflict, "user already exists", nil)
		}
		return response.Error(c, fiber.StatusInternalServerError, "failed to register user", nil)
	}

	return response.Success(c, fiber.StatusCreated, "user registered successfully", user)
}

// login godoc
// @Summary		Login user
// @Description	Login user
// @Tags			auth
// @Accept		json
// @Produce		json
// @Param			request	body		loginReq	true	"login dto"
// @Success		200
// @Failure		401
// @Router			/auth/login [post]
func (h *AuthHandler) login(c fiber.Ctx) error {
	req := c.Locals("body").(loginReq)

	accessToken, refreshToken, err := h.svc.Login(c.Context(), req.Email, req.Password, c.IP(), c.Get("User-Agent"))
	if err != nil {
		return response.Error(c, fiber.StatusUnauthorized, err.Error(), nil)
	}

	cookie.SetCookie(c, cookie.RefreshToken, refreshToken, h.svc.GetRefreshTokenTTL())

	return response.Success(c, fiber.StatusOK, "login successful", fiber.Map{
		"access_token": accessToken,
	})
}

// refresh godoc
// @Summary		Refresh JWT tokens
// @Description	Refresh JWT tokens
// @Tags			auth
// @Accept		json
// @Produce		json
// @Success		200
// @Failure		401
// @Router			/auth/refresh [post]
func (h *AuthHandler) refresh(c fiber.Ctx) error {
	refreshToken := cookie.GetCookie(c, cookie.RefreshToken)
	if refreshToken == "" {
		return response.Error(c, fiber.StatusUnauthorized, "refresh token not found", nil)
	}

	newAccessToken, newRefreshToken, err := h.svc.Refresh(c.Context(), refreshToken, c.IP(), c.Get("User-Agent"))
	if err != nil {
		return response.Error(c, fiber.StatusUnauthorized, err.Error(), nil)
	}

	cookie.SetCookie(c, cookie.RefreshToken, newRefreshToken, h.svc.GetRefreshTokenTTL())

	return response.Success(c, fiber.StatusOK, "tokens successful refreshed", fiber.Map{
		"access_token": newAccessToken,
	})
}

// logout godoc
// @Summary		Logout user
// @Description	Logout user
// @Tags			auth
// @Accept		json
// @Produce		json
// @Success		200
// @Failure		401
// @Router			/auth/logout [post]
func (h *AuthHandler) logout(c fiber.Ctx) error {
	refreshToken := cookie.GetCookie(c, cookie.RefreshToken)
	if refreshToken != "" {
		_ = h.svc.Logout(c.Context(), refreshToken)
	}

	cookie.ClearCookie(c, cookie.RefreshToken)
	return response.Success(c, fiber.StatusOK, "logout successful", nil)
}
