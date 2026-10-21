package httpapi

import (
	"strconv"

	"app-sync-backend/internal/auth"
	g "app-sync-backend/internal/google"
	"app-sync-backend/internal/storage"
	"app-sync-backend/internal/syncgoogle"
	"github.com/gofiber/fiber/v3"
)

func registrarSincronizacaoGoogle(app *fiber.App, banco *storage.Banco, login *auth.Google) {
	service := syncgoogle.New(banco)
	client := func(c fiber.Ctx) g.Client {
		return g.Client{Request: func(method, path string, body, result any) error {
			return login.SheetsRequest(c, method, path, body, result)
		}}
	}
	type input struct {
		ConfigVersao *int64 `json:"config_versao"`
		Periodo      string `json:"periodo"`
		Confirmacao  string `json:"confirmacao"`
	}
	for _, action := range []string{"inspecionar", "preparar"} {
		app.Post("/integracoes/google/"+action, func(c fiber.Ctx) error {
			session, err := login.SheetsSession(c)
			if err != nil {
				return err
			}
			in, err := lerJSON[input](c)
			if err != nil {
				return err
			}
			if in.ConfigVersao == nil || *in.ConfigVersao < 0 {
				return fiber.NewError(400, "Informe a versão da configuração")
			}
			var result syncgoogle.Inspection
			if action == "inspecionar" {
				result, err = service.Inspect(c.Context(), client(c), session, *in.ConfigVersao, in.Periodo)
			} else {
				result, err = service.Prepare(c.Context(), client(c), session, in.Confirmacao, *in.ConfigVersao, in.Periodo)
			}
			if err != nil {
				return err
			}
			return c.JSON(result)
		})
	}
	app.Post("/aulas/:id/sincronizar", func(c fiber.Ctx) error {
		if _, err := login.SheetsSession(c); err != nil {
			return err
		}
		id, err := strconv.ParseInt(c.Params("id"), 10, 64)
		if err != nil || id <= 0 {
			return fiber.NewError(400, "ID inválido")
		}
		in, err := lerJSON[struct {
			ConfigVersao *int64 `json:"config_versao"`
			Versao       *int64 `json:"versao"`
		}](c)
		if err != nil {
			return err
		}
		if in.ConfigVersao == nil || *in.ConfigVersao < 0 || in.Versao == nil || *in.Versao <= 0 {
			return fiber.NewError(400, "Informe as versões da aula e da configuração")
		}
		result, err := service.Sync(c.Context(), client(c), *in.ConfigVersao, id, *in.Versao)
		if err != nil {
			return err
		}
		return c.JSON(result)
	})
}
