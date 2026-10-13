package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestHTTPResponses(t *testing.T) {
	app, _ := novaAPITeste(t)
	// Estas rotas existem apenas no teste para exercitar o tratamento de erros.
	app.Get("/bad-request", func(c fiber.Ctx) error {
		return fmt.Errorf("validação: %w", fiber.NewError(400, "Entrada inválida"))
	})
	app.Get("/internal-error", func(c fiber.Ctx) error {
		return errors.New("detalhe interno que não deve ser exposto")
	})
	app.Get("/fiber-error", func(c fiber.Ctx) error {
		return fiber.NewError(503, "detalhe interno de infraestrutura")
	})
	tests := []struct {
		path   string
		status int
		body   map[string]string
	}{
		{"/health", 200, map[string]string{"status": "ok"}},
		{"/missing", 404, map[string]string{"error": "Not Found"}},
		{"/bad-request", 400, map[string]string{"error": "Entrada inválida"}},
		{"/internal-error", 500, map[string]string{"error": "Erro interno do servidor"}},
		{"/fiber-error", 503, map[string]string{"error": "Erro interno do servidor"}},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			response, err := app.Test(httptest.NewRequest("GET", tt.path, nil))
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			if response.StatusCode != tt.status {
				t.Errorf("status = %d; esperado %d", response.StatusCode, tt.status)
			}
			mediaType, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
			if err != nil || mediaType != "application/json" {
				t.Errorf("Content-Type inesperado: %q", response.Header.Get("Content-Type"))
			}
			var body map[string]string
			if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(body, tt.body) {
				t.Errorf("JSON = %v; esperado %v", body, tt.body)
			}
		})
	}
}
