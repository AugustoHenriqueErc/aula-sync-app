package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"app-sync-backend/internal/aula"
)

var ErrNaoEncontrada = errors.New("aula não encontrada")

// Registro separa os dados de persistência do modelo de domínio.
type Registro struct {
	ID             int64
	Aula           aula.Aula
	DuracaoMinutos int
}

func (b *Banco) Inserir(ctx context.Context, a aula.Aula) (Registro, error) {
	duracao, err := a.DuracaoMinutos()
	if err != nil {
		return Registro{}, err
	}
	result, err := b.db.ExecContext(ctx, `INSERT INTO aulas
  (data, inicio, fim, turma, conteudo, observacoes, intervalo_minutos, duracao_minutos)
  VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		a.Data, a.Inicio, a.Fim, a.Turma, a.Conteudo, a.Observacoes, a.IntervaloMinutos, duracao)
	if err != nil {
		return Registro{}, fmt.Errorf("inserir aula: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Registro{}, fmt.Errorf("obter ID da aula: %w", err)
	}
	return Registro{ID: id, Aula: a, DuracaoMinutos: duracao}, nil
}

const colunasAula = "id, data, inicio, fim, turma, conteudo, observacoes, intervalo_minutos, duracao_minutos"

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
		&r.Aula.Conteudo, &r.Aula.Observacoes, &r.Aula.IntervaloMinutos, &r.DuracaoMinutos)
	return r, err
}
