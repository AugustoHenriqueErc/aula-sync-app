package storage

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"strings"

	"app-sync-backend/internal/aula"
)

//go:embed migrations/007_conteudos.sql
var setimaMigracao string

var ErrConteudoDuplicado = errors.New("Já existe um conteúdo com esse nome")

type Conteudo struct {
	ID   int64  `json:"id"`
	Nome string `json:"nome"`
}

func migrarConteudos(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, setimaMigracao); err != nil {
		return err
	}
	rows, err := tx.QueryContext(ctx, "SELECT conteudo FROM aulas ORDER BY id")
	if err != nil {
		return err
	}
	var nomes []string
	for rows.Next() {
		var nome string
		if err := rows.Scan(&nome); err != nil {
			rows.Close()
			return err
		}
		if nome = strings.TrimSpace(nome); nome != "" {
			nomes = append(nomes, nome)
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, nome := range nomes {
		if _, err := tx.ExecContext(ctx, "INSERT INTO conteudos(nome,nome_normalizado) VALUES (?,?) ON CONFLICT(nome_normalizado) DO NOTHING", nome, normalizarNome(nome)); err != nil {
			return err
		}
	}
	_, err = tx.ExecContext(ctx, "PRAGMA user_version = 7")
	return err
}

func (b *Banco) CriarConteudo(ctx context.Context, nome string) (Conteudo, error) {
	nome = strings.TrimSpace(nome)
	if nome == "" {
		return Conteudo{}, &aula.ErroValidacao{Campo: "Nome", Mensagem: "Informe o nome do conteúdo"}
	}
	result, err := b.db.ExecContext(ctx, "INSERT INTO conteudos(nome,nome_normalizado) VALUES (?,?) ON CONFLICT(nome_normalizado) DO NOTHING", nome, normalizarNome(nome))
	if err != nil {
		return Conteudo{}, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return Conteudo{}, err
	}
	if n == 0 {
		return Conteudo{}, ErrConteudoDuplicado
	}
	id, err := result.LastInsertId()
	return Conteudo{ID: id, Nome: nome}, err
}

func (b *Banco) ListarConteudos(ctx context.Context) ([]Conteudo, error) {
	rows, err := b.db.QueryContext(ctx, "SELECT id,nome FROM conteudos ORDER BY nome_normalizado,id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	conteudos := make([]Conteudo, 0)
	for rows.Next() {
		var conteudo Conteudo
		if err := rows.Scan(&conteudo.ID, &conteudo.Nome); err != nil {
			return nil, err
		}
		conteudos = append(conteudos, conteudo)
	}
	return conteudos, rows.Err()
}
