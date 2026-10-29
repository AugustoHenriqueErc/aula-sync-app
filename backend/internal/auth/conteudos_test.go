package auth

import (
	"net/http"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
)

// Exercita o middleware comum com sessão válida; os handlers reais são testados
// em httpapi, incluindo o bloqueio de GET e POST sem sessão.
func TestMiddlewareProtegeOrigemDoCadastroDeConteudos(t *testing.T) {
	g := New(testConfig())
	g.sessions["conteudos-session"] = session{Expires: time.Now().Add(time.Hour)}
	app := fiber.New()
	g.Register(app)
	chamadas := 0
	app.Post("/conteudos", func(c fiber.Ctx) error {
		chamadas++
		return c.SendStatus(201)
	})
	cookie := &http.Cookie{Name: sessionCookie, Value: "conteudos-session"}
	for _, origem := range []string{"", "https://evil.test", g.config.AppURL + ".evil.test"} {
		res := request(t, app, "POST", "/conteudos", cookie, origem)
		res.Body.Close()
		if res.StatusCode != 403 || chamadas != 0 {
			t.Fatalf("origem %q: status=%d chamadas=%d", origem, res.StatusCode, chamadas)
		}
	}
	res := request(t, app, "POST", "/conteudos", cookie, g.config.AppURL)
	res.Body.Close()
	if res.StatusCode != 201 || chamadas != 1 {
		t.Fatalf("origem válida: status=%d chamadas=%d", res.StatusCode, chamadas)
	}
}
