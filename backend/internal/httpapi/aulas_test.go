package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"mime"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"app-sync-backend/internal/aula"
	"app-sync-backend/internal/storage"

	"github.com/gofiber/fiber/v3"
)

const corpoValido = `{"data":"2026-10-13","inicio":"08:00","fim":"10:00","turma_id":1,"conteudo":"Leitura","observacoes":"Exemplo","intervalo_minutos":15}`

func novaAPITeste(t *testing.T) (*fiber.App, *storage.Banco) {
	t.Helper()
	banco, err := storage.Abrir(context.Background(), filepath.Join(t.TempDir(), "aulas.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { banco.Close() })
	if _, err := banco.CriarTurma(context.Background(), "Turma fictícia"); err != nil {
		t.Fatal(err)
	}
	return New(banco), banco
}

func requisicao(t *testing.T, app *fiber.App, metodo, caminho, corpo, contentType string, status int) ([]byte, http.Header) {
	t.Helper()
	req := httptest.NewRequest(metodo, caminho, strings.NewReader(corpo))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != status {
		t.Fatalf("%s %s: status %d, esperado %d; corpo %s", metodo, caminho, resp.StatusCode, status, body)
	}
	tipo, _, err := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if err != nil || tipo != "application/json" {
		t.Fatalf("resposta não é JSON: %s", resp.Header.Get("Content-Type"))
	}
	if !json.Valid(body) {
		t.Fatalf("JSON inválido: %s", body)
	}
	return body, resp.Header
}

func TestCadastrarConsultarEConferirSQLite(t *testing.T) {
	app, banco := novaAPITeste(t)
	vazio, _ := requisicao(t, app, "GET", "/aulas", "", "", 200)
	var empty historicoResponse
	if err := json.Unmarshal(vazio, &empty); err != nil {
		t.Fatal(err)
	}
	if empty.Aulas == nil || len(empty.Aulas) != 0 || empty.TotalRegistros != 0 || empty.DuracaoTotalMinutos != 0 {
		t.Fatalf("lista vazia = %s", vazio)
	}
	body, headers := requisicao(t, app, "POST", "/aulas", corpoValido, "application/json; charset=utf-8", 201)
	var recebido map[string]any
	if err := json.Unmarshal(body, &recebido); err != nil {
		t.Fatal(err)
	}
	id, ok := recebido["id"].(float64)
	if !ok || id <= 0 {
		t.Fatalf("ID inválido: %v", recebido)
	}
	esperado := map[string]any{
		"turma_id": float64(1), "id": id, "data": "2026-10-13", "inicio": "08:00", "fim": "10:00",
		"turma": "Turma fictícia", "conteudo": "Leitura", "observacoes": "Exemplo",
		"intervalo_minutos": float64(15), "duracao_minutos": float64(105),
	}
	if !reflect.DeepEqual(recebido, esperado) {
		t.Fatalf("contrato inesperado: %v", recebido)
	}
	location := "/aulas/" + strconv.FormatInt(int64(id), 10)
	if headers.Get("Location") != location {
		t.Fatalf("Location = %s", headers.Get("Location"))
	}
	consulta, _ := requisicao(t, app, "GET", location, "", "", 200)
	if string(consulta) != string(body) {
		t.Fatalf("consulta difere do cadastro: %s", consulta)
	}
	lista, _ := requisicao(t, app, "GET", "/aulas", "", "", 200)
	var page historicoResponse
	if err := json.Unmarshal(lista, &page); err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(page.Aulas)
	if string(encoded) != "["+string(body)+"]" || page.TotalRegistros != 1 || page.DuracaoTotalMinutos != 105 {
		t.Fatalf("lista difere do cadastro: %s", lista)
	}
	registro, err := banco.Buscar(context.Background(), int64(id))
	aulaEsperada := aula.Aula{Data: "2026-10-13", Inicio: "08:00", Fim: "10:00", Turma: "Turma fictícia", Conteudo: "Leitura", Observacoes: "Exemplo", IntervaloMinutos: 15}
	if err != nil || registro.Aula != aulaEsperada || registro.DuracaoMinutos != 105 {
		t.Fatalf("SQLite não corresponde ao JSON: %+v, %v", registro, err)
	}
}

func TestListagemOrdenada(t *testing.T) {
	app, _ := novaAPITeste(t)
	var ids []int64
	for _, horario := range []struct{ data, inicio, fim string }{
		{"2026-10-12", "08:00", "10:00"}, {"2026-10-13", "08:00", "10:00"},
		{"2026-10-13", "10:00", "12:00"}, {"2026-10-13", "10:00", "12:00"},
	} {
		var input map[string]any
		if err := json.Unmarshal([]byte(corpoValido), &input); err != nil {
			t.Fatal(err)
		}
		input["data"], input["inicio"], input["fim"] = horario.data, horario.inicio, horario.fim
		encoded, err := json.Marshal(input)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := requisicao(t, app, "POST", "/aulas", string(encoded), "application/json", 201)
		var r aulaResponse
		if err := json.Unmarshal(body, &r); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, r.ID)
	}
	body, _ := requisicao(t, app, "GET", "/aulas", "", "", 200)
	var page historicoResponse
	if err := json.Unmarshal(body, &page); err != nil {
		t.Fatal(err)
	}
	lista := page.Aulas
	if len(lista) != 4 {
		t.Fatalf("total = %d", len(lista))
	}
	for i, r := range lista {
		if r.ID != ids[3-i] {
			t.Fatalf("ordem incorreta: %+v", lista)
		}
	}
}

func TestRejeitarJSONInvalidoSemGravar(t *testing.T) {
	app, banco := novaAPITeste(t)
	for _, tt := range []struct {
		nome, corpo, tipo string
		status            int
	}{
		{"sem content-type", corpoValido, "", 415},
		{"texto", corpoValido, "text/plain", 415},
		{"formulário", "data=2026-10-13", "application/x-www-form-urlencoded", 415},
		{"vazio", "", "application/json", 400},
		{"malformado", "{", "application/json", 400},
		{"null", "null", "application/json", 400},
		{"array", "[]", "application/json", 400},
		{"escalar", "1", "application/json", 400},
		{"objetos concatenados", corpoValido + "{}", "application/json", 400},
		{"lixo após JSON", corpoValido + "x", "application/json", 400},
		{"campo desconhecido", `{"extra":1}`, "application/json", 400},
		{"ID fornecido", `{"id":1}`, "application/json", 400},
		{"duração fornecida", `{"duracao_minutos":900}`, "application/json", 400},
		{"tipo de data", `{"data":123}`, "application/json", 400},
		{"intervalo string", `{"intervalo_minutos":"15"}`, "application/json", 400},
		{"intervalo fracionário", `{"intervalo_minutos":1.5}`, "application/json", 400},
		{"intervalo excede inteiro", `{"intervalo_minutos":999999999999999999999999}`, "application/json", 400},
	} {
		t.Run(tt.nome, func(t *testing.T) {
			body, _ := requisicao(t, app, "POST", "/aulas", tt.corpo, tt.tipo, tt.status)
			var erro map[string]string
			if err := json.Unmarshal(body, &erro); err != nil {
				t.Fatal(err)
			}
			if len(erro) != 1 || erro["error"] == "" {
				t.Fatalf("formato de erro: %s", body)
			}
		})
	}
	registros, err := banco.Listar(context.Background())
	if err != nil || len(registros) != 0 {
		t.Fatalf("JSON inválido gravou dados: %v, %v", registros, err)
	}
}

func TestErrosPorCampo(t *testing.T) {
	app, banco := novaAPITeste(t)
	for _, tt := range []struct {
		campo string
		valor any
	}{
		{"data", "2026-02-29"}, {"data", nil}, {"inicio", "8:00"},
		{"fim", "07:00"}, {"turma_id", 0}, {"conteudo", ""},
		{"intervalo_minutos", -1}, {"intervalo_minutos", 120},
	} {
		t.Run(tt.campo, func(t *testing.T) {
			var input map[string]any
			if err := json.Unmarshal([]byte(corpoValido), &input); err != nil {
				t.Fatal(err)
			}
			input[tt.campo] = tt.valor
			encoded, err := json.Marshal(input)
			if err != nil {
				t.Fatal(err)
			}
			body, _ := requisicao(t, app, "POST", "/aulas", string(encoded), "application/json", 422)
			var erro map[string]string
			if err := json.Unmarshal(body, &erro); err != nil {
				t.Fatal(err)
			}
			if len(erro) != 2 || erro["field"] != tt.campo || erro["error"] == "" {
				t.Fatalf("erro de campo: %s", body)
			}
		})
	}
	body, _ := requisicao(t, app, "POST", "/aulas", "{}", "application/json", 422)
	if !strings.Contains(string(body), `"field":"turma_id"`) {
		t.Fatalf("campo ausente: %s", body)
	}
	registros, err := banco.Listar(context.Background())
	if err != nil || len(registros) != 0 {
		t.Fatalf("aula inválida gravou dados: %v, %v", registros, err)
	}
}

func TestCamposOpcionais(t *testing.T) {
	app, _ := novaAPITeste(t)
	for _, body := range []string{
		`{"data":"2026-10-13","inicio":"08:00","fim":"09:00","turma_id":1,"conteudo":"Leitura"}`,
		`{"data":"2026-10-13","inicio":"08:00","fim":"09:00","turma_id":1,"conteudo":"Leitura","observacoes":null,"intervalo_minutos":null}`,
	} {
		resposta, _ := requisicao(t, app, "POST", "/aulas", body, "application/json", 201)
		var r aulaResponse
		if err := json.Unmarshal(resposta, &r); err != nil {
			t.Fatal(err)
		}
		if r.DuracaoMinutos != 60 || r.IntervaloMinutos != 0 || r.Observacoes != "" {
			t.Fatalf("valores padrão: %+v", r)
		}
	}
}

func TestIDInvalidoInexistenteEParametros(t *testing.T) {
	app, _ := novaAPITeste(t)
	for _, id := range []string{"abc", "0", "-1", "1.5", "+1", "9223372036854775808"} {
		requisicao(t, app, "GET", "/aulas/"+id, "", "", 400)
	}
	body, _ := requisicao(t, app, "GET", "/aulas/12345", "", "", 404)
	if string(body) != `{"error":"Aula não encontrada"}` {
		t.Fatalf("404 inesperado: %s", body)
	}
	requisicao(t, app, "GET", "/aulas?limit=1", "", "", 400)
}

func TestFalhaSQLiteNaoExpoeDetalhes(t *testing.T) {
	app, banco := novaAPITeste(t)
	if err := banco.Close(); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct{ metodo, caminho, corpo string }{
		{"POST", "/aulas", corpoValido}, {"GET", "/aulas", ""}, {"GET", "/aulas/1", ""},
	} {
		body, _ := requisicao(t, app, tt.metodo, tt.caminho, tt.corpo, "application/json", 500)
		if string(body) != `{"error":"Erro interno do servidor"}` {
			t.Fatalf("erro expôs detalhes: %s", body)
		}
	}
	// Saúde HTTP não afirma disponibilidade do banco.
	requisicao(t, app, "GET", "/health", "", "", 200)
}

// app.Test retorna erro de transporte para corpos grandes; usamos HTTP real
// para conferir o status e o JSON que chegam ao cliente nesse caso.
func TestLimiteDoCorpoHTTP(t *testing.T) {
	app, banco := novaAPITeste(t)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	encerrado := make(chan error, 1)
	go func() { encerrado <- app.Listener(listener, fiber.ListenConfig{DisableStartupMessage: true}) }()
	t.Cleanup(func() {
		if err := app.ShutdownWithTimeout(5 * time.Second); err != nil {
			t.Error(err)
		}
		listener.Close()
		select {
		case <-encerrado:
		case <-time.After(5 * time.Second):
			t.Error("servidor de teste não encerrou")
		}
	})
	req, err := http.NewRequest("POST", "http://"+listener.Addr().String()+"/aulas", strings.NewReader(strings.Repeat(" ", 64*1024+1)))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 5 * time.Second}
	defer client.CloseIdleConnections()
	response, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != 413 {
		t.Fatalf("status = %d; esperado 413", response.StatusCode)
	}
	var body map[string]string
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body) != 1 || body["error"] == "" {
		t.Fatalf("erro inesperado: %v", body)
	}
	registros, err := banco.Listar(context.Background())
	if err != nil || len(registros) != 0 {
		t.Fatalf("corpo grande gravou dados: %v, %v", registros, err)
	}
}
