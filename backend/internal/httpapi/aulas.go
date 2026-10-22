package httpapi

import (
	"net/url"
	"strconv"
	"strings"
	"time"

	"app-sync-backend/internal/aula"
	"app-sync-backend/internal/storage"

	"github.com/gofiber/fiber/v3"
)

// Os DTOs definem o contrato JSON sem acoplar o domínio ou o SQLite ao HTTP.
type criarAulaRequest struct {
	Data             string `json:"data"`
	Inicio           string `json:"inicio"`
	Fim              string `json:"fim"`
	TurmaID          int64  `json:"turma_id"`
	Conteudo         string `json:"conteudo"`
	Observacoes      string `json:"observacoes"`
	IntervaloMinutos int    `json:"intervalo_minutos"`
}

type aulaResponse struct {
	Google           *storage.EstadoGoogle `json:"google,omitempty"`
	Versao           int64                 `json:"versao"`
	Cancelada        bool                  `json:"cancelada"`
	UpdatedAt        string                `json:"updated_at"`
	TurmaID          int64                 `json:"turma_id"`
	ID               int64                 `json:"id"`
	Data             string                `json:"data"`
	Inicio           string                `json:"inicio"`
	Fim              string                `json:"fim"`
	Turma            string                `json:"turma"`
	Conteudo         string                `json:"conteudo"`
	Observacoes      string                `json:"observacoes"`
	IntervaloMinutos int                   `json:"intervalo_minutos"`
	DuracaoMinutos   int                   `json:"duracao_minutos"`
}

func respostaAula(r storage.Registro) aulaResponse {
	return aulaResponse{
		Versao: r.Versao, Cancelada: r.Cancelada, UpdatedAt: r.UpdatedAt,
		TurmaID: r.TurmaID, ID: r.ID, Data: r.Aula.Data, Inicio: r.Aula.Inicio, Fim: r.Aula.Fim,
		Turma: r.Aula.Turma, Conteudo: r.Aula.Conteudo, Observacoes: r.Aula.Observacoes,
		IntervaloMinutos: r.Aula.IntervaloMinutos, DuracaoMinutos: r.DuracaoMinutos,
	}
}

func registrarRotasAulas(app *fiber.App, banco *storage.Banco) {
	registrarRotaRelatorio(app, banco)
	registrarRotasCorrecao(app, banco)
	app.Post("/aulas", func(c fiber.Ctx) error {
		entrada, err := lerJSON[criarAulaRequest](c)
		if err != nil {
			return err
		}

		if entrada.TurmaID <= 0 {
			return c.Status(422).JSON(fiber.Map{"error": "Selecione uma turma ou estudante", "field": "turma_id"})
		}
		turma, err := banco.BuscarTurma(c.Context(), entrada.TurmaID)
		if err != nil {
			return err
		}
		registro, err := banco.Inserir(c.Context(), aula.Aula{
			Data: entrada.Data, Inicio: entrada.Inicio, Fim: entrada.Fim, Turma: turma.Nome,
			Conteudo: entrada.Conteudo, Observacoes: entrada.Observacoes, IntervaloMinutos: entrada.IntervaloMinutos,
		})
		if err != nil {
			return err
		}
		c.Set(fiber.HeaderLocation, "/aulas/"+strconv.FormatInt(registro.ID, 10))
		return c.Status(fiber.StatusCreated).JSON(respostaAula(registro))
	})

	app.Get("/aulas", func(c fiber.Ctx) error {
		filtro, err := lerFiltro(string(c.Request().URI().QueryString()))
		if err != nil {
			return err
		}
		historico, err := banco.Historico(c.Context(), filtro)
		if err != nil {
			return err
		}
		resposta := historicoResponse{Aulas: make([]aulaResponse, 0, len(historico.Aulas)), Pagina: filtro.Pagina, PorPagina: 20, TotalRegistros: historico.TotalRegistros, DuracaoTotalMinutos: historico.DuracaoTotalMinutos}
		for _, r := range historico.Aulas {
			item := respostaAula(r)
			estado, err := banco.EstadoGoogle(c.Context(), r)
			if err != nil {
				return err
			}
			item.Google = &estado
			resposta.Aulas = append(resposta.Aulas, item)
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

type historicoResponse struct {
	Aulas               []aulaResponse `json:"aulas"`
	Pagina              int            `json:"pagina"`
	PorPagina           int            `json:"por_pagina"`
	TotalRegistros      int            `json:"total_registros"`
	DuracaoTotalMinutos int64          `json:"duracao_total_minutos"`
}

func lerFiltro(query string) (storage.Filtro, error) {
	f := storage.Filtro{Pagina: 1}
	params, err := url.ParseQuery(query)
	if err != nil {
		return f, fiber.NewError(400, "Parâmetros inválidos")
	}
	for k, values := range params {
		if len(values) != 1 || values[0] == "" {
			return f, fiber.NewError(400, "Parâmetro vazio ou repetido: "+k)
		}
		v := values[0]
		switch k {
		case "situacao":
			if v != "ativas" && v != "canceladas" && v != "todas" {
				return f, fiber.NewError(400, "Situação inválida")
			}
			f.Situacao = v
		case "data_inicial", "data_final":
			d, err := time.Parse("2006-01-02", v)
			if err != nil || d.Year() < 1 || d.Format("2006-01-02") != v {
				return f, fiber.NewError(400, "Data inválida: "+k)
			}
			if k == "data_inicial" {
				f.DataInicial = v
			} else {
				f.DataFinal = v
			}
		case "turma_id", "pagina":
			n, err := strconv.ParseInt(v, 10, 64)
			if err != nil || n <= 0 || strings.Trim(v, "0123456789") != "" {
				return f, fiber.NewError(400, "Inteiro positivo inválido: "+k)
			}
			if k == "turma_id" {
				f.TurmaID = n
			} else {
				if n > int64(int(^uint(0)>>1)/20) {
					return f, fiber.NewError(400, "Página muito grande")
				}
				f.Pagina = int(n)
			}
		default:
			return f, fiber.NewError(400, "Parâmetro desconhecido: "+k)
		}
	}
	if f.DataInicial != "" && f.DataFinal != "" && f.DataInicial > f.DataFinal {
		return f, fiber.NewError(400, "A data inicial deve ser anterior ou igual à final")
	}
	return f, nil
}
