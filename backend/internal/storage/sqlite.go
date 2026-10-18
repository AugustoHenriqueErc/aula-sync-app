// Package storage persiste aulas em SQLite, sem depender da camada HTTP.
package storage

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"net/url"
	"path/filepath"

	_ "modernc.org/sqlite"
)

const CaminhoPadrao = "aulas.db"

//go:embed migrations/001_create_aulas.sql
var primeiraMigracao string

type Banco struct{ db *sql.DB }

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
	case 1, 2:
		// A migração já foi aplicada. Não recriar nem apagar tabelas.
	default:
		return fmt.Errorf("versão de banco não suportada: %d (máxima: 2)", versao)
	}
	if versao < 2 {
		if err := migrarTurmas(ctx, tx); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("concluir migração: %w", err)
	}
	return nil
}
