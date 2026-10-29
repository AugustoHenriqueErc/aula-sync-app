package storage

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"app-sync-backend/internal/aula"
)

func exemplo() aula.Aula {
	return aula.Aula{Data: "2026-10-13", Inicio: "08:00", Fim: "10:00",
		Turma: "Turma fictícia", Conteudo: "Leitura", Observacoes: "Observação de teste", IntervaloMinutos: 15}
}

func abrirTeste(t *testing.T, caminho string) *Banco {
	t.Helper()
	banco, err := Abrir(context.Background(), caminho)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { banco.Close() })
	return banco
}

func TestMigracaoEPersistencia(t *testing.T) {
	ctx := context.Background()
	caminho := filepath.Join(t.TempDir(), "aulas com espaço?#.db")
	banco := abrirTeste(t, caminho)
	if _, err := os.Stat(caminho); err != nil {
		t.Fatal(err)
	}
	var versao int
	if err := banco.db.QueryRow("PRAGMA user_version").Scan(&versao); err != nil || versao != 7 {
		t.Fatalf("versão = %d, erro = %v; esperado 7", versao, err)
	}
	registros, err := banco.Listar(ctx)
	if err != nil || registros == nil || len(registros) != 0 {
		t.Fatalf("banco vazio: %v, %v", registros, err)
	}
	a := exemplo()
	// Aspas e texto parecido com SQL devem ser persistidos como dados.
	a.Conteudo = "Leitura d'água; DROP TABLE aulas; --"
	salvo, err := banco.Inserir(ctx, a)
	if err != nil {
		t.Fatal(err)
	}
	if salvo.ID <= 0 || salvo.Aula != a || salvo.DuracaoMinutos != 105 {
		t.Fatalf("registro inesperado: %+v", salvo)
	}
	if err := banco.Close(); err != nil {
		t.Fatal(err)
	}
	// Reabrir aplica a verificação de migração novamente, sem perder registros.
	reaberto := abrirTeste(t, caminho)
	obtido, err := reaberto.Buscar(ctx, salvo.ID)
	if err != nil || obtido != salvo {
		t.Fatalf("registro após reabrir = %+v, %v; esperado %+v", obtido, err, salvo)
	}
	if err := reaberto.migrar(ctx); err != nil {
		t.Fatal(err)
	}
	registros, err = reaberto.Listar(ctx)
	if err != nil || !reflect.DeepEqual(registros, []Registro{salvo}) {
		t.Fatalf("reaplicação alterou dados: %v, %v", registros, err)
	}
}

func TestListagemOrdenadaEBuscaAusente(t *testing.T) {
	ctx := context.Background()
	banco := abrirTeste(t, filepath.Join(t.TempDir(), "aulas.db"))
	for _, id := range []int64{1, 0, -1} {
		if _, err := banco.Buscar(ctx, id); !errors.Is(err, ErrNaoEncontrada) {
			t.Fatalf("ID %d: esperado ErrNaoEncontrada, obtido %v", id, err)
		}
	}
	var salvos []Registro
	for _, horario := range []struct{ data, inicio, fim string }{
		{"2026-10-12", "08:00", "10:00"},
		{"2026-10-13", "08:00", "10:00"},
		{"2026-10-13", "10:00", "12:00"},
		{"2026-10-13", "10:00", "12:00"},
	} {
		a := exemplo()
		a.Data, a.Inicio, a.Fim = horario.data, horario.inicio, horario.fim
		salvo, err := banco.Inserir(ctx, a)
		if err != nil {
			t.Fatal(err)
		}
		salvos = append(salvos, salvo)
	}
	got, err := banco.Listar(ctx)
	want := []Registro{salvos[3], salvos[2], salvos[1], salvos[0]}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("ordenação = %+v, %v; esperado %+v", got, err, want)
	}
}

func TestRepositorioRejeitaAulaInvalida(t *testing.T) {
	ctx := context.Background()
	banco := abrirTeste(t, filepath.Join(t.TempDir(), "aulas.db"))
	for _, tt := range []struct {
		campo   string
		alterar func(*aula.Aula)
	}{
		{"Data", func(a *aula.Aula) { a.Data = "2026-02-29" }},
		{"Inicio", func(a *aula.Aula) { a.Inicio = "8:00" }},
		{"Fim", func(a *aula.Aula) { a.Fim = "07:00" }},
		{"Turma", func(a *aula.Aula) { a.Turma = "\u00a0\u2003" }},
		{"Conteudo", func(a *aula.Aula) { a.Conteudo = " \t\n" }},
		{"IntervaloMinutos", func(a *aula.Aula) { a.IntervaloMinutos = 120 }},
	} {
		t.Run(tt.campo, func(t *testing.T) {
			a := exemplo()
			tt.alterar(&a)
			registro, err := banco.Inserir(ctx, a)
			var validacao *aula.ErroValidacao
			if !errors.As(err, &validacao) || validacao.Campo != tt.campo {
				t.Fatalf("erro inesperado: %v", err)
			}
			if registro != (Registro{}) {
				t.Fatalf("retornou registro para entrada inválida: %+v", registro)
			}
		})
	}
	registros, err := banco.Listar(ctx)
	if err != nil || len(registros) != 0 {
		t.Fatalf("entrada inválida foi persistida: %v, %v", registros, err)
	}
}

func TestRestricoesSQL(t *testing.T) {
	banco := abrirTeste(t, filepath.Join(t.TempDir(), "aulas.db"))
	turma, err := banco.CriarTurma(context.Background(), "Turma")
	if err != nil {
		t.Fatal(err)
	}
	const insert = `INSERT INTO aulas (data, inicio, fim, turma, conteudo, observacoes, intervalo_minutos, duracao_minutos, turma_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	for _, tt := range []struct {
		nome   string
		coluna int
		valor  any
	}{
		{"data nula", 0, nil}, {"data inválida", 0, "2026-02-29"},
		{"data que produz NULL", 0, "2026-13-01"}, {"data fora do formato", 0, "2026-1-01"},
		{"ano zero", 0, "0000-01-01"}, {"dia inexistente", 0, "2026-04-31"},
		{"hora inválida", 1, "24:00"}, {"minuto inválido", 1, "08:60"},
		{"hora sem zero", 1, "8:00"}, {"hora com segundos", 1, "08:00:00"},
		{"término inválido", 2, "24:00"}, {"término anterior", 2, "07:00"}, {"término igual", 2, "08:00"},
		{"turma vazia", 3, ""}, {"turma em branco", 3, " \t\n"}, {"conteúdo vazio", 4, ""},
		{"observações nulas", 5, nil}, {"intervalo negativo", 6, -1},
		{"intervalo igual à duração", 6, 120}, {"intervalo maior", 6, 121},
		{"intervalo fracionário", 6, 1.5}, {"intervalo como texto", 6, "quinze"},
		{"duração inconsistente", 7, 60}, {"duração zero", 7, 0}, {"duração negativa", 7, -1},
		{"duração fracionária", 7, 105.5},
	} {
		t.Run(tt.nome, func(t *testing.T) {
			args := []any{"2026-10-13", "08:00", "10:00", "Turma", "Leitura", "", 15, 105, turma.ID}
			args[tt.coluna] = tt.valor
			if _, err := banco.db.Exec(insert, args...); err == nil {
				t.Fatal("SQL aceitou registro inválido")
			}
		})
	}
	registros, err := banco.Listar(context.Background())
	if err != nil || len(registros) != 0 {
		t.Fatalf("restrições deixaram registros: %v, %v", registros, err)
	}
	// As restrições não devem rejeitar limites permitidos pelo domínio.
	for _, data := range []string{"0001-01-01", "9999-12-31", "2000-02-29"} {
		a := exemplo()
		a.Data, a.Inicio, a.Fim, a.IntervaloMinutos, a.Observacoes = data, "00:00", "23:59", 1438, ""
		if _, err := banco.Inserir(context.Background(), a); err != nil {
			t.Fatalf("data válida %s rejeitada: %v", data, err)
		}
	}
}

func TestMigracaoReverteFalha(t *testing.T) {
	caminho := filepath.Join(t.TempDir(), "conflito.db")
	db, err := sql.Open("sqlite", caminho)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	// A tabela de conflito faz CREATE INDEX falhar após CREATE TABLE aulas.
	if _, err := db.Exec("CREATE TABLE aulas_data_inicio_id (valor TEXT)"); err != nil {
		t.Fatal(err)
	}
	if banco, err := Abrir(context.Background(), caminho); err == nil {
		banco.Close()
		t.Fatal("migração deveria falhar com nome de índice ocupado")
	}
	var versao, tabelas int
	if err := db.QueryRow("PRAGMA user_version").Scan(&versao); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow("SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name = 'aulas'").Scan(&tabelas); err != nil {
		t.Fatal(err)
	}
	if versao != 0 || tabelas != 0 {
		t.Fatalf("rollback falhou: versão %d, tabelas %d", versao, tabelas)
	}
	if _, err := db.Exec("DROP TABLE aulas_data_inicio_id"); err != nil {
		t.Fatal(err)
	}
	abrirTeste(t, caminho)
}

func TestVersaoFuturaRejeitada(t *testing.T) {
	caminho := filepath.Join(t.TempDir(), "futuro.db")
	banco := abrirTeste(t, caminho)
	if _, err := banco.db.Exec("PRAGMA user_version = 8"); err != nil {
		t.Fatal(err)
	}
	banco.Close()
	if novo, err := Abrir(context.Background(), caminho); err == nil {
		novo.Close()
		t.Fatal("versão futura deveria ser rejeitada")
	}
	db, err := sql.Open("sqlite", caminho)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var versao int
	if err := db.QueryRow("PRAGMA user_version").Scan(&versao); err != nil || versao != 8 {
		t.Fatalf("versão futura alterada: %d, %v", versao, err)
	}
}

func TestCaminhoPadraoEErrosDeAbertura(t *testing.T) {
	t.Chdir(t.TempDir())
	banco := abrirTeste(t, "")
	if _, err := os.Stat(CaminhoPadrao); err != nil {
		t.Fatal(err)
	}
	banco.Close()
	if b, err := Abrir(context.Background(), filepath.Join("inexistente", "aulas.db")); err == nil {
		b.Close()
		t.Fatal("deveria rejeitar diretório ausente")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if b, err := Abrir(ctx, "cancelado.db"); err == nil {
		b.Close()
		t.Fatal("deveria respeitar contexto cancelado")
	}
}
