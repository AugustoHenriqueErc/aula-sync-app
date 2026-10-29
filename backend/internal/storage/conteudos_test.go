package storage

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"reflect"
	"sync"
	"testing"

	"app-sync-backend/internal/aula"
)

func TestConteudosPersistenciaNormalizacaoEConcorrencia(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "conteudos.db")
	b := abrirTeste(t, path)
	vazios, err := b.ListarConteudos(ctx)
	if err != nil || vazios == nil || len(vazios) != 0 {
		t.Fatalf("lista vazia: %v %v", vazios, err)
	}
	for _, nome := range []string{"", " \t\n\u00a0\u2003"} {
		_, err := b.CriarConteudo(ctx, nome)
		var validacao *aula.ErroValidacao
		if !errors.As(err, &validacao) || validacao.Campo != "Nome" {
			t.Fatalf("nome vazio: %v", err)
		}
	}
	primeiro, err := b.CriarConteudo(ctx, "  Zonas\n  climáticas  ")
	if err != nil || primeiro.ID <= 0 || primeiro.Nome != "Zonas\n  climáticas" {
		t.Fatalf("multilinha: %+v %v", primeiro, err)
	}
	if _, err := b.CriarConteudo(ctx, "ZONAS CLIMÁTICAS"); !errors.Is(err, ErrConteudoDuplicado) {
		t.Fatalf("duplicidade multilinha: %v", err)
	}
	var wg sync.WaitGroup
	resultados := make(chan error, 3)
	for _, nome := range []string{" Frações ", "FRAÇÕES", "frações"} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := b.CriarConteudo(ctx, nome)
			resultados <- err
		}()
	}
	wg.Wait()
	close(resultados)
	sucessos, duplicatas := 0, 0
	for err := range resultados {
		switch {
		case err == nil:
			sucessos++
		case errors.Is(err, ErrConteudoDuplicado):
			duplicatas++
		default:
			t.Fatal(err)
		}
	}
	if sucessos != 1 || duplicatas != 2 {
		t.Fatalf("sucessos=%d duplicatas=%d", sucessos, duplicatas)
	}
	if _, err := b.CriarConteudo(ctx, "fracoes"); err != nil {
		t.Fatal("acentos devem permanecer distintos", err)
	}
	antes, err := b.ListarConteudos(ctx)
	if err != nil || len(antes) != 3 || antes[0].Nome != "fracoes" || normalizarNome(antes[1].Nome) != "frações" || antes[2] != primeiro {
		t.Fatalf("ordem: %+v %v", antes, err)
	}
	if err := b.Close(); err != nil {
		t.Fatal(err)
	}
	b = abrirTeste(t, path)
	depois, err := b.ListarConteudos(ctx)
	if err != nil || !reflect.DeepEqual(antes, depois) {
		t.Fatalf("persistência: %+v %v", depois, err)
	}
}

// Constrói o esquema anterior sem aplicar a migração de conteúdos.
func bancoV6Conteudos(t *testing.T) (*Banco, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "v6.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	db.SetMaxOpenConns(1)
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(primeiraMigracao); err != nil {
		t.Fatal(err)
	}
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if err := migrarTurmas(context.Background(), tx); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(terceiraMigracao + quartaMigracao + quintaMigracao + sextaMigracao); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	return &Banco{db: db}, path
}

func TestMigracaoConteudosPreservaAulasAuditoriaEGoogle(t *testing.T) {
	ctx := context.Background()
	b, path := bancoV6Conteudos(t)
	for _, nome := range []string{" Leitura\n  e revisão ", "LEITURA E REVISÃO", "Somente cancelada"} {
		a := exemplo()
		a.Conteudo = nome
		if _, err := b.Inserir(ctx, a); err != nil {
			t.Fatal(err)
		}
	}
	cancelada := true
	if _, err := b.Alterar(ctx, 3, Alteracao{Versao: 1, Cancelada: &cancelada}); err != nil {
		t.Fatal(err)
	}
	config, err := b.ConfigurarGoogle(ctx, "https://docs.google.com/spreadsheets/d/planilha123/edit", 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := b.SalvarVinculoGoogle(ctx, config.Destino.ID, VinculoGoogle{Periodo: "2026-10", Estado: "nao_verificada"}); err != nil {
		t.Fatal(err)
	}
	if err := b.IniciarEnvioGoogle(ctx, config.Destino.ID, 1, "2026-10", "snapshot remoto", 2); err != nil {
		t.Fatal(err)
	}
	if err := b.ConfirmarEnvioGoogle(ctx, config.Destino.ID, 1, "2026-10", 1, "snapshot remoto"); err != nil {
		t.Fatal(err)
	}
	antes, err := b.Listar(ctx)
	if err != nil {
		t.Fatal(err)
	}
	envios, err := b.EnviosGoogle(ctx, config.Destino.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	var auditoria string
	if err := b.db.QueryRow("SELECT estado_anterior FROM aula_alteracoes WHERE aula_id=3").Scan(&auditoria); err != nil {
		t.Fatal(err)
	}
	b.Close()
	b = abrirTeste(t, path)
	conteudos, err := b.ListarConteudos(ctx)
	esperado := []Conteudo{{ID: 1, Nome: "Leitura\n  e revisão"}, {ID: 2, Nome: "Somente cancelada"}}
	if err != nil || !reflect.DeepEqual(conteudos, esperado) {
		t.Fatalf("catálogo migrado: %+v %v", conteudos, err)
	}
	depois, err := b.Listar(ctx)
	if err != nil || !reflect.DeepEqual(antes, depois) {
		t.Fatalf("aulas alteradas: %+v %v", depois, err)
	}
	var auditoriaDepois string
	if err := b.db.QueryRow("SELECT estado_anterior FROM aula_alteracoes WHERE aula_id=3").Scan(&auditoriaDepois); err != nil || auditoria != auditoriaDepois {
		t.Fatal("auditoria alterada", err)
	}
	enviosDepois, err := b.EnviosGoogle(ctx, config.Destino.ID, 1)
	if err != nil || !reflect.DeepEqual(envios, enviosDepois) {
		t.Fatal("sincronização alterada", err)
	}
	configDepois, err := b.ConfiguracaoGoogle(ctx)
	if err != nil || !reflect.DeepEqual(config, configDepois) {
		t.Fatal("configuração alterada", err)
	}
	if err := b.Verificar(ctx); err != nil {
		t.Fatal(err)
	}
	b.Close()
	b = abrirTeste(t, path)
	reabertos, err := b.ListarConteudos(ctx)
	if err != nil || !reflect.DeepEqual(conteudos, reabertos) {
		t.Fatal("migração reaplicada", reabertos, err)
	}
}

func TestMigracaoConteudosRollback(t *testing.T) {
	b, _ := bancoV6Conteudos(t)
	// Falha depois do CREATE TABLE: a consulta não consegue ler a coluna esperada.
	if _, err := b.db.Exec("ALTER TABLE aulas RENAME COLUMN conteudo TO conteudo_indisponivel"); err != nil {
		t.Fatal(err)
	}
	if err := b.migrar(context.Background()); err == nil {
		t.Fatal("migração deveria falhar")
	}
	var versao, tabelas int
	if err := b.db.QueryRow("PRAGMA user_version").Scan(&versao); err != nil {
		t.Fatal(err)
	}
	if err := b.db.QueryRow("SELECT count(*) FROM sqlite_master WHERE name='conteudos'").Scan(&tabelas); err != nil {
		t.Fatal(err)
	}
	if versao != 6 || tabelas != 0 {
		t.Fatalf("rollback incompleto: versão=%d tabelas=%d", versao, tabelas)
	}
	if _, err := b.db.Exec("ALTER TABLE aulas RENAME COLUMN conteudo_indisponivel TO conteudo"); err != nil {
		t.Fatal(err)
	}
	if err := b.migrar(context.Background()); err != nil {
		t.Fatal("retomada da migração", err)
	}
}
