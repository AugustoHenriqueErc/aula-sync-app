package storage

import "context"

type Filtro struct {
	DataInicial, DataFinal string
	TurmaID                int64
	Pagina                 int
	Situacao               string
}
type Historico struct {
	Aulas               []Registro
	TotalRegistros      int
	DuracaoTotalMinutos int64
}

// A transação mantém página e totais na mesma visão dos dados.
func (b *Banco) Historico(ctx context.Context, f Filtro) (Historico, error) {
	return b.consultarHistorico(ctx, f, true)
}

// Relatorio usa a mesma transação, filtros e totais, sem limitar a uma página.
func (b *Banco) Relatorio(ctx context.Context, f Filtro) (Historico, error) {
	return b.consultarHistorico(ctx, f, false)
}

func (b *Banco) consultarHistorico(ctx context.Context, f Filtro, paginar bool) (Historico, error) {
	h := Historico{Aulas: make([]Registro, 0)}
	where := " WHERE 1=1"
	args := []any{}
	switch f.Situacao {
	case "todas":
	case "canceladas":
		where += " AND cancelada = 1"
	default:
		where += " AND cancelada = 0"
	}
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
	if err := tx.QueryRowContext(ctx, "SELECT count(*),coalesce(sum(CASE WHEN cancelada = 0 THEN duracao_minutos ELSE 0 END),0) FROM aulas"+where, args...).Scan(&h.TotalRegistros, &h.DuracaoTotalMinutos); err != nil {
		return h, err
	}
	if f.Pagina < 1 {
		f.Pagina = 1
	}
	query := "SELECT " + colunasAula + " FROM aulas" + where + " ORDER BY data DESC,inicio DESC,id DESC"
	if paginar {
		query += " LIMIT ? OFFSET ?"
		args = append(args, 20, (f.Pagina-1)*20)
	}
	rows, err := tx.QueryContext(ctx, query, args...)
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
