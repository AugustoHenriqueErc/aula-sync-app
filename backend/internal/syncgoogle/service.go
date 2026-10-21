// Package syncgoogle coordena o SQLite e o Sheets, um envio explícito por aula.
package syncgoogle

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	g "app-sync-backend/internal/google"
	"app-sync-backend/internal/storage"
	"github.com/gofiber/fiber/v3"
)

type confirmation struct {
	Session string
	Config  int64
	Period  string
	State   string
	SheetID *int64
	Expires time.Time
}
type Service struct {
	Banco         *storage.Banco
	confirmations map[string]confirmation
}

func New(b *storage.Banco) *Service {
	return &Service{Banco: b, confirmations: map[string]confirmation{}}
}

type Inspection struct {
	Periodo      string `json:"periodo"`
	Aba          string `json:"aba"`
	Estado       string `json:"estado"`
	Mensagem     string `json:"mensagem"`
	Confirmacao  string `json:"confirmacao,omitempty"`
	SheetID      *int64 `json:"sheet_id"`
	ConfigVersao int64  `json:"config_versao"`
	PlanilhaID   string `json:"planilha_id"`
}

func conflict(s string) error { return fiber.NewError(409, s) }
func (s *Service) config(ctx context.Context, version int64) (storage.ConfigGoogle, error) {
	c, err := s.Banco.ConfiguracaoGoogle(ctx)
	if err != nil {
		return c, err
	}
	if c.Destino == nil {
		return c, fiber.NewError(422, "Salve uma planilha primeiro")
	}
	if c.Versao != version {
		return c, conflict("O destino mudou. Recarregue a configuração e inspecione novamente")
	}
	return c, nil
}
func period(p string) (string, error) {
	if len(p) != 7 {
		return "", fiber.NewError(400, "Período inválido")
	}
	_, name, err := g.Periodo(p + "-01")
	if err != nil {
		return "", fiber.NewError(400, "Período inválido")
	}
	return name, nil
}
func (s *Service) inspect(ctx context.Context, c storage.ConfigGoogle, p string, book g.Spreadsheet) (Inspection, error) {
	name, err := period(p)
	i := Inspection{Periodo: p, Aba: name, Estado: "ausente", Mensagem: "Confirme a criação da aba mensal com os cabeçalhos.", ConfigVersao: c.Versao, PlanilhaID: c.Destino.PlanilhaID}
	if err != nil {
		return i, err
	}
	v, err := s.Banco.VinculoGoogle(ctx, c.Destino.ID, p)
	if err != nil {
		return i, err
	}
	for _, sh := range book.Sheets {
		if sh.Properties.Title == name {
			id := sh.Properties.ID
			i.SheetID = &id
			i.Estado, i.Mensagem = sh.State()
			break
		}
	}
	if v.SheetID != nil && (i.SheetID == nil || *v.SheetID != *i.SheetID || (v.Modelo != 0 && v.Modelo != g.Modelo)) {
		i.Estado = "incompativel"
		i.Mensagem = "A aba vinculada foi excluída, renomeada ou substituída. Restaure a aba original antes de enviar."
	}
	// O ID só é vinculado quando a estrutura está pronta. Vínculos anteriores nunca são redirecionados.
	id := v.SheetID
	if i.Estado == "pronta" {
		id = i.SheetID
	}
	modelo := v.Modelo
	if v.SheetID == nil && i.Estado == "pronta" {
		modelo = g.Modelo
	}
	err = s.Banco.SalvarVinculoGoogle(ctx, c.Destino.ID, storage.VinculoGoogle{Periodo: p, SheetID: id, Estado: i.Estado, Modelo: modelo})
	return i, err
}
func (s *Service) Inspect(ctx context.Context, client g.Client, session string, version int64, p string) (Inspection, error) {
	s.Banco.GoogleMu.Lock()
	defer s.Banco.GoogleMu.Unlock()
	c, err := s.config(ctx, version)
	if err != nil {
		return Inspection{}, err
	}
	if _, err = period(p); err != nil {
		return Inspection{}, err
	}
	book, err := client.Read(c.Destino.PlanilhaID)
	if err != nil {
		return Inspection{}, err
	}
	i, err := s.inspect(ctx, c, p, book)
	if err != nil {
		return i, err
	}
	for k, v := range s.confirmations {
		if time.Now().After(v.Expires) {
			delete(s.confirmations, k)
		}
	}
	if i.Estado == "ausente" || i.Estado == "vazia" {
		if len(s.confirmations) >= 1000 {
			return i, conflict("Muitas inspeções pendentes; aguarde alguns minutos")
		}
		var b [32]byte
		if _, err = rand.Read(b[:]); err != nil {
			return i, err
		}
		i.Confirmacao = hex.EncodeToString(b[:])
		s.confirmations[i.Confirmacao] = confirmation{session, version, p, i.Estado, i.SheetID, time.Now().Add(10 * time.Minute)}
	}
	return i, nil
}
func (s *Service) Prepare(ctx context.Context, client g.Client, session, token string, version int64, p string) (Inspection, error) {
	s.Banco.GoogleMu.Lock()
	defer s.Banco.GoogleMu.Unlock()
	c, err := s.config(ctx, version)
	if err != nil {
		return Inspection{}, err
	}
	confirmation, ok := s.confirmations[token]
	if !ok || confirmation.Session != session || confirmation.Config != version || confirmation.Period != p || time.Now().After(confirmation.Expires) {
		return Inspection{}, conflict("Confirmação expirada ou de outra sessão/destino. Inspecione novamente")
	}
	book, err := client.Read(c.Destino.PlanilhaID)
	if err != nil {
		return Inspection{}, err
	}
	i, err := s.inspect(ctx, c, p, book)
	if err != nil {
		return i, err
	}
	if confirmation.SheetID != nil && (i.SheetID == nil || *confirmation.SheetID != *i.SheetID) {
		return i, conflict("A aba mudou desde a confirmação; inspecione novamente")
	}
	if i.Estado == "pronta" {
		return i, nil
	} // Repetição após perda de resposta não reescreve cabeçalhos.
	if i.Estado != confirmation.State {
		return i, conflict("A estrutura mudou desde a inspeção; inspecione novamente")
	}
	var requests []any
	var id int64
	if i.Estado == "ausente" {
		// IDs explícitos permitem criar aba + cabeçalhos no mesmo batch atômico.
		for _, sh := range book.Sheets {
			if sh.Properties.ID >= id {
				id = sh.Properties.ID + 1
			}
		}
		if id > 2147483647 {
			return i, conflict("Não foi possível reservar o identificador da aba")
		}
		requests = append(requests, g.AddSheet(id, i.Aba))
	} else if i.Estado == "vazia" {
		id = *i.SheetID
		for _, sh := range book.Sheets {
			if sh.Properties.ID == id {
				requests = append(requests, g.Grow(sh, 0)...)
				break
			}
		}
	} else {
		return i, conflict(i.Mensagem)
	}
	requests = append(requests, g.Update(id, 0, g.Header()))
	if err = client.Batch(c.Destino.PlanilhaID, requests); err != nil {
		return i, err
	}
	book, err = client.Read(c.Destino.PlanilhaID)
	if err != nil {
		return i, err
	}
	i, err = s.inspect(ctx, c, p, book)
	if err == nil && i.Estado != "pronta" {
		err = conflict("A preparação não foi confirmada; inspecione novamente")
	}
	return i, err
}
func row(r storage.Registro, key string) g.Row {
	state := "ativa"
	if r.Cancelada {
		state = "cancelada"
	}
	return g.Row{Values: []g.Cell{g.Text(key), g.Number(r.Versao), g.Text(r.Aula.Data), g.Text(r.Aula.Inicio), g.Text(r.Aula.Fim), g.Number(int64(r.Aula.IntervaloMinutos)), g.Text(r.Aula.Turma), g.Text(r.Aula.Conteudo), g.Text(r.Aula.Observacoes), g.Number(int64(r.DuracaoMinutos)), g.Text(state), g.Text(r.UpdatedAt)}}
}
func sheet(book g.Spreadsheet, id int64) (g.Sheet, error) {
	for _, sh := range book.Sheets {
		if sh.Properties.ID == id {
			return sh, nil
		}
	}
	return g.Sheet{}, conflict("Aba vinculada não encontrada")
}
func locate(sh g.Sheet, key string) (int, g.Row, error) {
	index := -1
	var found g.Row
	for n, r := range sh.Rows() {
		if n == 0 || len(r.Values) == 0 {
			continue
		}
		if r.Values[0].String() == key {
			if index >= 0 {
				return 0, g.Row{}, conflict("Chave de aula duplicada na planilha. Revise as linhas antes de enviar")
			}
			index = n
			found = r
		}
	}
	return index, found, nil
}
func findEnvio(envios []storage.EnvioGoogle, p string) storage.EnvioGoogle {
	for _, e := range envios {
		if e.Periodo == p {
			return e
		}
	}
	return storage.EnvioGoogle{Periodo: p}
}

// write verifica a linha pela chave e pela última cópia confirmada, nunca apenas pelo número.
func (s *Service) write(ctx context.Context, client g.Client, c storage.ConfigGoogle, r storage.Registro, p string, wanted g.Row) error {
	book, err := client.Read(c.Destino.PlanilhaID)
	if err != nil {
		return err
	}
	i, err := s.inspect(ctx, c, p, book)
	if err != nil {
		return err
	}
	if i.Estado != "pronta" {
		return conflict(i.Mensagem)
	}
	sh, err := sheet(book, *i.SheetID)
	if err != nil {
		return err
	}
	index, current, err := locate(sh, wanted.Values[0].String())
	if err != nil {
		return err
	}
	envios, err := s.Banco.EnviosGoogle(ctx, c.Destino.ID, r.ID)
	if err != nil {
		return err
	}
	e := findEnvio(envios, p)
	// Uma chave em aba alheia ao histórico local é conflito, não um novo registro.
	for _, other := range book.Sheets {
		if other.Properties.ID == sh.Properties.ID {
			continue
		}
		found, _, scanErr := locate(other, wanted.Values[0].String())
		if scanErr != nil {
			return scanErr
		}
		if found < 0 {
			continue
		}
		known := false
		for _, sent := range envios {
			if sent.Confirmado == "" && sent.Proposto == "" {
				continue
			}
			binding, readErr := s.Banco.VinculoGoogle(ctx, c.Destino.ID, sent.Periodo)
			if readErr != nil {
				return readErr
			}
			if binding.SheetID != nil && *binding.SheetID == other.Properties.ID {
				known = true
				break
			}
		}
		if !known {
			return conflict("A chave desta aula aparece em outra aba sem vínculo local. Revise a duplicação antes de enviar")
		}
	}

	if index >= 0 && g.Equal(current, wanted) {
		if err = s.Banco.IniciarEnvioGoogle(ctx, c.Destino.ID, r.ID, p, g.Encode(wanted), index); err != nil {
			return err
		}
		return s.Banco.ConfirmarEnvioGoogle(ctx, c.Destino.ID, r.ID, p, g.Version(wanted), g.Encode(wanted))
	}
	if index >= 0 {
		previous, parseErr := g.Decode(e.Confirmado)
		if parseErr != nil || !g.Equal(previous, current) {
			return conflict("A linha foi alterada manualmente ou não corresponde ao último envio confirmado. Revise a planilha")
		}
		if g.Version(current) > g.Version(wanted) {
			return conflict("A planilha contém uma versão mais recente; envio bloqueado")
		}
	} else {
		if e.Confirmado != "" {
			return conflict("A linha enviada anteriormente foi removida ou sua chave foi alterada. Restaure-a antes de enviar")
		}
		index = len(sh.Rows())
		if e.Proposto != "" {
			index = e.Linha
			rows := sh.Rows()
			if index < 1 || (index < len(rows) && len(g.Trim(rows[index]).Values) > 0) {
				return conflict("A linha reservada para um envio sem confirmação foi ocupada. Revise a planilha")
			}
		}
		if index < 1 {
			index = 1
		}
	}
	// Um intento desconhecido precisa ser resolvido antes de substituir a proposta.
	if e.Proposto != "" {
		proposed, parseErr := g.Decode(e.Proposto)
		if parseErr != nil || !g.Equal(proposed, wanted) {
			return conflict("Há um envio anterior sem confirmação. Confira o resultado antes de enviar outra versão")
		}
	}
	if err = s.Banco.IniciarEnvioGoogle(ctx, c.Destino.ID, r.ID, p, g.Encode(wanted), index); err != nil {
		return err
	}
	requests := g.Grow(sh, index)
	requests = append(requests, g.Update(*i.SheetID, index, wanted))
	if err = client.Batch(c.Destino.PlanilhaID, requests); err != nil {
		return err
	}
	book, err = client.Read(c.Destino.PlanilhaID)
	if err != nil {
		return err
	}
	i, err = s.inspect(ctx, c, p, book)
	if err != nil {
		return err
	}
	if i.Estado != "pronta" {
		return conflict(i.Mensagem)
	}
	sh, err = sheet(book, *i.SheetID)
	if err != nil {
		return err
	}
	_, current, err = locate(sh, wanted.Values[0].String())
	if err != nil {
		return err
	}
	if !g.Equal(current, wanted) {
		return conflict("O conteúdo enviado não foi confirmado. Inspecione antes de tentar novamente")
	}
	return s.Banco.ConfirmarEnvioGoogle(ctx, c.Destino.ID, r.ID, p, g.Version(wanted), g.Encode(wanted))
}
func (s *Service) Sync(ctx context.Context, client g.Client, configVersion, id, version int64) (result storage.EstadoGoogle, err error) {
	s.Banco.GoogleMu.Lock()
	defer s.Banco.GoogleMu.Unlock()
	c, err := s.config(ctx, configVersion)
	if err != nil {
		return result, err
	}
	r, err := s.Banco.Buscar(ctx, id)
	if err != nil {
		return result, err
	}
	if r.Versao != version {
		return result, storage.ErrConflito
	}
	if pending, readErr := s.Banco.OutroEnvioPendenteGoogle(ctx, c.Destino.ID, id); readErr != nil {
		return result, readErr
	} else if pending != 0 {
		return result, conflict(fmt.Sprintf("O envio da aula %d está sem confirmação. Tente enviá-la novamente antes de continuar", pending))
	}
	p := r.Aula.Data[:7]
	install, err := s.Banco.InstalacaoGoogle(ctx)
	if err != nil {
		return result, err
	}
	key := g.Key(install, id)
	// Cria o vínculo de pendência antes de qualquer tentativa para registrar também falhas de leitura.
	v, err := s.Banco.VinculoGoogle(ctx, c.Destino.ID, p)
	if err != nil {
		return result, err
	}
	if err = s.Banco.SalvarVinculoGoogle(ctx, c.Destino.ID, v); err != nil {
		return result, err
	}
	if err = s.Banco.RegistrarTentativaGoogle(ctx, c.Destino.ID, id, p); err != nil {
		return result, err
	}
	defer func() {
		if err != nil {
			message := "Não foi possível confirmar o envio; inspecione e tente novamente"
			var fe *fiber.Error
			if errors.As(err, &fe) {
				message = fe.Message
			}
			if saveErr := s.Banco.FalhaEnvioGoogle(context.WithoutCancel(ctx), c.Destino.ID, id, p, message); saveErr != nil {
				err = fmt.Errorf("registrar falha de sincronização: %w", saveErr)
			}
		}
	}()
	book, err := client.Read(c.Destino.PlanilhaID)
	if err != nil {
		return result, err
	}
	i, err := s.inspect(ctx, c, p, book)
	if err != nil {
		return result, err
	}
	if i.Estado != "pronta" {
		return result, conflict(i.Mensagem)
	}
	envios, err := s.Banco.EnviosGoogle(ctx, c.Destino.ID, id)
	if err != nil {
		return result, err
	}
	// Recupera intentos, inclusive de uma versão anterior ou de mudança de mês parcialmente concluída.
	for _, e := range envios {
		if e.Proposto != "" {
			proposed, parseErr := g.Decode(e.Proposto)
			if parseErr != nil {
				return result, parseErr
			}
			if err = s.write(ctx, client, c, r, e.Periodo, proposed); err != nil {
				return result, err
			}
		}
	}
	envios, err = s.Banco.EnviosGoogle(ctx, c.Destino.ID, id)
	if err != nil {
		return result, err
	}
	// Antes de ativar o novo mês, retira a representação anterior da totalização.
	for _, e := range envios {
		if e.Periodo == p || e.Confirmado == "" {
			continue
		}
		old, parseErr := g.Decode(e.Confirmado)
		if parseErr != nil {
			return result, parseErr
		}
		if len(old.Values) != len(g.Cabecalhos) {
			return result, conflict("Referência de envio inválida")
		}
		old.Values[1] = g.Number(r.Versao)
		old.Values[10] = g.Text("movida")
		old.Values[11] = g.Text(r.UpdatedAt)
		if err = s.write(ctx, client, c, r, e.Periodo, old); err != nil {
			return result, err
		}
	}
	if err = s.write(ctx, client, c, r, p, row(r, key)); err != nil {
		return result, err
	}
	return s.Banco.EstadoGoogle(ctx, r)
}
