package handler

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/middleware"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/models"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/pkg/auth"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/pkg/response"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/user/service"
)

type AdminUserHandler struct {
	svc service.UserService
}

func NewAdminUserHandler(svc service.UserService) *AdminUserHandler {
	return &AdminUserHandler{svc: svc}
}

func (h *AdminUserHandler) RegisterRoutes(admin fiber.Router) {
	g := admin.Group("/users")
	g.Get("", middleware.ValidateQuery[listUsersQuery](), h.listUsers)
	g.Get("/:id", h.getUserByID)
	g.Patch("/:id", middleware.Validate[adminUpdateUserReq](), h.updateUser)
	g.Delete("/:id", h.deleteUser)
}

type listUsersQuery struct {
	Page     int              `query:"page" json:"page" validate:"omitempty,min=1"`
	Limit    int              `query:"limit" json:"limit" validate:"omitempty,min=1,max=100"`
	Search   *string          `query:"search" json:"search" validate:"omitempty,max=100"`
	Role     *models.UserRole `query:"role" json:"role" validate:"omitempty,oneof=default admin"`
	IsBanned *bool            `query:"is_banned" json:"is_banned" validate:"omitempty"`
}

type adminUpdateUserReq struct {
	Email           *string          `json:"email" validate:"omitempty,email" example:"admin@example.com"`
	Role            *models.UserRole `json:"role" validate:"omitempty,oneof=default admin" example:"admin"`
	Password        *string          `json:"password" validate:"omitempty,min=6" example:"newpassword123"`
	AvatarURL       *string          `json:"avatar_url" validate:"omitempty,url" example:"https://example.com/avatar.png"`
	IsBanned        *bool            `json:"is_banned" validate:"omitempty" example:"false"`
	IsEmailVerified *bool            `json:"is_email_verified" validate:"omitempty" example:"true"`
}

// listUsers godoc
// @Summary		List all users
// @Description	Get a paginated list of users with optional filters by search, role, and ban status. Requires admin role
// @Tags			admin-users
// @Accept		json
// @Produce		json
// @Security	BearerAuth
// @Param			page		query		int		false	"Page number (default: 1)"		minimum(1)
// @Param			limit		query		int		false	"Items per page (default: 10)"	minimum(1)	maximum(100)
// @Param			search		query		string	false	"Search by email"
// @Param			role		query		string	false	"Filter by role"	Enums(default, admin)
// @Param			is_banned	query		bool	false	"Filter by ban status"
// @Success		200			{object}	response.UserResponse{data=models.PaginatedUsersResponse}
// @Failure		401			{object}	response.ErrorResponse	"Unauthorized"
// @Failure		403			{object}	response.ErrorResponse	"Forbidden"
// @Failure		500			{object}	response.ErrorResponse	"Internal server error"
// @Router		/admin/users [get]
func (h *AdminUserHandler) listUsers(c fiber.Ctx) error {
	query := c.Locals("query").(listUsersQuery)

	if query.Page <= 0 {
		query.Page = 1
	}
	if query.Limit <= 0 {
		query.Limit = 10
	}

	filter := models.ListUsersFilter{
		Search:   query.Search,
		Role:     query.Role,
		IsBanned: query.IsBanned,
		Page:     query.Page,
		Limit:    query.Limit,
	}

	users, total, err := h.svc.List(c.Context(), filter)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "failed to list users", nil)
	}

	totalPages := 0
	if query.Limit > 0 {
		totalPages = int((total + int64(query.Limit) - 1) / int64(query.Limit))
	}

	res := models.PaginatedUsersResponse{
		Users: users,
		Meta: models.PaginationMeta{
			Page:       query.Page,
			Limit:      query.Limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}

	return response.Success(c, fiber.StatusOK, "success", res)
}

// getUserByID godoc
// @Summary		Get user by ID
// @Description	Get a specific user by their UUID. Requires admin role
// @Tags			admin-users
// @Accept		json
// @Produce		json
// @Security	BearerAuth
// @Param			id	path		string	true	"User ID (UUID)"
// @Success		200	{object}	response.UserResponse{data=models.User}
// @Failure		400	{object}	response.ErrorResponse	"Invalid user ID"
// @Failure		401	{object}	response.ErrorResponse	"Unauthorized"
// @Failure		403	{object}	response.ErrorResponse	"Forbidden"
// @Failure		404	{object}	response.ErrorResponse	"User not found"
// @Router		/admin/users/{id} [get]
func (h *AdminUserHandler) getUserByID(c fiber.Ctx) error {
	id := c.Params("id")
	if _, err := uuid.Parse(id); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid user id", nil)
	}

	user, err := h.svc.GetByID(c.Context(), id)
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "user not found", nil)
	}

	return response.Success(c, fiber.StatusOK, "success", user)
}

// updateUser godoc
// @Summary		Update user
// @Description	Update a user's profile by their UUID. Admins cannot demote or ban themselves. Requires admin role
// @Tags			admin-users
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			id		path		string				true	"User ID (UUID)"
// @Param			request	body		adminUpdateUserReq	true	"Update user request"
// @Success		200		{object}	response.UserResponse{data=models.User}
// @Failure		400		{object}	response.ErrorResponse	"Cannot demote/ban yourself or invalid input"
// @Failure		401		{object}	response.ErrorResponse	"Unauthorized"
// @Failure		403		{object}	response.ErrorResponse	"Forbidden"
// @Failure		404		{object}	response.ErrorResponse	"User not found"
// @Failure		409		{object}	response.ErrorResponse	"Email already taken"
// @Failure		500		{object}	response.ErrorResponse	"Internal server error"
// @Router		/admin/users/{id} [patch]
func (h *AdminUserHandler) updateUser(c fiber.Ctx) error {
	targetID := c.Params("id")
	if _, err := uuid.Parse(targetID); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid user id", nil)
	}

	adminID, ok := auth.GetMe(c)
	if !ok {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized", nil)
	}

	req := c.Locals("body").(adminUpdateUserReq)

	input := service.AdminUpdateUserInput{
		Email:           req.Email,
		Role:            req.Role,
		Password:        req.Password,
		AvatarURL:       req.AvatarURL,
		IsBanned:        req.IsBanned,
		IsEmailVerified: req.IsEmailVerified,
	}

	user, err := h.svc.AdminUpdate(c.Context(), targetID, adminID, input)
	if err != nil {
		if errors.Is(err, service.ErrCannotDemoteSelf) || errors.Is(err, service.ErrCannotBanSelf) {
			return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
		}
		if errors.Is(err, service.ErrUserAlreadyExists) {
			return response.Error(c, fiber.StatusConflict, err.Error(), nil)
		}
		if errors.Is(err, service.ErrUserNotFound) {
			return response.Error(c, fiber.StatusNotFound, err.Error(), nil)
		}
		return response.Error(c, fiber.StatusInternalServerError, "failed to update user", nil)
	}

	return response.Success(c, fiber.StatusOK, "user updated successfully", user)
}

// deleteUser godoc
// @Summary		Delete user
// @Description	Delete a user by their UUID. Admins cannot delete themselves. Requires admin role
// @Tags			admin-users
// @Accept		json
// @Produce		json
// @Security	BearerAuth
// @Param			id	path	string	true	"User ID (UUID)"
// @Success		200	{object}	response.SuccessResponse
// @Failure		400	{object}	response.ErrorResponse	"Cannot delete yourself"
// @Failure		401	{object}	response.ErrorResponse	"Unauthorized"
// @Failure		403	{object}	response.ErrorResponse	"Forbidden"
// @Failure		404	{object}	response.ErrorResponse	"User not found"
// @Failure		500	{object}	response.ErrorResponse	"Internal server error"
// @Router		/admin/users/{id} [delete]
func (h *AdminUserHandler) deleteUser(c fiber.Ctx) error {
	targetID := c.Params("id")
	if _, err := uuid.Parse(targetID); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid user id", nil)
	}

	adminID, ok := auth.GetMe(c)
	if !ok {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized", nil)
	}

	if err := h.svc.AdminDelete(c.Context(), targetID, adminID); err != nil {
		if errors.Is(err, service.ErrCannotDeleteSelf) {
			return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
		}
		if errors.Is(err, service.ErrUserNotFound) {
			return response.Error(c, fiber.StatusNotFound, err.Error(), nil)
		}
		return response.Error(c, fiber.StatusInternalServerError, "failed to delete user", nil)
	}

	return response.Success(c, fiber.StatusOK, "user deleted successfully", nil)
}
