package middleware

import (
	"github.com/gofiber/fiber/v3"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/pkg/response"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/pkg/validator"
)

func Validate[T any]() fiber.Handler {
	return func(c fiber.Ctx) error {
		var body T
		if err := c.Bind().JSON(&body); err != nil {
			return response.Error(c, fiber.StatusBadRequest, "invalid request body", nil)
		}

		if errors := validator.Validate(body); len(errors) > 0 {
			return response.Error(c, fiber.StatusUnprocessableEntity, "validation failed", errors)
		}

		c.Locals("body", body)
		return c.Next()
	}
}
