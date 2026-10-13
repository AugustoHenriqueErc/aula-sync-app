package aula

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func aulaExemplo() Aula {
	return Aula{
		Data: "2026-10-13", Inicio: "08:00", Fim: "09:00",
		Turma: "Turma de demonstração", Conteudo: "Leitura",
	}
}

func TestDuracaoMinutos(t *testing.T) {
	tests := []struct {
		nome, data, inicio, fim string
		intervalo, esperado     int
	}{
		{"aula comum", "2026-10-13", "08:00", "09:00", 0, 60},
		{"com intervalo", "2026-10-13", "08:00", "10:00", 15, 105},
		{"sem arredondamento", "2026-10-13", "08:07", "09:02", 0, 55},
		{"duração mínima", "2026-10-13", "08:00", "08:01", 0, 1},
		{"intervalo deixa um minuto", "2026-10-13", "08:00", "09:00", 59, 1},
		{"limites do dia", "2026-10-13", "00:00", "23:59", 0, 1439},
		{"ano bissexto", "2024-02-29", "08:00", "09:00", 0, 60},
		{"século bissexto", "2000-02-29", "08:00", "09:00", 0, 60},
		{"ano mínimo", "0001-01-01", "08:00", "09:00", 0, 60},
		{"ano máximo", "9999-12-31", "08:00", "09:00", 0, 60},
	}
	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			a := aulaExemplo()
			a.Data, a.Inicio, a.Fim, a.IntervaloMinutos = tt.data, tt.inicio, tt.fim, tt.intervalo
			if err := a.Validar(); err != nil {
				t.Fatalf("aula válida rejeitada: %v", err)
			}
			got, err := a.DuracaoMinutos()
			if err != nil || got != tt.esperado {
				t.Fatalf("duração = %d, %v; esperado %d minutos", got, err, tt.esperado)
			}
		})
	}
}

func TestAulaInvalida(t *testing.T) {
	tests := []struct {
		nome, campo string
		alterar     func(*Aula)
	}{
		{"data ausente", "Data", func(a *Aula) { a.Data = "" }},
		{"data inexistente", "Data", func(a *Aula) { a.Data = "2026-02-29" }},
		{"século não bissexto", "Data", func(a *Aula) { a.Data = "1900-02-29" }},
		{"dia inexistente", "Data", func(a *Aula) { a.Data = "2026-04-31" }},
		{"mês inválido", "Data", func(a *Aula) { a.Data = "2026-13-01" }},
		{"ano zero", "Data", func(a *Aula) { a.Data = "0000-01-01" }},
		{"ano fora do limite", "Data", func(a *Aula) { a.Data = "10000-01-01" }},
		{"formato brasileiro", "Data", func(a *Aula) { a.Data = "13/10/2026" }},
		{"data sem zeros", "Data", func(a *Aula) { a.Data = "2026-1-1" }},
		{"data com espaço", "Data", func(a *Aula) { a.Data = "2026-10-13 " }},
		{"início ausente", "Inicio", func(a *Aula) { a.Inicio = "" }},
		{"início sem zero", "Inicio", func(a *Aula) { a.Inicio = "8:00" }},
		{"início com segundos", "Inicio", func(a *Aula) { a.Inicio = "08:00:00" }},
		{"início com espaço", "Inicio", func(a *Aula) { a.Inicio = " 08:00" }},
		{"início com fuso", "Inicio", func(a *Aula) { a.Inicio = "08:00Z" }},
		{"minuto inválido", "Inicio", func(a *Aula) { a.Inicio = "08:60" }},
		{"hora inválida", "Inicio", func(a *Aula) { a.Inicio = "24:00" }},
		{"término ausente", "Fim", func(a *Aula) { a.Fim = "" }},
		{"término sem zero", "Fim", func(a *Aula) { a.Fim = "9:00" }},
		{"término com segundos", "Fim", func(a *Aula) { a.Fim = "09:00:00" }},
		{"término inválido", "Fim", func(a *Aula) { a.Fim = "24:00" }},
		{"término igual", "Fim", func(a *Aula) { a.Fim = "08:00" }},
		{"término anterior", "Fim", func(a *Aula) { a.Fim = "07:59" }},
		{"atravessa meia-noite", "Fim", func(a *Aula) { a.Inicio = "23:00"; a.Fim = "01:00" }},
		{"turma ausente", "Turma", func(a *Aula) { a.Turma = "" }},
		{"turma em branco", "Turma", func(a *Aula) { a.Turma = " \t\n" }},
		{"conteúdo ausente", "Conteudo", func(a *Aula) { a.Conteudo = "" }},
		{"conteúdo em branco", "Conteudo", func(a *Aula) { a.Conteudo = " \t\n" }},
		{"intervalo negativo", "IntervaloMinutos", func(a *Aula) { a.IntervaloMinutos = -1 }},
		{"intervalo igual à duração", "IntervaloMinutos", func(a *Aula) { a.IntervaloMinutos = 60 }},
		{"intervalo maior que duração", "IntervaloMinutos", func(a *Aula) { a.IntervaloMinutos = 61 }},
	}
	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			a := aulaExemplo()
			tt.alterar(&a)
			got, durationErr := a.DuracaoMinutos()
			if got != 0 {
				t.Errorf("aula inválida retornou duração %d", got)
			}
			for _, err := range []error{a.Validar(), durationErr} {
				var validationErr *ErroValidacao
				if !errors.As(err, &validationErr) {
					t.Fatalf("esperado ErroValidacao, recebido %v", err)
				}
				if validationErr.Campo != tt.campo {
					t.Errorf("campo = %q; esperado %q", validationErr.Campo, tt.campo)
				}
				if validationErr.Mensagem == "" || !strings.Contains(err.Error(), tt.campo) {
					t.Errorf("erro sem contexto: %v", err)
				}
			}
		})
	}
}

func TestCalculoNaoModificaAula(t *testing.T) {
	a := aulaExemplo()
	a.Turma = "  Turma de demonstração  "
	a.Conteudo = "  Leitura  "
	a.Observacoes = "Observação opcional preservada."
	original := a
	for i := 0; i < 2; i++ {
		got, err := a.DuracaoMinutos()
		if err != nil || got != 60 {
			t.Fatalf("duração = %d, %v; esperado 60", got, err)
		}
		if err := a.Validar(); err != nil {
			t.Fatal(err)
		}
	}
	if a != original {
		t.Fatal("o cálculo modificou os dados da aula")
	}
	a.Fim = "10:00"
	got, err := a.DuracaoMinutos()
	if err != nil || got != 120 {
		t.Fatalf("duração após alterar horário = %d, %v; esperado 120", got, err)
	}
}

func TestCalculoIndependeDoFusoLocal(t *testing.T) {
	original := time.Local
	t.Cleanup(func() { time.Local = original })
	// Não usar t.Parallel: este teste modifica a configuração global de fuso.
	for _, offset := range []int{-3 * 3600, 0, 14 * 3600} {
		time.Local = time.FixedZone("fuso de teste", offset)
		a := aulaExemplo()
		a.Data, a.Inicio, a.Fim = "2018-11-04", "00:00", "02:00"
		got, err := a.DuracaoMinutos()
		if err != nil || got != 120 {
			t.Fatalf("fuso %d: duração = %d, %v; esperado 120", offset, got, err)
		}
	}
}
