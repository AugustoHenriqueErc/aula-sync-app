package storage

import (
	"context"
	"database/sql"
	"errors"
)

var ErrExclusaoPendente = errors.New("Existe uma exclusão aguardando confirmação. Conclua a exclusão antes de alterar aulas, sincronizar ou trocar o destino")

type Exclusao struct {
	DestinoID     int64
	RemoverGoogle bool
}

func (b *Banco) Exclusao(ctx context.Context, id int64) (*Exclusao, error) {
	var e Exclusao
	var destino sql.NullInt64
	err := b.db.QueryRowContext(ctx, "SELECT destino_id,remover_google FROM aula_exclusoes WHERE aula_id=?", id).Scan(&destino, &e.RemoverGoogle)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	e.DestinoID = destino.Int64
	return &e, err
}
func (b *Banco) ExclusaoGooglePendente(ctx context.Context, id int64) (bool, error) {
	e, err := b.Exclusao(ctx, id)
	return e != nil, err
}
func (b *Banco) AlgumaExclusaoGoogle(ctx context.Context) (bool, error) {
	var n int
	err := b.db.QueryRowContext(ctx, "SELECT count(*) FROM aula_exclusoes WHERE remover_google=1").Scan(&n)
	return n > 0, err
}

// IniciarExclusao e FinalizarExclusao são chamados sob GoogleMu pelo serviço.
func (b *Banco) IniciarExclusao(ctx context.Context, id int64, destino *int64, google bool) error {
	_, err := b.db.ExecContext(ctx, "INSERT INTO aula_exclusoes(aula_id,destino_id,remover_google) VALUES(?,?,?)", id, destino, google)
	return err
}
func (b *Banco) FinalizarExclusao(ctx context.Context, id int64) error {
	tx, err := b.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// Remove inclusive os snapshots da trilha e dos envios: exclusão definitiva.
	for _, table := range []string{"aula_exclusoes", "google_sincronizacoes", "aula_alteracoes", "aulas"} {
		column := "aula_id"
		if table == "aulas" {
			column = "id"
		}
		if _, err = tx.ExecContext(ctx, "DELETE FROM "+table+" WHERE "+column+"=?", id); err != nil {
			return err
		}
	}
	return tx.Commit()
}
func (b *Banco) AulaTemEnvioIncerto(ctx context.Context, id int64) (bool, error) {
	var n int
	err := b.db.QueryRowContext(ctx, "SELECT count(*) FROM google_sincronizacoes WHERE aula_id=? AND conteudo_proposto IS NOT NULL", id).Scan(&n)
	return n > 0, err
}
