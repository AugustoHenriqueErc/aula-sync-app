package httpapi

import (
	"errors"
	"github.com/gofiber/fiber/v3"
)

// New configura a API sem abrir uma porta, permitindo testes HTTP em memória.
func New() *fiber.App {
	app := fiber.New(fiber.Config{ErrorHandler: handleError})
	app.Get("/", func(c fiber.Ctx) error {
		return c.SendString("Olá Este é o Backend Mínimo de aula-sync-app!")
	})
	app.Get("/health", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})
	return app
}

func handleError(c fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := "Erro interno do servidor"
	var httpError *fiber.Error
	if errors.As(err, &httpError) {
		code = httpError.Code
		if code < fiber.StatusInternalServerError {
			message = httpError.Message
		}
	}
	return c.Status(code).JSON(fiber.Map{"error": message})
}
