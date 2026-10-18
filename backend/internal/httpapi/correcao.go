package httpapi

import (
	"app-sync-backend/internal/aula"
	"app-sync-backend/internal/storage"
	"bytes"
	"encoding/json"
	"github.com/gofiber/fiber/v3"
	"io"
	"log/slog"
	"mime"
	"strconv"
	"strings"
)

func lerJSON[T any](c fiber.Ctx) (*T, error) {
	tipo, _, err := mime.ParseMediaType(c.Get(fiber.HeaderContentType))
	if err != nil || tipo != "application/json" {
		return nil, fiber.NewError(415, "Use Content-Type: application/json")
	}
	var entrada *T
	d := json.NewDecoder(bytes.NewReader(c.Body()))
	d.DisallowUnknownFields()
	if err := d.Decode(&entrada); err != nil || entrada == nil {
		return nil, fiber.NewError(400, "Envie um objeto JSON com os campos e tipos documentados")
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return nil, fiber.NewError(400, "Envie um único objeto JSON")
	}
	return entrada, nil
}
func idDaAula(c fiber.Ctx) (int64, error) {
	v := c.Params("id")
	id, err := strconv.ParseInt(v, 10, 64)
	if err != nil || id <= 0 || strings.Trim(v, "0123456789") != "" {
		return 0, fiber.NewError(400, "O ID deve ser um inteiro positivo")
	}
	return id, nil
}
func registrarRotasCorrecao(app *fiber.App, banco *storage.Banco) {
	app.Put("/aulas/:id", func(c fiber.Ctx) error {
		id, err := idDaAula(c)
		if err != nil {
			return err
		}
		entrada, err := lerJSON[struct {
			criarAulaRequest
			Versao int64 `json:"versao"`
		}](c)
		if err != nil {
			return err
		}
		if entrada.Versao < 1 {
			return fiber.NewError(400, "Informe a versão atual da aula")
		}
		r, err := banco.Alterar(c.Context(), id, storage.Alteracao{Versao: entrada.Versao, TurmaID: entrada.TurmaID, Aula: aula.Aula{Data: entrada.Data, Inicio: entrada.Inicio, Fim: entrada.Fim, Conteudo: entrada.Conteudo, Observacoes: entrada.Observacoes, IntervaloMinutos: entrada.IntervaloMinutos}})
		if err != nil {
			return err
		}
		slog.Info("aula_alterada", "aula_id", r.ID, "versao", r.Versao, "operacao", "editar")
		return c.JSON(respostaAula(r))
	})
	app.Patch("/aulas/:id/situacao", func(c fiber.Ctx) error {
		id, err := idDaAula(c)
		if err != nil {
			return err
		}
		entrada, err := lerJSON[struct {
			Versao    int64 `json:"versao"`
			Cancelada *bool `json:"cancelada"`
		}](c)
		if err != nil {
			return err
		}
		if entrada.Versao < 1 || entrada.Cancelada == nil {
			return fiber.NewError(400, "Informe versão e cancelada (true ou false)")
		}
		r, err := banco.Alterar(c.Context(), id, storage.Alteracao{Versao: entrada.Versao, Cancelada: entrada.Cancelada})
		if err != nil {
			return err
		}
		slog.Info("aula_alterada", "aula_id", r.ID, "versao", r.Versao, "cancelada", r.Cancelada)
		return c.JSON(respostaAula(r))
	})
}
