package storage

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

func TestExcluirRemoveDadosETrilhaSemReutilizarID(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "test.db")
	b := abrirTeste(t, path)
	r, err := b.Inserir(ctx, exemplo())
	if err != nil {
		t.Fatal(err)
	}
	yes := true
	r, err = b.Alterar(ctx, r.ID, Alteracao{Versao: r.Versao, Cancelada: &yes})
	if err != nil {
		t.Fatal(err)
	}
	if err = b.IniciarExclusao(ctx, r.ID, nil, false); err != nil {
		t.Fatal(err)
	}
	if err = b.FinalizarExclusao(ctx, r.ID); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"aulas", "aula_alteracoes", "google_sincronizacoes", "aula_exclusoes"} {
		var n int
		if err = b.db.QueryRow("SELECT count(*) FROM " + table).Scan(&n); err != nil || n != 0 {
			t.Fatal(table, n, err)
		}
	}
	b.Close()
	b = abrirTeste(t, path)
	next, err := b.Inserir(ctx, exemplo())
	if err != nil || next.ID <= r.ID {
		t.Fatal("ID após reinício", next, err)
	}
	if err = b.Verificar(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestMigracaoSeisIniciaContadorComMaiorIDExistente(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "versao5.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(primeiraMigracao); err != nil {
		t.Fatal(err)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = migrarTurmas(ctx, tx); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(terceiraMigracao + quartaMigracao + quintaMigracao); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(`INSERT INTO turmas(id,nome,nome_normalizado) VALUES(1,'Teste','teste'); INSERT INTO aulas(id,data,inicio,fim,turma,conteudo,observacoes,intervalo_minutos,duracao_minutos,turma_id) VALUES(50,'2026-10-01','08:00','09:00','Teste','Exemplo','',0,60,1)`); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	db.Close()
	b := abrirTeste(t, path)
	if _, err = b.Buscar(ctx, 50); err != nil {
		t.Fatal("migração perdeu aula", err)
	}
	if err = b.IniciarExclusao(ctx, 50, nil, false); err != nil {
		t.Fatal(err)
	}
	if err = b.FinalizarExclusao(ctx, 50); err != nil {
		t.Fatal(err)
	}
	next, err := b.Inserir(ctx, exemplo())
	if err != nil || next.ID != 51 {
		t.Fatal("contador não preservou maior ID da versão 5", next, err)
	}
}
