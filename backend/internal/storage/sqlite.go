// Package storage persiste aulas em SQLite, sem depender da camada HTTP.
package storage

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"net/url"
	"path/filepath"
	"sync"

	_ "modernc.org/sqlite"
)

const CaminhoPadrao = "aulas.db"

//go:embed migrations/001_create_aulas.sql
var primeiraMigracao string

//go:embed migrations/003_correcao.sql
var terceiraMigracao string

//go:embed migrations/004_google_config.sql
var quartaMigracao string

//go:embed migrations/005_google_sync.sql
var quintaMigracao string

//go:embed migrations/006_exclusao.sql
var sextaMigracao string

// GoogleMu serializa configuração, correções e operações remotas neste processo.
type Banco struct {
	db       *sql.DB
	GoogleMu sync.Mutex
}

// Abrir cria ou abre um arquivo local e aplica as migrações pendentes.
// Um caminho vazio usa aulas.db no diretório de execução. A pasta deve existir.
func Abrir(ctx context.Context, caminho string) (*Banco, error) {
	if caminho == "" {
		caminho = CaminhoPadrao
	}
	absoluto, err := filepath.Abs(caminho)
	if err != nil {
		return nil, fmt.Errorf("resolver caminho do banco: %w", err)
	}
	// A URL evita interpretar caracteres como ? e # do nome como opções do driver.
	dsn := url.URL{Scheme: "file", Path: filepath.ToSlash(absoluto)}
	options := url.Values{}
	options.Set("mode", "rwc")
	options.Add("_pragma", "busy_timeout(5000)")
	options.Add("_pragma", "foreign_keys(1)")
	options.Set("_txlock", "immediate")
	dsn.RawQuery = options.Encode()
	db, err := sql.Open("sqlite", dsn.String())
	if err != nil {
		return nil, fmt.Errorf("abrir SQLite: %w", err)
	}
	// Uma conexão é suficiente para o uso local e serializa operações neste processo.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	banco := &Banco{db: db}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("conectar ao SQLite: %w", err)
	}
	if err := banco.migrar(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return banco, nil
}

func (b *Banco) Close() error { return b.db.Close() }

func (b *Banco) migrar(ctx context.Context) error {
	tx, err := b.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("iniciar migração: %w", err)
	}
	defer tx.Rollback()
	var versao int
	if err := tx.QueryRowContext(ctx, "PRAGMA user_version").Scan(&versao); err != nil {
		return fmt.Errorf("consultar versão do banco: %w", err)
	}
	switch versao {
	case 0:
		if _, err := tx.ExecContext(ctx, primeiraMigracao); err != nil {
			return fmt.Errorf("aplicar migração 001: %w", err)
		}
		if _, err := tx.ExecContext(ctx, "PRAGMA user_version = 1"); err != nil {
			return fmt.Errorf("registrar migração 001: %w", err)
		}
	case 1, 2, 3, 4, 5, 6, 7:
		// A migração já foi aplicada. Não recriar nem apagar tabelas.
	default:
		return fmt.Errorf("versão de banco não suportada: %d (máxima: 7)", versao)
	}
	if versao < 2 {
		if err := migrarTurmas(ctx, tx); err != nil {
			return err
		}
	}
	if versao < 3 {
		if _, err := tx.ExecContext(ctx, terceiraMigracao); err != nil {
			return fmt.Errorf("aplicar migração 003: %w", err)
		}
	}
	if versao < 4 {
		if _, err := tx.ExecContext(ctx, quartaMigracao); err != nil {
			return fmt.Errorf("aplicar migração 004: %w", err)
		}
	}
	if versao < 5 {
		if _, err := tx.ExecContext(ctx, quintaMigracao); err != nil {
			return fmt.Errorf("aplicar migração 005: %w", err)
		}
	}
	if versao < 6 {
		if _, err := tx.ExecContext(ctx, sextaMigracao); err != nil {
			return fmt.Errorf("aplicar migração 006: %w", err)
		}
	}
	if versao < 7 {
		if err := migrarConteudos(ctx, tx); err != nil {
			return fmt.Errorf("aplicar migração 007: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("concluir migração: %w", err)
	}
	return nil
}
