package httpapi

import (
	"encoding/json"
	"testing"
)

func TestConfiguracaoGoogleHTTP(t *testing.T) {
	app, _ := novaAPITeste(t)
	body, _ := requisicao(t, app, "GET", "/integracoes/google/configuracao", "", "", 200)
	if string(body) != `{"versao":0,"destino":null}` {
		t.Fatalf("config inicial: %s", body)
	}
	body, _ = requisicao(t, app, "PUT", "/integracoes/google/configuracao", `{"url":"https://docs.google.com/spreadsheets/d/teste/edit?usp=sharing#gid=0","versao":0}`, "application/json", 200)
	var c configGoogleResponse
	if err := json.Unmarshal(body, &c); err != nil || c.Versao != 1 || c.Destino == nil || c.Destino.URL != "https://docs.google.com/spreadsheets/d/teste/edit" {
		t.Fatalf("salvar: %s %v", body, err)
	}
	requisicao(t, app, "PUT", "/integracoes/google/configuracao", `{"url":"https://docs.google.com/spreadsheets/d/outro/edit","versao":0}`, "application/json", 409)
	for _, body := range []string{`null`, `{}`, `{"url":"x","versao":null}`, `{"url":"x","versao":-1}`, `{"url":"x","versao":1.5}`, `{"url":"x","versao":1,"extra":true}`} {
		requisicao(t, app, "PUT", "/integracoes/google/configuracao", body, "application/json", 400)
	}
	requisicao(t, app, "PUT", "/integracoes/google/configuracao", `{"url":"https://example.com/","versao":1}`, "application/json", 422)
	requisicao(t, app, "POST", "/aulas", corpoValido, "application/json", 201)
	body, _ = requisicao(t, app, "GET", "/integracoes/google/planejamento", "", "", 200)
	var p struct {
		Configuracao configGoogleResponse `json:"configuracao"`
		Meses        []struct {
			Aba string `json:"aba_esperada"`
		} `json:"meses"`
		Remota string `json:"verificacao_remota"`
	}
	if err := json.Unmarshal(body, &p); err != nil || p.Remota != "nao_realizada" || p.Configuracao.Versao != 1 || len(p.Meses) != 1 || p.Meses[0].Aba != "Registros aulas - 10/2026" {
		t.Fatalf("plano: %s %v", body, err)
	}
}
