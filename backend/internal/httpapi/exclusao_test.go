package httpapi

import (
	"context"
	"strconv"
	"testing"
)

func TestExcluirHTTPExigeCancelamentoVersaoEEscolhaExplicita(t *testing.T) {
	app, b := novaAPITeste(t)
	requisicao(t, app, "POST", "/aulas", corpoValido, "application/json", 201)
	path := "/aulas/1"
	for _, body := range []string{`{}`, `null`, `{"versao":1,"config_versao":0}`, `{"versao":1,"config_versao":0,"remover_google":false,"extra":true}`} {
		requisicao(t, app, "DELETE", path, body, "application/json", 400)
	}
	requisicao(t, app, "DELETE", path, `{"versao":1,"config_versao":0,"remover_google":false}`, "application/json", 409)
	requisicao(t, app, "PATCH", path+"/situacao", `{"versao":1,"cancelada":true}`, "application/json", 200)
	requisicao(t, app, "DELETE", path, `{"versao":1,"config_versao":0,"remover_google":false}`, "application/json", 409)
	requisicao(t, app, "DELETE", path, `{"versao":2,"config_versao":0,"remover_google":true}`, "application/json", 422)
	requisicao(t, app, "DELETE", path, `{"versao":2,"config_versao":0,"remover_google":false}`, "application/json", 204)
	requisicao(t, app, "GET", path, "", "", 404)
	records, err := b.Listar(context.Background())
	if err != nil || len(records) != 0 {
		t.Fatal(records, err)
	}
	requisicao(t, app, "POST", "/aulas", corpoValido, "application/json", 201)
	records, err = b.Listar(context.Background())
	if err != nil || len(records) != 1 || records[0].ID <= 1 {
		t.Fatal("ID reutilizado", strconv.FormatInt(records[0].ID, 10), err)
	}
}
