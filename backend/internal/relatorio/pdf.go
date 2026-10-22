// Package relatorio apresenta os dados locais em um modelo fixo de PDF.
package relatorio

import (
	"bytes"
	"fmt"
	"strings"
	"time"
	"unicode"

	"app-sync-backend/internal/storage"

	"github.com/go-pdf/fpdf"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
)

func duracao(m int64) string { return fmt.Sprintf("%dh%02d", m/60, m%60) }
func data(s string) string {
	d, err := time.Parse("2006-01-02", s)
	if err != nil {
		return s
	}
	return d.Format("02/01/2006")
}

// PDF não consulta serviços externos nem altera os registros recebidos.
func PDF(h storage.Historico, f storage.Filtro) ([]byte, error) {
	p := fpdf.New("P", "mm", "A4", "")
	p.SetMargins(16, 16, 16)
	p.SetAutoPageBreak(true, 20)
	p.AddUTF8FontFromBytes("Go", "", goregular.TTF)
	p.AddUTF8FontFromBytes("Go", "B", gobold.TTF)
	p.SetTitle("Relatório de aulas", true)
	p.SetAuthor("Aula Sync App", true)
	p.AliasNbPages("")
	p.SetHeaderFunc(func() {
		p.SetTextColor(30, 52, 70)
		p.SetFont("Go", "B", 16)
		p.CellFormat(0, 9, "Relatório de aulas", "", 1, "L", false, 0, "")
		p.SetFont("Go", "", 9)
		p.CellFormat(0, 6, "Aula Sync App · Registros locais", "", 1, "L", false, 0, "")
		p.Ln(4)
	})
	p.SetFooterFunc(func() {
		p.SetY(-14)
		p.SetTextColor(90, 90, 90)
		p.SetFont("Go", "", 9)
		p.CellFormat(0, 5, fmt.Sprintf("Página %d de {nb}", p.PageNo()), "", 0, "R", false, 0, "")
	})
	p.AddPage()
	// MultiCell quebra inclusive palavras extensas e continua em outras páginas.
	texto := func(s string) {
		p.SetFont("Go", "", 10)
		p.SetTextColor(35, 35, 35)
		// Mantém quebras de linha; remove controles sem representação impressa.
		s = strings.Map(func(r rune) rune {
			if r == '\t' {
				return ' '
			}
			if unicode.IsControl(r) && r != '\n' {
				return -1
			}
			return r
		}, s)
		p.MultiCell(0, 5.5, s, "", "L", false)
	}
	inicial, final := "sem limite inicial", "sem limite final"
	if f.DataInicial != "" {
		inicial = data(f.DataInicial)
	}
	if f.DataFinal != "" {
		final = data(f.DataFinal)
	}
	texto("Período: " + inicial + " a " + final)
	turma := "Todas as turmas e estudantes"
	if f.TurmaID > 0 {
		turma = fmt.Sprintf("ID %d", f.TurmaID)
		if len(h.Aulas) > 0 {
			turma = h.Aulas[0].Aula.Turma
		}
	}
	texto("Turma/estudante: " + turma)
	situacao := "Ativas"
	if f.Situacao == "todas" {
		situacao = "Todas"
	}
	if f.Situacao == "canceladas" {
		situacao = "Canceladas"
	}
	texto("Situação: " + situacao)
	texto(fmt.Sprintf("%d aula(s) · Carga horária total: %s", h.TotalRegistros, duracao(h.DuracaoTotalMinutos)))
	texto("Aulas canceladas não entram na carga horária total.")
	p.Ln(4)
	if len(h.Aulas) == 0 {
		texto("Nenhuma aula encontrada para os filtros aplicados.")
	}
	for _, r := range h.Aulas {
		if p.GetY() > 245 {
			p.AddPage()
		}
		p.SetDrawColor(200, 210, 215)
		p.Line(16, p.GetY(), 194, p.GetY())
		p.Ln(3)
		texto(fmt.Sprintf("%s · Aula #%d · %s", data(r.Aula.Data), r.ID, r.Aula.Turma))
		if r.Cancelada {
			texto("CANCELADA · Fora da carga horária total")
		}
		texto(fmt.Sprintf("Horário: %s às %s · Intervalo: %d min · Duração: %s", r.Aula.Inicio, r.Aula.Fim, r.Aula.IntervaloMinutos, duracao(int64(r.DuracaoMinutos))))
		texto("Conteúdo: " + r.Aula.Conteudo)
		if r.Aula.Observacoes != "" {
			texto("Observações: " + r.Aula.Observacoes)
		}
		p.Ln(4)
	}
	var buf bytes.Buffer
	err := p.Output(&buf)
	return buf.Bytes(), err
}
