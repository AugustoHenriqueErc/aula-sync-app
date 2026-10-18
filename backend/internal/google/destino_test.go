package google

import "testing"

func TestLinkDaPlanilha(t *testing.T) {
	for _, link := range []string{"https://docs.google.com/spreadsheets/d/teste_A-1/edit?usp=sharing#gid=123", " https://docs.google.com/spreadsheets/d/teste_A-1/ ", "https://docs.google.com/spreadsheets/d/teste_A-1"} {
		id, err := PlanilhaID(link)
		if err != nil || id != "teste_A-1" || URLPlanilha(id) != "https://docs.google.com/spreadsheets/d/teste_A-1/edit" {
			t.Fatalf("%q: %s %v", link, id, err)
		}
	}
	for _, link := range []string{"", "http://docs.google.com/spreadsheets/d/id/edit", "https://docs.google.com.evil.test/spreadsheets/d/id/edit", "https://docs.google.com@evil.test/spreadsheets/d/id/edit", "https://x@docs.google.com/spreadsheets/d/id/edit", "https://docs.google.com:443/spreadsheets/d/id/edit", "https://docs.google.com/document/d/id/edit", "https://docs.google.com/spreadsheets/d/e/id/pubhtml", "https://docs.google.com/spreadsheets/d/id/other", "https://docs.google.com/spreadsheets/d/id%2Fother/edit", "https://docs.google.com/spreadsheets/d//edit"} {
		if _, err := PlanilhaID(link); err == nil {
			t.Fatalf("link inválido aceito: %s", link)
		}
	}
}
func TestPeriodoPelaDataDaAula(t *testing.T) {
	for _, tt := range []struct{ data, periodo, aba string }{{"2026-10-31", "2026-10", "Registros aulas - 10/2026"}, {"2026-11-01", "2026-11", "Registros aulas - 11/2026"}, {"2027-10-01", "2027-10", "Registros aulas - 10/2027"}, {"2024-02-29", "2024-02", "Registros aulas - 02/2024"}, {"0001-01-01", "0001-01", "Registros aulas - 01/0001"}} {
		p, a, err := Periodo(tt.data)
		if err != nil || p != tt.periodo || a != tt.aba {
			t.Fatalf("%s: %s %s %v", tt.data, p, a, err)
		}
	}
	for _, data := range []string{"2026-02-29", "2026-13-01", "0000-01-01", "2026-1-01", "2026-10-18T00:00:00Z"} {
		if _, _, err := Periodo(data); err == nil {
			t.Fatalf("data inválida: %s", data)
		}
	}
}
