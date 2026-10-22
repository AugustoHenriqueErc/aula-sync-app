# Validações das etapas de implementação

Roteiros de teste e registros de validação das etapas 1 a 11, reunidos a partir
do [README](../README.md). Os comandos devem ser executados a partir da raiz do
repositório ou do diretório indicado em cada seção.

Os registros preservam o contexto de cada etapa e podem descrever versões
anteriores da aplicação. Para acompanhar o estado atual, consulte o
[checklist de implementação](../CHECKLIST_IMPLEMENTACAO.md). A validação da
etapa 12 está no [roteiro do relatório PDF](relatorio-pdf.md).

## Validação da etapa 1

Em 12/10/2026, foram executados em uma cópia temporária sem `node_modules`,
`dist` ou binário prévio: `npm ci`, formatação, checagem Svelte/TypeScript,
build do frontend, download e verificação de módulos Go, `go vet` e build do
backend. Todos passaram. O cache global de Go foi reutilizado; este teste
não equivale a uma máquina nova ou à validação de outros sistemas operacionais.

O frontend em 5173 e o backend em 3000 responderam HTTP 200 com o conteúdo
esperado. O componente Svelte também foi servido corretamente. Essa conferência
foi feita por HTTP, sem revisão visual em navegador. Os processos de teste
foram encerrados ao final.

Após atualizar `devalue` e `source-map-js` dentro das faixas permitidas,
o npm reportou zero vulnerabilidades. A revisão dos arquivos versionados
não encontrou artefatos locais ou padrões comuns de credenciais; isso não é
uma auditoria completa do histórico do Git.

As mensagens de commit por checkbox estão em
[docs/commits-etapa-1.md](commits-etapa-1.md). Nenhum commit foi executado.

## Testes da etapa 2

Em `backend/`, execute `go test -v ./...`. Os testes HTTP usam `httptest` e
`app.Test` do Fiber, sem abrir portas reais. Conferem status, cabeçalho JSON,
contrato de saúde, rota inexistente, erro de validação e ocultação de detalhes
em erros 500 e 503. Os testes da configuração cobrem porta padrão, alternativa,
limites e valores inválidos. As rotas que provocam erros existem somente nos testes.

## Testar a etapa 3

1. Inicie o backend em `backend/` com `go run ./cmd/api`.
2. Inicie o frontend em `frontend/` com `npm run dev -- --host 127.0.0.1 --port 5173 --strictPort`.
3. Abra http://127.0.0.1:5173 e confira **Backend conectado.**.
4. No DevTools, abra a aba Network e confira a chamada `/api/health`, status 200
   e JSON `{"status":"ok"}`. O estado **Verificando conexão…** pode ser breve;
   use a limitação de velocidade da aba Network para observá-lo.
5. Encerre o Go com Ctrl+C e clique em **Verificar novamente**: a página deve
   mostrar **Falha na conexão.** e orientar a verificar se o backend está ligado.
6. Reinicie o Go e clique novamente: o sucesso deve voltar sem recarregar a página.
7. Confira que o botão fica desabilitado durante a consulta e pode receber foco
   por Tab e ser acionado por Enter. Em largura de celular, a página deve caber
   sem rolagem horizontal.

A consulta tem limite de cinco segundos e confere se o JSON contém `status: "ok"`.
Respostas HTTP sem sucesso, JSON inválido ou incompatível e falhas de rede
produzem mensagens de erro. O estado é uma union TypeScript discriminada pelo
campo `status`; os dados recebidos são `unknown` até a validação. `onMount`
inicia a consulta ao montar o componente e cancela a requisição ao desmontá-lo.

Em 13/10/2026, passaram `npm run format:check`, `npm run check` (zero erros e
avisos), `npm run build`, testes Go, `go vet` e build do backend. A validação no
Chrome headless usou Go na porta 33003 e Vite na 5174, comprovando o proxy com
porta alternativa, carregamento, sucesso, backend desligado e recuperação real.
A interceptação de requisições no navegador foi usada apenas para simular JSON
incompatível e timeout. A captura em largura de 375px foi revisada, sem rolagem
horizontal. Os servidores e o navegador de teste foram encerrados ao final.

## Testar a etapa 4

A etapa 4 implementa o domínio no backend. Não é necessário iniciar servidores
para testar o cálculo:

```bash
cd backend
go test -v ./internal/aula
go test ./...
go vet ./...
```

Os testes usam tabelas: cada linha descreve uma entrada e seu resultado esperado.
Por exemplo, 08:00–10:00 menos 15 minutos de intervalo resulta em 105 minutos.
Datas inválidas, formatos incorretos, campos obrigatórios em branco, término
anterior ou igual ao início e intervalos inválidos devem retornar erro por campo.
Há casos de ano bissexto, duração mínima, limites do dia, repetição do cálculo e
independência do fuso local. A duração é calculada sob demanda e não fica guardada
em um campo que possa se tornar desatualizado.

Em 13/10/2026, os testes de todos os pacotes, `go vet ./...` e o build do backend
passaram. `go test -cover ./...` registrou 100% de cobertura de instruções no
pacote `internal/aula`; isso não substitui a validação das hipóteses com Maria
Luzia. Os exemplos e as decisões de datas, intervalos e unidade de duração estão
em [docs/dominio-aula.md](dominio-aula.md).

## Testar a etapa 5

A persistência pode ser testada sem iniciar frontend ou API. Em `backend/`:

```bash
export DATABASE_PATH="$PWD/aulas-estudo.db"
go run ./cmd/db migrar
go run ./cmd/db inserir-exemplo
go run ./cmd/db listar
unset DATABASE_PATH
```

O exemplo fictício deve aparecer com ID e duração de 105 minutos. Cada comando
é um processo separado: os dados permanecem no arquivo após o encerramento.
Repetir `migrar` preserva os registros; repetir `inserir-exemplo` cria outra aula.
Sem `DATABASE_PATH`, o caminho padrão é `aulas.db` no diretório de execução.
O comando Go não carrega `.env` automaticamente; a pasta do banco deve existir.

Para os testes automatizados em bancos temporários e o build do comando:

```bash
CGO_ENABLED=0 go test -v ./internal/storage
go test ./...
go vet ./...
CGO_ENABLED=0 go build -o /tmp/aula-sync-db ./cmd/db
```

O driver, a migração, as restrições, as operações e os limites desta etapa estão
em [docs/persistencia-sqlite.md](persistencia-sqlite.md). A etapa 6 conecta
essas operações à API; o comando de estudo continua disponível.

## Testar a etapa 6

Inicie a API em `backend/` com `DATABASE_PATH=aulas-api-estudo.db go run ./cmd/api`.
Em outro terminal, crie primeiro uma turma via `POST /turmas` e ajuste
`turma_id` em `docs/exemplos/aula.json` para o ID retornado, conforme o contrato v2.
Na raiz do projeto:

```bash
curl --include -X POST http://localhost:3000/aulas \
  -H 'Content-Type: application/json' \
  --data-binary @docs/exemplos/aula.json
curl --include http://localhost:3000/aulas
curl --include http://localhost:3000/aulas/1
```

O cadastro responde 201, `Location` e o registro com duração de 105 minutos.
Substitua `1` pelo ID recebido. Reinicie a API com o mesmo caminho de banco e
consulte novamente: a aula deve permanecer. A listagem retorna um objeto com página, registros e totais,
em ordem de data, início e ID decrescentes; consulte o contrato v2 para filtros.

O [contrato JSON v3](api-v3.md) documenta campos, exemplos, códigos de erro e
requisições inválidas para testar. Em `backend/`, execute
`go test -v ./internal/httpapi`, `go test ./...` e `go vet ./...`.

Em 13/10/2026, passaram a suíte com `CGO_ENABLED=0`, `go vet` e o build da API.
Os testes HTTP usam SQLite temporário, com comparação entre JSON e registro salvo,
ordenação, validação de conteúdo/tipos/campos, IDs inválidos/inexistentes e falha do
banco sem exposição de detalhes. O limite de corpo foi testado por HTTP real.
Também foi validado o binário na porta 33006: cadastro, consultas, erros, leitura
direta por SQL, persistência após reinício, encerramento por SIGINT/SIGTERM e
recusa de inicialização com caminho de banco inválido. Os processos foram
encerrados e os arquivos temporários removidos. O formulário que consome esse
contrato está implementado na etapa 7.

## Testar a etapa 7

Com backend e frontend ligados, abra http://127.0.0.1:5173 e preencha uma aula
fictícia: data válida, início 08:00, término 10:00, intervalo 15 e conteúdo; crie ou selecione uma turma/estudante.
A prévia deve indicar **105 minutos**. Clique em **Salvar aula**: a confirmação
usa a duração retornada pelo servidor, e o formulário é limpo para outra aula.

Confira também campos obrigatórios vazios, término anterior ao início e intervalo
que ocupa toda a aula. Desligue o backend e tente salvar outro registro: os campos
devem permanecer preenchidos. Durante o envio, o formulário fica bloqueado.
O [roteiro do formulário](formulario-aula.md) inclui teclado, celular,
comparação com os dados salvos e as limitações de uma tentativa após falha de rede.

Formatação, checagem e build passaram. A validação no Chrome cobriu cadastro real,
erros locais e do servidor, teclado, bloqueio de envios repetidos, falha e
recuperação da conexão, timeout e largura de 375 px. As regras de aula ainda são
hipóteses do MVP pendentes de revisão com Maria Luzia.


## Etapa 8 — Histórico e turmas

Ao iniciar, a API migra o SQLite para a versão 2, preservando aulas e agrupando
nomes equivalentes em um catálogo. Aulas antigas mantêm seu texto de turma;
o filtro usa o ID compartilhado. O nome exibido no catálogo vem da primeira
aula (menor ID), com espaços normalizados. Não foi aplicado nenhum script
manual aos bancos locais durante o desenvolvimento desta etapa.

Na interface, use **Criar turma ou estudante**, informe um nome e clique em
**Criar e selecionar**. Selecione essa opção ao cadastrar uma aula. No histórico,
use **Filtrar**, **Limpar**, **Anterior** e **Próxima**. O total corresponde a
todas as aulas do filtro, mesmo ao mudar de página.

Consulte o [contrato v3](api-v3.md) e o
[roteiro de teste da etapa 8](historico-aulas.md).


## Etapa 9 — Editar, cancelar e recuperar dados

No histórico, clique **Editar** para carregar a aula no formulário e depois
**Salvar alterações**. **Cancelar aula** preserva o registro e o retira da carga
horária; use o filtro **Canceladas** para consultar ou **Reativar aula** para
incluí-lo novamente. A consulta padrão mostra somente as aulas ativas.

Duas abas não podem sobrescrever silenciosamente a mesma versão. Um conflito
preserva os campos: feche a edição e abra novamente a aula para conferir os dados
atuais. O SQLite migra automaticamente para a versão 3 ao iniciar.

Para executar o teste de interface completo, na raiz:

```bash
node scripts/test-etapa9.cjs
```

Consulte o [contrato v3](api-v3.md) e o
[roteiro de correção e backup](correcao-backup.md), que inclui os comandos de
backup consistente e restauração em outro arquivo. Nenhuma integração Google foi
adicionada nesta etapa.


## Etapa 10 — Configurar a planilha de teste

Na seção **Integração com Google Sheets**, cole o link e clique em **Salvar
destino**. A configuração fica no SQLite e pode ser alterada sem editar código.
A configuração e os meses são atualizados automaticamente ao abrir essa aba e
após salvar o destino.
A data da aula determina a aba esperada, como `Registros aulas - 10/2026`.

Salvar o link não valida permissões nem cria abas ou envia aulas. A API migra o
banco para a versão 6 e mantém os dados existentes. O endereço da planilha de
teste não está fixado no código nem nos exemplos versionados.

O [contrato e roteiro da etapa 10](google-etapa10.md) descreve endpoints,
modelo de estados, cabeçalhos propostos e o fluxo de login Google com OAuth.
O backend acessa diretamente a API Sheets usando a conta do usuário logado.
Informe uma planilha própria ou compartilhada com permissão de edição nessa
conta. Login e autorização estão implementados, com sessões e tokens somente em
memória. Consulte [configuração e testes do login Google](login-google.md).
Apps Script foi descartado. Login e leitura real foram confirmados pelo desenvolvedor;
a revisão institucional e os demais cenários reais de autorização permanecem pendentes.

A [revisão de cenários e validação da conta](google-validacao-etapa10.md)
detalha os resultados esperados, o roteiro de conferência das permissões e as
evidências necessárias para encerrar a preparação.

Para testar o fluxo local e a configuração no navegador:

```bash
node scripts/test-etapa9.cjs --google
```

## Etapa 11 — Preparar abas e sincronizar aulas

Na **Integração Google**, recarregue a prévia, inspecione o mês e confirme a
preparação se a aba estiver ausente ou vazia. No **Histórico**, use **Sincronizar
com Google** na aula desejada. Correções e cancelamentos exigem novo envio.
Uma falha mantém a aula local e oferece nova tentativa.

Consulte [como testar a etapa 11](google-etapa11.md), incluindo mudança de
mês, troca de planilha, alterações manuais e limites de concorrência.

```bash
node scripts/test-etapa9.cjs --sync
```

O teste de navegador usa dados temporários e respostas Sheets simuladas; a
validação de escrita na planilha real deve ser feita pelo aplicativo.

As [melhorias de experiência do frontend](melhorias-frontend.md) incluem
perfil com foto, convite à sincronização após cadastro, cores no histórico e
ações em lote sobre todas as páginas dos filtros aplicados. O documento registra
também os ajustes posteriores de cancelamento e exclusão.

Os [ajustes finais da integração](ajustes-integracao-etapa11.md) acrescentam
aviso de aula já sincronizada, cancelamento em vermelho, exclusão definitiva com
opção de remover no Google, preparação de abas pelo histórico e leitura ao salvar
o destino. A detecção de dois cadastros iguais permanece uma pendência futura.
