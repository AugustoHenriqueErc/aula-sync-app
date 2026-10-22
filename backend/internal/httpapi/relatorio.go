package httpapi

import (
	"app-sync-backend/internal/relatorio"
	"app-sync-backend/internal/storage"

	"github.com/gofiber/fiber/v3"
)

func registrarRotaRelatorio(app *fiber.App, banco *storage.Banco) {
	app.Get("/aulas/relatorio.pdf", func(c fiber.Ctx) error {
		filtro, err := lerFiltro(string(c.Request().URI().QueryString()))
		if err != nil {
			return err
		}
		historico, err := banco.Relatorio(c.Context(), filtro)
		if err != nil {
			return err
		}
		pdf, err := relatorio.PDF(historico, filtro)
		if err != nil {
			return err
		}
		c.Set(fiber.HeaderContentType, "application/pdf")
		c.Set(fiber.HeaderContentDisposition, `attachment; filename="relatorio-aulas.pdf"`)
		c.Set(fiber.HeaderCacheControl, "no-store")
		return c.Send(pdf)
	})
}
