// Package google contém regras locais de destino; não faz chamadas de rede.
package google

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var caminhoPlanilha = regexp.MustCompile(`^/spreadsheets/d/([A-Za-z0-9_-]+)(?:/(?:edit|view|preview|copy))?/?$`)

// PlanilhaID aceita links de edição/compartilhamento, mas nunca usa gid como
// destino mensal. Validar o formato não comprova existência ou permissão.
func PlanilhaID(link string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(link))
	if err != nil || u.Scheme != "https" || u.Host != "docs.google.com" || u.User != nil || u.RawPath != "" {
		return "", fmt.Errorf("Informe um link HTTPS de planilha do Google Sheets")
	}
	partes := caminhoPlanilha.FindStringSubmatch(u.Path)
	if len(partes) != 2 {
		return "", fmt.Errorf("Use o link da planilha no formato https://docs.google.com/spreadsheets/d/ID/edit")
	}
	return partes[1], nil
}
func URLPlanilha(id string) string { return "https://docs.google.com/spreadsheets/d/" + id + "/edit" }
func Periodo(data string) (string, string, error) {
	d, err := time.Parse("2006-01-02", data)
	if err != nil || d.Year() < 1 || d.Format("2006-01-02") != data {
		return "", "", fmt.Errorf("Data inválida para determinar a aba mensal")
	}
	return d.Format("2006-01"), "Registros aulas - " + d.Format("01/2006"), nil
}
