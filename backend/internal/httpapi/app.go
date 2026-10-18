package httpapi

import (
	"errors"
	"log/slog"

	"app-sync-backend/internal/aula"
	"app-sync-backend/internal/storage"

	"github.com/gofiber/fiber/v3"
)

// New configura a API sem abrir uma porta, permitindo testes HTTP em memória.
func New(banco *storage.Banco) *fiber.App {
	app := fiber.New(fiber.Config{ErrorHandler: handleError, BodyLimit: 64 * 1024})
	app.Get("/", func(c fiber.Ctx) error {
		return c.SendString("Olá Este é o Backend Mínimo de aula-sync-app!")
	})
	app.Get("/health", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})
	registrarRotasAulas(app, banco)
	registrarRotasTurmas(app, banco)
	return app
}

func handleError(c fiber.Ctx, err error) error {
	if errors.Is(err, storage.ErrConflito) || errors.Is(err, storage.ErrCancelada) {
		return c.Status(409).JSON(fiber.Map{"error": err.Error()})
	}
	if errors.Is(err, storage.ErrTurmaDuplicada) {
		return c.Status(409).JSON(fiber.Map{"error": err.Error()})
	}
	if errors.Is(err, storage.ErrTurmaAusente) {
		return c.Status(422).JSON(fiber.Map{"error": err.Error(), "field": "turma_id"})
	}
	var validacao *aula.ErroValidacao
	if errors.As(err, &validacao) {
		campos := map[string]string{
			"Nome": "nome", "Data": "data", "Inicio": "inicio", "Fim": "fim", "Turma": "turma",
			"Conteudo": "conteudo", "Observacoes": "observacoes", "IntervaloMinutos": "intervalo_minutos",
		}
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error": validacao.Mensagem, "field": campos[validacao.Campo],
		})
	}
	if errors.Is(err, storage.ErrNaoEncontrada) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Aula não encontrada"})
	}

	code := fiber.StatusInternalServerError
	message := "Erro interno do servidor"
	var httpError *fiber.Error
	if errors.As(err, &httpError) {
		code = httpError.Code
		if code < fiber.StatusInternalServerError {
			message = httpError.Message
		}
	}
	if code >= 500 {
		slog.Error("falha_http", "status", code)
	}
	return c.Status(code).JSON(fiber.Map{"error": message})
}
