package auth

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-jose/go-jose/v4"
	"github.com/gofiber/fiber/v3"
)

func TestConfig(t *testing.T) {
	valid := map[string]string{"GOOGLE_CLIENT_ID": "client", "GOOGLE_CLIENT_SECRET": "secret", "GOOGLE_REDIRECT_URI": "http://localhost:5173/api/auth/google/callback", "APP_URL": "http://localhost:5173"}
	for _, tt := range []struct {
		name, key, value string
		ok               bool
	}{
		{"valid", "", "", true}, {"partial", "GOOGLE_CLIENT_SECRET", "", false}, {"external callback", "GOOGLE_REDIRECT_URI", "https://evil.test/api/auth/google/callback", false}, {"http production", "APP_URL", "http://app.example", false}, {"origin path", "APP_URL", "http://localhost:5173/app", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := ConfigFromEnv(func(k string) string {
				if k == tt.key {
					return tt.value
				}
				return valid[k]
			})
			if (err == nil) != tt.ok {
				t.Fatalf("config=%+v err=%v", cfg, err)
			}
		})
	}
	c, err := ConfigFromEnv(func(string) string { return "" })
	if err != nil || New(c) != nil {
		t.Fatal("modo local deve funcionar sem OAuth")
	}
}
func appFor(g *Google) *fiber.App {
	app := fiber.New()
	g.Register(app)
	app.Get("/private", func(c fiber.Ctx) error { return c.SendString("private") })
	app.Post("/private", func(c fiber.Ctx) error { return c.SendStatus(204) })
	app.Get("/sheet", func(c fiber.Ctx) error {
		title, err := g.ReadSheet(c, "sheet123")
		if err != nil {
			return err
		}
		return c.SendString(title)
	})
	return app
}
func request(t *testing.T, app *fiber.App, method, path string, cookie *http.Cookie, origin string) *http.Response {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	res, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { res.Body.Close() })
	return res
}
func cookieNamed(t *testing.T, res *http.Response, name string) *http.Cookie {
	t.Helper()
	for _, c := range res.Cookies() {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("cookie %s ausente", name)
	return nil
}
func testConfig() Config {
	return Config{"client", "secret", "http://localhost:5173/api/auth/google/callback", "http://localhost:5173"}
}
func TestLoginAndCallback(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	wrong, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	for _, variant := range []string{"valid", "state", "cookie", "expired_flow", "denied", "nonce", "audience", "issuer", "expired_token", "signature", "azp", "no_sheets"} {
		t.Run(variant, func(t *testing.T) {
			g := New(testConfig())
			g.verify = oidc.NewVerifier("https://accounts.google.com", &oidc.StaticKeySet{PublicKeys: []crypto.PublicKey{&key.PublicKey}}, &oidc.Config{ClientID: "client"}).Verify
			app := appFor(g)
			start := request(t, app, "POST", "/auth/google/login", nil, g.config.AppURL)
			if start.StatusCode != 303 {
				t.Fatal(start.StatusCode)
			}
			flowCookieValue := cookieNamed(t, start, flowCookie)
			if !flowCookieValue.HttpOnly || flowCookieValue.SameSite != http.SameSiteLaxMode {
				t.Fatal("cookie inseguro")
			}
			u, _ := url.Parse(start.Header.Get("Location"))
			q := u.Query()
			if q.Get("code_challenge_method") != "S256" || q.Get("code_challenge") == "" || q.Get("nonce") == "" {
				t.Fatal("PKCE/nonce ausente")
			}
			f := g.flows[flowCookieValue.Value]
			claims := map[string]any{"iss": "https://accounts.google.com", "sub": "user-a", "aud": "client", "exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Unix(), "nonce": f.Nonce, "email": "a@example.test", "email_verified": true, "name": "Pessoa A"}
			signingKey := key
			switch variant {
			case "nonce":
				claims["nonce"] = "wrong"
			case "audience":
				claims["aud"] = "other"
			case "issuer":
				claims["iss"] = "https://evil.test"
			case "expired_token":
				claims["exp"] = time.Now().Add(-time.Hour).Unix()
			case "signature":
				signingKey = wrong
			case "azp":
				claims["azp"] = "other"
			}
			signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: signingKey}, nil)
			if err != nil {
				t.Fatal(err)
			}
			payload, _ := json.Marshal(claims)
			obj, err := signer.Sign(payload)
			if err != nil {
				t.Fatal(err)
			}
			jwt, err := obj.CompactSerialize()
			if err != nil {
				t.Fatal(err)
			}
			exchanges := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				exchanges++
				r.ParseForm()
				if r.Form.Get("code_verifier") != f.Verifier || r.Form.Get("redirect_uri") != g.config.RedirectURI {
					t.Error("troca sem vínculo PKCE/redirect")
				}
				w.Header().Set("Content-Type", "application/json")
				scope := sheetsScope
				if variant == "no_sheets" {
					scope = "openid email profile"
				}
				json.NewEncoder(w).Encode(map[string]any{"access_token": "private-access-token", "token_type": "Bearer", "expires_in": 3600, "id_token": jwt, "scope": scope})
			}))
			defer server.Close()
			g.oauth.Endpoint.TokenURL = server.URL
			state := f.State
			callbackCookie := flowCookieValue
			switch variant {
			case "state":
				state = "wrong"
			case "cookie":
				callbackCookie = nil
			case "expired_flow":
				f.Expires = time.Now().Add(-time.Second)
				g.flows[flowCookieValue.Value] = f
			}
			path := "/auth/google/callback?state=" + state + "&code=code"
			if variant == "denied" {
				path = "/auth/google/callback?state=" + state + "&error=access_denied"
			}
			res := request(t, app, "GET", path, callbackCookie, "")
			success := variant == "valid" || variant == "no_sheets"
			if success {
				if res.Header.Get("Location") != g.config.AppURL+"/" {
					t.Fatal(res.Header.Get("Location"))
				}
				cookie := cookieNamed(t, res, sessionCookie)
				status := request(t, app, "GET", "/auth/session", cookie, "")
				body, _ := io.ReadAll(status.Body)
				if strings.Contains(string(body), "private-access-token") || !strings.Contains(string(body), "user-a") {
					t.Fatalf("resposta sessão: %s", body)
				}
				if request(t, app, "GET", "/private", cookie, "").StatusCode != 200 {
					t.Fatal("sessão recusada")
				}
				if request(t, app, "POST", "/private", cookie, "https://evil.test").StatusCode != 403 {
					t.Fatal("CSRF aceito")
				}
				if request(t, app, "POST", "/private", cookie, g.config.AppURL).StatusCode != 204 {
					t.Fatal("mutação legítima recusada")
				}
				if variant == "no_sheets" && request(t, app, "GET", "/sheet", cookie, "").StatusCode != 403 {
					t.Fatal("scope ausente aceito")
				}
				replay := request(t, app, "GET", path, flowCookieValue, "")
				if !strings.Contains(replay.Header.Get("Location"), "estado_invalido") {
					t.Fatal("replay aceito")
				}
				if request(t, app, "POST", "/auth/logout", cookie, g.config.AppURL).StatusCode != 204 || request(t, app, "GET", "/private", cookie, "").StatusCode != 401 {
					t.Fatal("logout não invalidou sessão")
				}
			} else {
				if !strings.Contains(res.Header.Get("Location"), "auth_error=") || len(g.sessions) != 0 {
					t.Fatal("callback inválido aceito")
				}
			}
			if (variant == "state" || variant == "cookie" || variant == "expired_flow" || variant == "denied") && exchanges != 0 {
				t.Fatal("callback inválido chegou à troca de token")
			}
		})
	}
}

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestSessionAndSheet(t *testing.T) {
	g := New(testConfig())
	app := appFor(g)
	if request(t, app, "GET", "/private", nil, "").StatusCode != 401 {
		t.Fatal("rota sem sessão aberta")
	}
	if request(t, app, "POST", "/auth/google/login", nil, "https://evil.test").StatusCode != 403 {
		t.Fatal("login CSRF")
	}
	for _, status := range []int{200, 401, 403, 404, 429, 500} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			g.sessions["session-a"] = session{User: User{Subject: "a"}, Token: "token-a", Sheets: true, Expires: time.Now().Add(time.Hour)}
			g.sessions["session-b"] = session{User: User{Subject: "b"}, Token: "token-b", Sheets: true, Expires: time.Now().Add(time.Hour)}
			g.client = &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
				if r.Method != "GET" || r.Header.Get("Authorization") != "Bearer token-a" || r.URL.Host != "sheets.googleapis.com" {
					t.Error("método, destino ou conta incorretos")
				}
				return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(`{"spreadsheetId":"sheet123","properties":{"title":"Teste"}}`)), Header: make(http.Header)}, nil
			})}
			cookie := &http.Cookie{Name: sessionCookie, Value: "session-a"}
			res := request(t, app, "GET", "/sheet", cookie, "")
			expected := status
			if status == 404 {
				expected = 403
			}
			if status == 429 || status == 500 {
				expected = 502
			}
			if res.StatusCode != expected {
				t.Fatalf("status %d esperado %d", res.StatusCode, expected)
			}
			if status == 401 && request(t, app, "GET", "/private", cookie, "").StatusCode != 401 {
				t.Fatal("revogação não invalidou sessão")
			}
			if _, ok := g.sessions["session-b"]; !ok {
				t.Fatal("revogação afetou outra conta")
			}
		})
	}
	g.sessions["expired"] = session{Expires: time.Now().Add(-time.Second)}
	if request(t, app, "GET", "/private", &http.Cookie{Name: sessionCookie, Value: "expired"}, "").StatusCode != 401 {
		t.Fatal("sessão expirada aceita")
	}
	g.sessions["old"] = session{Expires: time.Now().Add(time.Hour)}
	request(t, app, "POST", "/auth/google/login", &http.Cookie{Name: sessionCookie, Value: "old"}, g.config.AppURL)
	if _, ok := g.sessions["old"]; ok {
		t.Fatal("troca de conta preservou sessão anterior")
	}
}
func TestSecureCookie(t *testing.T) {
	g := New(Config{"client", "secret", "https://app.example/api/auth/google/callback", "https://app.example"})
	res := request(t, appFor(g), "POST", "/auth/google/login", nil, g.config.AppURL)
	if !cookieNamed(t, res, flowCookie).Secure {
		t.Fatal("cookie HTTPS sem Secure")
	}
}

// A assinatura é sempre validada pelo verificador OIDC real nos testes acima.
