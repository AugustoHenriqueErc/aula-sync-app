# Ajustes da etapa 11 — reenvio, cancelamento, exclusão e destino

Comportamento aprovado pelo desenvolvedor em 21/10/2026. A exclusão passa a ser
definitiva, substituindo a restrição anterior de somente preservar/cancelar aulas.
O relatório PDF continua na etapa 12.

## Reenvio da mesma aula

Uma aula cuja versão atual já foi confirmada no destino atual não será enviada
novamente. A interface informa:

> Esta aula já está na planilha. Nenhum novo registro foi enviado.

O backend também faz essa verificação, sem chamar o Google nem incrementar o
contador de tentativas. A resposta mantém `estado: sincronizado` e acrescenta
`ja_sincronizada: true` e `mensagem`. Uma aula editada ou cancelada tem outra versão
e pode atualizar a linha existente. Uma tentativa anterior sem confirmação ainda
pode ser reconciliada e não é bloqueada como reenvio confirmado.

### Pendência futura — dois cadastros iguais

Detectar dois cadastros diferentes com data, horários, turma e demais dados iguais
será tratado posteriormente, conforme solicitação do desenvolvedor. O bloqueio
atual refere-se somente ao reenvio da mesma aula/versão no mesmo destino.

## Cancelamento no histórico

O filtro inicial e **Limpar** usam **Todas**. A aula cancelada permanece visível
em vermelho e deixa de contar na carga horária. Filtros explícitos de **Ativas**
ou **Canceladas** continuam disponíveis.

- Antes do envio: **Cancelamento pendente de sincronização**.
- Depois do envio: **Cancelamento sincronizado**, ainda em vermelho.
- Reativar retorna a aula à situação ativa, com nova versão pendente de envio.

A sincronização de cancelamento preserva a representação no Google com situação
`cancelada`. Somente o comando de exclusão remove os dados definitivamente.

## Exclusão definitiva

**Excluir registro** aparece nas aulas canceladas. O diálogo identifica a aula,
explica a exclusão definitiva e oferece a opção, inicialmente desmarcada:
**Excluir também o registro da planilha configurada**. **Voltar** ou Escape
fecham o diálogo sem excluir. Durante a operação, aguardar seu resultado.

Regras:

1. Versões da aula e da configuração precisam coincidir com as apresentadas.
2. A aula deve estar cancelada. Se já foi enviada à planilha configurada, o
   cancelamento deve ser sincronizado antes de excluir.
3. Um envio sem resultado confirmado deve ser reconciliado antes da exclusão.
4. Sem marcar a opção Google, remover os dados locais e preservar os registros
   remotos. Sem destino configurado, a exclusão é somente local.
5. Marcando a opção Google, conferir a chave da aula, IDs das abas, cabeçalhos e
   último conteúdo confirmado. Alteração manual, chave duplicada ou estrutura
   incompatível bloqueiam a exclusão. Limpar também as representações em meses
   anteriores da mesma planilha. Planilhas anteriormente usadas como outros
   destinos são preservadas; a opção refere-se ao destino configurado.
6. A remoção no Google limpa as 12 células do registro, sem deslocar outras
   linhas. Podem restar linhas vazias. A posição é reservada e não será
   reutilizada por novos envios, inclusive após reiniciar a API.
7. Persistir o intento antes da escrita remota. Se o Google falhar ou a resposta
   se perder, manter a aula local, com aviso no histórico, permitindo **Excluir registro** novamente com
   a mesma opção já selecionada no diálogo. Enquanto houver uma exclusão remota sem confirmação, bloquear
   sincronização, troca de destino e alterações da aula em exclusão.
8. Confirmada a remoção, apagar a aula, os snapshots da trilha, os envios e o
   intento numa transação local. A aula desaparece dos filtros, totais e consultas.
   Não há lixeira nem restauração pela interface; um novo cadastro é uma nova aula.
9. A migração 006 mantém um contador crescente de IDs. Uma nova aula nunca
   reutiliza a chave externa da aula excluída, mesmo quando o registro remoto
   foi preservado por escolha do usuário.

O SQLite não depende de disponibilidade do Google para a exclusão somente local,
respeitadas as condições acima. Mantém-se o limite de uma instância da API por
banco/identidade e a ausência de transação distribuída com editores externos.

## Salvar e verificar o destino

**Salvar destino** persiste a configuração e chama a leitura autenticada existente.
O botão separado **Verificar leitura com minha conta** foi removido.

- Sucesso: **Destino salvo e acesso de leitura confirmado**, com título da planilha.
- Falha de leitura: **Destino salvo, mas não foi possível confirmar o acesso à
  planilha**, com orientação do erro. A configuração permanece salva e as aulas
  locais permanecem disponíveis. Salvar novamente repete a verificação.
- Falha no salvamento local: informar o erro, preservar o link digitado e não
  apresentar leitura como confirmada.

Leitura não comprova escrita. A escrita continua sendo conferida na preparação
e no envio. Ao abrir a aba, a configuração e os meses continuam sendo atualizados
sem sobrescrever o link em rascunho.

## Preparação pelo histórico

Ao sincronizar uma aula, inspecionar seu mês. Se a aba estiver ausente, perguntar:

> A aba Registros aulas - 10/2026 não existe. Deseja criá-la com os cabeçalhos e enviar esta aula?

O popup também identifica a planilha. Aba existente e vazia pede confirmação
para criar os cabeçalhos. Confirmando, usar o token da inspeção, preparar e conferir
a aba e continuar o envio. Recusando, preservar a aula como pendente.
Cabeçalhos incompatíveis bloqueiam a escrita e mostram o problema.

O mesmo fluxo atende cadastro com envio imediato e envio em lote. No lote, recusar
a preparação interrompe o restante, informando quantas aulas já foram concluídas.
Sessão, revisão do destino e período continuam vinculados à confirmação no backend.

## Contratos adicionais

`DELETE /aulas/:id`, com sessão quando OAuth configurado e Origin correto:

```json
{"versao":2,"config_versao":1,"remover_google":true}
```

A opção é obrigatória. Exclusão remota exige sessão com escopo Sheets.
Sucesso: 204 sem corpo. Entrada inválida: 400; falta de sessão/permissão: 401/403;
aula ausente: 404; versão, cancelamento pendente, estrutura ou resultado incerto:
409; configuração/login ausente: 422.

Migração 006: `aula_sequencia`, `aula_exclusoes` e `google_abas.proxima_linha`.
A migração preserva registros, vínculos existentes e identidade da instalação.

## Como testar

1. Reiniciar a API com OAuth, entrar e salvar o destino. Conferir leitura automática
   e ausência do botão antigo. Tentar uma planilha inacessível: o destino deve
   permanecer salvo, com erro de leitura claro.
2. Cadastrar aula fictícia de um mês sem aba. Pelo histórico, sincronizar e recusar
   a criação: nada deve ser preparado. Repetir, aceitar e conferir o registro.
3. Tentar reenviar a mesma aula confirmada: conferir o aviso e ausência de nova
   linha. Editar e enviar: conferir atualização da linha original.
4. Cancelar a aula no filtro **Todas**: conferir vermelho, total sem sua duração
   e mensagem de cancelamento pendente. Sincronizar e conferir a nova mensagem.
5. Abrir **Excluir registro**, voltar e conferir preservação. Excluir somente no
   aplicativo: deve desaparecer localmente e permanecer no Google como cancelada.
6. Cadastrar outra aula, enviar, cancelar e sincronizar. Excluir marcando a opção
   Google: conferir remoção local e células vazias no destino. Outras aulas e
   cabeçalhos devem permanecer intactos.
7. Simular falta de escrita na exclusão: manter a aula local; corrigir a permissão
   e tentar novamente com a mesma opção, sem remover outra aula.
8. Cadastrar novamente após excluir: deve receber novo ID, nova chave externa e
   nova posição, preservando registros mantidos na planilha.

Verificação automatizada:

```bash
# backend/
go test -race ./...
go vet ./...
go build ./...

# frontend/
npm run check
npm run build
npm run format:check

# raiz, com Chrome
node scripts/test-etapa9.cjs --ux
```

Os testes usam bancos temporários, simulador Sheets e respostas simuladas no
navegador. Escrita/exclusão reais devem ser conferidas pelo aplicativo na planilha
de teste. Referência técnica: [UpdateCellsRequest e máscara de campos](https://developers.google.com/workspace/sheets/api/reference/rest/v4/spreadsheets/request#updatecellsrequest).
