package handler

import (
	"github.com/gofiber/fiber/v3"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/pkg/auth"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/pkg/response"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/sessions/service"
)

type SessionHandler struct {
	svc service.SessionService
}

func NewSessionHandler(svc service.SessionService) *SessionHandler {
	return &SessionHandler{svc: svc}
}

func (h *SessionHandler) RegisterRoutes(protected fiber.Router) {
	g := protected.Group("/auth/sessions")
	g.Get("/", h.getSessions)
	g.Patch("/:sessionId/block", h.blockSession)
	g.Delete("/:sessionId", h.deleteSession)
	g.Delete("/", h.deleteAllSessions)
}

// getSessions godoc
// @Summary		Get active sessions
// @Description	Get all active sessions of the authenticated user
// @Tags			sessions
// @Accept		json
// @Produce		json
// @Security		BearerAuth
// @Success		200	{object}	response.UserResponse{data=[]models.Session}
// @Failure		401	{object}	response.ErrorResponse	"Unauthorized"
// @Failure		500	{object}	response.ErrorResponse	"Internal server error"
// @Router			/auth/sessions [get]
func (h *SessionHandler) getSessions(c fiber.Ctx) error {
	userID, ok := auth.GetMe(c)
	if !ok {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized", nil)
	}

	sessions, err := h.svc.GetByUserID(c.Context(), userID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "failed to fetch sessions", nil)
	}

	return response.Success(c, fiber.StatusOK, "success", sessions)
}

// blockSession godoc
// @Summary		Block session
// @Description	Block a specific session by its ID. The blocked session's tokens will be invalidated
// @Tags			sessions
// @Accept		json
// @Produce		json
// @Security		BearerAuth
// @Param			sessionId	path	string	true	"Session ID (UUID)"
// @Success		200	{object}	response.SuccessResponse
// @Failure		401	{object}	response.ErrorResponse	"Unauthorized"
// @Failure		500	{object}	response.ErrorResponse	"Internal server error"
// @Router			/auth/sessions/{sessionId}/block [patch]
func (h *SessionHandler) blockSession(c fiber.Ctx) error {
	userID, ok := auth.GetMe(c)
	if !ok {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized", nil)
	}

	sessionID := c.Params("sessionId")
	if err := h.svc.Block(c.Context(), sessionID, userID); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "failed to block session", nil)
	}

	return response.Success(c, fiber.StatusOK, "session blocked", nil)
}

// deleteSession godoc
// @Summary		Delete session
// @Description	Delete a specific session by its ID. This will immediately revoke the session
// @Tags			sessions
// @Accept		json
// @Produce		json
// @Security		BearerAuth
// @Param			sessionId	path	string	true	"Session ID (UUID)"
// @Success		200	{object}	response.SuccessResponse
// @Failure		401	{object}	response.ErrorResponse	"Unauthorized"
// @Failure		500	{object}	response.ErrorResponse	"Internal server error"
// @Router			/auth/sessions/{sessionId} [delete]
func (h *SessionHandler) deleteSession(c fiber.Ctx) error {
	userID, ok := auth.GetMe(c)
	if !ok {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized", nil)
	}

	sessionID := c.Params("sessionId")
	if err := h.svc.DeleteByID(c.Context(), sessionID, userID); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "failed to delete session", nil)
	}

	return response.Success(c, fiber.StatusOK, "session deleted", nil)
}

// deleteAllSessions godoc
// @Summary		Delete all sessions
// @Description	Delete all active sessions of the authenticated user. This will revoke all tokens
// @Tags			sessions
// @Accept		json
// @Produce		json
// @Security		BearerAuth
// @Success		200	{object}	response.SuccessResponse
// @Failure		401	{object}	response.ErrorResponse	"Unauthorized"
// @Failure		500	{object}	response.ErrorResponse	"Internal server error"
// @Router			/auth/sessions [delete]
func (h *SessionHandler) deleteAllSessions(c fiber.Ctx) error {
	userID, ok := auth.GetMe(c)
	if !ok {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized", nil)
	}

	if err := h.svc.DeleteAllByUserID(c.Context(), userID); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "failed to delete sessions", nil)
	}

	return response.Success(c, fiber.StatusOK, "all sessions deleted", nil)
}
