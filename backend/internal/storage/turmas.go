package storage

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"strings"

	"app-sync-backend/internal/aula"
)

//go:embed migrations/002_turmas.sql
var segundaMigracao string

var ErrTurmaDuplicada = errors.New("Já existe uma turma ou estudante com esse nome")
var ErrTurmaAusente = errors.New("Turma ou estudante não encontrado")

type Turma struct {
	ID   int64  `json:"id"`
	Nome string `json:"nome"`
}

func normalizarNome(nome string) string {
	return strings.ToLower(strings.Join(strings.Fields(nome), " "))
}

func migrarTurmas(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, segundaMigracao); err != nil {
		return err
	}
	rows, err := tx.QueryContext(ctx, "SELECT id, turma FROM aulas ORDER BY id")
	if err != nil {
		return err
	}
	type antiga struct {
		id   int64
		nome string
	}
	var antigas []antiga
	for rows.Next() {
		var a antiga
		if err := rows.Scan(&a.id, &a.nome); err != nil {
			rows.Close()
			return err
		}
		antigas = append(antigas, a)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, a := range antigas {
		chave := normalizarNome(a.nome)
		if _, err := tx.ExecContext(ctx, "INSERT INTO turmas(nome,nome_normalizado) VALUES (?,?) ON CONFLICT(nome_normalizado) DO NOTHING", strings.Join(strings.Fields(a.nome), " "), chave); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO aulas_novas (turma_id,id,data,inicio,fim,turma,conteudo,observacoes,intervalo_minutos,duracao_minutos)
   SELECT (SELECT id FROM turmas WHERE nome_normalizado = ?),id,data,inicio,fim,turma,conteudo,observacoes,intervalo_minutos,duracao_minutos FROM aulas WHERE id = ?`, chave, a.id); err != nil {
			return err
		}
	}
	_, err = tx.ExecContext(ctx, `DROP TABLE aulas; ALTER TABLE aulas_novas RENAME TO aulas;
 CREATE INDEX aulas_data_inicio_id ON aulas(data DESC,inicio DESC,id DESC);
 CREATE INDEX aulas_turma_data ON aulas(turma_id,data DESC);
 PRAGMA user_version = 2;`)
	return err
}
func (b *Banco) CriarTurma(ctx context.Context, nome string) (Turma, error) {
	nome = strings.Join(strings.Fields(nome), " ")
	if nome == "" {
		return Turma{}, &aula.ErroValidacao{Campo: "Nome", Mensagem: "Informe o nome da turma ou estudante"}
	}
	result, err := b.db.ExecContext(ctx, "INSERT INTO turmas(nome,nome_normalizado) VALUES (?,?) ON CONFLICT(nome_normalizado) DO NOTHING", nome, normalizarNome(nome))
	if err != nil {
		return Turma{}, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return Turma{}, err
	}
	if n == 0 {
		return Turma{}, ErrTurmaDuplicada
	}
	id, err := result.LastInsertId()
	return Turma{ID: id, Nome: nome}, err
}
func (b *Banco) BuscarTurma(ctx context.Context, id int64) (Turma, error) {
	var t Turma
	err := b.db.QueryRowContext(ctx, "SELECT id,nome FROM turmas WHERE id = ?", id).Scan(&t.ID, &t.Nome)
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrTurmaAusente
	}
	return t, err
}
func (b *Banco) ListarTurmas(ctx context.Context) ([]Turma, error) {
	rows, err := b.db.QueryContext(ctx, "SELECT id,nome FROM turmas ORDER BY nome_normalizado,id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ts := make([]Turma, 0)
	for rows.Next() {
		var t Turma
		if err := rows.Scan(&t.ID, &t.Nome); err != nil {
			return nil, err
		}
		ts = append(ts, t)
	}
	return ts, rows.Err()
}
