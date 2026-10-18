# Etapa 10 — Configuração local e contrato de integração

Status: preparação local implementada; execução no Google pertence à etapa 11.
A autorização e a implantação do Apps Script ainda precisam ser validadas na
conta de teste. A revisão institucional com Maria Luzia continua pendente.

## O que está disponível

A seção **Integração com Google Sheets** permite salvar o link da planilha no
SQLite. O backend aceita URLs HTTPS de `docs.google.com/spreadsheets/d/ID`,
com os sufixos usuais de visualização/edição, e guarda apenas o identificador.
Parâmetros de compartilhamento e fragmentos como `gid` não escolhem a aba mensal.
O link exibido é normalizado para `https://docs.google.com/spreadsheets/d/ID/edit`.
URLs de publicação, outros domínios, credenciais na URL e caminhos incompatíveis
são rejeitados. Validar a sintaxe não confirma acesso, existência ou estrutura.

A prévia agrupa todas as aulas locais por mês/ano, informa a aba esperada e
conta as canceladas separadamente. A duração considera somente as ativas.
Clique **Recarregar configuração e prévia** após cadastrar ou editar aulas para
atualizar essa visualização. O botão também substitui o link digitado pelo salvo.

Salvar o mesmo identificador com a versão atual não cria outro destino. Trocar
para outra planilha preserva os destinos anteriores; voltar a uma já usada
reutiliza seu identificador local. Cada troca incrementa a versão da configuração.
Uma versão antiga recebe 409, preservando o link digitado até o usuário recarregar.

Nenhum endpoint deste incremento faz requisições ao Google, cria abas, grava
cabeçalhos ou marca aulas como sincronizadas. A inspeção feita pelo conector no
ambiente de desenvolvimento não é uma funcionalidade da aplicação.

## API local

### GET /integracoes/google/configuracao

Antes da primeira configuração:

```json
{"versao":0,"destino":null}
```

Depois de salvar:

```json
{
  "versao":1,
  "destino":{
    "id":1,
    "planilha_id":"ID_DA_PLANILHA",
    "url":"https://docs.google.com/spreadsheets/d/ID_DA_PLANILHA/edit"
  }
}
```

### PUT /integracoes/google/configuracao

Content-Type application/json; corpo:

```json
{"url":"https://docs.google.com/spreadsheets/d/ID_DA_PLANILHA/edit?usp=sharing","versao":0}
```

Responde 200 com a configuração salva. A versão é obrigatória e não negativa.
JSON inválido, campos desconhecidos e versão ausente/inválida retornam 400;
link inválido retorna 422 com `field: url`; conflito retorna 409; falha interna
retorna 500 genérico. Não há campo para escolher livremente uma aba.

### GET /integracoes/google/planejamento

Retorna configuração e agrupamento local no mesmo snapshot do banco:

```json
{
  "configuracao":{"versao":0,"destino":null},
  "meses":[{
    "periodo":"2026-10",
    "aba_esperada":"Registros aulas - 10/2026",
    "aulas":3,
    "canceladas":1,
    "duracao_ativa_minutos":150
  }],
  "verificacao_remota":"nao_realizada"
}
```

Sem aulas: `meses: []`. A configuração pode estar ausente e a prévia ainda pode
ser calculada. Isso não significa que o destino está pronto para enviar.

## SQLite — migração 004

- `google_configuracao`: singleton, destino ativo, versão e identidade aleatória
  da instalação. A identidade é criada uma vez e preservada em backups.
- `google_destinos`: identificadores únicos de planilhas e data de cadastro.
- `google_abas`: vínculo único entre destino e período, `sheet_id` opcional,
  estado de preparação e data de verificação.
- `google_sincronizacoes`: referência à aula e ao vínculo mensal, versão enviada,
  estado, contador de tentativas, última tentativa e erro.

Configurar uma planilha não preenche `google_abas` nem `google_sincronizacoes`.
Essas tabelas são o modelo persistente preparado para a etapa 11; ainda não há
operações de envio. A migração mantém aulas, turmas, alterações e backups locais.

A identidade da instalação distingue bancos independentes com a mesma aula de
ID 1. Restaurar um backup conserva essa identidade para recuperar os mesmos
registros externos. Uma cópia usada como instalação independente deverá ganhar
outra identidade antes de sincronizar; não executar duas cópias divergentes como
se fossem instalações diferentes com a mesma identidade.

## Contrato proposto para o adaptador Apps Script

O backend Go será o único cliente do adaptador. O navegador conversa somente
com a API local. Proposta: projeto Apps Script independente da planilha,
publicado como Web App e executado pela conta responsável pelo ambiente de teste.
Isso permite trocar o identificador de planilha sem trocar o código do script.
A execução como proprietário utiliza as permissões dessa conta, conforme a
[documentação de Web Apps](https://developers.google.com/apps-script/guides/web).

O script usará o identificador para abrir a planilha. `openById` exige o escopo
`https://www.googleapis.com/auth/spreadsheets`; não limitar indevidamente a
`spreadsheets.currentonly` em um adaptador independente com destino configurável.
Não são necessários escopos de Gmail, Docs ou Drive para esse desenho inicial.
Fonte: [SpreadsheetApp.openById](https://developers.google.com/apps-script/reference/spreadsheet/spreadsheet-app#openById(String)).

### Autenticação e configuração do ambiente

Proposta de MVP a validar na conta de teste: Web App acessível ao backend sem
sessão de navegador, com segredo aleatório compartilhado enviado no corpo JSON
sobre HTTPS e verificado antes de qualquer acesso à planilha. O backend manterá
o segredo no ambiente do servidor, e o script nas propriedades do projeto.
Não é o link da planilha nem a URL do Web App que autoriza a chamada.

Os nomes previstos são `GOOGLE_APPS_SCRIPT_URL` e `GOOGLE_APPS_SCRIPT_SECRET` no
backend, e `API_SECRET` nas Script Properties. Ainda não são lidos pelo código
atual. Não colocar segredo em variáveis VITE_, URLs, logs, exemplos versionados
ou respostas da API. Gerar um segredo de pelo menos 32 bytes aleatórios no
ambiente autorizado quando o adaptador for implantado. O serviço de propriedades
é descrito em [Properties Service](https://developers.google.com/apps-script/guides/properties).

A viabilidade dessa publicação depende da conta e da política institucional.
Não foi validada apenas porque o conector conseguiu ler a planilha. Se o ambiente
exigir autenticação Google de cada chamada, revisar o mecanismo antes de iniciar
o adaptador; não tornar a planilha pública para contornar essa restrição.

### Operações previstas (ainda não disponíveis)

| Operação | Entrada relevante | Resultado |
| --- | --- | --- |
| inspecionar | ID da planilha, período, nome esperado, versão do modelo | Identidade da aba, estado, cabeçalhos encontrados e diferenças |
| preparar | Mesmo destino e período, ação confirmada e referência da inspeção | Aba pronta, somente após conferir que a inspeção ainda é válida |
| sincronizar | Identidade da instalação, aula e versão, destino/período, dados | Registro inserido/atualizado ou conflito, com identidade e versão confirmadas |

Envelope conceitual de requisição: `contrato`, `request_id`, `segredo`, `operacao`,
`planilha_id`, `periodo` e `payload`. O corpo de resposta não contém o segredo:
`contrato`, `request_id`, `ok`, `resultado` ou `erro {codigo,mensagem}`.
O backend deve conferir o envelope além do status HTTP, pois uma página HTML de
login ou um corpo inesperado não representa sucesso.

Códigos de domínio previstos: `SEM_ACESSO`, `ABA_AUSENTE`, `ABA_VAZIA`,
`ESTRUTURA_INCOMPATIVEL`, `CONFIRMACAO_DESATUALIZADA`, `VERSAO_ANTIGA` e
`FALHA_TEMPORARIA`. Falta de acesso nunca será traduzida como aba ausente.

### Período e preparação confiável

1. O backend obtém `AAAA-MM` da data validada da aula e deriva o nome esperado.
2. Antes do primeiro vínculo, o adaptador procura exatamente esse nome. Se não
   encontrar, retorna ausência; não cria nada durante a inspeção.
3. Depois da criação ou de encontrar uma aba compatível, vincula `sheet_id`,
   período e versão do modelo. O ID é estável em relação à posição da aba;
   consulte [Sheet.getSheetId](https://developers.google.com/apps-script/reference/spreadsheet/sheet#getSheetId()).
4. Em cada escrita, comparar ID vinculado, nome esperado e cabeçalhos, e conferir
   que a data de cada aula pertence ao período. Renomeação divergente, exclusão
   ou outra aba ocupando o nome causam conflito, não redirecionamento silencioso.
5. A confirmação local deve ficar vinculada à revisão da configuração, destino,
   período e ação proposta. Reinspecionar sob bloqueio antes de criar/preparar;
   uma mudança de destino ou estrutura exige nova conferência.
6. Se a aba aparecer entre inspeção e confirmação, validar a existente em vez
   de criar duplicata. Repetir a confirmação não deve sobrescrever cabeçalhos.

Uma aba só será considerada vazia após inspeção de sua área utilizada, incluindo
fórmulas. A ausência de valores em poucas células não autoriza a preparação.

### Modelo de colunas proposto — versão 1

Ordem inicial na linha 1 (a revisar com Maria Luzia):

| Cabeçalho | Conteúdo |
| --- | --- |
| chave_aula | Identidade da instalação + ID local |
| versao | Versão da aula enviada |
| data | Data civil AAAA-MM-DD, sem conversão de fuso |
| inicio | HH:MM |
| fim | HH:MM |
| intervalo_minutos | Inteiro |
| turma_estudante | Nome no registro local |
| conteudo | Texto literal |
| observacoes | Texto literal |
| duracao_minutos | Inteiro calculado no backend |
| situacao | ativa, cancelada ou movida |
| updated_at | Data UTC da atualização local |

A coluna de duração preserva o valor da aula; totalizações devem considerar
somente `situacao=ativa`. Datas, horários e textos serão gravados como valores
literais, sem interpretar o conteúdo do usuário como fórmula.

Para o primeiro contrato, cabeçalhos precisam coincidir exatamente em nome e
ordem; cabeçalhos extras preenchidos são incompatíveis. Fórmulas em cabeçalhos
ou células que a aplicação precisaria sobrescrever também bloqueiam o envio.
Não adaptar silenciosamente o modelo institucional: apresentar diferenças.

### Idempotência, versões e falhas

- Chave lógica: instalação + ID da aula, dentro de cada planilha. Não depender
  apenas do número da linha, que pode mudar quando a planilha é reorganizada.
- Antes de inserir, localizar essa chave; atualizar a mesma linha quando existe.
  Chave duplicada ou alteração manual divergente gera conflito e exige revisão.
- Versão antiga nunca sobrescreve uma mais nova. Repetir a mesma versão e mesmo
  conteúdo confirma o resultado já obtido; divergência não é sucesso.
- O adaptador deverá serializar preparação e escritas concorrentes. Confirmar o
  resultado após perda de resposta antes de repetir uma operação.
- Timeout proposto: 10 segundos no backend. Interromper a espera não prova que
  a escrita parou no Google. Primeiro corte com nova tentativa manual, sem loop
  automático; inspecionar novamente antes de reenviar.
- Estado efetivo só será sincronizado se a versão local, destino e período
  coincidirem com o envio confirmado e o vínculo estiver pronto. Uma edição
  torna o envio antigo pendente, mesmo que seu registro histórico diga sucesso.
- Cancelamentos atualizam situação e versão; não apagam a linha.
- Mudança de mês na mesma planilha: marcar a representação anterior como movida
  antes de ativar a nova, sob bloqueio. Repetição deve concluir uma operação
  parcial, evitando duas representações ativas. Uma falha intermediária pode
  deixar o Google temporariamente sem a aula ativa, mas o SQLite mantém o total.
- Troca de planilha: não mover nem apagar linhas anteriores. O novo envio é uma
  ação explícita, com estado independente; referências anteriores permanecem.

Essa política deve ser comprovada na etapa 11 com falhas e chamadas concorrentes;
o contrato não equivale a uma integração já validada.

## Como testar este incremento

1. Inicie backend e frontend conforme o README, com banco fictício separado.
2. Cole o link de teste em Integração com Google Sheets e salve.
3. Recarregue a página e reinicie a API com o mesmo banco: o destino deve persistir.
4. Cadastre aulas em outubro/2026, novembro/2026 e outubro/2027; recarregue a
   prévia e confira três nomes distintos. Nenhuma aba deve ser criada no Google.
5. Troque a planilha e depois volte: os registros locais permanecem intactos.
6. Abra a configuração em duas abas. Salve um destino em uma e tente salvar na
   outra: deve receber conflito e manter o texto até recarregar.
7. Tente link de outro domínio: deve mostrar erro, preservando o destino salvo.

Backend: `go test ./...`, `go vet ./...` e build em `backend/`.
Frontend: `npm run format:check`, `npm run check` e `npm run build` em `frontend/`.
Teste de regressão e configuração em Chrome, na raiz:

```bash
node scripts/test-etapa9.cjs --google
```

O nome do script anterior foi preservado; `--google` acrescenta os cenários da
etapa 10. Usa banco temporário e um link fictício por padrão; para validar um link
específico, defina `GOOGLE_TEST_SHEET_URL` apenas no processo de teste. O script
não acessa a planilha. Requisitos e funcionamento do Chrome estão no roteiro da
[etapa 9](correcao-backup.md).

## Pendências para encerrar a etapa 10

- Confirmar conta responsável, propriedade e permissão de escrita da planilha.
- Validar a opção de implantação/autenticação do Apps Script nessa conta.
- Revisar o modelo proposto com a usuária/responsável e ajustar antes do envio.
- Comprovar escopos e configuração de segredos no ambiente autorizado.

A planilha de teste foi fornecida pelo desenvolvedor. Uma leitura pelo conector
confirmou uma única aba inicial e ausência de valores no intervalo A1:Z5; não
foi feita inspeção da grade inteira nem escrita no arquivo. Isso não atesta que
o Apps Script já esteja autorizado ou que a verificação remota esteja implementada.
