package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"app-sync-backend/internal/httpapi"
	"app-sync-backend/internal/storage"

	"github.com/gofiber/fiber/v3"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))
	if err := executar(); err != nil {
		log.Fatal(err)
	}
}

func executar() error {
	port, err := serverPort(os.Getenv("PORT"))
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	banco, err := storage.Abrir(ctx, os.Getenv("DATABASE_PATH"))
	if err != nil {
		return err
	}
	defer banco.Close()

	app := httpapi.New(banco)
	encerrado := make(chan error, 1)
	app.Hooks().OnPostShutdown(func(err error) error {
		encerrado <- err
		return nil
	})
	err = app.Listen(":"+port, fiber.ListenConfig{
		GracefulContext: ctx, ShutdownTimeout: 5 * time.Second,
	})
	if err != nil {
		return err
	}
	// Listen pode retornar antes do término das requisições em andamento.
	// Aguardar o hook evita fechar o SQLite antes do encerramento do servidor.
	return <-encerrado
}

func serverPort(value string) (string, error) {
	if value == "" {
		return "3000", nil
	}
	number, err := strconv.Atoi(value)
	if err != nil || number < 1 || number > 65535 {
		return "", fmt.Errorf("PORT deve ser um número inteiro entre 1 e 65535")
	}
	return strconv.Itoa(number), nil
}
