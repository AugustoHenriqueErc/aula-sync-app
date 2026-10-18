package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestEditarCancelarReativarEAuditoria(t *testing.T) {
	ctx := context.Background()
	b := abrirTeste(t, filepath.Join(t.TempDir(), "aulas.db"))
	original, err := b.Inserir(ctx, exemplo())
	if err != nil {
		t.Fatal(err)
	}
	a := original.Aula
	a.Fim = "11:00"
	editada, err := b.Alterar(ctx, original.ID, Alteracao{Versao: 1, TurmaID: original.TurmaID, Aula: a})
	if err != nil {
		t.Fatal(err)
	}
	if editada.DuracaoMinutos != 165 || editada.Versao != 2 || editada.UpdatedAt == original.UpdatedAt {
		t.Fatalf("edição: %+v", editada)
	}
	if _, err := time.Parse(time.RFC3339Nano, editada.UpdatedAt); err != nil {
		t.Fatal(err)
	}
	var snapshot string
	if err := b.db.QueryRow("SELECT estado_anterior FROM aula_alteracoes WHERE aula_id=? AND versao_anterior=1", original.ID).Scan(&snapshot); err != nil {
		t.Fatal(err)
	}
	var anterior Registro
	if err := json.Unmarshal([]byte(snapshot), &anterior); err != nil || anterior != original {
		t.Fatalf("snapshot: %+v %v", anterior, err)
	}
	cancelar := true
	cancelada, err := b.Alterar(ctx, original.ID, Alteracao{Versao: 2, Cancelada: &cancelar})
	if err != nil || !cancelada.Cancelada || cancelada.Versao != 3 {
		t.Fatalf("cancelamento: %+v %v", cancelada, err)
	}
	for _, situacao := range []string{"", "ativas", "canceladas", "todas"} {
		h, err := b.Historico(ctx, Filtro{Situacao: situacao})
		n := 0
		if situacao == "canceladas" || situacao == "todas" {
			n = 1
		}
		if err != nil || h.TotalRegistros != n || h.DuracaoTotalMinutos != 0 {
			t.Fatalf("situação %s: %+v %v", situacao, h, err)
		}
	}
	if _, err := b.Alterar(ctx, original.ID, Alteracao{Versao: 3, TurmaID: original.TurmaID, Aula: a}); !errors.Is(err, ErrCancelada) {
		t.Fatalf("editou cancelada: %v", err)
	}
	// Repetir o mesmo estado e versão não inventa uma nova alteração.
	igual, err := b.Alterar(ctx, original.ID, Alteracao{Versao: 3, Cancelada: &cancelar})
	if err != nil || igual != cancelada {
		t.Fatal("repetição alterou estado", err)
	}
	cancelar = false
	reativada, err := b.Alterar(ctx, original.ID, Alteracao{Versao: 3, Cancelada: &cancelar})
	if err != nil || reativada.Cancelada || reativada.Versao != 4 {
		t.Fatalf("reativação: %+v %v", reativada, err)
	}
	h, err := b.Historico(ctx, Filtro{})
	if err != nil || h.DuracaoTotalMinutos != 165 {
		t.Fatalf("total: %+v %v", h, err)
	}
	var n int
	if err := b.db.QueryRow("SELECT count(*) FROM aula_alteracoes").Scan(&n); err != nil || n != 3 {
		t.Fatalf("auditoria: %d %v", n, err)
	}
}
func TestAlteracaoConcorrenteEInvalida(t *testing.T) {
	ctx := context.Background()
	b := abrirTeste(t, filepath.Join(t.TempDir(), "aulas.db"))
	r, err := b.Inserir(ctx, exemplo())
	if err != nil {
		t.Fatal(err)
	}
	entrada := Alteracao{Versao: 1, TurmaID: r.TurmaID, Aula: r.Aula}
	invalida := entrada
	invalida.Aula.Fim = "07:00"
	if _, err := b.Alterar(ctx, r.ID, invalida); err == nil {
		t.Fatal("aceitou horário inválido")
	}
	inexistente := entrada
	inexistente.TurmaID = 999
	if _, err := b.Alterar(ctx, r.ID, inexistente); !errors.Is(err, ErrTurmaAusente) {
		t.Fatalf("turma inexistente: %v", err)
	}
	if _, err := b.Alterar(ctx, 999, entrada); !errors.Is(err, ErrNaoEncontrada) {
		t.Fatalf("ID inexistente: %v", err)
	}
	antes, err := b.Buscar(ctx, r.ID)
	if err != nil || antes != r {
		t.Fatal("entrada inválida alterou registro")
	}
	var wg sync.WaitGroup
	resultados := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, err := b.Alterar(ctx, r.ID, entrada); resultados <- err }()
	}
	wg.Wait()
	close(resultados)
	ok, conflitos := 0, 0
	for err := range resultados {
		if err == nil {
			ok++
		} else if errors.Is(err, ErrConflito) {
			conflitos++
		} else {
			t.Fatal(err)
		}
	}
	if ok != 1 || conflitos != 1 {
		t.Fatalf("sucessos=%d conflitos=%d", ok, conflitos)
	}
}
func TestFalhaAuditoriaReverteEdicao(t *testing.T) {
	ctx := context.Background()
	b := abrirTeste(t, filepath.Join(t.TempDir(), "aulas.db"))
	r, err := b.Inserir(ctx, exemplo())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.db.Exec(`CREATE TRIGGER falha_auditoria BEFORE INSERT ON aula_alteracoes BEGIN SELECT RAISE(ABORT,'falha simulada'); END`); err != nil {
		t.Fatal(err)
	}
	a := r.Aula
	a.Fim = "11:00"
	if _, err := b.Alterar(ctx, r.ID, Alteracao{Versao: 1, TurmaID: r.TurmaID, Aula: a}); err == nil {
		t.Fatal("auditoria deveria falhar")
	}
	atual, err := b.Buscar(ctx, r.ID)
	if err != nil || atual != r {
		t.Fatal("edição parcial persistiu", err)
	}
}
func TestBackupERestauracao(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	b := abrirTeste(t, filepath.Join(dir, "origem.db"))
	r, err := b.Inserir(ctx, exemplo())
	if err != nil {
		t.Fatal(err)
	}
	cancelar := true
	r, err = b.Alterar(ctx, r.ID, Alteracao{Versao: 1, Cancelada: &cancelar})
	if err != nil {
		t.Fatal(err)
	}
	destino := filepath.Join(dir, "backup com espaço?#.db")
	if err := b.Backup(ctx, destino); err != nil {
		t.Fatal(err)
	}
	if err := b.Backup(ctx, destino); err == nil {
		t.Fatal("sobrescreveu backup")
	}
	vazio := filepath.Join(dir, "existente.db")
	if err := os.WriteFile(vazio, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := b.Backup(ctx, vazio); err == nil {
		t.Fatal("sobrescreveu arquivo vazio existente")
	}
	// Alterar a origem não pode alterar o snapshot.
	cancelar = false
	if _, err := b.Alterar(ctx, r.ID, Alteracao{Versao: 2, Cancelada: &cancelar}); err != nil {
		t.Fatal(err)
	}
	dados, err := os.ReadFile(destino)
	if err != nil {
		t.Fatal(err)
	}
	restaurado := filepath.Join(dir, "restaurado.db")
	if err := os.WriteFile(restaurado, dados, 0600); err != nil {
		t.Fatal(err)
	}
	copia := abrirTeste(t, restaurado)
	if err := copia.Verificar(ctx); err != nil {
		t.Fatal(err)
	}
	obtido, err := copia.Buscar(ctx, r.ID)
	if err != nil || !reflect.DeepEqual(obtido, r) {
		t.Fatalf("restauração: %+v %v", obtido, err)
	}
	var n int
	if err := copia.db.QueryRow("SELECT count(*) FROM aula_alteracoes").Scan(&n); err != nil || n != 1 {
		t.Fatalf("auditoria restaurada: %d %v", n, err)
	}
}
func TestMigracao3PreservaVersao2(t *testing.T) {
	ctx := context.Background()
	caminho := filepath.Join(t.TempDir(), "v2.db")
	db, err := sql.Open("sqlite", caminho)
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
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO turmas(id,nome,nome_normalizado) VALUES (7,'Augusto','augusto'); INSERT INTO aulas(id,turma_id,data,inicio,fim,turma,conteudo,intervalo_minutos,duracao_minutos) VALUES (42,7,'2026-10-18','08:00','09:00','Augusto','Original',0,60)`); err != nil {
		t.Fatal(err)
	}
	db.Close()
	b := abrirTeste(t, caminho)
	r, err := b.Buscar(ctx, 42)
	if err != nil || r.Versao != 1 || r.Cancelada || r.UpdatedAt != "" || r.TurmaID != 7 || r.DuracaoMinutos != 60 {
		t.Fatalf("migração: %+v %v", r, err)
	}
	if err := b.migrar(ctx); err != nil {
		t.Fatal(err)
	}
}
