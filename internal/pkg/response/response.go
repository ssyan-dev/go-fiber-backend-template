package response

import "github.com/gofiber/fiber/v3"

type response struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}

func Success(c fiber.Ctx, status int, message string, data any) error {
	return c.Status(status).JSON(response{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func Error(c fiber.Ctx, status int, message string, data any) error {
	return c.Status(status).JSON(response{
		Success: false,
		Message: message,
		Data:    data,
	})
}

func Empty(c fiber.Ctx, status int) error {
	return c.Status(status).JSON(response{Success: true})
}
