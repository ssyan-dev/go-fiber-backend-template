package auth

import (
	"github.com/gofiber/fiber/v3"
)

const (
	UserIDLocals = "user_id"
)

func GetMe(c fiber.Ctx) (string, bool) {
	userID, ok := c.Locals(UserIDLocals).(string)
	if !ok {
		return "", false
	}
	return userID, true
}
