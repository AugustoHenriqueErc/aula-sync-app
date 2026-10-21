package google

import (
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"strconv"
)

var Cabecalhos = []string{"chave_aula", "versao", "data", "inicio", "fim", "intervalo_minutos", "turma_estudante", "conteudo", "observacoes", "duracao_minutos", "situacao", "updated_at"}

const Modelo = 1

type Cell struct {
	Value map[string]any `json:"userEnteredValue,omitempty"`
}
type Row struct {
	Values []Cell `json:"values"`
}
type Sheet struct {
	Properties struct {
		ID    int64  `json:"sheetId"`
		Title string `json:"title"`
		Type  string `json:"sheetType"`
		Grid  struct {
			Rows    int `json:"rowCount"`
			Columns int `json:"columnCount"`
		} `json:"gridProperties"`
	} `json:"properties"`
	Data []struct {
		StartRow    int   `json:"startRow"`
		StartColumn int   `json:"startColumn"`
		Rows        []Row `json:"rowData"`
	} `json:"data"`
}
type Spreadsheet struct {
	ID     string  `json:"spreadsheetId"`
	Sheets []Sheet `json:"sheets"`
}

// Request é a fronteira HTTP: autenticação continua no backend, nunca no navegador.
type Client struct {
	Request func(method, path string, body, result any) error
}

func (c Client) Read(id string) (Spreadsheet, error) {
	var p Spreadsheet
	fields := "spreadsheetId,sheets(properties(sheetId,title,sheetType,gridProperties),data(startRow,startColumn,rowData(values(userEnteredValue))))"
	err := c.Request("GET", url.PathEscape(id)+"?fields="+url.QueryEscape(fields), nil, &p)
	if err == nil && p.ID != id {
		err = fmt.Errorf("resposta inesperada da planilha")
	}
	return p, err
}
func (c Client) Batch(id string, requests []any) error {
	var result map[string]any
	return c.Request("POST", url.PathEscape(id)+":batchUpdate", map[string]any{"requests": requests}, &result)
}
func Text(s string) Cell      { return Cell{map[string]any{"stringValue": s}} }
func Number(n int64) Cell     { return Cell{map[string]any{"numberValue": float64(n)}} }
func (c Cell) String() string { v, _ := c.Value["stringValue"].(string); return v }
func (c Cell) Empty() bool {
	return len(c.Value) == 0 || (len(c.Value) == 1 && c.String() == "" && c.Value["stringValue"] != nil)
}
func (s Sheet) Rows() []Row {
	var result []Row
	for _, d := range s.Data {
		for i, row := range d.Rows {
			n := d.StartRow + i
			for len(result) <= n {
				result = append(result, Row{})
			}
			for len(result[n].Values) < d.StartColumn+len(row.Values) {
				result[n].Values = append(result[n].Values, Cell{})
			}
			copy(result[n].Values[d.StartColumn:], row.Values)
		}
	}
	return result
}
func Trim(row Row) Row {
	for len(row.Values) > 0 && row.Values[len(row.Values)-1].Empty() {
		row.Values = row.Values[:len(row.Values)-1]
	}
	return row
}
func Equal(a, b Row) bool {
	a = Trim(a)
	b = Trim(b)
	if len(a.Values) != len(b.Values) {
		return false
	}
	for i := range a.Values {
		if a.Values[i].Empty() && b.Values[i].Empty() {
			continue
		}
		if !reflect.DeepEqual(a.Values[i].Value, b.Values[i].Value) {
			return false
		}
	}
	return true
}
func Header() Row {
	r := Row{}
	for _, h := range Cabecalhos {
		r.Values = append(r.Values, Text(h))
	}
	return r
}
func (s Sheet) State() (string, string) {
	if s.Properties.Type != "GRID" {
		return "incompativel", "A aba precisa ser uma grade comum."
	}
	rows := s.Rows()
	empty := true
	for _, r := range rows {
		if len(Trim(r).Values) > 0 {
			empty = false
			break
		}
	}
	if empty {
		return "vazia", "A aba está vazia; confirme a criação dos cabeçalhos."
	}
	if len(rows) == 0 || !Equal(rows[0], Header()) {
		return "incompativel", "Cabeçalhos diferentes do modelo. Esperados, na ordem: " + fmt.Sprint(Cabecalhos)
	}
	return "pronta", "Cabeçalhos compatíveis. A escrita será verificada durante a operação."
}
func Update(sheetID int64, index int, row Row) any {
	return map[string]any{"updateCells": map[string]any{"start": map[string]any{"sheetId": sheetID, "rowIndex": index, "columnIndex": 0}, "rows": []Row{row}, "fields": "userEnteredValue"}}
}
func AddSheet(id int64, title string) any {
	return map[string]any{"addSheet": map[string]any{"properties": map[string]any{"sheetId": id, "title": title, "gridProperties": map[string]any{"rowCount": 1000, "columnCount": len(Cabecalhos)}}}}
}
func Grow(s Sheet, index int) []any {
	var req []any
	if index >= s.Properties.Grid.Rows {
		req = append(req, map[string]any{"appendDimension": map[string]any{"sheetId": s.Properties.ID, "dimension": "ROWS", "length": index - s.Properties.Grid.Rows + 1}})
	}
	if s.Properties.Grid.Columns < len(Cabecalhos) {
		req = append(req, map[string]any{"appendDimension": map[string]any{"sheetId": s.Properties.ID, "dimension": "COLUMNS", "length": len(Cabecalhos) - s.Properties.Grid.Columns}})
	}
	return req
}
func Encode(r Row) string          { b, _ := json.Marshal(r); return string(b) }
func Decode(s string) (Row, error) { var r Row; err := json.Unmarshal([]byte(s), &r); return r, err }
func Version(r Row) int64 {
	if len(r.Values) < 2 {
		return 0
	}
	n, ok := r.Values[1].Value["numberValue"].(float64)
	if !ok {
		return 0
	}
	i := int64(n)
	if float64(i) != n {
		return 0
	}
	return i
}
func Key(install string, id int64) string { return install + ":" + strconv.FormatInt(id, 10) }
