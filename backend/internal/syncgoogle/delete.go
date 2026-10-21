package syncgoogle

import (
	"context"

	g "app-sync-backend/internal/google"
	"app-sync-backend/internal/storage"
	"github.com/gofiber/fiber/v3"
)

// Delete só remove uma aula cancelada, sob a mesma exclusão mútua dos envios.
// Se a resposta do Google se perder, o intento impede novos envios até a reconciliação.
func (s *Service) Delete(ctx context.Context, client g.Client, configVersion, id, version int64, removeGoogle bool) error {
	s.Banco.GoogleMu.Lock()
	defer s.Banco.GoogleMu.Unlock()
	c, err := s.Banco.ConfiguracaoGoogle(ctx)
	if err != nil {
		return err
	}
	if c.Versao != configVersion {
		return conflict("O destino mudou. Atualize o histórico antes de excluir")
	}
	r, err := s.Banco.Buscar(ctx, id)
	if err != nil {
		return err
	}
	if r.Versao != version {
		return storage.ErrConflito
	}
	if !r.Cancelada {
		return conflict("Cancele a aula antes de excluí-la definitivamente")
	}
	intent, err := s.Banco.Exclusao(ctx, id)
	if err != nil {
		return err
	}
	var destination *int64
	if c.Destino != nil {
		destination = &c.Destino.ID
	}
	if removeGoogle && destination == nil {
		return fiber.NewError(422, "Configure uma planilha antes de excluir no Google")
	}
	if intent != nil {
		if intent.RemoverGoogle != removeGoogle || (removeGoogle && intent.DestinoID != *destination) {
			return conflict("Retome a exclusão com as mesmas opções da tentativa anterior")
		}
	} else {
		if pending, readErr := s.Banco.AlgumaExclusaoGoogle(ctx); readErr != nil {
			return readErr
		} else if pending {
			return storage.ErrExclusaoPendente
		}
		// Não descarta um intento de envio cujo resultado ainda pode chegar do Google.
		if pending, readErr := s.Banco.AulaTemEnvioIncerto(ctx, id); readErr != nil {
			return readErr
		} else if pending {
			return conflict("Há um envio sem confirmação. Sincronize o cancelamento antes de excluir")
		}
		if destination != nil {
			envios, readErr := s.Banco.EnviosGoogle(ctx, *destination, id)
			if readErr != nil {
				return readErr
			}
			for _, e := range envios {
				if e.Confirmado == "" {
					continue
				}
				previous, decodeErr := g.Decode(e.Confirmado)
				if decodeErr != nil || len(previous.Values) != len(g.Cabecalhos) {
					return conflict("Referência de envio inválida; confira a sincronização")
				}
				if previous.Values[10].String() != "movida" && (previous.Values[10].String() != "cancelada" || e.Versao != r.Versao) {
					return conflict("Sincronize o cancelamento desta aula antes de excluir")
				}
			}
		}
		// A inspeção remota antecede o intento; conflitos conhecidos não deixam a aula bloqueada.
		if removeGoogle {
			if _, err = s.deletions(ctx, client, c, r); err != nil {
				return err
			}
		}
		if err = s.Banco.IniciarExclusao(ctx, id, destination, removeGoogle); err != nil {
			return err
		}
	}
	if removeGoogle {
		requests, readErr := s.deletions(ctx, client, c, r)
		if readErr != nil {
			return readErr
		}
		if len(requests) > 0 {
			if err = client.Batch(c.Destino.PlanilhaID, requests); err != nil {
				return err
			}
		}
		remaining, readErr := s.deletions(ctx, client, c, r)
		if readErr != nil {
			return readErr
		}
		if len(remaining) > 0 {
			return conflict("A remoção no Google não foi confirmada. Tente excluir novamente com a mesma opção")
		}
	}
	return s.Banco.FinalizarExclusao(ctx, id)
}
func (s *Service) deletions(ctx context.Context, client g.Client, c storage.ConfigGoogle, r storage.Registro) ([]any, error) {
	book, err := client.Read(c.Destino.PlanilhaID)
	if err != nil {
		return nil, err
	}
	install, err := s.Banco.InstalacaoGoogle(ctx)
	if err != nil {
		return nil, err
	}
	key := g.Key(install, r.ID)
	envios, err := s.Banco.EnviosGoogle(ctx, c.Destino.ID, r.ID)
	if err != nil {
		return nil, err
	}
	var requests []any
	for _, sh := range book.Sheets {
		index, current, locateErr := locate(sh, key)
		if locateErr != nil {
			return nil, locateErr
		}
		if index < 0 {
			continue
		}
		found := false
		for _, e := range envios {
			v, readErr := s.Banco.VinculoGoogle(ctx, c.Destino.ID, e.Periodo)
			if readErr != nil {
				return nil, readErr
			}
			if v.SheetID == nil || *v.SheetID != sh.Properties.ID {
				continue
			}
			name, readErr := period(e.Periodo)
			if readErr != nil {
				return nil, readErr
			}
			state, _ := sh.State()
			if name != sh.Properties.Title || state != "pronta" || v.Modelo != g.Modelo {
				return nil, conflict("A estrutura da aba mudou. Restaure a aba antes de excluir")
			}
			expected, decodeErr := g.Decode(e.Confirmado)
			if decodeErr != nil || !g.Equal(expected, current) {
				return nil, conflict("A linha foi alterada manualmente. Restaure o último conteúdo confirmado antes de excluir")
			}
			found = true
			break
		}
		if !found {
			return nil, conflict("A chave da aula está em uma aba sem vínculo confirmado. Revise a planilha antes de excluir")
		}
		// Reserva inclusive posições antigas da migração v5 antes de limpá-las.
		for _, e := range envios {
			v, readErr := s.Banco.VinculoGoogle(ctx, c.Destino.ID, e.Periodo)
			if readErr != nil {
				return nil, readErr
			}
			if v.SheetID != nil && *v.SheetID == sh.Properties.ID {
				if err = s.Banco.GarantirLinhaGoogle(ctx, c.Destino.ID, e.Periodo, len(sh.Rows())); err != nil {
					return nil, err
				}
				break
			}
		}
		requests = append(requests, g.ClearRecord(sh.Properties.ID, index))
	}
	return requests, nil
}
