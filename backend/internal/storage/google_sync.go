package storage

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type VinculoGoogle struct {
	Periodo string
	SheetID *int64
	Estado  string
	Modelo  int
}
type EnvioGoogle struct {
	Periodo                                     string
	Versao                                      int64
	Estado                                      string
	Tentativas                                  int
	UltimaTentativa, Erro, Confirmado, Proposto string
	Linha                                       int
}

func (b *Banco) InstalacaoGoogle(ctx context.Context) (string, error) {
	var s string
	err := b.db.QueryRowContext(ctx, "SELECT instalacao_id FROM google_configuracao WHERE id=1").Scan(&s)
	return s, err
}
func (b *Banco) VinculoGoogle(ctx context.Context, destino int64, periodo string) (VinculoGoogle, error) {
	v := VinculoGoogle{Periodo: periodo, Estado: "nao_verificada"}
	var id sql.NullInt64
	var modelo sql.NullInt64
	err := b.db.QueryRowContext(ctx, "SELECT sheet_id,estado,modelo FROM google_abas WHERE destino_id=? AND periodo=?", destino, periodo).Scan(&id, &v.Estado, &modelo)
	if errors.Is(err, sql.ErrNoRows) {
		return v, nil
	}
	if id.Valid {
		v.SheetID = &id.Int64
	}
	v.Modelo = int(modelo.Int64)
	return v, err
}
func (b *Banco) SalvarVinculoGoogle(ctx context.Context, destino int64, v VinculoGoogle) error {
	_, err := b.db.ExecContext(ctx, `INSERT INTO google_abas(destino_id,periodo,sheet_id,estado,verificada_em,modelo) VALUES(?,?,?,?,?,?) ON CONFLICT(destino_id,periodo) DO UPDATE SET sheet_id=coalesce(google_abas.sheet_id,excluded.sheet_id),estado=excluded.estado,verificada_em=excluded.verificada_em,modelo=excluded.modelo`, destino, v.Periodo, v.SheetID, v.Estado, time.Now().UTC().Format(time.RFC3339Nano), v.Modelo)
	return err
}
func (b *Banco) EnviosGoogle(ctx context.Context, destino, aula int64) ([]EnvioGoogle, error) {
	rows, err := b.db.QueryContext(ctx, `SELECT periodo,coalesce(versao_enviada,0),estado,tentativas,coalesce(ultima_tentativa,''),coalesce(erro,''),coalesce(conteudo_confirmado,''),coalesce(conteudo_proposto,''),coalesce(linha_proposta,0) FROM google_sincronizacoes WHERE destino_id=? AND aula_id=? ORDER BY periodo`, destino, aula)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []EnvioGoogle{}
	for rows.Next() {
		var e EnvioGoogle
		if err := rows.Scan(&e.Periodo, &e.Versao, &e.Estado, &e.Tentativas, &e.UltimaTentativa, &e.Erro, &e.Confirmado, &e.Proposto, &e.Linha); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// O intento é persistido ANTES da escrita: permite reconhecer sucesso com resposta perdida.
func (b *Banco) IniciarEnvioGoogle(ctx context.Context, destino, aula int64, periodo, proposto string, linha int) error {
	_, err := b.db.ExecContext(ctx, `INSERT INTO google_sincronizacoes(aula_id,destino_id,periodo,estado,tentativas,ultima_tentativa,conteudo_proposto,linha_proposta) VALUES(?,?,?,'pendente',0,?,?,?) ON CONFLICT(aula_id,destino_id,periodo) DO UPDATE SET estado='pendente',ultima_tentativa=excluded.ultima_tentativa,erro=NULL,conteudo_proposto=excluded.conteudo_proposto,linha_proposta=excluded.linha_proposta`, aula, destino, periodo, time.Now().UTC().Format(time.RFC3339Nano), proposto, linha)
	return err
}
func (b *Banco) ConfirmarEnvioGoogle(ctx context.Context, destino, aula int64, periodo string, versao int64, conteudo string) error {
	_, err := b.db.ExecContext(ctx, `UPDATE google_sincronizacoes SET estado='sincronizado',versao_enviada=?,conteudo_confirmado=?,conteudo_proposto=NULL,linha_proposta=NULL,erro=NULL WHERE destino_id=? AND aula_id=? AND periodo=?`, versao, conteudo, destino, aula, periodo)
	return err
}
func (b *Banco) FalhaEnvioGoogle(ctx context.Context, destino, aula int64, periodo, mensagem string) error {
	_, err := b.db.ExecContext(ctx, `INSERT INTO google_sincronizacoes(aula_id,destino_id,periodo,estado,tentativas,ultima_tentativa,erro) VALUES(?,?,?,'erro',1,?,?) ON CONFLICT(aula_id,destino_id,periodo) DO UPDATE SET estado='erro',erro=excluded.erro,ultima_tentativa=excluded.ultima_tentativa`, aula, destino, periodo, time.Now().UTC().Format(time.RFC3339Nano), mensagem)
	return err
}

type EstadoGoogle struct {
	Estado          string `json:"estado"`
	DestinoID       int64  `json:"destino_id"`
	ConfigVersao    int64  `json:"config_versao"`
	Periodo         string `json:"periodo"`
	Tentativas      int    `json:"tentativas"`
	UltimaTentativa string `json:"ultima_tentativa,omitempty"`
	Erro            string `json:"erro,omitempty"`
}

func (b *Banco) EstadoGoogle(ctx context.Context, r Registro) (EstadoGoogle, error) {
	c, err := b.ConfiguracaoGoogle(ctx)
	e := EstadoGoogle{Estado: "sem_destino", ConfigVersao: c.Versao, Periodo: r.Aula.Data[:7]}
	if err != nil || c.Destino == nil {
		return e, err
	}
	e.DestinoID = c.Destino.ID
	e.Estado = "aguardando_preparacao"
	v, err := b.VinculoGoogle(ctx, c.Destino.ID, e.Periodo)
	if err != nil {
		return e, err
	}
	if v.Estado == "pronta" {
		e.Estado = "pendente"
	}
	envios, err := b.EnviosGoogle(ctx, c.Destino.ID, r.ID)
	if err != nil {
		return e, err
	}
	for _, s := range envios {
		if s.Periodo == e.Periodo {
			e.Tentativas = s.Tentativas
			e.UltimaTentativa = s.UltimaTentativa
			e.Erro = s.Erro
			if s.Estado == "erro" {
				e.Estado = "erro"
			}
			if s.Estado == "sincronizado" && s.Versao == r.Versao && v.Estado == "pronta" {
				e.Estado = "sincronizado"
			}
		}
	}
	return e, nil
}

// Não libera a linha de uma escrita incerta para outra aula antes da reconciliação.
func (b *Banco) OutroEnvioPendenteGoogle(ctx context.Context, destino, aula int64) (int64, error) {
	var id int64
	err := b.db.QueryRowContext(ctx, "SELECT aula_id FROM google_sincronizacoes WHERE destino_id=? AND aula_id!=? AND conteudo_proposto IS NOT NULL LIMIT 1", destino, aula).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return id, err
}

// Uma tentativa corresponde a uma ação explícita, inclusive quando a leitura falha.
func (b *Banco) RegistrarTentativaGoogle(ctx context.Context, destino, aula int64, periodo string) error {
	_, err := b.db.ExecContext(ctx, `INSERT INTO google_sincronizacoes(aula_id,destino_id,periodo,estado,tentativas,ultima_tentativa) VALUES(?,?,?,'pendente',1,?) ON CONFLICT(aula_id,destino_id,periodo) DO UPDATE SET estado='pendente',tentativas=tentativas+1,ultima_tentativa=excluded.ultima_tentativa,erro=NULL`, aula, destino, periodo, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}
