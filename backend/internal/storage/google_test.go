package storage

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"reflect"
	"testing"
)

func TestConfigGooglePersistenciaETroca(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "aulas.db")
	b := abrirTeste(t, path)
	c, err := b.ConfiguracaoGoogle(ctx)
	if err != nil || c.Versao != 0 || c.Destino != nil {
		t.Fatalf("inicial: %+v %v", c, err)
	}
	r, err := b.Inserir(ctx, exemplo())
	if err != nil {
		t.Fatal(err)
	}
	a, err := b.ConfigurarGoogle(ctx, "https://docs.google.com/spreadsheets/d/teste-A/edit?usp=sharing#gid=555", 0)
	if err != nil || a.Versao != 1 {
		t.Fatalf("salvar: %+v %v", a, err)
	}
	igual, err := b.ConfigurarGoogle(ctx, "https://docs.google.com/spreadsheets/d/teste-A/edit", 1)
	if err != nil || !reflect.DeepEqual(a, igual) {
		t.Fatal("link equivalente alterou a configuração", err)
	}
	if _, err := b.ConfigurarGoogle(ctx, "https://docs.google.com/spreadsheets/d/teste-B/edit", 0); !errors.Is(err, ErrConfigConflito) {
		t.Fatalf("conflito: %v", err)
	}
	if _, err := b.ConfigurarGoogle(ctx, "https://evil.test/", 1); err == nil {
		t.Fatal("link inválido aceito")
	}
	outra, err := b.ConfigurarGoogle(ctx, "https://docs.google.com/spreadsheets/d/teste-B/edit", 1)
	if err != nil || outra.Destino.ID == a.Destino.ID {
		t.Fatal("troca não criou destino distinto", err)
	}
	volta, err := b.ConfigurarGoogle(ctx, "https://docs.google.com/spreadsheets/d/teste-A/edit", 2)
	if err != nil || volta.Destino.ID != a.Destino.ID || volta.Versao != 3 {
		t.Fatalf("retorno: %+v %v", volta, err)
	}
	var n int
	for _, tabela := range []string{"google_abas", "google_sincronizacoes"} {
		if err := b.db.QueryRow("SELECT count(*) FROM " + tabela).Scan(&n); err != nil || n != 0 {
			t.Fatal("configuração criou estado remoto", tabela, err)
		}
	}
	var instalacao string
	if err := b.db.QueryRow("SELECT instalacao_id FROM google_configuracao").Scan(&instalacao); err != nil || len(instalacao) != 32 {
		t.Fatal("identidade inválida", err)
	}
	b.Close()
	b2 := abrirTeste(t, path)
	persistida, err := b2.ConfiguracaoGoogle(ctx)
	if err != nil || !reflect.DeepEqual(volta, persistida) {
		t.Fatalf("persistência: %+v %v", persistida, err)
	}
	aula, err := b2.Buscar(ctx, r.ID)
	if err != nil || aula != r {
		t.Fatal("troca alterou aula", err)
	}
	var id string
	if err := b2.db.QueryRow("SELECT instalacao_id FROM google_configuracao").Scan(&id); err != nil || id != instalacao {
		t.Fatal("identidade mudou ao reabrir", err)
	}
}
func TestPlanejamentoGoogleMeses(t *testing.T) {
	ctx := context.Background()
	b := abrirTeste(t, filepath.Join(t.TempDir(), "plano.db"))
	for _, data := range []string{"2026-10-31", "2026-11-01", "2027-10-01"} {
		a := exemplo()
		a.Data = data
		if _, err := b.Inserir(ctx, a); err != nil {
			t.Fatal(err)
		}
	}
	cancelar := true
	if _, err := b.Alterar(ctx, 1, Alteracao{Versao: 1, Cancelada: &cancelar}); err != nil {
		t.Fatal(err)
	}
	p, err := b.PlanejamentoGoogle(ctx)
	if err != nil || len(p.Meses) != 3 || p.Configuracao.Destino != nil {
		t.Fatalf("plano: %+v %v", p, err)
	}
	if p.Meses[0].Aba != "Registros aulas - 10/2027" || p.Meses[1].Aba != "Registros aulas - 11/2026" || p.Meses[2].Canceladas != 1 || p.Meses[2].Minutos != 0 {
		t.Fatalf("meses: %+v", p.Meses)
	}
}
func TestMigracao4ReverteFalhaEPreservaDados(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "v3.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(primeiraMigracao); err != nil {
		t.Fatal(err)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := migrarTurmas(ctx, tx); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(terceiraMigracao); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO turmas VALUES (1,'Teste','teste'); INSERT INTO aulas(turma_id,data,inicio,fim,turma,conteudo,intervalo_minutos,duracao_minutos) VALUES (1,'2026-10-18','08:00','09:00','Teste','Preservar',0,60); CREATE TABLE google_abas (id INTEGER);`); err != nil {
		t.Fatal(err)
	}
	if novo, err := Abrir(ctx, path); err == nil {
		novo.Close()
		t.Fatal("conflito deveria falhar")
	}
	var v int
	if err := db.QueryRow("PRAGMA user_version").Scan(&v); err != nil || v != 3 {
		t.Fatal("rollback mudou versão", v, err)
	}
	var n int
	if err := db.QueryRow("SELECT count(*) FROM sqlite_master WHERE name='google_destinos'").Scan(&n); err != nil || n != 0 {
		t.Fatal("migração parcial persistiu", err)
	}
	if _, err := db.Exec("DROP TABLE google_abas"); err != nil {
		t.Fatal(err)
	}
	db.Close()
	b := abrirTeste(t, path)
	r, err := b.Buscar(ctx, 1)
	if err != nil || r.Aula.Conteudo != "Preservar" || r.DuracaoMinutos != 60 {
		t.Fatal("dados alterados", err)
	}
	if err := b.migrar(ctx); err != nil {
		t.Fatal(err)
	}
	// A identidade da instalação e a configuração acompanham o backup.
	config, err := b.ConfigurarGoogle(ctx, "https://docs.google.com/spreadsheets/d/teste/edit", 0)
	if err != nil {
		t.Fatal(err)
	}
	destino := filepath.Join(t.TempDir(), "backup.db")
	if err := b.Backup(ctx, destino); err != nil {
		t.Fatal(err)
	}
	copia := abrirTeste(t, destino)
	got, err := copia.ConfiguracaoGoogle(ctx)
	if err != nil || !reflect.DeepEqual(config, got) {
		t.Fatal("backup perdeu configuração", err)
	}
}

func TestMigracoesCincoESeisPreservamConfiguracaoEIdentidade(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "quatro.db")
	// Constrói uma instalação real de versão 4, antes da nova migração.
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
	if _, err = tx.Exec(terceiraMigracao + quartaMigracao); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec("INSERT INTO google_destinos(id,planilha_id,criado_em) VALUES(1,'test','agora'); UPDATE google_configuracao SET destino_id=1,versao=7,instalacao_id='identidade-preservada'"); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	db.Close()
	b := abrirTeste(t, path)
	c, err := b.ConfiguracaoGoogle(ctx)
	if err != nil || c.Versao != 7 || c.Destino.PlanilhaID != "test" {
		t.Fatal(c, err)
	}
	id, err := b.InstalacaoGoogle(ctx)
	if err != nil || id != "identidade-preservada" {
		t.Fatal(id, err)
	}
	var v int
	if err = b.db.QueryRow("PRAGMA user_version").Scan(&v); err != nil || v != 7 {
		t.Fatal(v, err)
	}
}
