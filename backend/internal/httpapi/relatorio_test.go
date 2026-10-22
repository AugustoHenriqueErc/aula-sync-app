package httpapi

import (
	"context"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"app-sync-backend/internal/aula"
	"app-sync-backend/internal/storage"

	"github.com/gofiber/fiber/v3"
)

func baixarPDF(t *testing.T, app *fiber.App, query string) string {
	t.Helper()
	res, err := app.Test(httptest.NewRequest("GET", "/aulas/relatorio.pdf?"+query, nil), fiber.TestConfig{Timeout: 0})
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 200 || res.Header.Get("Content-Type") != "application/pdf" || !strings.Contains(res.Header.Get("Content-Disposition"), "attachment;") || res.Header.Get("Cache-Control") != "no-store" || !strings.HasPrefix(string(b), "%PDF-") {
		t.Fatalf("resposta inválida: %d %v", res.StatusCode, res.Header)
	}
	path := filepath.Join(t.TempDir(), "relatorio.pdf")
	if err := os.WriteFile(path, b, 0600); err != nil {
		t.Fatal(err)
	}
	// Artefatos opcionais para revisão visual, sempre com dados fictícios.
	if dir := os.Getenv("AULA_PDF_TEST_DIR"); dir != "" {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, t.Name()+".pdf"), b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := exec.LookPath("pdftotext"); err != nil {
		t.Skip("PDF gerado; instale Poppler para conferir o texto extraído")
	}
	out, err := exec.Command("pdftotext", "-layout", path, "-").CombinedOutput()
	if err != nil {
		t.Fatalf("PDF ilegível: %s %v", out, err)
	}
	return string(out)
}

func contemPDF(t *testing.T, texto string, esperados ...string) {
	t.Helper()
	for _, s := range esperados {
		if !strings.Contains(texto, s) {
			t.Fatalf("PDF não contém %q:\n%s", s, texto)
		}
	}
}

func TestRelatorioPDFTotalFiltrosCorrecaoECancelamento(t *testing.T) {
	app, banco := novaAPITeste(t)
	ctx := context.Background()
	var registros []storage.Registro
	for i, fim := range []string{"09:00", "09:30", "08:45"} {
		r, err := banco.Inserir(ctx, aula.Aula{Data: fmt.Sprintf("2026-10-%02d", i+1), Inicio: "08:00", Fim: fim, Turma: "José · Reforço", Conteudo: "Leitura, ação e revisão"})
		if err != nil {
			t.Fatal(err)
		}
		registros = append(registros, r)
	}
	texto := baixarPDF(t, app, "situacao=todas&pagina=2")
	contemPDF(t, texto, "3 aula(s)", "3h15", "José · Reforço", "Leitura, ação e revisão", "01/10/2026", "03/10/2026")
	contemPDF(t, baixarPDF(t, app, "data_inicial=2026-10-02&data_final=2026-10-02"), "1 aula(s)", "1h30")
	contemPDF(t, baixarPDF(t, app, "turma_id=1"), "0 aula(s)", "0h00", "Nenhuma aula encontrada")
	a := registros[0].Aula
	a.Conteudo, a.Fim = "Conteúdo corrigido", "10:00"
	if _, err := banco.Alterar(ctx, registros[0].ID, storage.Alteracao{Versao: 1, TurmaID: registros[0].TurmaID, Aula: a}); err != nil {
		t.Fatal(err)
	}
	cancelada := true
	if _, err := banco.Alterar(ctx, registros[1].ID, storage.Alteracao{Versao: 1, Cancelada: &cancelada}); err != nil {
		t.Fatal(err)
	}
	contemPDF(t, baixarPDF(t, app, "situacao=todas"), "3 aula(s)", "2h45", "CANCELADA", "Conteúdo corrigido")
	contemPDF(t, baixarPDF(t, app, "situacao=ativas"), "2 aula(s)", "2h45")
	contemPDF(t, baixarPDF(t, app, "situacao=canceladas"), "1 aula(s)", "0h00")
}

func TestRelatorioPDFTodasPaginasTextosLongos(t *testing.T) {
	app, banco := novaAPITeste(t)
	for i := 1; i <= 25; i++ {
		_, err := banco.Inserir(context.Background(), aula.Aula{Data: "2026-10-01", Inicio: "08:00", Fim: "09:00", Turma: "Estudante fictício", Conteudo: fmt.Sprintf("MARCADOR%03d %s FIM%03d", i, strings.Repeat("Revisão de matemática e interpretação. ", 40), i), Observacoes: strings.Repeat("A", 300)})
		if err != nil {
			t.Fatal(err)
		}
	}
	texto := baixarPDF(t, app, "pagina=2")
	contemPDF(t, texto, "25 aula(s)", "25h00", "Página 2 de", "Revisão de matemática")
	for i := 1; i <= 25; i++ {
		contemPDF(t, texto, fmt.Sprintf("MARCADOR%03d", i), fmt.Sprintf("FIM%03d", i))
	}
}

func TestRelatorioPDFVazioEErros(t *testing.T) {
	app, banco := novaAPITeste(t)
	contemPDF(t, baixarPDF(t, app, ""), "0 aula(s)", "0h00", "Nenhuma aula encontrada")
	for _, q := range []string{"data_inicial=2026-02-30", "situacao=invalida", "pagina=0", "turma_id=1&turma_id=2", "outro=1"} {
		requisicao(t, app, "GET", "/aulas/relatorio.pdf?"+q, "", "", 400)
	}
	if err := banco.Close(); err != nil {
		t.Fatal(err)
	}
	requisicao(t, app, "GET", "/aulas/relatorio.pdf", "", "", 500)
}
