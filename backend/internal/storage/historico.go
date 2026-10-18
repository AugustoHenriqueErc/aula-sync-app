package storage

import "context"

type Filtro struct {
	DataInicial, DataFinal string
	TurmaID                int64
	Pagina                 int
}
type Historico struct {
	Aulas               []Registro
	TotalRegistros      int
	DuracaoTotalMinutos int64
}

// A transação mantém página e totais na mesma visão dos dados.
func (b *Banco) Historico(ctx context.Context, f Filtro) (Historico, error) {
	h := Historico{Aulas: make([]Registro, 0)}
	where := " WHERE 1=1"
	args := []any{}
	if f.DataInicial != "" {
		where += " AND data >= ?"
		args = append(args, f.DataInicial)
	}
	if f.DataFinal != "" {
		where += " AND data <= ?"
		args = append(args, f.DataFinal)
	}
	if f.TurmaID > 0 {
		where += " AND turma_id = ?"
		args = append(args, f.TurmaID)
	}
	tx, err := b.db.BeginTx(ctx, nil)
	if err != nil {
		return h, err
	}
	defer tx.Rollback()
	if err := tx.QueryRowContext(ctx, "SELECT count(*),coalesce(sum(duracao_minutos),0) FROM aulas"+where, args...).Scan(&h.TotalRegistros, &h.DuracaoTotalMinutos); err != nil {
		return h, err
	}
	if f.Pagina < 1 {
		f.Pagina = 1
	}
	args = append(args, 20, (f.Pagina-1)*20)
	rows, err := tx.QueryContext(ctx, "SELECT "+colunasAula+" FROM aulas"+where+" ORDER BY data DESC,inicio DESC,id DESC LIMIT ? OFFSET ?", args...)
	if err != nil {
		return h, err
	}
	for rows.Next() {
		r, err := lerRegistro(rows)
		if err != nil {
			rows.Close()
			return h, err
		}
		h.Aulas = append(h.Aulas, r)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return h, err
	}
	return h, tx.Commit()
}
