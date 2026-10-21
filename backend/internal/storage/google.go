package storage

import (
	"context"
	"database/sql"
	"errors"
	"time"

	destino "app-sync-backend/internal/google"
)

var ErrConfigConflito = errors.New("A configuração mudou em outra aba. Recarregue antes de salvar novamente")

type DestinoGoogle struct {
	ID         int64
	PlanilhaID string
}
type ConfigGoogle struct {
	Versao  int64
	Destino *DestinoGoogle
}
type MesGoogle struct {
	Periodo, Aba      string
	Aulas, Canceladas int
	Minutos           int64
}
type PlanoGoogle struct {
	Configuracao ConfigGoogle
	Meses        []MesGoogle
}

const consultaConfigGoogle = `SELECT c.versao,d.id,d.planilha_id FROM google_configuracao c LEFT JOIN google_destinos d ON d.id=c.destino_id WHERE c.id=1`

func lerConfigGoogle(row interface{ Scan(...any) error }) (ConfigGoogle, error) {
	var c ConfigGoogle
	var id sql.NullInt64
	var planilha sql.NullString
	err := row.Scan(&c.Versao, &id, &planilha)
	if id.Valid {
		c.Destino = &DestinoGoogle{ID: id.Int64, PlanilhaID: planilha.String}
	}
	return c, err
}
func (b *Banco) ConfiguracaoGoogle(ctx context.Context) (ConfigGoogle, error) {
	return lerConfigGoogle(b.db.QueryRowContext(ctx, consultaConfigGoogle))
}

// A troca preserva os destinos anteriores e não cria vínculos de abas ou envios.
func (b *Banco) ConfigurarGoogle(ctx context.Context, link string, versao int64) (ConfigGoogle, error) {
	b.GoogleMu.Lock()
	defer b.GoogleMu.Unlock()
	if pending, err := b.AlgumaExclusaoGoogle(ctx); err != nil {
		return ConfigGoogle{}, err
	} else if pending {
		return ConfigGoogle{}, ErrExclusaoPendente
	}
	id, err := destino.PlanilhaID(link)
	if err != nil {
		return ConfigGoogle{}, err
	}
	tx, err := b.db.BeginTx(ctx, nil)
	if err != nil {
		return ConfigGoogle{}, err
	}
	defer tx.Rollback()
	atual, err := lerConfigGoogle(tx.QueryRowContext(ctx, consultaConfigGoogle))
	if err != nil {
		return ConfigGoogle{}, err
	}
	if atual.Versao != versao {
		return ConfigGoogle{}, ErrConfigConflito
	}
	if atual.Destino != nil && atual.Destino.PlanilhaID == id {
		return atual, nil
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO google_destinos(planilha_id,criado_em) VALUES (?,?) ON CONFLICT(planilha_id) DO NOTHING`, id, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		return ConfigGoogle{}, err
	}
	var destinoID int64
	if err := tx.QueryRowContext(ctx, "SELECT id FROM google_destinos WHERE planilha_id=?", id).Scan(&destinoID); err != nil {
		return ConfigGoogle{}, err
	}
	if _, err := tx.ExecContext(ctx, "UPDATE google_configuracao SET destino_id=?,versao=versao+1 WHERE id=1", destinoID); err != nil {
		return ConfigGoogle{}, err
	}
	if err := tx.Commit(); err != nil {
		return ConfigGoogle{}, err
	}
	return ConfigGoogle{Versao: atual.Versao + 1, Destino: &DestinoGoogle{ID: destinoID, PlanilhaID: id}}, nil
}
func (b *Banco) PlanejamentoGoogle(ctx context.Context) (PlanoGoogle, error) {
	p := PlanoGoogle{Meses: make([]MesGoogle, 0)}
	tx, err := b.db.BeginTx(ctx, nil)
	if err != nil {
		return p, err
	}
	defer tx.Rollback()
	p.Configuracao, err = lerConfigGoogle(tx.QueryRowContext(ctx, consultaConfigGoogle))
	if err != nil {
		return p, err
	}
	rows, err := tx.QueryContext(ctx, `SELECT substr(data,1,7),count(*),sum(cancelada),coalesce(sum(CASE WHEN cancelada=0 THEN duracao_minutos ELSE 0 END),0) FROM aulas GROUP BY substr(data,1,7) ORDER BY substr(data,1,7) DESC`)
	if err != nil {
		return p, err
	}
	for rows.Next() {
		var m MesGoogle
		if err := rows.Scan(&m.Periodo, &m.Aulas, &m.Canceladas, &m.Minutos); err != nil {
			rows.Close()
			return p, err
		}
		_, m.Aba, err = destino.Periodo(m.Periodo + "-01")
		if err != nil {
			rows.Close()
			return p, err
		}
		p.Meses = append(p.Meses, m)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return p, err
	}
	return p, tx.Commit()
}
