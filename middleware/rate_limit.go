// middleware/rate_limit.go
package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

func RateLimit() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        5,               // 5 ta so'rov
		Expiration: 1 * time.Minute, // 1 daqiqada
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP() // IP bo'yicha limit
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(429).JSON(fiber.Map{
				"error": "Juda ko'p so'rov. Bir daqiqa kuting.",
			})
		},
	})
}
