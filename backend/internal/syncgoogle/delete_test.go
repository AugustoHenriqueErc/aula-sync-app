package syncgoogle

import (
	"context"
	"errors"
	"testing"

	g "app-sync-backend/internal/google"
	"app-sync-backend/internal/storage"
)

func cancel(t *testing.T, s *Service, r storage.Registro) storage.Registro {
	t.Helper()
	yes := true
	r, err := s.Banco.Alterar(context.Background(), r.ID, storage.Alteracao{Versao: r.Versao, Cancelada: &yes})
	if err != nil {
		t.Fatal(err)
	}
	return r
}
func TestReenvioConfirmadoNaoChamaGoogleNemContaTentativa(t *testing.T) {
	s, f, r, c := setup(t)
	ctx := context.Background()
	prepare(t, s, f, c, "2026-10")
	syncAula(t, s, f, c, r)
	before, _ := s.Banco.EstadoGoogle(ctx, r)
	reads, writes := f.reads, f.writes
	result, err := s.Sync(ctx, f.client(), c.Versao, r.ID, r.Versao)
	if err != nil || !result.JaSincronizada || result.Mensagem == "" || result.Tentativas != before.Tentativas || f.reads != reads || f.writes != writes {
		t.Fatalf("reenvio: %+v %v", result, err)
	}
	r = edit(t, s, r, "2026-10-19")
	syncAula(t, s, f, c, r)
	if f.writes != writes+1 {
		t.Fatal("edição deixou de atualizar a linha")
	}
}
func TestExclusaoLocalDefinitivaReservaID(t *testing.T) {
	s, f, r, c := setup(t)
	ctx := context.Background()
	if err := s.Delete(ctx, f.client(), c.Versao, r.ID, r.Versao, false); err == nil {
		t.Fatal("excluiu aula ativa")
	}
	r = cancel(t, s, r)
	if err := s.Delete(ctx, f.client(), c.Versao, r.ID, r.Versao-1, false); err == nil {
		t.Fatal("aceitou versão antiga")
	}
	if err := s.Delete(ctx, f.client(), c.Versao-1, r.ID, r.Versao, false); err == nil {
		t.Fatal("aceitou destino antigo")
	}
	reads := f.reads
	if err := s.Delete(ctx, f.client(), c.Versao, r.ID, r.Versao, false); err != nil {
		t.Fatal(err)
	}
	if f.reads != reads {
		t.Fatal("exclusão local dependeu do Google")
	}
	if _, err := s.Banco.Buscar(ctx, r.ID); !errors.Is(err, storage.ErrNaoEncontrada) {
		t.Fatal("aula não foi removida", err)
	}
	next, err := s.Banco.Inserir(ctx, r.Aula)
	if err != nil || next.ID <= r.ID {
		t.Fatal("reutilizou ID excluído", next, err)
	}
}
func TestExclusaoRemotaPreservaOutrasLinhasERetomaAposRespostaPerdida(t *testing.T) {
	s, f, r, c := setup(t)
	ctx := context.Background()
	prepare(t, s, f, c, "2026-10")
	syncAula(t, s, f, c, r)
	other, err := s.Banco.Inserir(ctx, r.Aula)
	if err != nil {
		t.Fatal(err)
	}
	syncAula(t, s, f, c, other)
	r = cancel(t, s, r)
	if err = s.Delete(ctx, f.client(), c.Versao, r.ID, r.Versao, true); err == nil {
		t.Fatal("excluiu antes de sincronizar cancelamento")
	}
	syncAula(t, s, f, c, r)
	f.loseResponse = true
	if err = s.Delete(ctx, f.client(), c.Versao, r.ID, r.Versao, true); err == nil {
		t.Fatal("esperava resposta perdida")
	}
	if _, err = s.Banco.Buscar(ctx, r.ID); err != nil {
		t.Fatal("apagou local antes de confirmar")
	}
	if _, err = s.Sync(ctx, f.client(), c.Versao, other.ID, other.Versao); !errors.Is(err, storage.ErrExclusaoPendente) {
		t.Fatal("permitiu envio durante exclusão", err)
	}
	if _, err = s.Banco.ConfigurarGoogle(ctx, "https://docs.google.com/spreadsheets/d/other/edit", c.Versao); !errors.Is(err, storage.ErrExclusaoPendente) {
		t.Fatal("permitiu troca de destino", err)
	}
	if _, err = s.Banco.Alterar(ctx, r.ID, storage.Alteracao{Versao: r.Versao, Cancelada: new(bool)}); !errors.Is(err, storage.ErrExclusaoPendente) {
		t.Fatal("permitiu reativação", err)
	}
	if err = s.Delete(ctx, f.client(), c.Versao, r.ID, r.Versao, false); err == nil {
		t.Fatal("descartou exclusão remota pendente")
	}
	estado, readErr := s.Banco.EstadoGoogle(ctx, r)
	if readErr != nil || !estado.ExclusaoPendente || !estado.ExclusaoGoogle {
		t.Fatalf("exclusão pendente não aparece no histórico: %+v %v", estado, readErr)
	}
	writes := f.writes
	s = New(s.Banco)
	if err = s.Delete(ctx, f.client(), c.Versao, r.ID, r.Versao, true); err != nil {
		t.Fatal(err)
	}
	if f.writes != writes || len(f.book.Sheets[1].Rows()) != 3 {
		t.Fatal("excluiu outra linha ao repetir")
	}
	other = edit(t, s, other, "2026-10-19")
	syncAula(t, s, f, c, other)
	if f.book.Sheets[1].Rows()[2].Values[2].String() != other.Aula.Data {
		t.Fatal("perdeu vínculo após deslocamento de linha")
	}
}
func TestExclusaoLocalPreservaGoogleEExclusaoRemotaRemoveMesesAnteriores(t *testing.T) {
	for _, remote := range []bool{false, true} {
		t.Run(map[bool]string{false: "local", true: "google"}[remote], func(t *testing.T) {
			s, f, r, c := setup(t)
			ctx := context.Background()
			prepare(t, s, f, c, "2026-10")
			syncAula(t, s, f, c, r)
			r = edit(t, s, r, "2026-11-01")
			prepare(t, s, f, c, "2026-11")
			syncAula(t, s, f, c, r)
			r = cancel(t, s, r)
			syncAula(t, s, f, c, r)
			if err := s.Delete(ctx, f.client(), c.Versao, r.ID, r.Versao, remote); err != nil {
				t.Fatal(err)
			}
			want := 2
			if remote {
				want = 1
			}
			if len(f.book.Sheets[1].Rows()) != want || len(f.book.Sheets[2].Rows()) != want {
				t.Fatal("resultado incorreto nos meses")
			}
			next, err := s.Banco.Inserir(ctx, r.Aula)
			if err != nil || next.ID <= r.ID {
				t.Fatal("ID reutilizado")
			}
			syncAula(t, s, f, c, next)
			if len(f.book.Sheets[2].Rows()) != 3 {
				t.Fatal("novo cadastro sobrescreveu a aula excluída")
			}
		})
	}
}
func TestExclusaoBloqueiaConflitoERecuperaFalhaDePermissao(t *testing.T) {
	for _, scenario := range []string{"manual", "duplicada", "renomeada", "acesso", "escrita"} {
		t.Run(scenario, func(t *testing.T) {
			s, f, r, c := setup(t)
			ctx := context.Background()
			prepare(t, s, f, c, "2026-10")
			syncAula(t, s, f, c, r)
			r = cancel(t, s, r)
			syncAula(t, s, f, c, r)
			rows := f.book.Sheets[1].Rows()
			switch scenario {
			case "manual":
				rows[1].Values[7].Value = map[string]any{"stringValue": "manual"}
				setRows(&f.book.Sheets[1], rows)
			case "duplicada":
				rows = append(rows, rows[1])
				setRows(&f.book.Sheets[1], rows)
			case "renomeada":
				f.book.Sheets[1].Properties.Title = "Outra"
			case "acesso":
				f.failRead = true
			case "escrita":
				f.failWrite = true
			}
			if err := s.Delete(ctx, f.client(), c.Versao, r.ID, r.Versao, true); err == nil {
				t.Fatal("aceitou conflito/falha")
			}
			if _, err := s.Banco.Buscar(ctx, r.ID); err != nil {
				t.Fatal("perdeu aula local")
			}
			if scenario == "escrita" {
				f.failWrite = false
				if err := s.Delete(ctx, f.client(), c.Versao, r.ID, r.Versao, true); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestLimpezaTardiaNaoApagaNovoCadastro(t *testing.T) {
	s, f, r, c := setup(t)
	ctx := context.Background()
	prepare(t, s, f, c, "2026-10")
	syncAula(t, s, f, c, r)
	r = cancel(t, s, r)
	syncAula(t, s, f, c, r)
	if err := s.Delete(ctx, f.client(), c.Versao, r.ID, r.Versao, true); err != nil {
		t.Fatal(err)
	}
	next, err := s.Banco.Inserir(ctx, r.Aula)
	if err != nil {
		t.Fatal(err)
	}
	syncAula(t, s, f, c, next)
	// Simula conclusão tardia de uma chamada de limpeza já recebida pelo Google.
	if err = f.client().Batch(c.Destino.PlanilhaID, []any{g.ClearRecord(f.book.Sheets[1].Properties.ID, 1)}); err != nil {
		t.Fatal(err)
	}
	install, err := s.Banco.InstalacaoGoogle(ctx)
	if err != nil {
		t.Fatal(err)
	}
	index, _, err := locate(f.book.Sheets[1], g.Key(install, next.ID))
	if err != nil || index <= 1 {
		t.Fatal("novo cadastro reutilizou posição apagada", index, err)
	}
}
