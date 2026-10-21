package google

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClienteSheetsHTTPMascaraEValoresLiterais(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method == "GET" {
			fields := r.URL.Query().Get("fields")
			if !strings.Contains(fields, "userEnteredValue") || r.URL.Query().Get("ranges") != "" {
				t.Error("leitura parcial da grade")
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"spreadsheetId":"teste","sheets":[{"properties":{"sheetId":0,"title":"Aba","sheetType":"GRID"},"data":[{"startRow":2,"startColumn":4,"rowData":[{"values":[{"userEnteredValue":{"formulaValue":"=\"\""}}]}]}]}]}`))
			return
		}
		if r.URL.Path != "/teste:batchUpdate" {
			t.Error(r.URL.Path)
		}
		var body struct{ Requests []map[string]json.RawMessage }
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if len(body.Requests) != 2 {
			t.Error("esperava criação e cabeçalhos no mesmo batch")
		}
		w.Write([]byte(`{"replies":[{},{}]}`))
	}))
	defer server.Close()
	client := Client{Request: func(method, path string, body, result any) error {
		raw, _ := json.Marshal(body)
		req, _ := http.NewRequest(method, server.URL+"/"+path, strings.NewReader(string(raw)))
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer res.Body.Close()
		return json.NewDecoder(res.Body).Decode(result)
	}}
	book, err := client.Read("teste")
	if err != nil {
		t.Fatal(err)
	}
	state, _ := book.Sheets[0].State()
	if state != "incompativel" {
		t.Fatal("fórmula distante tratada como vazio")
	}
	if err := client.Batch("teste", []any{AddSheet(1, "Registros aulas - 10/2026"), Update(1, 0, Header())}); err != nil {
		t.Fatal(err)
	}
	if requests != 2 {
		t.Fatal(requests)
	}
	row := Row{Values: []Cell{Text("=1+1"), Number(105)}}
	b, _ := json.Marshal(Update(1, 1, row))
	if strings.Contains(string(b), "formulaValue") || !strings.Contains(string(b), "numberValue") {
		t.Fatal("valor não literal")
	}
}
