// Package auth mantém sessões do protótipo em memória. Nenhum token é persistido.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gofiber/fiber/v3"
	"golang.org/x/oauth2"
)

const sheetsScope = "https://www.googleapis.com/auth/spreadsheets"
const sessionCookie = "aula_session"
const flowCookie = "aula_oauth"

type Config struct{ ClientID, ClientSecret, RedirectURI, AppURL string }

// ConfigFromEnv falha em configuração parcial; ausência total mantém o modo local.
func ConfigFromEnv(get func(string) string) (Config, error) {
	c := Config{get("GOOGLE_CLIENT_ID"), get("GOOGLE_CLIENT_SECRET"), get("GOOGLE_REDIRECT_URI"), get("APP_URL")}
	if c == (Config{}) {
		return c, nil
	}
	if c.ClientID == "" || c.ClientSecret == "" || c.RedirectURI == "" || c.AppURL == "" {
		return Config{}, errors.New("configure GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET, GOOGLE_REDIRECT_URI e APP_URL juntos")
	}
	app, err := url.Parse(c.AppURL)
	if err != nil || app.Host == "" || app.User != nil || app.RawQuery != "" || app.Fragment != "" || (app.Path != "" && app.Path != "/") {
		return Config{}, errors.New("APP_URL deve ser a origem do frontend")
	}
	if app.Scheme != "https" && !(app.Scheme == "http" && (app.Hostname() == "localhost" || app.Hostname() == "127.0.0.1")) {
		return Config{}, errors.New("APP_URL exige HTTPS, exceto em localhost ou 127.0.0.1")
	}
	callback, err := url.Parse(c.RedirectURI)
	if err != nil || callback.Scheme != app.Scheme || callback.Host != app.Host || callback.User != nil || callback.RawQuery != "" || callback.Fragment != "" || (callback.Path != "/api/auth/google/callback" && callback.Path != "/auth/google/callback") {
		return Config{}, errors.New("GOOGLE_REDIRECT_URI deve apontar para /api/auth/google/callback na origem de APP_URL (ou /auth/google/callback sem proxy)")
	}
	c.AppURL = strings.TrimSuffix(c.AppURL, "/")
	return c, nil
}

type User struct {
	Subject string `json:"sub"`
	Name    string `json:"nome"`
	Email   string `json:"email"`
}
type session struct {
	User    User
	Token   string
	Sheets  bool
	Expires time.Time
}
type flow struct {
	State, Nonce, Verifier string
	Expires                time.Time
}
type Google struct {
	config   Config
	oauth    oauth2.Config
	client   *http.Client
	verify   func(context.Context, string) (*oidc.IDToken, error)
	mu       sync.Mutex
	sessions map[string]session
	flows    map[string]flow
}

func New(c Config) *Google {
	if c.ClientID == "" {
		return nil
	}
	client := &http.Client{Timeout: 8 * time.Second}
	ctx := oidc.ClientContext(context.Background(), client)
	verifier := oidc.NewVerifier("https://accounts.google.com", oidc.NewRemoteKeySet(ctx, "https://www.googleapis.com/oauth2/v3/certs"), &oidc.Config{ClientID: c.ClientID})
	return &Google{config: c, client: client, verify: verifier.Verify,
		oauth:    oauth2.Config{ClientID: c.ClientID, ClientSecret: c.ClientSecret, RedirectURL: c.RedirectURI, Scopes: []string{"openid", "email", "profile", sheetsScope}, Endpoint: oauth2.Endpoint{AuthURL: "https://accounts.google.com/o/oauth2/v2/auth", TokenURL: "https://oauth2.googleapis.com/token", AuthStyle: oauth2.AuthStyleInParams}},
		sessions: map[string]session{}, flows: map[string]flow{}}
}
func random() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("fonte aleatória indisponível")
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
func equal(a, b string) bool { return a != "" && subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1 }
func (g *Google) cookie(c fiber.Ctx, name, value string, seconds int) {
	c.Cookie(&fiber.Cookie{Name: name, Value: value, Path: "/", HTTPOnly: true, Secure: strings.HasPrefix(g.config.AppURL, "https://"), SameSite: "Lax", MaxAge: seconds})
}
func (g *Google) cleanLocked() {
	now := time.Now()
	for k, v := range g.flows {
		if !now.Before(v.Expires) {
			delete(g.flows, k)
		}
	}
	for k, v := range g.sessions {
		if !now.Before(v.Expires) {
			delete(g.sessions, k)
		}
	}
}
func (g *Google) current(c fiber.Ctx) (session, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.cleanLocked()
	s, ok := g.sessions[c.Cookies(sessionCookie)]
	return s, ok
}
func (g *Google) forget(c fiber.Ctx) {
	g.mu.Lock()
	delete(g.sessions, c.Cookies(sessionCookie))
	g.mu.Unlock()
	g.cookie(c, sessionCookie, "", -1)
}
func (g *Google) origin(c fiber.Ctx) error {
	if c.Get("Origin") != g.config.AppURL {
		return fiber.NewError(403, "Origem da solicitação não permitida")
	}
	return nil
}

// Register deve ser chamado antes das rotas privadas. Health continua público.
func (g *Google) Register(app *fiber.App) {
	app.Get("/auth/session", func(c fiber.Ctx) error {
		c.Set("Cache-Control", "no-store")
		if g == nil {
			return c.JSON(fiber.Map{"configurado": false, "usuario": nil, "sheets_autorizado": false})
		}
		s, ok := g.current(c)
		if !ok {
			return c.JSON(fiber.Map{"configurado": true, "usuario": nil, "sheets_autorizado": false})
		}
		return c.JSON(fiber.Map{"configurado": true, "usuario": s.User, "sheets_autorizado": s.Sheets})
	})
	if g == nil {
		return
	}
	app.Use(func(c fiber.Ctx) error {
		c.Set("Cache-Control", "no-store")
		c.Set("Referrer-Policy", "no-referrer")
		return c.Next()
	})
	app.Post("/auth/google/login", g.login)
	app.Get("/auth/google/callback", g.callback)
	app.Post("/auth/logout", func(c fiber.Ctx) error {
		if err := g.origin(c); err != nil {
			return err
		}
		g.forget(c)
		return c.SendStatus(204)
	})
	app.Use(func(c fiber.Ctx) error {
		if _, ok := g.current(c); !ok {
			return fiber.NewError(401, "Sessão ausente ou expirada. Entre com Google novamente.")
		}
		if c.Method() != "GET" && c.Method() != "HEAD" {
			if err := g.origin(c); err != nil {
				return err
			}
		}
		return c.Next()
	})
}
func (g *Google) login(c fiber.Ctx) error {
	if err := g.origin(c); err != nil {
		return err
	}
	f := flow{random(), random(), oauth2.GenerateVerifier(), time.Now().Add(10 * time.Minute)}
	id := random()
	g.mu.Lock()
	g.cleanLocked()
	if len(g.flows) >= 1000 {
		g.mu.Unlock()
		return fiber.NewError(429, "Muitas tentativas de login; tente mais tarde")
	}
	delete(g.flows, c.Cookies(flowCookie))
	g.flows[id] = f
	g.mu.Unlock()
	g.forget(c)
	g.cookie(c, flowCookie, id, 600)
	return c.Redirect().Status(303).To(g.oauth.AuthCodeURL(f.State, oauth2.SetAuthURLParam("nonce", f.Nonce), oauth2.SetAuthURLParam("prompt", "select_account"), oauth2.S256ChallengeOption(f.Verifier)))
}
func (g *Google) callback(c fiber.Ctx) error {
	g.mu.Lock()
	g.cleanLocked()
	f, ok := g.flows[c.Cookies(flowCookie)]
	delete(g.flows, c.Cookies(flowCookie))
	g.mu.Unlock()
	g.cookie(c, flowCookie, "", -1)
	fail := func(code string) error { return c.Redirect().Status(303).To(g.config.AppURL + "/?auth_error=" + code) }
	if !ok || !equal(f.State, c.Query("state")) {
		return fail("estado_invalido")
	}
	if c.Query("error") != "" {
		return fail("consentimento_recusado")
	}
	if c.Query("code") == "" {
		return fail("codigo_ausente")
	}
	ctx, cancel := context.WithTimeout(c.Context(), 8*time.Second)
	defer cancel()
	ctx = context.WithValue(ctx, oauth2.HTTPClient, g.client)
	token, err := g.oauth.Exchange(ctx, c.Query("code"), oauth2.VerifierOption(f.Verifier))
	if err != nil {
		return fail("falha_autorizacao")
	}
	raw, _ := token.Extra("id_token").(string)
	id, err := g.verify(ctx, raw)
	if err != nil || !equal(id.Nonce, f.Nonce) || id.Subject == "" {
		return fail("identidade_invalida")
	}
	var claims struct {
		Name            string `json:"name"`
		Email           string `json:"email"`
		Verified        bool   `json:"email_verified"`
		AuthorizedParty string `json:"azp"`
	}
	if err := id.Claims(&claims); err != nil || !claims.Verified || (claims.AuthorizedParty != "" && claims.AuthorizedParty != g.config.ClientID) {
		return fail("identidade_invalida")
	}
	if token.AccessToken == "" || !strings.EqualFold(token.TokenType, "Bearer") || token.Expiry.IsZero() || !time.Now().Before(token.Expiry) {
		return fail("falha_autorizacao")
	}
	scope, _ := token.Extra("scope").(string)
	sheets := false
	for _, s := range strings.Fields(scope) {
		if s == sheetsScope {
			sheets = true
		}
	}
	expires := time.Now().Add(time.Hour)
	if token.Expiry.Before(expires) {
		expires = token.Expiry
	}
	if id.Expiry.Before(expires) {
		expires = id.Expiry
	}
	sid := random()
	g.mu.Lock()
	g.cleanLocked()
	if len(g.sessions) >= 1000 {
		g.mu.Unlock()
		return fail("limite_sessoes")
	}
	delete(g.sessions, c.Cookies(sessionCookie))
	g.sessions[sid] = session{User{id.Subject, claims.Name, claims.Email}, token.AccessToken, sheets, expires}
	g.mu.Unlock()
	g.cookie(c, sessionCookie, sid, int(time.Until(expires).Seconds()))
	return c.Redirect().Status(303).To(g.config.AppURL + "/")
}

// ReadSheet faz somente uma leitura de metadados. Não comprova permissão de escrita.
func (g *Google) ReadSheet(c fiber.Ctx, id string) (string, error) {
	if g == nil {
		return "", fiber.NewError(503, "Login Google não configurado")
	}
	s, ok := g.current(c)
	if !ok {
		return "", fiber.NewError(401, "Entre com Google novamente")
	}
	if !s.Sheets {
		return "", fiber.NewError(403, "Autorize o acesso ao Google Sheets entrando com Google novamente")
	}
	ctx, cancel := context.WithTimeout(c.Context(), 8*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", "https://sheets.googleapis.com/v4/spreadsheets/"+url.PathEscape(id)+"?fields=spreadsheetId,properties(title)", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+s.Token)
	res, err := g.client.Do(req)
	if err != nil {
		return "", fiber.NewError(502, "Não foi possível consultar o Google; tente novamente")
	}
	defer res.Body.Close()
	if res.StatusCode == 401 {
		g.forget(c)
		return "", fiber.NewError(401, "Autorização expirada ou revogada. Entre com Google novamente")
	}
	if res.StatusCode == 403 || res.StatusCode == 404 {
		return "", fiber.NewError(403, "A planilha não está acessível à conta conectada. Confira o link, as permissões e a configuração da API Sheets")
	}
	if res.StatusCode != 200 {
		return "", fiber.NewError(502, "Google indisponível; tente novamente")
	}
	var data struct {
		ID         string `json:"spreadsheetId"`
		Properties struct {
			Title string `json:"title"`
		} `json:"properties"`
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, 64*1024)).Decode(&data); err != nil || data.ID != id {
		return "", fiber.NewError(502, "Resposta inesperada do Google")
	}
	if _, ok := g.current(c); !ok {
		return "", fiber.NewError(401, "Sessão encerrada durante a consulta")
	}
	return data.Properties.Title, nil
}
