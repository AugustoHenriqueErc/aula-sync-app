# Etapa 11 — Preparação e sincronização com Google Sheets

Implementação disponível, com testes automatizados e de interface aprovados em
20/10/2026. A escrita na planilha real ainda precisa ser validada pelo desenvolvedor.
O login e a leitura de `test-aula-sync-app` foram confirmados pelo desenvolvedor
nesta conversa. Isso não comprova permissão de escrita nem aprovação dos cabeçalhos
pela responsável. A implementação da etapa 11 foi autorizada explicitamente;
as pendências institucionais da etapa 10 continuam registradas separadamente.

## Como testar no aplicativo

1. Reinicie o backend no terminal com as variáveis OAuth já configuradas:
   `cd backend` e `go run ./cmd/api` (a partir da raiz). A migração 005 é automática
   e preserva aulas, destinos e identidade da instalação. Reiniciar encerra a sessão.
2. Mantenha o frontend em `http://localhost:5173`; entre com Google novamente.
3. Cadastre uma aula fictícia, por exemplo 18/10/2026, 08:00–10:00, intervalo de
   15 minutos. A duração esperada é 105 minutos.
4. Em **Integração Google**, salve a planilha de teste e clique em **Recarregar
   configuração e prévia**. Confira o link em **Abrir planilha configurada**.
5. No mês correspondente, clique em **Inspecionar aba mensal**. A inspeção lê
   a planilha, mas não grava nela.
6. Se a aba estiver ausente ou completamente vazia, clique em **Confirmar
   preparação de Registros aulas - 10/2026**. Confira o destino no diálogo.
   **Agora não** dispensa a preparação; a aula continua salva localmente.
7. Após aparecer **Cabeçalhos compatíveis**, abra o **Histórico** e clique em
   **Sincronizar com Google** na aula desejada. Confirme o envio.
8. Confira a linha na aba mensal: data, horários, textos literais, duração 105 e
   situação `ativa`. A aba inicial e as demais abas devem permanecer intactas.
9. Repita o envio: deve continuar existindo uma única linha para a aula.
10. Edite a aula, envie novamente e confira a atualização da mesma linha. Cancele,
    use o filtro **Canceladas**, envie e confira `cancelada`; reative e envie de novo.
11. Mude a data para novembro. Inspecione e prepare novembro antes de enviar.
    A linha anterior deve ficar `movida`, e a de novembro, `ativa`.
12. Teste uma planilha sem permissão de edição: o erro deve aparecer, preservando
    a aula local. Corrija a permissão e use **Tentar envio novamente**.
13. Altere manualmente o conteúdo de uma linha já enviada, depois edite a aula
    local e tente enviá-la: deve bloquear a sobrescrita. Restaure o conteúdo
    anterior da planilha para permitir a atualização local.
14. Troque o destino por outra planilha de teste. O histórico deve mostrar uma
    nova pendência. Prepare e envie explicitamente; a planilha anterior é preservada.

Registrar os resultados reais, inclusive a conta com permissão de escrita,
sem credenciais ou dados pessoais. Não marcar a validação remota como concluída
somente porque os testes com simuladores passaram.

## Comportamento e persistência

- O cadastro continua local. Não há envio automático, lote, fila ou retentativa
  automática. A data da aula determina o mês de destino.
- A inspeção lê os valores inseridos de toda a grade, incluindo fórmulas, e exige
  os 12 cabeçalhos exatos do [modelo proposto](google-etapa10.md). Colunas extras
  preenchidas na linha de cabeçalho também bloqueiam o uso.
- Preparação exige uma confirmação aleatória, válida por dez minutos e vinculada
  à sessão, revisão da configuração, período e ação inspecionada. O backend
  reinspeciona antes de escrever; logout/troca de conta ou destino exige nova inspeção.
- A criação da aba e de seus cabeçalhos usa um único `spreadsheets.batchUpdate`.
  Uma preparação repetida reconhece a aba já pronta, sem reescrever seu cabeçalho.
- O vínculo guarda `sheet_id` e versão do modelo. Renomear, excluir ou substituir
  uma aba já vinculada bloqueia o envio; restaure a aba original. Não há redirecionamento
  silencioso para outra aba com o mesmo nome.
- A chave externa é `identidade_da_instalação:ID_da_aula`, dentro de cada planilha.
  A linha é localizada pela chave a cada operação; não se depende de sua posição.
  Duplicações, fórmulas ou alterações manuais na linha a atualizar causam conflito.
- A migração 005 acrescenta o modelo do vínculo, a cópia do conteúdo confirmado,
  a proposta de escrita e a linha reservada. A proposta é salva antes da chamada
  remota e só é removida depois de conferir a leitura de retorno.
- Uma resposta perdida pode ser recuperada por nova tentativa, inclusive após
  reiniciar a API. O backend reconcilia a proposta anterior antes de enviar uma
  versão nova. Enquanto outra aula tiver escrita sem confirmação no destino,
  o envio é bloqueado indicando seu ID; tente primeiro essa aula novamente.
- Mudança de mês marca as representações anteriores como `movida` antes de ativar
  a atual. Uma falha intermediária pode deixar temporariamente nenhuma linha
  ativa no Google. A nova tentativa conclui o fluxo; o total local permanece correto.
- Cancelamentos preservam a linha com `cancelada`. Somente `ativa` deve entrar
  nas somas feitas na planilha; a coluna de duração mantém a duração original.
- Trocar a planilha mantém os vínculos e envios anteriores, sem herdar o estado
  de outra planilha. Voltar a um destino conhecido recupera seu histórico.
- Cada ação explícita de sincronização conta como tentativa, inclusive falhas
  de leitura. O histórico mostra estado, última tentativa e erro do destino atual.
  **Último envio confirmado** descreve a última conferência, não monitoramento contínuo.

## Limites deste primeiro corte

O bloqueio serializa operações de uma instância do backend, incluindo troca de
configuração e correções locais durante um envio. Execute uma única instância
por banco/identidade. A API do Sheets não oferece, neste fluxo, uma comparação
atômica do conteúdo lido com a escrita posterior: alterações feitas simultaneamente
por outros editores ou processos entre essas chamadas não têm a mesma garantia
que alterações já presentes na inspeção. Evite editar manualmente durante o envio.
Um timeout também não cancela comprovadamente uma operação já recebida pelo Google;
a recuperação verifica e reutiliza o intento, mas não oferece uma transação
atômica distribuída entre SQLite e Sheets.

As chamadas ao Google usam o timeout de oito segundos do cliente OAuth existente,
sem retentativas automáticas. A leitura aceita até 16 MiB de resposta; exceder
esse limite ou receber JSON incompleto bloqueia a operação, nunca classifica a
planilha como vazia. A abordagem de leitura integral prioriza o MVP e não se
propõe a sincronizar planilhas grandes.

O modelo permanece uma proposta institucional. Contas continuam compartilhando
a instalação local conforme a etapa 10; isolamento por usuária e publicação são
trabalhos da etapa 13. O relatório PDF pertence à etapa 12.

## Contrato HTTP

Todas as escritas exigem sessão com escopo Sheets e Origin correto. Tokens ficam
apenas no backend. O modo local sem OAuth não permite operações remotas.

| Método e rota | Corpo JSON |
| --- | --- |
| `POST /integracoes/google/inspecionar` | `{"config_versao":1,"periodo":"2026-10"}` |
| `POST /integracoes/google/preparar` | `{"config_versao":1,"periodo":"2026-10","confirmacao":"token_da_inspecao"}` |
| `POST /aulas/1/sincronizar` | `{"config_versao":1,"versao":1}` |

Inspeção/preparação respondem `periodo`, `aba`, `estado`, `mensagem`, `sheet_id`,
`config_versao`, `planilha_id` e, quando necessário, `confirmacao`.
Sincronização responde o estado local efetivo. O `GET /aulas` acrescenta a cada
item `google`: estado, destino, versão da configuração, período, tentativas,
última tentativa e erro. Endpoints individuais de cadastro/consulta conservam
seu contrato anterior.

Respostas: 400 entrada inválida; 401 sessão ausente/revogada; 403 falta de escopo,
origem ou acesso; 404 aula ausente; 409 conflito ou resultado remoto não confirmado;
422 ausência de configuração/login; 500 falha interna sem conteúdo sensível.
Não interpretar 403/404 do Google como ausência de aba.

## Verificação automatizada

Em `backend/`:

```bash
go test -race ./...
go vet ./...
go build ./...
```

Em `frontend/`:

```bash
npm run check
npm run build
npm run format:check
```

Na raiz, com Chrome instalado:

```bash
node scripts/test-etapa9.cjs --sync
```

Os testes Go cobrem transporte HTTP, OAuth da sessão, erros de permissão,
migração v4 → v5, inspeção com fórmula distante, criação confirmada, repetição,
concorrência, perda de resposta, recuperação, correções/cancelamentos, conflitos,
troca de destino e reconciliação de mês/ano. Usam SQLite temporário e simulador
do Sheets, sem credenciais reais. O Chrome testa o fluxo local existente e os
controles novos com respostas de sincronização simuladas no navegador.

Referências consultadas: [requisições em lote](https://developers.google.com/workspace/sheets/api/guides/batch),
[leitura e escrita de células](https://developers.google.com/workspace/sheets/api/guides/values)
e [operações de abas](https://developers.google.com/workspace/sheets/api/samples/sheet).
