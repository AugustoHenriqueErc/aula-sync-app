# Etapa 10 — Configuração local e contrato de integração

Atualização em 20/10/2026: login/leitura reais confirmados pelo desenvolvedor.
As operações de inspeção, preparação e envio descritas como futuras neste registro
da etapa 10 foram implementadas na etapa 11. Consulte o [estado atual e roteiro](google-etapa11.md).
A revisão institucional continua pendente.


Status: preparação local, login e leitura de teste implementados; escritas no
Google pertencem à etapa 11.
Login Google, OAuth e leitura de metadados implementados; validação com conta
real pendente. Roteiro em [login-google.md](login-google.md). A revisão institucional com Maria Luzia continua pendente.

A [revisão técnica dos cenários e o roteiro de validação da conta](google-validacao-etapa10.md)
registram o que já foi analisado e as evidências externas ainda necessárias.

Decisão confirmada pelo desenvolvedor em 19/10/2026: integração direta do backend
Go com a API do Google Sheets, usando OAuth da mesma conta Google utilizada no
login. Apps Script e o segredo compartilhado da proposta anterior foram descartados.
O usuário informa uma planilha própria ou compartilhada na qual tenha permissão
de edição. Login e consentimento estão implementados; ensaio remoto real pendente.

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
A configuração e os meses são atualizados ao abrir a aba de integração e após
salvar o destino. O rascunho do link é preservado ao navegar entre abas.

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

## Contrato para integração direta com Google Sheets

O backend Go acessará a API do Google Sheets diretamente em nome do usuário
logado. O navegador conversa com a API local e participa do redirecionamento
para login/consentimento do Google. Não haverá Apps Script, conta executora
central ou segredo compartilhado com um Web App.

### Login, autorização e conta executora

1. O usuário entra com Google. OpenID Connect identifica a conta; OAuth 2.0
   autoriza o acesso à API. São responsabilidades distintas no mesmo fluxo.
2. O backend valida a identidade e vincula a sessão e as credenciais à mesma
   conta, identificada pelo `sub` validado, sem confiar em e-mail enviado pelo frontend.
3. O usuário cola o link de uma planilha própria ou compartilhada. Ser proprietário
   não é necessário; ter somente leitura não autoriza escrita.
4. Inspeção, preparação e envio usam exclusivamente a autorização da conta da
   sessão. O link identifica o destino, mas não concede permissão.
5. Troca de conta invalida confirmações anteriores e exige revalidar o destino.
   Nunca reutilizar credenciais de outra conta. Logout encerra a sessão; revogação
   ou expiração sem renovação possível exige reconexão antes de acessar o Google.

A configuração atual é um singleton local. Ela não comprova suporte a múltiplos
usuários: o vínculo entre sessão, conta e credenciais foi implementado em memória,
mas aulas e destino ainda são compartilhados na instalação.
O destino deve ser revalidado para cada conta, sem herdar autorização do último
usuário. Isso não define compartilhamento dos registros locais entre usuários;
as regras de acesso aos dados locais deverão ser revisadas antes do piloto.

### Autenticação e configuração do ambiente

Implementado na etapa 10 o login e o consentimento com Authorization Code no
backend, proteção `state` contra CSRF, validação do ID token (assinatura, emissor,
audiência, validade e `nonce`) e sessão com cookie HttpOnly, SameSite adequado e
Secure em produção. O backend mantém tokens e segredo OAuth fora do frontend,
URLs da aplicação, logs e arquivos versionados. PKCE complementa a proteção do fluxo.

Configurar projeto Google Cloud com Sheets API habilitada, tela de consentimento,
usuários de teste e cliente OAuth Web com URI de retorno exata. As variáveis implementadas
são `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `GOOGLE_REDIRECT_URI` e `APP_URL`.
Sessões e access tokens permanecem apenas em memória por até uma hora, limitados
pela validade dos tokens. Não há renovação nem refresh token: expiração e reinício
exigem novo login. Logout elimina a sessão e seu token; não revoga consentimento
no Google. Configuração e testes em [login-google.md](login-google.md).

Escopos previstos: `openid email profile` para identidade e
`https://www.googleapis.com/auth/spreadsheets` para planilhas informadas por link.
O escopo de Sheets é amplo e não fica limitado ao link salvo: o backend deve
restringir operações ao destino confirmado. `drive.file` é uma alternativa mais
restrita, mas colar um link não concede por si só acesso ao arquivo nesse escopo;
exigiria um fluxo de seleção/autorização de arquivo a ser avaliado separadamente.
Não solicitar acesso a Gmail ou Docs. Consentimento OAuth e permissão de edição
no arquivo são verificações distintas; leitura bem-sucedida não comprova escrita.

A etapa 10 valida login, vínculo da conta e autorização com leitura de teste,
sem criar abas ou enviar aulas. A etapa 11 implementa inspeção completa, preparação
e sincronização. A etapa 13 revisa controles de acesso e segurança para produção.

Referências: [OAuth para aplicações web](https://developers.google.com/identity/protocols/oauth2/web-server),
[OpenID Connect](https://developers.google.com/identity/openid-connect/openid-connect) e
[escopos da API Sheets](https://developers.google.com/workspace/sheets/api/scopes).

### Operações previstas (ainda não disponíveis)

| Operação | Entrada relevante | Resultado |
| --- | --- | --- |
| inspecionar | ID da planilha, período, nome esperado, versão do modelo | Identidade da aba, estado, cabeçalhos encontrados e diferenças |
| preparar | Mesmo destino e período, ação confirmada e referência da inspeção | Aba pronta, somente após conferir que a inspeção ainda é válida |
| sincronizar | Identidade da instalação, aula e versão, destino/período, dados | Registro inserido/atualizado ou conflito, com identidade e versão confirmadas |

As operações acima são contratos internos do backend, não endpoints de Apps
Script nem envelopes enviados ao Google. O cliente Sheets usa os endpoints e
corpos oficiais da API com access token da conta da sessão no cabeçalho
`Authorization: Bearer`. O ID token identifica o usuário; não substitui o access
token nas chamadas ao Sheets. O backend traduz respostas e erros da API para os
resultados de domínio e nunca expõe credenciais nas respostas ao frontend.

Códigos de domínio previstos: `SEM_ACESSO`, `ABA_AUSENTE`, `ABA_VAZIA`,
`ESTRUTURA_INCOMPATIVEL`, `CONFIRMACAO_DESATUALIZADA`, `VERSAO_ANTIGA` e
`FALHA_TEMPORARIA`. Falta de acesso nunca será traduzida como aba ausente.

### Período e preparação confiável

1. O backend obtém `AAAA-MM` da data validada da aula e deriva o nome esperado.
2. Antes do primeiro vínculo, o adaptador procura exatamente esse nome. Se não
   encontrar, retorna ausência; não cria nada durante a inspeção.
3. Depois da criação ou de encontrar uma aba compatível, vincula `sheet_id`,
   período e versão do modelo. O ID é estável em relação à posição da aba;
   consulte [SheetProperties.sheetId](https://developers.google.com/workspace/sheets/api/reference/rest/v4/spreadsheets/sheets#SheetProperties).
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
- Para detectar alterações manuais, a etapa 11 deverá guardar uma referência
  do conteúdo da última escrita confirmada e compará-la antes de atualizar.
  Também deverá persistir a versão do modelo usada no vínculo mensal; essas
  referências ainda não estão representadas nas tabelas atuais.
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
- Validar login Google, consentimento OAuth e vínculo da sessão à conta executora com uma conta real.
- Revisar o modelo proposto com a usuária/responsável e ajustar antes do envio.
- Comprovar escopos e configuração de segredos no ambiente autorizado.

A planilha de teste foi fornecida pelo desenvolvedor. Uma leitura pelo conector
confirmou uma única aba inicial e ausência de valores no intervalo A1:Z5; não
foi feita inspeção da grade inteira nem escrita no arquivo. Isso não atesta que
o aplicativo tenha autorização OAuth; a nova leitura de teste ainda precisa
ser executada com a conta real.
