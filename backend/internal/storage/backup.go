package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// Backup cria um snapshot consistente, inclusive quando a API usa o banco.
// Reserva o destino com O_EXCL: nunca sobrescreve uma cópia existente.
func (b *Banco) Backup(ctx context.Context, destino string) error {
	if destino == "" {
		return fmt.Errorf("informe o arquivo de destino")
	}
	absoluto, err := filepath.Abs(destino)
	if err != nil {
		return err
	}
	arquivo, err := os.OpenFile(absoluto, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("reservar destino do backup: %w", err)
	}
	if err := arquivo.Close(); err != nil {
		return err
	}
	// Em caso de falha, o destino pode estar incompleto; não anunciar sucesso.
	if _, err := b.db.ExecContext(ctx, "VACUUM INTO ?", absoluto); err != nil {
		return fmt.Errorf("backup incompleto (não utilizar o destino): %w", err)
	}
	arquivo, err = os.OpenFile(absoluto, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer arquivo.Close()
	if err := arquivo.Sync(); err != nil {
		return fmt.Errorf("sincronizar backup: %w", err)
	}
	return nil
}
func (b *Banco) Verificar(ctx context.Context) error {
	var resultado string
	if err := b.db.QueryRowContext(ctx, "PRAGMA integrity_check").Scan(&resultado); err != nil {
		return err
	}
	if resultado != "ok" {
		return fmt.Errorf("o banco falhou na verificação de integridade")
	}
	var violacoes int
	if err := b.db.QueryRowContext(ctx, "SELECT count(*) FROM pragma_foreign_key_check").Scan(&violacoes); err != nil {
		return err
	}
	if violacoes != 0 {
		return fmt.Errorf("o banco contém referências inválidas")
	}
	return nil
}
