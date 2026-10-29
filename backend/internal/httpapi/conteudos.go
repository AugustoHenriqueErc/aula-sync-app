package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"mime"

	"app-sync-backend/internal/storage"

	"github.com/gofiber/fiber/v3"
)

func registrarRotasConteudos(app *fiber.App, banco *storage.Banco) {
	app.Get("/conteudos", func(c fiber.Ctx) error {
		if len(c.Request().URI().QueryString()) > 0 {
			return fiber.NewError(400, "Esta consulta não aceita parâmetros")
		}
		conteudos, err := banco.ListarConteudos(c.Context())
		if err != nil {
			return err
		}
		return c.JSON(conteudos)
	})
	app.Post("/conteudos", func(c fiber.Ctx) error {
		tipo, _, err := mime.ParseMediaType(c.Get(fiber.HeaderContentType))
		if err != nil || tipo != "application/json" {
			return fiber.NewError(415, "Use Content-Type: application/json")
		}
		var entrada *struct {
			Nome string `json:"nome"`
		}
		d := json.NewDecoder(bytes.NewReader(c.Body()))
		d.DisallowUnknownFields()
		if err := d.Decode(&entrada); err != nil || entrada == nil {
			return fiber.NewError(400, "Envie um objeto JSON com nome")
		}
		if err := d.Decode(new(any)); err != io.EOF {
			return fiber.NewError(400, "Envie um único objeto JSON")
		}
		conteudo, err := banco.CriarConteudo(c.Context(), entrada.Nome)
		if err != nil {
			return err
		}
		return c.Status(201).JSON(conteudo)
	})
}
