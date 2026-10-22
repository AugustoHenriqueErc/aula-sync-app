package httpapi

import (
	"net/http/httptest"
	"testing"

	"app-sync-backend/internal/auth"
)

func TestOAuthProtegeRotasDaAplicacao(t *testing.T) {
	_, banco := novaAPITeste(t)
	login := auth.New(auth.Config{ClientID: "test-client", ClientSecret: "test-secret", RedirectURI: "http://localhost:5173/api/auth/google/callback", AppURL: "http://localhost:5173"})
	app := NewWithAuth(banco, login)
	for _, path := range []string{"/aulas", "/aulas/relatorio.pdf", "/turmas", "/integracoes/google/configuracao", "/integracoes/google/planejamento", "/integracoes/google/acesso"} {
		res, err := app.Test(httptest.NewRequest("GET", path, nil))
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != 401 {
			t.Fatalf("%s: status %d; esperado 401", path, res.StatusCode)
		}
	}
	for _, path := range []string{"/integracoes/google/inspecionar", "/integracoes/google/preparar", "/aulas/1/sincronizar"} {
		res, err := app.Test(httptest.NewRequest("POST", path, nil))
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != 401 {
			t.Fatalf("%s: status %d", path, res.StatusCode)
		}
	}
	for _, path := range []string{"/health", "/auth/session"} {
		res, err := app.Test(httptest.NewRequest("GET", path, nil))
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != 200 {
			t.Fatalf("%s: status %d; esperado 200", path, res.StatusCode)
		}
	}
}
