package cookie

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/config"
)

const (
	RefreshToken = "refresh_token"
)

func SetCookie(c fiber.Ctx, name, token string, ttl time.Duration) {
	c.Cookie(&fiber.Cookie{
		Name:     name,
		Value:    token,
		Expires:  time.Now().Add(ttl),
		HTTPOnly: true,
		Secure:   config.IsProduction,
		SameSite: "lax",
		Path:     "/",
	})
}

func GetCookie(c fiber.Ctx, name string) string {
	return c.Cookies(name)
}

func ClearCookie(c fiber.Ctx, name string) {
	c.Cookie(&fiber.Cookie{
		Name:     name,
		Value:    "",
		Expires:  time.Now().Add(-time.Hour),
		HTTPOnly: true,
		Secure:   config.IsProduction,
		SameSite: "Lax",
		Path:     "/",
	})
}
