package handler

import (
	"errors"

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
	g.Get("/verify-email", h.verifyEmail)
	g.Post("/verify-email", h.verifyEmail)
	g.Post("/resend-email-verification", middleware.Validate[resendEmailVerificationReq](), h.resendEmailVerification)
	g.Post("/forgot-password", middleware.Validate[forgotPasswordReq](), h.forgotPassword)
	g.Post("/reset-password", middleware.Validate[resetPasswordReq](), h.resetPassword)
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

type verifyEmailReq struct {
	Code string `json:"code"`
}

type resendEmailVerificationReq struct {
	Email string `json:"email" validate:"required,email"`
}

type forgotPasswordReq struct {
	Email string `json:"email" validate:"required,email"`
}

type resetPasswordReq struct {
	Code            string `json:"code" validate:"required"`
	Password        string `json:"password" validate:"required,min=6"`
	PasswordConfirm string `json:"passwordConfirm" validate:"required,min=6"`
}

// register godoc
// @Summary		Register new user
// @Description	Create a new user
// @Tags			auth
// @Accept			json
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
		if errors.Is(err, service.ErrUserAlreadyExists) {
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
// @Accept			json
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
// @Accept			json
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
// @Accept			json
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

// verifyEmail godoc
// @Summary		Verify email
// @Description	Verify email address using verification code
// @Tags			auth
// @Accept		json
// @Produce		json
// @Param			code	query		string			false	"Verification code"
// @Param			request	body		verifyEmailReq	false	"Verification code dto"
// @Success		200
// @Failure		400
// @Failure		500
// @Router			/auth/verify-email [get]
// @Router			/auth/verify-email [post]
func (h *AuthHandler) verifyEmail(c fiber.Ctx) error {
	code := c.Query("code")
	if code == "" {
		var req verifyEmailReq
		if err := c.Bind().Body(&req); err == nil {
			code = req.Code
		}
	}

	if code == "" {
		return response.Error(c, fiber.StatusBadRequest, "code is required", nil)
	}

	if err := h.svc.VerifyEmail(c.Context(), code); err != nil {
		if errors.Is(err, service.ErrInvalidCode) {
			return response.Error(c, fiber.StatusBadRequest, "invalid or expired verification code", nil)
		}
		return response.Error(c, fiber.StatusInternalServerError, "failed to verify email", nil)
	}

	return response.Success(c, fiber.StatusOK, "email verified successfully", nil)
}

// resendEmailVerification godoc
// @Summary		Resend email verification code
// @Description	Resend email verification code to user email
// @Tags			auth
// @Accept		json
// @Produce		json
// @Param			request	body		resendEmailVerificationReq	true	"Resend email verification dto"
// @Success		200
// @Failure		400
// @Failure		404
// @Failure		500
// @Router			/auth/resend-email-verification [post]
func (h *AuthHandler) resendEmailVerification(c fiber.Ctx) error {
	req := c.Locals("body").(resendEmailVerificationReq)

	if err := h.svc.ResendEmailVerification(c.Context(), req.Email); err != nil {
		if errors.Is(err, service.ErrEmailAlreadyVerified) {
			return response.Error(c, fiber.StatusBadRequest, "email is already verified", nil)
		}
		if errors.Is(err, service.ErrUserNotFound) {
			return response.Error(c, fiber.StatusNotFound, "user not found", nil)
		}
		return response.Error(c, fiber.StatusInternalServerError, "failed to resend verification code", nil)
	}

	return response.Success(c, fiber.StatusOK, "verification email sent", nil)
}

// forgotPassword godoc
// @Summary		Forgot password
// @Description	Send code to email
// @Tags			auth
// @Accept			json
// @Produce		json
// @Param			request	body		forgotPasswordReq	true	"Forgot password dto"
// @Success		200
// @Failure		400
// @Failure		404
// @Failure		500
// @Router			/auth/forgot-password [post]
func (h *AuthHandler) forgotPassword(c fiber.Ctx) error {
	req := c.Locals("body").(forgotPasswordReq)

	if err := h.svc.ForgotPassword(c.Context(), req.Email); err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			return response.Error(c, fiber.StatusNotFound, "user not found", nil)
		}
		return response.Error(c, fiber.StatusInternalServerError, "failed to send password reset code", nil)
	}

	return response.Success(c, fiber.StatusOK, "password reset code sent", nil)
}

// resetPassword godoc
// @Summary		Reset password
// @Description	Reset password via code
// @Tags			auth
// @Accept			json
// @Produce		json
// @Param			request	body		resetPasswordReq	true	"Reset password dto"
// @Success		200
// @Failure		400
// @Failure		500
// @Router			/auth/reset-password [post]
func (h *AuthHandler) resetPassword(c fiber.Ctx) error {
	req := c.Locals("body").(resetPasswordReq)

	if req.Password != req.PasswordConfirm {
		return response.Error(c, fiber.StatusBadRequest, "password didn't match", nil)
	}

	if err := h.svc.ResetPassword(c.Context(), req.Code, req.Password); err != nil {
		if errors.Is(err, service.ErrInvalidCode) {
			return response.Error(c, fiber.StatusBadRequest, "invalid or expired verification code", nil)
		}
		return response.Error(c, fiber.StatusInternalServerError, "failed to reset password", nil)
	}

	return response.Success(c, fiber.StatusOK, "password reset successfully", nil)
}
