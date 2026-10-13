package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"mime"
	"strconv"
	"strings"

	"app-sync-backend/internal/aula"
	"app-sync-backend/internal/storage"

	"github.com/gofiber/fiber/v3"
)

// Os DTOs definem o contrato JSON sem acoplar o domínio ou o SQLite ao HTTP.
type criarAulaRequest struct {
	Data             string `json:"data"`
	Inicio           string `json:"inicio"`
	Fim              string `json:"fim"`
	Turma            string `json:"turma"`
	Conteudo         string `json:"conteudo"`
	Observacoes      string `json:"observacoes"`
	IntervaloMinutos int    `json:"intervalo_minutos"`
}

type aulaResponse struct {
	ID               int64  `json:"id"`
	Data             string `json:"data"`
	Inicio           string `json:"inicio"`
	Fim              string `json:"fim"`
	Turma            string `json:"turma"`
	Conteudo         string `json:"conteudo"`
	Observacoes      string `json:"observacoes"`
	IntervaloMinutos int    `json:"intervalo_minutos"`
	DuracaoMinutos   int    `json:"duracao_minutos"`
}

func respostaAula(r storage.Registro) aulaResponse {
	return aulaResponse{
		ID: r.ID, Data: r.Aula.Data, Inicio: r.Aula.Inicio, Fim: r.Aula.Fim,
		Turma: r.Aula.Turma, Conteudo: r.Aula.Conteudo, Observacoes: r.Aula.Observacoes,
		IntervaloMinutos: r.Aula.IntervaloMinutos, DuracaoMinutos: r.DuracaoMinutos,
	}
}

func registrarRotasAulas(app *fiber.App, banco *storage.Banco) {
	app.Post("/aulas", func(c fiber.Ctx) error {
		contentType, _, err := mime.ParseMediaType(c.Get(fiber.HeaderContentType))
		if err != nil || contentType != "application/json" {
			return fiber.NewError(fiber.StatusUnsupportedMediaType, "Use Content-Type: application/json")
		}
		// Decodificar para um ponteiro permite diferenciar um objeto de null.
		var entrada *criarAulaRequest
		decoder := json.NewDecoder(bytes.NewReader(c.Body()))
		decoder.DisallowUnknownFields()
		erroJSON := fiber.NewError(fiber.StatusBadRequest, "Envie um único objeto JSON com os campos e tipos documentados")
		if err := decoder.Decode(&entrada); err != nil || entrada == nil {
			return erroJSON
		}
		// Um segundo valor ou conteúdo extra não deve passar silenciosamente.
		if err := decoder.Decode(new(any)); err != io.EOF {
			return erroJSON
		}

		registro, err := banco.Inserir(c.Context(), aula.Aula{
			Data: entrada.Data, Inicio: entrada.Inicio, Fim: entrada.Fim, Turma: entrada.Turma,
			Conteudo: entrada.Conteudo, Observacoes: entrada.Observacoes, IntervaloMinutos: entrada.IntervaloMinutos,
		})
		if err != nil {
			return err
		}
		c.Set(fiber.HeaderLocation, "/aulas/"+strconv.FormatInt(registro.ID, 10))
		return c.Status(fiber.StatusCreated).JSON(respostaAula(registro))
	})

	app.Get("/aulas", func(c fiber.Ctx) error {
		if len(c.Request().URI().QueryString()) != 0 {
			return fiber.NewError(fiber.StatusBadRequest, "A listagem ainda não aceita parâmetros de consulta")
		}
		registros, err := banco.Listar(c.Context())
		if err != nil {
			return err
		}
		resposta := make([]aulaResponse, 0, len(registros))
		for _, registro := range registros {
			resposta = append(resposta, respostaAula(registro))
		}
		return c.JSON(resposta)
	})

	app.Get("/aulas/:id", func(c fiber.Ctx) error {
		parametro := c.Params("id")
		id, err := strconv.ParseInt(parametro, 10, 64)
		if err != nil || id <= 0 || strings.Trim(parametro, "0123456789") != "" {
			return fiber.NewError(fiber.StatusBadRequest, "O ID deve ser um inteiro positivo")
		}
		registro, err := banco.Buscar(c.Context(), id)
		if err != nil {
			return err
		}
		return c.JSON(respostaAula(registro))
	})
}
