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
	if len(os.Args) != 2 {
		return fmt.Errorf("uso: go run ./cmd/db migrar|inserir-exemplo|listar")
	}
	comando := os.Args[1]
	if comando != "migrar" && comando != "inserir-exemplo" && comando != "listar" {
		return fmt.Errorf("comando desconhecido: %s", comando)
	}
	ctx := context.Background()
	banco, err := storage.Abrir(ctx, os.Getenv("DATABASE_PATH"))
	if err != nil {
		return err
	}
	defer banco.Close()
	switch comando {
	case "migrar":
		fmt.Println("Banco pronto: migração 001 aplicada ou já existente.")
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
