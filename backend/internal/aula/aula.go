// Package aula contém as hipóteses de domínio do MVP, sem dependências de HTTP ou banco.
package aula

import (
	"strings"
	"time"
)

// Aula descreve um registro em um único dia, com horários civis no formato HH:MM.
// As regras propostas estão documentadas em docs/dominio-aula.md.
type Aula struct {
	Data             string
	Inicio           string
	Fim              string
	Turma            string
	Conteudo         string
	Observacoes      string
	IntervaloMinutos int
}

// ErroValidacao identifica o primeiro campo que impede o uso do registro.
type ErroValidacao struct {
	Campo    string
	Mensagem string
}

func (e *ErroValidacao) Error() string {
	return e.Campo + ": " + e.Mensagem
}

// Validar verifica todos os campos sem modificar a aula.
func (a Aula) Validar() error {
	_, err := a.DuracaoMinutos()
	return err
}

// DuracaoMinutos valida a aula e retorna os minutos líquidos, sem arredondamento.
// Não consulta o relógio atual nem interpreta os horários como instantes UTC.
func (a Aula) DuracaoMinutos() (int, error) {
	data, err := time.Parse("2006-01-02", a.Data)
	if err != nil || data.Year() < 1 || data.Format("2006-01-02") != a.Data {
		return 0, &ErroValidacao{"Data", "informe uma data válida no formato AAAA-MM-DD, com ano de 0001 a 9999"}
	}
	inicio, err := minutosDoDia(a.Inicio, "Inicio")
	if err != nil {
		return 0, err
	}
	fim, err := minutosDoDia(a.Fim, "Fim")
	if err != nil {
		return 0, err
	}
	if strings.TrimSpace(a.Turma) == "" {
		return 0, &ErroValidacao{"Turma", "informe a turma ou o estudante"}
	}
	if strings.TrimSpace(a.Conteudo) == "" {
		return 0, &ErroValidacao{"Conteudo", "informe o conteúdo da aula"}
	}
	if fim <= inicio {
		return 0, &ErroValidacao{"Fim", "o término deve ser posterior ao início, no mesmo dia"}
	}
	duracaoBruta := fim - inicio
	if a.IntervaloMinutos < 0 || a.IntervaloMinutos >= duracaoBruta {
		return 0, &ErroValidacao{"IntervaloMinutos", "o intervalo deve ser não negativo e menor que a duração bruta da aula"}
	}
	return duracaoBruta - a.IntervaloMinutos, nil
}

func minutosDoDia(value, campo string) (int, error) {
	horario, err := time.Parse("15:04", value)
	if err != nil || horario.Format("15:04") != value {
		return 0, &ErroValidacao{campo, "informe um horário válido no formato HH:MM, entre 00:00 e 23:59"}
	}
	return horario.Hour()*60 + horario.Minute(), nil
}
