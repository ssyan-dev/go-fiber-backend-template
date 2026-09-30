package handler

import (
	"github.com/gofiber/fiber/v3"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/middleware"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/pkg/auth"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/pkg/response"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/user/service"
)

type UserHandler struct {
	svc service.UserService
}

func NewUserHandler(svc service.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

func (h *UserHandler) RegisterRoutes(protected fiber.Router) {
	g := protected.Group("/users")
	g.Get("/me", h.getMe)
	g.Patch("/me", middleware.Validate[updateUserReq](), h.updateMe)
}

// getMe godoc
// @Summary		Get current user
// @Description	Get the authenticated user's profile information
// @Tags			users
// @Accept		json
// @Produce		json
// @Security	BearerAuth
// @Success		200	{object}	response.UserResponse{data=models.User}
// @Failure		401	{object}	response.ErrorResponse	"Unauthorized"
// @Failure		404	{object}	response.ErrorResponse	"User not found"
// @Router			/users/me [get]
func (h *UserHandler) getMe(c fiber.Ctx) error {
	userID, ok := auth.GetMe(c)
	if !ok {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized", nil)
	}

	user, err := h.svc.GetByID(c.Context(), userID)
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "user not found", nil)
	}

	return response.Success(c, fiber.StatusOK, "success", user)
}

type updateUserReq struct {
	Email           *string `json:"email" validate:"omitempty,email" example:"newemail@example.com"`
	CurrentPassword *string `json:"current_password" validate:"omitempty,min=6" example:"oldpassword123"`
	NewPassword     *string `json:"new_password" validate:"omitempty,min=6" example:"newpassword123"`
	AvatarURL       *string `json:"avatar_url" validate:"omitempty,url" example:"https://example.com/avatar.png"`
}

// updateMe godoc
// @Summary		Update current user
// @Description	Update the authenticated user's profile
// @Tags			users
// @Accept		json
// @Produce		json
// @Security	BearerAuth
// @Param			request	body		updateUserReq	true	"Update profile request"
// @Success		200	{object}	response.SuccessResponse
// @Failure		400	{object}	response.ErrorResponse	"Invalid current password or password required"
// @Failure		401	{object}	response.ErrorResponse	"Unauthorized"
// @Failure		500	{object}	response.ErrorResponse	"Internal server error"
// @Router			/users/me [patch]
func (h *UserHandler) updateMe(c fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(string)
	if !ok {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized", nil)
	}

	req := c.Locals("body").(updateUserReq)

	if err := h.svc.Update(c.Context(), userID, req.Email, req.CurrentPassword, req.NewPassword, req.AvatarURL); err != nil {
		if err == service.ErrInvalidCurrentPassword {
			return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
		}
		if err == service.ErrPasswordRequired {
			return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
		}
		return response.Error(c, fiber.StatusInternalServerError, "failed to update profile", nil)
	}

	return response.Success(c, fiber.StatusOK, "success update", nil)
}
