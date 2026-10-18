// Comando de estudo da persistência. Não cadastra aulas pela API.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"app-sync-backend/internal/aula"
	"app-sync-backend/internal/storage"
)

func main() {
	if err := executar(); err != nil {
		log.Fatal(err)
	}
}

func executar() error {
	if len(os.Args) < 2 {
		return fmt.Errorf("uso: go run ./cmd/db migrar|inserir-exemplo|listar|verificar|backup DESTINO")
	}
	comando := os.Args[1]
	if comando != "migrar" && comando != "inserir-exemplo" && comando != "listar" && comando != "backup" && comando != "verificar" {
		return fmt.Errorf("comando desconhecido: %s", comando)
	}
	if (comando == "backup" && len(os.Args) != 3) || (comando != "backup" && len(os.Args) != 2) {
		return fmt.Errorf("argumentos inválidos; backup exige um arquivo de destino")
	}
	if comando == "backup" || comando == "verificar" {
		caminho := os.Getenv("DATABASE_PATH")
		if caminho == "" {
			caminho = storage.CaminhoPadrao
		}
		if _, err := os.Stat(caminho); err != nil {
			return fmt.Errorf("o banco de origem deve existir: %w", err)
		}
	}
	ctx := context.Background()
	banco, err := storage.Abrir(ctx, os.Getenv("DATABASE_PATH"))
	if err != nil {
		return err
	}
	defer banco.Close()
	switch comando {
	case "backup":
		if err := banco.Backup(ctx, os.Args[2]); err != nil {
			return err
		}
		fmt.Println("Backup concluído.")
	case "verificar":
		if err := banco.Verificar(ctx); err != nil {
			return err
		}
		fmt.Println("Integridade e referências verificadas.")
	case "migrar":
		fmt.Println("Banco pronto: migrações até a versão 4 aplicadas.")
	case "inserir-exemplo":
		registro, err := banco.Inserir(ctx, aula.Aula{
			Data: "2026-10-13", Inicio: "08:00", Fim: "10:00", Turma: "Turma fictícia",
			Conteudo: "Leitura de demonstração", Observacoes: "Exemplo da etapa 5", IntervaloMinutos: 15,
		})
		if err != nil {
			return err
		}
		fmt.Printf("Aula fictícia inserida: ID %d, duração %d minutos.\n", registro.ID, registro.DuracaoMinutos)
	case "listar":
		registros, err := banco.Listar(ctx)
		if err != nil {
			return err
		}
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(registros)
	}
	return nil
}
