package health

import (
	"sync/atomic"

	"github.com/gofiber/fiber/v2"
)

func RegisterRoutes(app *fiber.App, ready *atomic.Bool) {

	app.Get("/healthz", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	app.Get("/readyz", func(c *fiber.Ctx) error {
		if ready.Load() {
			return c.SendStatus(fiber.StatusOK)
		}
		return c.SendStatus(fiber.StatusServiceUnavailable)
	})
}
