package httpapi

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

func TestCorrecaoHTTP(t *testing.T) {
	app, _ := novaAPITeste(t)
	body, _ := requisicao(t, app, "POST", "/aulas", corpoValido, "application/json", 201)
	var r aulaResponse
	if err := json.Unmarshal(body, &r); err != nil {
		t.Fatal(err)
	}
	if r.Versao != 1 || r.Cancelada || r.UpdatedAt == "" {
		t.Fatalf("metadados: %s", body)
	}
	editar := strings.TrimSuffix(corpoValido, "}") + `,"versao":1}`
	editar = strings.Replace(editar, `"fim":"10:00"`, `"fim":"11:00"`, 1)
	body, _ = requisicao(t, app, "PUT", "/aulas/1", editar, "application/json", 200)
	if err := json.Unmarshal(body, &r); err != nil || r.DuracaoMinutos != 165 || r.Versao != 2 {
		t.Fatalf("edição: %s %v", body, err)
	}
	requisicao(t, app, "PUT", "/aulas/1", editar, "application/json", 409)
	requisicao(t, app, "PUT", "/aulas/999", editar, "application/json", 404)
	requisicao(t, app, "PUT", "/aulas/abc", editar, "application/json", 400)
	requisicao(t, app, "PUT", "/aulas/1", corpoValido, "application/json", 400)
	requisicao(t, app, "PUT", "/aulas/1", strings.Replace(strings.Replace(editar, `"versao":1`, `"versao":2`, 1), `"fim":"11:00"`, `"fim":"07:00"`, 1), "application/json", 422)
	for _, entrada := range []string{`{}`, `{"versao":2}`, `{"versao":2,"cancelada":null}`, `{"versao":2,"cancelada":"true"}`, `{"versao":2,"cancelada":true,"extra":1}`} {
		requisicao(t, app, "PATCH", "/aulas/1/situacao", entrada, "application/json", 400)
	}
	requisicao(t, app, "PATCH", "/aulas/1/situacao", `{"versao":2,"cancelada":true}`, "application/json", 200)
	requisicao(t, app, "PUT", "/aulas/1", strings.Replace(editar, `"versao":1`, `"versao":3`, 1), "application/json", 409)
	body, _ = requisicao(t, app, "GET", "/aulas?situacao=todas", "", "", 200)
	var h historicoResponse
	if err := json.Unmarshal(body, &h); err != nil || h.TotalRegistros != 1 || h.DuracaoTotalMinutos != 0 || !h.Aulas[0].Cancelada {
		t.Fatalf("cancelamento: %s %v", body, err)
	}
	requisicao(t, app, "GET", "/aulas?situacao=errada", "", "", 400)
	requisicao(t, app, "PATCH", "/aulas/1/situacao", `{"versao":3,"cancelada":false}`, "application/json", 200)
	body, _ = requisicao(t, app, "GET", "/aulas", "", "", 200)
	if err := json.Unmarshal(body, &h); err != nil || h.TotalRegistros != 1 || h.DuracaoTotalMinutos != 165 {
		t.Fatalf("reativação: %s %v", body, err)
	}
}
func TestLogsNaoIncluemDadosDaAula(t *testing.T) {
	var out bytes.Buffer
	antigo := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&out, nil)))
	defer slog.SetDefault(antigo)
	app, banco := novaAPITeste(t)
	corpo := strings.Replace(corpoValido, "Leitura", "CONTEUDO_PRIVADO", 1)
	requisicao(t, app, "POST", "/aulas", corpo, "application/json", 201)
	requisicao(t, app, "PUT", "/aulas/1", strings.TrimSuffix(corpo, "}")+`,"versao":1}`, "application/json", 200)
	banco.Close()
	requisicao(t, app, "GET", "/aulas", "", "", 500)
	logs := out.String()
	if !strings.Contains(logs, "aula_alterada") || !strings.Contains(logs, "falha_http") {
		t.Fatalf("logs ausentes: %s", logs)
	}
	for _, segredo := range []string{"CONTEUDO_PRIVADO", "Turma fictícia", "database is closed", "2026-10-13"} {
		if strings.Contains(logs, segredo) {
			t.Fatalf("log expôs dados: %s", logs)
		}
	}
	for _, linha := range strings.Split(strings.TrimSpace(logs), "\n") {
		if !json.Valid([]byte(linha)) {
			t.Fatalf("log não estruturado: %s", linha)
		}
	}
}
