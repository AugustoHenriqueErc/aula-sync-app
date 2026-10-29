# Catálogo de conteúdos — backend

Implementação da etapa B dos [ajustes antes do deploy](ajustes-antes-deploy.md).
O dropdown no formulário pertence à etapa C e ainda não foi implementado.

## Dados e migração

O catálogo é global no arquivo SQLite, assim como o de turmas/estudantes, e não
depende do Google. Não há vínculo com uma turma específica nem edição/exclusão
de opções nesta etapa.

A migração 007 cria `conteudos(id, nome, nome_normalizado)` e registra
`PRAGMA user_version = 7` na mesma transação. O backend aplica a migração ao
abrir o banco; bancos novos recebem todas as migrações anteriores também.
Uma falha desfaz a criação e a importação, sem avançar a versão. Um banco de
versão superior à suportada continua sendo rejeitado.

A importação lê todas as aulas atuais, inclusive canceladas, por ID crescente.
Cada chave normalizada gera uma opção, com o texto da primeira ocorrência.
Textos vazios são ignorados. Reabrir o banco não repete a importação; aulas
criadas posteriormente não alimentam o catálogo automaticamente.

No cadastro e na importação, removem-se espaços apenas das extremidades do nome;
espaços e quebras de linha internos são preservados na apresentação. A chave
normalizada usa as mesmas regras de turma/estudante: minúsculas e sequências de
espaços Unicode substituídas por um espaço. Por exemplo, `Leitura\n e revisão`
e `LEITURA E REVISÃO` representam uma única opção. Acentos permanecem distintos.
O índice único resolve também tentativas concorrentes de criar a mesma chave.

As aulas continuam armazenando seu próprio `conteudo` textual. A migração não
modifica aulas, versões, cancelamentos, auditoria ou estados de envio ao Google.
Não há `conteudo_id` novo no contrato de aulas. Os relatórios e o Sheets seguem
usando o texto registrado na aula.

## Contrato HTTP

Rotas diretas do backend abaixo; pelo proxy do Vite, acrescentar `/api`.
Quando OAuth está configurado, ambas exigem sessão e o POST também exige
`Origin` igual a `APP_URL`, como as rotas de turmas. Sem OAuth, seguem o modo
local existente. Não há permissão de planilha envolvida nestas rotas.

### GET /conteudos

Retorna HTTP 200 e lista ordenada por chave normalizada e ID:

```json
[{"id": 1, "nome": "Frações"}]
```

Lista vazia: `[]`. Não aceita parâmetros de consulta (400). Sem paginação nesta
etapa, seguindo o catálogo de turmas.

### POST /conteudos

Exige `Content-Type: application/json` (aceita `charset=utf-8`) e um único objeto:

```json
{"nome": "Frações"}
```

Retorna HTTP 201 com `{ "id": 1, "nome": "Frações" }`. O ID é gerado pelo banco.

| Status | Situação |
| --- | --- |
| 400 | JSON inválido, null, array, tipo incompatível, campo desconhecido ou múltiplos objetos |
| 401 | Sem sessão válida quando OAuth está ativo |
| 403 | Origem ausente ou diferente de APP_URL no POST autenticado |
| 409 | Nome já cadastrado segundo a chave normalizada |
| 415 | Content-Type ausente ou incompatível |
| 422 | Nome ausente, vazio ou composto somente por espaços; `field` é `nome` |
| 500 | Falha interna, sem expor detalhes do banco |

Os erros seguem `{ "error": "mensagem" }`, com `field` na validação de nome.
O limite de corpo de 64 KiB da aplicação continua aplicado.

## Como testar sem alterar o banco de desenvolvimento

Em `backend/`, abra uma API isolada com OAuth desabilitado e banco temporário:

```bash
conteudos_teste_dir=$(mktemp -d /tmp/aula-sync-conteudos-XXXXXX)
GOOGLE_CLIENT_ID= GOOGLE_CLIENT_SECRET= GOOGLE_REDIRECT_URI= APP_URL= \
  DATABASE_PATH="$conteudos_teste_dir/aulas.db" PORT=3001 go run ./cmd/api
```

Em outro terminal:

```bash
curl --fail http://127.0.0.1:3001/conteudos
curl --include http://127.0.0.1:3001/conteudos \
  -H 'Content-Type: application/json' --data '{"nome":"Frações"}'
curl --fail http://127.0.0.1:3001/conteudos
curl --include http://127.0.0.1:3001/conteudos \
  -H 'Content-Type: application/json' --data '{"nome":" FRAÇÕES "}'
```

Esperado: lista vazia, 201, lista contendo Frações e 409 na duplicata. Reinicie
usando o mesmo `DATABASE_PATH` para conferir persistência. Encerre com Ctrl+C.

Antes de aplicar ao banco de desenvolvimento, pare os processos que o usam e
faça uma cópia/backup, seguindo [correção e backup](correcao-backup.md). Atenção:
o comando `go run ./cmd/db backup` também abre e migra o banco antes de exportar;
para guardar a versão v6 original, faça o backup com o binário anterior ou copie
o arquivo com todos os processos encerrados antes de executar o código novo.
Nenhuma migração foi executada no banco de desenvolvimento durante estes testes.

## Verificação automatizada

Em `backend/`:

```bash
go test ./...
go vet ./...
go build -o /tmp/aula-sync-backend ./cmd/api
```

Os testes específicos cobrem catálogo vazio, multilinha, persistência, ordenação,
vazios Unicode, duplicidade e concorrência; banco v6 migrado com aulas canceladas,
preservação de aulas/auditoria/sincronização, reabertura e rollback/retomada.
Os testes HTTP conferem contrato e erros sem detalhes internos; a integração
protege GET e POST sem sessão, e o teste do middleware verifica o POST com sessão
válida e origens válidas/inválidas, sem chamar o Google.

Resultado nesta implementação: suíte completa, vet e build aprovados; arquivos
Go formatados e `git diff --check` sem problemas. Validação realizada em bancos
temporários, sem acessar o Google nem alterar o banco de desenvolvimento.
