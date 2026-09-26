package handler

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/middleware"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/pkg/response"
	vcService "github.com/ssyan-dev/go-fiber-backend-template/internal/verification_codes/service"
)

type VerificationHandler struct {
	vcSvc vcService.VerificationCodeService
}

func NewVerificationHandler(vcSvc vcService.VerificationCodeService) *VerificationHandler {
	return &VerificationHandler{
		vcSvc: vcSvc,
	}
}

func (h *VerificationHandler) RegisterRoutes(api fiber.Router) {
	g := api.Group("/verification")
	g.Get("/", h.verification)
	g.Post("/resend-email-verification", middleware.Validate[resendEmailVerificationReq](), h.resendEmailVerificationCode)
}

type resendEmailVerificationReq struct {
	Email string `json:"email" validate:"required,email"`
}

// verification godoc
// @Summary		Verification
// @Description	Verification (email verification, password reset, etc..)
// @Tags			verification
// @Accept		json
// @Produce		json
// @Success		200
// @Failure		400
// @Failure		500
// @Param			code		query	string	true	"Verification code"
// @Router		/verification [get]
func (h *VerificationHandler) verification(c fiber.Ctx) error {
	code := c.Query("code")
	if code == "" {
		return response.Error(c, fiber.StatusBadRequest, "code is required", nil)
	}

	if err := h.vcSvc.VerifyEmail(c.Context(), code); err != nil {
		if errors.Is(err, vcService.ErrInvalidCode) {
			return response.Error(c, fiber.StatusBadRequest, "invalid or expired verification code", nil)
		}
		return response.Error(c, fiber.StatusInternalServerError, "failed to verify email", nil)
	}

	return response.Success(c, fiber.StatusOK, "email verified successfully", nil)
}

// resendEmailVerification godoc
// @Summary		Resend Email Verification code
// @Description	Resend Email Verification code
// @Tags			verification
// @Accept		json
// @Produce		json
// @Param			request	body		resendEmailVerificationReq	true	"Resend email verification dto"
// @Success		200
// @Failure		400
// @Failure		500
// @Router		/verification/resend-email-verification [post]
func (h *VerificationHandler) resendEmailVerificationCode(c fiber.Ctx) error {
	req := c.Locals("body").(resendEmailVerificationReq)

	if err := h.vcSvc.ResendEmailVerificationCode(c.Context(), req.Email); err != nil {
		if errors.Is(err, vcService.ErrEmailAlreadyVerified) {
			return response.Error(c, fiber.StatusBadRequest, "email is already verified", nil)
		}
		if errors.Is(err, vcService.ErrUserNotFound) {
			return response.Error(c, fiber.StatusNotFound, "user not found", nil)
		}
		return response.Error(c, fiber.StatusInternalServerError, "failed to resend verification code", nil)
	}

	return response.Success(c, fiber.StatusOK, "verification email sent", nil)
}
