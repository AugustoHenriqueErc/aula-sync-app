package syncgoogle

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"app-sync-backend/internal/aula"
	g "app-sync-backend/internal/google"
	"app-sync-backend/internal/storage"
)

// Simula o contrato HTTP do Sheets, incluindo gravação concluída com resposta perdida.
type fakeSheets struct {
	book                              g.Spreadsheet
	writes                            int
	failRead, failWrite, loseResponse bool
	failAt                            int
}

func (f *fakeSheets) client() g.Client { return g.Client{Request: f.request} }
func (f *fakeSheets) request(method, path string, body, result any) error {
	if method == "GET" {
		if f.failRead {
			return errors.New("leitura indisponível")
		}
		b, _ := json.Marshal(f.book)
		return json.Unmarshal(b, result)
	}
	if f.failWrite || (f.failAt > 0 && f.writes+1 == f.failAt) {
		return errors.New("escrita indisponível")
	}
	f.writes++
	raw, _ := json.Marshal(body)
	var batch struct {
		Requests []struct {
			Add *struct {
				Properties struct {
					ID    int64  `json:"sheetId"`
					Title string `json:"title"`
				}
			} `json:"addSheet"`
			Update *struct {
				Start struct {
					ID  int64 `json:"sheetId"`
					Row int   `json:"rowIndex"`
				}
				Rows []g.Row
			} `json:"updateCells"`
			Append *struct {
				ID        int64 `json:"sheetId"`
				Dimension string
				Length    int
			} `json:"appendDimension"`
		}
	}
	if err := json.Unmarshal(raw, &batch); err != nil {
		return err
	}
	for _, r := range batch.Requests {
		if r.Add != nil {
			for _, sh := range f.book.Sheets {
				if sh.Properties.Title == r.Add.Properties.Title {
					return errors.New("aba duplicada")
				}
			}
			sh := makeSheet(r.Add.Properties.ID, r.Add.Properties.Title)
			f.book.Sheets = append(f.book.Sheets, sh)
		}
		if r.Update != nil {
			for i := range f.book.Sheets {
				sh := &f.book.Sheets[i]
				if sh.Properties.ID != r.Update.Start.ID {
					continue
				}
				rows := sh.Rows()
				for len(rows) < r.Update.Start.Row+len(r.Update.Rows) {
					rows = append(rows, g.Row{})
				}
				copy(rows[r.Update.Start.Row:], r.Update.Rows)
				setRows(sh, rows)
			}
		}
		if r.Append != nil {
			for i := range f.book.Sheets {
				sh := &f.book.Sheets[i]
				if sh.Properties.ID == r.Append.ID {
					if r.Append.Dimension == "ROWS" {
						sh.Properties.Grid.Rows += r.Append.Length
					} else {
						sh.Properties.Grid.Columns += r.Append.Length
					}
				}
			}
		}
	}
	if f.loseResponse {
		f.loseResponse = false
		return errors.New("resposta perdida após escrita")
	}
	return nil
}
func makeSheet(id int64, title string) g.Sheet {
	var sh g.Sheet
	sh.Properties.ID = id
	sh.Properties.Title = title
	sh.Properties.Type = "GRID"
	sh.Properties.Grid.Rows = 1000
	sh.Properties.Grid.Columns = 26
	return sh
}
func setRows(sh *g.Sheet, rows []g.Row) {
	raw, _ := json.Marshal(map[string]any{"properties": sh.Properties, "data": []any{map[string]any{"rowData": rows}}})
	_ = json.Unmarshal(raw, sh)
}
func setup(t *testing.T) (*Service, *fakeSheets, storage.Registro, storage.ConfigGoogle) {
	t.Helper()
	ctx := context.Background()
	b, err := storage.Abrir(ctx, filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { b.Close() })
	r, err := b.Inserir(ctx, aula.Aula{Data: "2026-10-18", Inicio: "08:00", Fim: "10:00", Turma: "Teste", Conteudo: "=texto literal", IntervaloMinutos: 15})
	if err != nil {
		t.Fatal(err)
	}
	c, err := b.ConfigurarGoogle(ctx, "https://docs.google.com/spreadsheets/d/test/edit", 0)
	if err != nil {
		t.Fatal(err)
	}
	return New(b), &fakeSheets{book: g.Spreadsheet{ID: "test", Sheets: []g.Sheet{makeSheet(0, "Página inicial")}}}, r, c
}
func prepare(t *testing.T, s *Service, f *fakeSheets, c storage.ConfigGoogle, p string) {
	t.Helper()
	i, err := s.Inspect(context.Background(), f.client(), "session", c.Versao, p)
	if err != nil {
		t.Fatal(err)
	}
	if i.Estado == "pronta" {
		return
	}
	i, err = s.Prepare(context.Background(), f.client(), "session", i.Confirmacao, c.Versao, p)
	if err != nil || i.Estado != "pronta" {
		t.Fatalf("preparar: %+v %v", i, err)
	}
}
func syncAula(t *testing.T, s *Service, f *fakeSheets, c storage.ConfigGoogle, r storage.Registro) {
	t.Helper()
	state, err := s.Sync(context.Background(), f.client(), c.Versao, r.ID, r.Versao)
	if err != nil || state.Estado != "sincronizado" {
		t.Fatalf("sincronizar: %+v %v", state, err)
	}
}
func edit(t *testing.T, s *Service, r storage.Registro, date string) storage.Registro {
	t.Helper()
	a := r.Aula
	a.Data = date
	a.Conteudo = "Corrigida"
	out, err := s.Banco.Alterar(context.Background(), r.ID, storage.Alteracao{Versao: r.Versao, TurmaID: r.TurmaID, Aula: a})
	if err != nil {
		t.Fatal(err)
	}
	return out
}
func TestFluxoRepeticaoEdicaoCancelamentoEMudancaDeMes(t *testing.T) {
	s, f, r, c := setup(t)
	ctx := context.Background()
	prepare(t, s, f, c, "2026-10")
	syncAula(t, s, f, c, r)
	writes := f.writes
	syncAula(t, s, f, c, r)
	if f.writes != writes || len(f.book.Sheets[1].Rows()) != 2 {
		t.Fatal("repetição duplicou escrita/linha")
	}
	if f.book.Sheets[1].Rows()[1].Values[7].String() != "=texto literal" {
		t.Fatal("texto interpretado como fórmula")
	}
	r = edit(t, s, r, "2026-10-19")
	state, _ := s.Banco.EstadoGoogle(ctx, r)
	if state.Estado != "pendente" {
		t.Fatalf("edição não ficou pendente: %+v", state)
	}
	syncAula(t, s, f, c, r)
	cancel := true
	r, err := s.Banco.Alterar(ctx, r.ID, storage.Alteracao{Versao: r.Versao, Cancelada: &cancel})
	if err != nil {
		t.Fatal(err)
	}
	syncAula(t, s, f, c, r)
	if f.book.Sheets[1].Rows()[1].Values[10].String() != "cancelada" {
		t.Fatal("cancelamento não refletido")
	}
	cancel = false
	r, err = s.Banco.Alterar(ctx, r.ID, storage.Alteracao{Versao: r.Versao, Cancelada: &cancel})
	if err != nil {
		t.Fatal(err)
	}
	r = edit(t, s, r, "2027-10-01")
	prepare(t, s, f, c, "2027-10")
	syncAula(t, s, f, c, r)
	if f.book.Sheets[1].Rows()[1].Values[10].String() != "movida" || f.book.Sheets[2].Rows()[1].Values[10].String() != "ativa" {
		t.Fatal("mudança de ano não reconciliada")
	}
	r = edit(t, s, r, "2026-10-20")
	syncAula(t, s, f, c, r)
	if len(f.book.Sheets[1].Rows()) != 2 || f.book.Sheets[2].Rows()[1].Values[10].String() != "movida" {
		t.Fatal("retorno ao mês duplicou aula")
	}
}
func TestPreparacaoConfirmacaoEProtecaoDeConteudo(t *testing.T) {
	s, f, _, c := setup(t)
	ctx := context.Background()
	i, err := s.Inspect(ctx, f.client(), "session", c.Versao, "2026-10")
	if err != nil || i.Estado != "ausente" || f.writes != 0 {
		t.Fatal("inspeção gravou")
	}
	if _, err = s.Prepare(ctx, f.client(), "outra", i.Confirmacao, c.Versao, "2026-10"); err == nil {
		t.Fatal("aceitou confirmação de outra sessão")
	}
	if _, err = s.Prepare(ctx, f.client(), "session", i.Confirmacao, c.Versao, "2026-11"); err == nil {
		t.Fatal("aceitou outro mês")
	}
	f.loseResponse = true
	if _, err = s.Prepare(ctx, f.client(), "session", i.Confirmacao, c.Versao, "2026-10"); err == nil {
		t.Fatal("esperava resposta perdida")
	}
	if _, err = s.Prepare(ctx, f.client(), "session", i.Confirmacao, c.Versao, "2026-10"); err != nil {
		t.Fatal(err)
	}
	if f.writes != 1 || len(f.book.Sheets) != 2 {
		t.Fatal("preparação duplicada")
	}
	// Fórmula distante que resulta em vazio ainda é conteúdo: não basta olhar A1:L5.
	sh := makeSheet(5, "Registros aulas - 11/2026")
	rows := make([]g.Row, 120)
	rows[119] = g.Row{Values: []g.Cell{{Value: map[string]any{"formulaValue": "=\"\""}}}}
	setRows(&sh, rows)
	f.book.Sheets = append(f.book.Sheets, sh)
	i, err = s.Inspect(ctx, f.client(), "session", c.Versao, "2026-11")
	if err != nil || i.Estado != "incompativel" || i.Confirmacao != "" {
		t.Fatalf("conteúdo distante ignorado: %+v %v", i, err)
	}
	// Aba completamente vazia requer confirmação.
	sh = makeSheet(6, "Registros aulas - 12/2026")
	f.book.Sheets = append(f.book.Sheets, sh)
	prepare(t, s, f, c, "2026-12")
}
func TestFalhaRespostaPerdidaERecuperacaoAposReinicio(t *testing.T) {
	s, f, r, c := setup(t)
	ctx := context.Background()
	prepare(t, s, f, c, "2026-10")
	f.loseResponse = true
	if _, err := s.Sync(ctx, f.client(), c.Versao, r.ID, r.Versao); err == nil {
		t.Fatal("esperava falha")
	}
	state, _ := s.Banco.EstadoGoogle(ctx, r)
	if state.Estado != "erro" {
		t.Fatal(state)
	}
	s = New(s.Banco)
	r = edit(t, s, r, "2026-10-19")
	syncAula(t, s, f, c, r)
	if len(f.book.Sheets[1].Rows()) != 2 || f.book.Sheets[1].Rows()[1].Values[2].String() != r.Aula.Data {
		t.Fatal("recuperação não reconciliou versão anterior")
	}
}
func TestBloqueiaMudancaManualDuplicacaoRenomeacaoEFormula(t *testing.T) {
	for _, scenario := range []string{"manual", "duplicada", "renomeada", "formula", "removida", "outra aba", "cabecalho extra"} {
		t.Run(scenario, func(t *testing.T) {
			s, f, r, c := setup(t)
			prepare(t, s, f, c, "2026-10")
			syncAula(t, s, f, c, r)
			r = edit(t, s, r, "2026-10-19")
			rows := f.book.Sheets[1].Rows()
			switch scenario {
			case "manual":
				rows[1].Values[7] = g.Text("edição no Google")
			case "duplicada":
				rows = append(rows, rows[1])
			case "renomeada":
				f.book.Sheets[1].Properties.Title = "Outro mês"
			case "formula":
				rows[1].Values[7] = g.Cell{Value: map[string]any{"formulaValue": "=1+1"}}
			case "removida":
				rows = rows[:1]
			case "outra aba":
				setRows(&f.book.Sheets[0], rows)
			case "cabecalho extra":
				rows[0].Values = append(rows[0].Values, g.Text("extra"))
			}
			setRows(&f.book.Sheets[1], rows)
			writes := f.writes
			if _, err := s.Sync(context.Background(), f.client(), c.Versao, r.ID, r.Versao); err == nil {
				t.Fatal("aceitou conflito")
			}
			if f.writes != writes {
				t.Fatal("sobrescreveu conflito")
			}
		})
	}
}
func TestFalhaDeAcessoPreservaLocalETrocaDeDestino(t *testing.T) {
	s, f, r, c := setup(t)
	ctx := context.Background()
	f.failRead = true
	if _, err := s.Sync(ctx, f.client(), c.Versao, r.ID, r.Versao); err == nil {
		t.Fatal("aceitou falha de leitura")
	}
	local, err := s.Banco.Buscar(ctx, r.ID)
	if err != nil || local != r {
		t.Fatal("perdeu aula")
	}
	f.failRead = false
	prepare(t, s, f, c, "2026-10")
	f.failWrite = true
	if _, err := s.Sync(ctx, f.client(), c.Versao, r.ID, r.Versao); err == nil {
		t.Fatal("aceitou falha de escrita")
	}
	f.failWrite = false
	syncAula(t, s, f, c, r)
	next, err := s.Banco.ConfigurarGoogle(ctx, "https://docs.google.com/spreadsheets/d/other/edit", c.Versao)
	if err != nil {
		t.Fatal(err)
	}
	state, _ := s.Banco.EstadoGoogle(ctx, r)
	if state.Estado == "sincronizado" {
		t.Fatal("herdou estado de outro destino")
	}
	if _, err := s.Sync(ctx, f.client(), c.Versao, r.ID, r.Versao); err == nil {
		t.Fatal("aceitou destino antigo")
	}
	other := &fakeSheets{book: g.Spreadsheet{ID: "other"}}
	prepare(t, s, other, next, "2026-10")
	syncAula(t, s, other, next, r)
	if len(f.book.Sheets[1].Rows()) != 2 {
		t.Fatal("alterou destino anterior")
	}
}
func TestConcorrenciaNaoDuplica(t *testing.T) {
	s, f, r, c := setup(t)
	prepare(t, s, f, c, "2026-10")
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for range 8 {
		wg.Go(func() { _, err := s.Sync(context.Background(), f.client(), c.Versao, r.ID, r.Versao); errs <- err })
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(f.book.Sheets[1].Rows()) != 2 || f.writes != 2 {
		t.Fatal("concorrência duplicou envio")
	}
}
func TestMudancaDeMesComFalhaIntermediaria(t *testing.T) {
	s, f, r, c := setup(t)
	prepare(t, s, f, c, "2026-10")
	syncAula(t, s, f, c, r)
	r = edit(t, s, r, "2026-11-01")
	prepare(t, s, f, c, "2026-11")
	f.failAt = f.writes + 2
	if _, err := s.Sync(context.Background(), f.client(), c.Versao, r.ID, r.Versao); err == nil {
		t.Fatal("esperava falha no novo mês")
	}
	if f.book.Sheets[1].Rows()[1].Values[10].String() != "movida" {
		t.Fatal("mês anterior ainda ativo")
	}
	f.failAt = 0
	syncAula(t, s, f, c, r)
	if len(f.book.Sheets[2].Rows()) != 2 {
		t.Fatal("recuperação duplicou aula")
	}
}
func TestIntentoBloqueiaOutraAulaEAbaOcupada(t *testing.T) {
	s, f, r, c := setup(t)
	ctx := context.Background()
	prepare(t, s, f, c, "2026-10")
	f.failWrite = true
	if _, err := s.Sync(ctx, f.client(), c.Versao, r.ID, r.Versao); err == nil {
		t.Fatal("esperava falha")
	}
	second, err := s.Banco.Inserir(ctx, r.Aula)
	if err != nil {
		t.Fatal(err)
	}
	f.failWrite = false
	if _, err = s.Sync(ctx, f.client(), c.Versao, second.ID, second.Versao); err == nil || !strings.Contains(err.Error(), "sem confirmação") {
		t.Fatal("não bloqueou outra aula", err)
	}
	rows := f.book.Sheets[1].Rows()
	rows = append(rows, g.Row{Values: []g.Cell{g.Text("conteúdo de outra pessoa")}})
	setRows(&f.book.Sheets[1], rows)
	if _, err = s.Sync(ctx, f.client(), c.Versao, r.ID, r.Versao); err == nil {
		t.Fatal("sobrescreveu linha ocupada")
	}
}

func TestInspecaoNaoRevalidaModeloDiferente(t *testing.T) {
	s, f, _, c := setup(t)
	ctx := context.Background()
	prepare(t, s, f, c, "2026-10")
	v, err := s.Banco.VinculoGoogle(ctx, c.Destino.ID, "2026-10")
	if err != nil {
		t.Fatal(err)
	}
	v.Modelo = 2
	if err = s.Banco.SalvarVinculoGoogle(ctx, c.Destino.ID, v); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		i, err := s.Inspect(ctx, f.client(), "session", c.Versao, "2026-10")
		if err != nil || i.Estado != "incompativel" {
			t.Fatalf("modelo diferente revalidado: %+v %v", i, err)
		}
	}
}
