package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"app-sync-backend/internal/httpapi"
)

func main() {
	port, err := serverPort(os.Getenv("PORT"))
	if err != nil {
		log.Fatal(err)
	}
	log.Fatal(httpapi.New().Listen(":" + port))
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
