package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"app-sync-backend/internal/aula"
)

var ErrConflito = errors.New("Esta aula foi alterada em outra operação. Recarregue o registro antes de tentar novamente")
var ErrCancelada = errors.New("Reative a aula antes de editar seus dados")

// Alteracao separa a atualização de dados da mudança de situação.
// Cancelada nil significa edição; um valor indica cancelar/reativar.
type Alteracao struct {
	Versao    int64
	TurmaID   int64
	Aula      aula.Aula
	Cancelada *bool
}

func (b *Banco) Alterar(ctx context.Context, id int64, entrada Alteracao) (Registro, error) {
	b.GoogleMu.Lock()
	defer b.GoogleMu.Unlock()
	tx, err := b.db.BeginTx(ctx, nil)
	if err != nil {
		return Registro{}, err
	}
	defer tx.Rollback()
	anterior, err := lerRegistro(tx.QueryRowContext(ctx, "SELECT "+colunasAula+" FROM aulas WHERE id = ?", id))
	if errors.Is(err, sql.ErrNoRows) {
		return Registro{}, ErrNaoEncontrada
	}
	if err != nil {
		return Registro{}, err
	}
	if anterior.Versao != entrada.Versao {
		return Registro{}, ErrConflito
	}
	atual := anterior
	operacao := "editar"
	if entrada.Cancelada != nil {
		if anterior.Cancelada == *entrada.Cancelada {
			return anterior, nil
		}
		atual.Cancelada = *entrada.Cancelada
		operacao = "reativar"
		if atual.Cancelada {
			operacao = "cancelar"
		}
	} else {
		if anterior.Cancelada {
			return Registro{}, ErrCancelada
		}
		var nome string
		err := tx.QueryRowContext(ctx, "SELECT nome FROM turmas WHERE id = ?", entrada.TurmaID).Scan(&nome)
		if errors.Is(err, sql.ErrNoRows) {
			return Registro{}, ErrTurmaAusente
		}
		if err != nil {
			return Registro{}, err
		}
		atual.Aula = entrada.Aula
		atual.Aula.Turma = nome
		atual.TurmaID = entrada.TurmaID
		atual.DuracaoMinutos, err = atual.Aula.DuracaoMinutos()
		if err != nil {
			return Registro{}, err
		}
	}
	atual.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	atual.Versao++
	result, err := tx.ExecContext(ctx, `UPDATE aulas SET data=?,inicio=?,fim=?,turma=?,conteudo=?,observacoes=?,intervalo_minutos=?,duracao_minutos=?,turma_id=?,cancelada=?,updated_at=?,versao=? WHERE id=? AND versao=?`, atual.Aula.Data, atual.Aula.Inicio, atual.Aula.Fim, atual.Aula.Turma, atual.Aula.Conteudo, atual.Aula.Observacoes, atual.Aula.IntervaloMinutos, atual.DuracaoMinutos, atual.TurmaID, atual.Cancelada, atual.UpdatedAt, atual.Versao, id, entrada.Versao)
	if err != nil {
		return Registro{}, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return Registro{}, err
	}
	if n != 1 {
		return Registro{}, ErrConflito
	}
	snapshot, err := json.Marshal(anterior)
	if err != nil {
		return Registro{}, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO aula_alteracoes(aula_id,versao_anterior,operacao,registrada_em,estado_anterior) VALUES (?,?,?,?,?)`, id, anterior.Versao, operacao, atual.UpdatedAt, string(snapshot)); err != nil {
		return Registro{}, err
	}
	if err := tx.Commit(); err != nil {
		return Registro{}, err
	}
	return atual, nil
}
