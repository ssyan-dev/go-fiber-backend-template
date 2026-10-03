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
	svc     service.AuthService
	limiter fiber.Handler
}

func NewAuthHandler(svc service.AuthService, limiter fiber.Handler) *AuthHandler {
	return &AuthHandler{
		svc:     svc,
		limiter: limiter,
	}
}

func (h *AuthHandler) RegisterRoutes(api fiber.Router) {
	authLimiter := h.limiter
	if authLimiter == nil {
		authLimiter = middleware.AuthLimiter(nil, nil)
	}
	g := api.Group("/auth", authLimiter)

	g.Post("/register", middleware.Validate[registerReq](), h.register)
	g.Post("/login", middleware.Validate[loginReq](), h.login)
	g.Post("/logout", h.logout)
	g.Post("/refresh", h.refresh)
	g.Get("/verify-email", h.verifyEmail)
	g.Post("/resend-email-verification", middleware.Validate[resendEmailVerificationReq](), h.resendEmailVerification)
	g.Post("/forgot-password", middleware.Validate[forgotPasswordReq](), h.forgotPassword)
	g.Post("/reset-password", middleware.Validate[resetPasswordReq](), h.resetPassword)
}

type registerReq struct {
	Email           string `json:"email" validate:"required,email" example:"user@example.com"`
	Password        string `json:"password" validate:"required,min=6" example:"password123"`
	PasswordConfirm string `json:"passwordConfirm" validate:"required,min=6" example:"password123"`
}

type loginReq struct {
	Email    string `json:"email" validate:"required,email" example:"user@example.com"`
	Password string `json:"password" validate:"required" example:"password123"`
}

type resendEmailVerificationReq struct {
	Email string `json:"email" validate:"required,email" example:"user@example.com"`
}

type forgotPasswordReq struct {
	Email string `json:"email" validate:"required,email" example:"user@example.com"`
}

type resetPasswordReq struct {
	Code            string `json:"code" validate:"required" example:"123456"`
	Password        string `json:"password" validate:"required,min=6" example:"newpassword123"`
	PasswordConfirm string `json:"passwordConfirm" validate:"required,min=6" example:"newpassword123"`
}

// register godoc
// @Summary		Register new user
// @Description	Create a new user account with email and password. A verification email will be sent if SMTP enabled and configured
// @Tags			auth
// @Accept			json
// @Produce		json
// @Param			request	body		registerReq	true	"Register request"
// @Success		201		{object}	response.UserResponse{data=models.User}
// @Failure		400		{object}	response.ErrorResponse	"Passwords don't match"
// @Failure		409		{object}	response.ErrorResponse	"User already exists"
// @Failure		500		{object}	response.ErrorResponse	"Internal server error"
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
// @Description	Authenticate user by email and password. Returns access token in body, sets refresh token as HTTP-only cookie
// @Tags			auth
// @Accept			json
// @Produce		json
// @Param			request	body		loginReq	true	"Login request"
// @Success		200		{object}	response.TokenResponse
// @Failure		401		{object}	response.ErrorResponse	"Invalid credentials"
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
// @Description	Issue new access and refresh tokens using the refresh token from HTTP-only cookie
// @Tags			auth
// @Accept			json
// @Produce		json
// @Success		200	{object}	response.TokenResponse
// @Failure		401	{object}	response.ErrorResponse	"Refresh token not found or invalid"
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
// @Description	Revoke current session and clear refresh token cookie
// @Tags			auth
// @Accept			json
// @Produce		json
// @Success		200	{object}	response.SuccessResponse
// @Failure		401	{object}	response.ErrorResponse	"Unauthorized"
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
// @Description	Verify email address using a verification code passed as a query parameter
// @Tags			auth
// @Produce		json
// @Param			code	query		string	true	"Verification code"
// @Success		200	{object}	response.SuccessResponse
// @Failure		400	{object}	response.ErrorResponse	"Code is required or invalid"
// @Failure		500	{object}	response.ErrorResponse	"Internal server error"
// @Router			/auth/verify-email [get]
func (h *AuthHandler) verifyEmail(c fiber.Ctx) error {
	code := c.Query("code")
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
// @Description	Resend a new verification code to the specified email address
// @Tags			auth
// @Accept		json
// @Produce		json
// @Param			request	body		resendEmailVerificationReq	true	"Resend verification request"
// @Success		200	{object}	response.SuccessResponse
// @Failure		400	{object}	response.ErrorResponse	"Email already verified"
// @Failure		404	{object}	response.ErrorResponse	"User not found"
// @Failure		500	{object}	response.ErrorResponse	"Internal server error"
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
// @Description	Send a password reset code to the specified email address
// @Tags			auth
// @Accept			json
// @Produce		json
// @Param			request	body		forgotPasswordReq	true	"Forgot password request"
// @Success		200	{object}	response.SuccessResponse
// @Failure		404	{object}	response.ErrorResponse	"User not found"
// @Failure		500	{object}	response.ErrorResponse	"Internal server error"
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
// @Description	Reset user password using a verification code sent via email
// @Tags			auth
// @Accept			json
// @Produce		json
// @Param			request	body		resetPasswordReq	true	"Reset password request"
// @Success		200	{object}	response.SuccessResponse
// @Failure		400	{object}	response.ErrorResponse	"Passwords don't match or invalid code"
// @Failure		500	{object}	response.ErrorResponse	"Internal server error"
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
