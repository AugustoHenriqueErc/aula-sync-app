package auth

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/gofiber/fiber/v3"
)

// SheetsSession só é usado internamente para vincular confirmações à sessão atual.
func (g *Google) SheetsSession(c fiber.Ctx) (string, error) {
	if g == nil {
		return "", fiber.NewError(422, "Configure o login Google para sincronizar")
	}
	s, ok := g.current(c)
	if !ok {
		return "", fiber.NewError(401, "Entre com Google novamente")
	}
	if !s.Sheets {
		return "", fiber.NewError(403, "Autorize o acesso ao Google Sheets entrando novamente")
	}
	return c.Cookies(sessionCookie), nil
}

// SheetsRequest recebe apenas caminhos construídos pelo cliente interno Sheets.
func (g *Google) SheetsRequest(c fiber.Ctx, method, path string, body, result any) error {
	if _, err := g.SheetsSession(c); err != nil {
		return err
	}
	s, ok := g.current(c)
	if !ok {
		return fiber.NewError(401, "Entre com Google novamente")
	}
	var data []byte
	var err error
	if body != nil {
		data, err = json.Marshal(body)
		if err != nil {
			return err
		}
	}
	req, err := http.NewRequestWithContext(c.Context(), method, "https://sheets.googleapis.com/v4/spreadsheets/"+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.Token)
	req.Header.Set("Content-Type", "application/json")
	res, err := g.client.Do(req)
	if err != nil {
		return fiber.NewError(409, "Não foi possível confirmar a operação no Google. Inspecione e tente novamente para conferir o resultado")
	}
	defer res.Body.Close()
	if res.StatusCode == 401 {
		g.forget(c)
		return fiber.NewError(401, "Autorização expirada ou revogada. Entre novamente")
	}
	if res.StatusCode == 403 || res.StatusCode == 404 {
		return fiber.NewError(403, "Sem acesso à planilha ou à escrita. Confira a conta, as permissões e a API Sheets")
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fiber.NewError(409, "Google não confirmou a operação. Inspecione a planilha e tente novamente")
	}
	// Limite explícito: uma leitura truncada nunca será interpretada como aba vazia.
	raw, err := io.ReadAll(io.LimitReader(res.Body, 16*1024*1024+1))
	if err != nil || len(raw) > 16*1024*1024 {
		return fiber.NewError(409, "Não foi possível inspecionar toda a planilha; operação bloqueada")
	}
	if err = json.Unmarshal(raw, result); err != nil {
		return fiber.NewError(409, "Resposta incompleta do Google; inspecione novamente")
	}
	if _, ok := g.current(c); !ok {
		return fiber.NewError(401, "Sessão encerrada durante a operação. Entre e confira o resultado")
	}
	return nil
}
