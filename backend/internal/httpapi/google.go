package httpapi

import (
	destino "app-sync-backend/internal/google"
	"app-sync-backend/internal/storage"
	"errors"
	"github.com/gofiber/fiber/v3"
)

type destinoGoogleResponse struct {
	ID         int64  `json:"id"`
	PlanilhaID string `json:"planilha_id"`
	URL        string `json:"url"`
}
type configGoogleResponse struct {
	Versao  int64                  `json:"versao"`
	Destino *destinoGoogleResponse `json:"destino"`
}

func respostaConfigGoogle(c storage.ConfigGoogle) configGoogleResponse {
	r := configGoogleResponse{Versao: c.Versao}
	if c.Destino != nil {
		r.Destino = &destinoGoogleResponse{ID: c.Destino.ID, PlanilhaID: c.Destino.PlanilhaID, URL: destino.URLPlanilha(c.Destino.PlanilhaID)}
	}
	return r
}
func registrarRotasGoogle(app *fiber.App, banco *storage.Banco) {
	app.Get("/integracoes/google/configuracao", func(c fiber.Ctx) error {
		config, err := banco.ConfiguracaoGoogle(c.Context())
		if err != nil {
			return err
		}
		return c.JSON(respostaConfigGoogle(config))
	})
	app.Put("/integracoes/google/configuracao", func(c fiber.Ctx) error {
		entrada, err := lerJSON[struct {
			URL    string `json:"url"`
			Versao *int64 `json:"versao"`
		}](c)
		if err != nil {
			return err
		}
		if entrada.Versao == nil || *entrada.Versao < 0 {
			return fiber.NewError(400, "Informe a versão atual da configuração")
		}
		if _, err := destino.PlanilhaID(entrada.URL); err != nil {
			return c.Status(422).JSON(fiber.Map{"error": err.Error(), "field": "url"})
		}
		config, err := banco.ConfigurarGoogle(c.Context(), entrada.URL, *entrada.Versao)
		if errors.Is(err, storage.ErrConfigConflito) {
			return fiber.NewError(409, err.Error())
		}
		if err != nil {
			return err
		}
		return c.JSON(respostaConfigGoogle(config))
	})
	app.Get("/integracoes/google/planejamento", func(c fiber.Ctx) error {
		plano, err := banco.PlanejamentoGoogle(c.Context())
		if err != nil {
			return err
		}
		type mes struct {
			Periodo    string `json:"periodo"`
			Aba        string `json:"aba_esperada"`
			Aulas      int    `json:"aulas"`
			Canceladas int    `json:"canceladas"`
			Minutos    int64  `json:"duracao_ativa_minutos"`
		}
		meses := make([]mes, 0, len(plano.Meses))
		for _, m := range plano.Meses {
			meses = append(meses, mes{m.Periodo, m.Aba, m.Aulas, m.Canceladas, m.Minutos})
		}
		return c.JSON(struct {
			Configuracao configGoogleResponse `json:"configuracao"`
			Meses        []mes                `json:"meses"`
			Verificacao  string               `json:"verificacao_remota"`
		}{respostaConfigGoogle(plano.Configuracao), meses, "nao_realizada"})
	})
}
