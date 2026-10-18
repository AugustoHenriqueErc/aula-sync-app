package storage

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"sync"
	"testing"
)

func TestMigrarNomesPreservaAulas(t *testing.T) {
	ctx := context.Background()
	caminho := filepath.Join(t.TempDir(), "antigo.db")
	db, err := sql.Open("sqlite", caminho)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(primeiraMigracao); err != nil {
		t.Fatal(err)
	}
	for _, nome := range []string{" Augusto ", "augusto", "AUGUSTO", "Maria   Luzia"} {
		if _, err = db.Exec(`INSERT INTO aulas(data,inicio,fim,turma,conteudo,intervalo_minutos,duracao_minutos) VALUES ('2026-10-01','08:00','09:00',?,'Conteúdo original',0,60)`, nome); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = db.Exec("PRAGMA user_version=1"); err != nil {
		t.Fatal(err)
	}
	db.Close()
	b := abrirTeste(t, caminho)
	ts, err := b.ListarTurmas(ctx)
	if err != nil || len(ts) != 2 {
		t.Fatalf("turmas: %v %v", ts, err)
	}
	as, err := b.Listar(ctx)
	if err != nil || len(as) != 4 {
		t.Fatalf("aulas: %v %v", as, err)
	}
	for i, r := range as {
		if r.ID != int64(4-i) || r.DuracaoMinutos != 60 || r.Aula.Conteudo != "Conteúdo original" {
			t.Fatalf("aula alterada: %+v", r)
		}
	}
	if as[1].TurmaID != as[2].TurmaID || as[2].TurmaID != as[3].TurmaID || as[3].Aula.Turma != " Augusto " {
		t.Fatal("agrupamento ou preservação incorreto")
	}
	if err := b.migrar(ctx); err != nil {
		t.Fatal(err)
	}
	var problemas int
	if err := b.db.QueryRow("SELECT count(*) FROM pragma_foreign_key_check").Scan(&problemas); err != nil || problemas != 0 {
		t.Fatalf("referências inválidas: %d %v", problemas, err)
	}
	if _, err := b.db.Exec("UPDATE aulas SET turma_id=999"); err == nil {
		t.Fatal("FK inexistente aceita")
	}
}

func TestMigracao2Rollback(t *testing.T) {
	caminho := filepath.Join(t.TempDir(), "rollback.db")
	db, err := sql.Open("sqlite", caminho)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec(primeiraMigracao + " PRAGMA user_version=1; CREATE TABLE turmas(id INTEGER);"); err != nil {
		t.Fatal(err)
	}
	if b, err := Abrir(context.Background(), caminho); err == nil {
		b.Close()
		t.Fatal("conflito deveria falhar")
	}
	var v int
	if err := db.QueryRow("PRAGMA user_version").Scan(&v); err != nil || v != 1 {
		t.Fatalf("versão alterada: %d %v", v, err)
	}
	var n int
	if err := db.QueryRow("SELECT count(*) FROM aulas").Scan(&n); err != nil {
		t.Fatal(err)
	}
}

func TestTurmasNormalizacaoEConcorrencia(t *testing.T) {
	b := abrirTeste(t, filepath.Join(t.TempDir(), "turmas.db"))
	ctx := context.Background()
	var wg sync.WaitGroup
	resultados := make(chan error, 3)
	for _, nome := range []string{"  ÁUGUSTO  Silva ", "áugusto silva", "Áugusto\tSilva"} {
		wg.Add(1)
		go func() { defer wg.Done(); _, err := b.CriarTurma(ctx, nome); resultados <- err }()
	}
	wg.Wait()
	close(resultados)
	sucessos, duplicatas := 0, 0
	for err := range resultados {
		if err == nil {
			sucessos++
		} else if errors.Is(err, ErrTurmaDuplicada) {
			duplicatas++
		} else {
			t.Fatal(err)
		}
	}
	if sucessos != 1 || duplicatas != 2 {
		t.Fatalf("sucessos=%d duplicatas=%d", sucessos, duplicatas)
	}
	if _, err := b.CriarTurma(ctx, "\u00a0\t"); err == nil {
		t.Fatal("nome vazio aceito")
	}
	if _, err := b.CriarTurma(ctx, "Augusto Silva"); err != nil {
		t.Fatal("acentos devem ser distintos", err)
	}
}

func TestHistoricoFiltrosETotais(t *testing.T) {
	b := abrirTeste(t, filepath.Join(t.TempDir(), "historico.db"))
	ctx := context.Background()
	var turmaID int64
	for _, tt := range []struct{ data, fim, turma string }{{"2026-09-30", "09:00", "Augusto"}, {"2026-10-01", "09:30", "Augusto"}, {"2026-10-31", "08:45", "Maria"}} {
		a := exemplo()
		a.Data = tt.data
		a.Fim = tt.fim
		a.Turma = tt.turma
		a.IntervaloMinutos = 0
		r, err := b.Inserir(ctx, a)
		if err != nil {
			t.Fatal(err)
		}
		if tt.turma == "Augusto" {
			turmaID = r.TurmaID
		}
	}
	for _, tt := range []struct {
		nome   string
		filtro Filtro
		n      int
		total  int64
	}{
		{"todos", Filtro{}, 3, 195}, {"extremos", Filtro{DataInicial: "2026-09-30", DataFinal: "2026-10-31"}, 3, 195},
		{"mes", Filtro{DataInicial: "2026-10-01", DataFinal: "2026-10-31"}, 2, 135},
		{"dia", Filtro{DataInicial: "2026-10-01", DataFinal: "2026-10-01"}, 1, 90},
		{"turma", Filtro{TurmaID: turmaID}, 2, 150}, {"combinado", Filtro{DataInicial: "2026-10-01", TurmaID: turmaID}, 1, 90},
		{"final", Filtro{DataFinal: "2026-09-30"}, 1, 60}, {"vazio", Filtro{DataInicial: "2027-01-01"}, 0, 0},
	} {
		t.Run(tt.nome, func(t *testing.T) {
			h, err := b.Historico(ctx, tt.filtro)
			if err != nil || len(h.Aulas) != tt.n || h.TotalRegistros != tt.n || h.DuracaoTotalMinutos != tt.total || h.Aulas == nil {
				t.Fatalf("%+v %v", h, err)
			}
		})
	}
}
func TestHistoricoPaginas(t *testing.T) {
	b := abrirTeste(t, filepath.Join(t.TempDir(), "paginas.db"))
	ctx := context.Background()
	for i := 0; i < 21; i++ {
		if _, err := b.Inserir(ctx, exemplo()); err != nil {
			t.Fatal(err)
		}
	}
	for pagina, n := range map[int]int{1: 20, 2: 1, 3: 0} {
		h, err := b.Historico(ctx, Filtro{Pagina: pagina})
		if err != nil || len(h.Aulas) != n || h.TotalRegistros != 21 || h.DuracaoTotalMinutos != 2205 {
			t.Fatalf("página %d: %+v %v", pagina, h, err)
		}
		if n > 0 && h.Aulas[0].ID != int64(21-(pagina-1)*20) {
			t.Fatal("ordem incorreta")
		}
	}
}
