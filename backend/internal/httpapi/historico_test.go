package httpapi

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestFiltrosInvalidos(t *testing.T) {
	app, _ := novaAPITeste(t)
	for _, q := range []string{"data_inicial=2026-02-29", "data_final=0000-01-01", "data_inicial=2026-11-01&data_final=2026-10-31", "pagina=0", "pagina=-1", "pagina=1.5", "pagina=9223372036854775807", "pagina=1&pagina=2", "turma_id=abc", "turma_id=0", "data_final=", "outro=1"} {
		requisicao(t, app, "GET", "/aulas?"+q, "", "", 400)
	}
}
func TestTurmasHTTP(t *testing.T) {
	app, _ := novaAPITeste(t)
	requisicao(t, app, "POST", "/turmas", `{"nome":"  Augusto  Silva "}`, "application/json", 201)
	requisicao(t, app, "POST", "/turmas", `{"nome":"augusto silva"}`, "application/json", 409)
	requisicao(t, app, "POST", "/turmas", `{"nome":" "}`, "application/json", 422)
	for _, body := range []string{`null`, `[]`, `{"nome":1}`, `{"nome":"A","extra":1}`, `{"nome":"A"}{}`} {
		requisicao(t, app, "POST", "/turmas", body, "application/json", 400)
	}
	requisicao(t, app, "POST", "/turmas", `{"nome":"A"}`, "text/plain", 415)
	body, _ := requisicao(t, app, "GET", "/turmas", "", "", 200)
	if !strings.Contains(string(body), `"nome":"Augusto Silva"`) {
		t.Fatalf("nome: %s", body)
	}
	requisicao(t, app, "POST", "/aulas", strings.Replace(corpoValido, `"turma_id":1`, `"turma_id":999`, 1), "application/json", 422)
	requisicao(t, app, "POST", "/aulas", strings.Replace(corpoValido, `"turma_id":1`, `"turma":"Augusto"`, 1), "application/json", 400)
}
func TestHistoricoHTTP(t *testing.T) {
	app, _ := novaAPITeste(t)
	for i := 0; i < 21; i++ {
		requisicao(t, app, "POST", "/aulas", corpoValido, "application/json", 201)
	}
	for _, q := range []string{"pagina=1", "pagina=2", "pagina=3", "pagina=2&turma_id=1&data_inicial=2026-10-13&data_final=2026-10-13"} {
		body, _ := requisicao(t, app, "GET", "/aulas?"+q, "", "", 200)
		var h historicoResponse
		if err := json.Unmarshal(body, &h); err != nil {
			t.Fatal(err)
		}
		want := 20
		if h.Pagina == 2 {
			want = 1
		}
		if h.Pagina == 3 {
			want = 0
		}
		if len(h.Aulas) != want || h.TotalRegistros != 21 || h.DuracaoTotalMinutos != 2205 || h.PorPagina != 20 {
			t.Fatalf("resposta: %s", body)
		}
	}
	body, _ := requisicao(t, app, "GET", "/aulas?turma_id=999", "", "", 200)
	var h historicoResponse
	if err := json.Unmarshal(body, &h); err != nil {
		t.Fatal(err)
	}
	if h.Aulas == nil || len(h.Aulas) != 0 || h.TotalRegistros != 0 || h.DuracaoTotalMinutos != 0 {
		t.Fatalf("vazio: %s", body)
	}
}
