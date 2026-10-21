package auth

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
)

func TestSheetsRequestUsaSessaoProtegeEscritaETrataFalhas(t *testing.T) {
	for _, scenario := range []string{"sucesso", "sem sessao", "origem", "sem escopo", "revogada", "somente leitura", "indisponivel", "resposta invalida", "logout durante chamada"} {
		t.Run(scenario, func(t *testing.T) {
			g := New(testConfig())
			g.sessions["session-a"] = session{User: User{Subject: "a"}, Token: "segredo-a", Sheets: true, Expires: time.Now().Add(time.Hour)}
			called := false
			g.client = &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
				called = true
				if r.Method != "POST" || r.URL.Host != "sheets.googleapis.com" || r.URL.Path != "/v4/spreadsheets/test:batchUpdate" || r.Header.Get("Authorization") != "Bearer segredo-a" {
					t.Error("requisição sem vínculo de conta/endpoint")
				}
				status := 200
				body := `{"replies":[]}`
				switch scenario {
				case "revogada":
					status = 401
				case "somente leitura":
					status = 403
				case "indisponivel":
					status = 503
				case "resposta invalida":
					body = `{"incompleta":`
				case "logout durante chamada":
					g.mu.Lock()
					delete(g.sessions, "session-a")
					g.mu.Unlock()
				}
				return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
			})}
			app := fiber.New()
			g.Register(app)
			app.Post("/write", func(c fiber.Ctx) error {
				var result any
				if err := g.SheetsRequest(c, "POST", "test:batchUpdate", map[string]any{"requests": []any{}}, &result); err != nil {
					return err
				}
				return c.JSON(result)
			})
			cookie := &http.Cookie{Name: sessionCookie, Value: "session-a"}
			origin := g.config.AppURL
			want := 200
			switch scenario {
			case "sem sessao":
				cookie = nil
				want = 401
			case "origem":
				origin = "https://other.test"
				want = 403
			case "sem escopo":
				s := g.sessions["session-a"]
				s.Sheets = false
				g.sessions["session-a"] = s
				want = 403
			case "revogada", "logout durante chamada":
				want = 401
			case "somente leitura":
				want = 403
			case "indisponivel", "resposta invalida":
				want = 409
			}
			res := request(t, app, "POST", "/write", cookie, origin)
			if res.StatusCode != want {
				t.Fatalf("status %d esperado %d", res.StatusCode, want)
			}
			if (scenario == "sem sessao" || scenario == "origem" || scenario == "sem escopo") && called {
				t.Fatal("chamou Google sem autorização")
			}
			b, _ := io.ReadAll(res.Body)
			if strings.Contains(string(b), "segredo-a") {
				t.Fatal("token exposto")
			}
			if scenario == "revogada" {
				if _, ok := g.sessions["session-a"]; ok {
					t.Fatal("sessão revogada preservada")
				}
			}
		})
	}
}
