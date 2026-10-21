package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"app-sync-backend/internal/aula"
)

var ErrNaoEncontrada = errors.New("aula não encontrada")

// Registro separa os dados de persistência do modelo de domínio.
type Registro struct {
	Versao         int64
	Cancelada      bool
	UpdatedAt      string
	ID             int64
	TurmaID        int64
	Aula           aula.Aula
	DuracaoMinutos int
}

func (b *Banco) Inserir(ctx context.Context, a aula.Aula) (Registro, error) {
	duracao, err := a.DuracaoMinutos()
	if err != nil {
		return Registro{}, err
	}
	tx, err := b.db.BeginTx(ctx, nil)
	if err != nil {
		return Registro{}, err
	}
	defer tx.Rollback()
	// O comando de estudo reutiliza ou cria o cadastro pelo nome. A API exige ID.
	if _, err = tx.ExecContext(ctx, "INSERT INTO turmas(nome,nome_normalizado) VALUES (?,?) ON CONFLICT(nome_normalizado) DO NOTHING", a.Turma, normalizarNome(a.Turma)); err != nil {
		return Registro{}, err
	}
	var turmaID int64
	if err = tx.QueryRowContext(ctx, "SELECT id FROM turmas WHERE nome_normalizado = ?", normalizarNome(a.Turma)).Scan(&turmaID); err != nil {
		return Registro{}, err
	}
	var id int64
	if err = tx.QueryRowContext(ctx, `UPDATE aula_sequencia SET ultimo=max(ultimo,coalesce((SELECT max(id) FROM aulas),0))+1 WHERE id=1 RETURNING ultimo`).Scan(&id); err != nil {
		return Registro{}, err
	}
	atualizada := time.Now().UTC().Format(time.RFC3339Nano)
	_, err = tx.ExecContext(ctx, `INSERT INTO aulas
  (id, data, inicio, fim, turma, conteudo, observacoes, intervalo_minutos, duracao_minutos, turma_id, updated_at)
  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, a.Data, a.Inicio, a.Fim, a.Turma, a.Conteudo, a.Observacoes, a.IntervaloMinutos, duracao, turmaID, atualizada)
	if err != nil {
		return Registro{}, fmt.Errorf("inserir aula: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return Registro{}, err
	}
	return Registro{Versao: 1, UpdatedAt: atualizada, ID: id, TurmaID: turmaID, Aula: a, DuracaoMinutos: duracao}, nil
}

const colunasAula = "id, data, inicio, fim, turma, conteudo, observacoes, intervalo_minutos, duracao_minutos, turma_id, versao, cancelada, updated_at"

func (b *Banco) Buscar(ctx context.Context, id int64) (Registro, error) {
	registro, err := lerRegistro(b.db.QueryRowContext(ctx, "SELECT "+colunasAula+" FROM aulas WHERE id = ?", id))
	if errors.Is(err, sql.ErrNoRows) {
		return Registro{}, ErrNaoEncontrada
	}
	if err != nil {
		return Registro{}, fmt.Errorf("buscar aula: %w", err)
	}
	return registro, nil
}

// Listar retorna todas as aulas da mais recente para a mais antiga.
// Início e ID, ambos decrescentes, desempatarão datas iguais.
func (b *Banco) Listar(ctx context.Context) ([]Registro, error) {
	rows, err := b.db.QueryContext(ctx, "SELECT "+colunasAula+" FROM aulas ORDER BY data DESC, inicio DESC, id DESC")
	if err != nil {
		return nil, fmt.Errorf("listar aulas: %w", err)
	}
	defer rows.Close()
	registros := make([]Registro, 0)
	for rows.Next() {
		registro, err := lerRegistro(rows)
		if err != nil {
			return nil, fmt.Errorf("ler aula: %w", err)
		}
		registros = append(registros, registro)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("percorrer aulas: %w", err)
	}
	return registros, nil
}

// sql.Row e sql.Rows oferecem Scan; a mesma leitura serve para busca e listagem.
func lerRegistro(row interface{ Scan(...any) error }) (Registro, error) {
	var r Registro
	err := row.Scan(&r.ID, &r.Aula.Data, &r.Aula.Inicio, &r.Aula.Fim, &r.Aula.Turma,
		&r.Aula.Conteudo, &r.Aula.Observacoes, &r.Aula.IntervaloMinutos, &r.DuracaoMinutos, &r.TurmaID, &r.Versao, &r.Cancelada, &r.UpdatedAt)
	return r, err
}
