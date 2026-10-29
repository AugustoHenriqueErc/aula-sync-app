package httpapi

import (
	"encoding/json"
	"reflect"
	"testing"

	"app-sync-backend/internal/storage"
)

func TestConteudosHTTP(t *testing.T) {
	app, _ := novaAPITeste(t)
	body, _ := requisicao(t, app, "GET", "/conteudos", "", "", 200)
	if string(body) != "[]" {
		t.Fatalf("lista vazia: %s", body)
	}
	body, _ = requisicao(t, app, "POST", "/conteudos", `{"nome":"  Leitura\n  e revisão  "}`, "application/json; charset=utf-8", 201)
	var criado storage.Conteudo
	if err := json.Unmarshal(body, &criado); err != nil || criado.ID <= 0 || criado.Nome != "Leitura\n  e revisão" {
		t.Fatalf("criação: %s %v", body, err)
	}
	requisicao(t, app, "POST", "/conteudos", `{"nome":"LEITURA E REVISÃO"}`, "application/json", 409)
	for _, entrada := range []string{`{}`, `{"nome":""}`, `{"nome":" \n\t\u00a0 "}`} {
		body, _ := requisicao(t, app, "POST", "/conteudos", entrada, "application/json", 422)
		var erro map[string]string
		if err := json.Unmarshal(body, &erro); err != nil || erro["field"] != "nome" {
			t.Fatalf("validação: %s %v", body, err)
		}
	}
	for _, entrada := range []string{"", `null`, `[]`, `{"nome":1}`, `{"nome":"A","extra":1}`, `{"nome":"A"}{}`, `{"nome":`} {
		requisicao(t, app, "POST", "/conteudos", entrada, "application/json", 400)
	}
	for _, tipo := range []string{"", "text/plain", "application/json; charset"} {
		requisicao(t, app, "POST", "/conteudos", `{"nome":"A"}`, tipo, 415)
	}
	requisicao(t, app, "GET", "/conteudos?pagina=1", "", "", 400)
	body, _ = requisicao(t, app, "GET", "/conteudos", "", "", 200)
	var lista []storage.Conteudo
	if err := json.Unmarshal(body, &lista); err != nil || !reflect.DeepEqual(lista, []storage.Conteudo{criado}) {
		t.Fatalf("lista: %s %v", body, err)
	}
}

func TestConteudosErroInternoNaoExpoeBanco(t *testing.T) {
	app, banco := novaAPITeste(t)
	banco.Close()
	for _, metodo := range []string{"GET", "POST"} {
		body, _ := requisicao(t, app, metodo, "/conteudos", `{"nome":"Conteúdo privado"}`, "application/json", 500)
		if string(body) != `{"error":"Erro interno do servidor"}` {
			t.Fatalf("erro exposto: %s", body)
		}
	}
}
