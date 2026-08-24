package main

import (
	"log"

	"github.com/gofiber/fiber/v3"
)

func main() {
	app := fiber.New()

	app.Get("/", func(c fiber.Ctx) error {
		return c.SendString("Olá Este é o Backend Mínimo de aula-sync-app!")
	})

	log.Fatal(app.Listen(":3000"))
}
