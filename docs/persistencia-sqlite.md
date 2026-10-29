> Ajustes antes do deploy: o esquema atual é v7, com catálogo de conteúdos.
> A migração importa textos existentes sem alterar aulas, auditoria ou envios.
> Consulte [catálogo de conteúdos](catalogo-conteudos.md).

> Etapa 9: o esquema passou a v3, com versão, situação, updated_at e trilha
> transacional de alterações. Consulte [correção e backup](correcao-backup.md).

> Evolução na etapa 8: o esquema atual é v2, com catálogo de turmas,
> nomes normalizados únicos e `turma_id` obrigatório nas aulas. A migração
> preserva os registros v1. Veja [histórico e migração](historico-aulas.md).
> Este documento mantém a explicação da base implementada na etapa 5.

# Persistência SQLite — etapa 5

## Driver e execução

Usamos `modernc.org/sqlite v1.58.0` com `database/sql`, a biblioteca padrão de Go.
O driver inclui SQLite e não depende de CGO, compilador C ou instalação do
executável `sqlite3`. A contrapartida é um conjunto maior de dependências Go e
uma primeira compilação mais demorada. A versão fica fixada no `go.mod`, com
checksums no `go.sum`; `go mod tidy` também classifica Fiber como dependência direta.
A toolchain do projeto continua Go 1.27.0. O primeiro download exige rede.

Referências: [documentação do driver](https://pkg.go.dev/modernc.org/sqlite@v1.58.0),
[restrições SQL](https://www.sqlite.org/lang_createtable.html) e
[tabelas STRICT](https://www.sqlite.org/stricttables.html).

## Arquivo e ciclo de vida

Os comandos `cmd/db` e `cmd/api` leem `DATABASE_PATH` do ambiente. O padrão é `aulas.db` relativo
ao diretório de execução; executando em `backend/`, o arquivo fica em
`backend/aulas.db`. Um caminho absoluto evita depender do diretório atual.
A pasta de destino deve existir e permitir escrita. Arquivos `.env` não são
carregados automaticamente pelo Go.

`storage.Abrir(ctx, caminho)` abre ou cria o arquivo, verifica a conexão e aplica
as migrações pendentes. Quem abre chama `Close()` ao terminar. Se a abertura ou
migração falhar, a conexão é fechada antes de retornar o erro. O caminho é tratado
como nome de arquivo, não como uma URI de configuração do SQLite.

O pool usa uma conexão por instância, suficiente para este MVP local. O driver
recebe `busy_timeout(5000)` em cada conexão para aguardar brevemente um bloqueio.
As transações usam `BEGIN IMMEDIATE`, reservando a escrita antes de consultar a
versão; isso evita duas inicializações aplicarem a mesma migração simultaneamente.
Usamos o journal padrão do SQLite, sem configurar WAL nesta etapa.

Desde a etapa 6, `cmd/api` também abre e migra o banco ao iniciar, e fecha a
conexão após o encerramento HTTP. Os endpoints estão no [contrato v1](api-v1.md).
`/health` continua verificando somente a resposta HTTP. Na etapa 7, o frontend
oferece o formulário de cadastro e mantém o diagnóstico de conexão.

## Migração e restrições

A migração `internal/storage/migrations/001_create_aulas.sql` é incorporada ao
binário por `go:embed`, sem depender de encontrar um arquivo SQL em produção.
`PRAGMA user_version` guarda a versão do esquema:

- Versão 0: cria a tabela e o índice e registra versão 1 na mesma transação.
- Versão 1: mantém a estrutura e os registros existentes.
- Qualquer outra versão: retorna erro, sem tentar rebaixar o banco.

Uma falha desfaz toda a migração. Não há exclusão automática de banco nem
migração destrutiva. Migrações aplicadas não devem ser editadas: uma mudança futura
exigirá outro arquivo numerado e sua aplicação explícita no código. Esta primeira
versão tem apenas um caso de migração, sem framework ou executor genérico.

`aulas` é uma tabela `STRICT`, com ID inteiro positivo, dados do domínio e duração
calculada. `NOT NULL` e `CHECK` restringem formatos, datas existentes, início e
fim no mesmo dia, textos obrigatórios, intervalo e duração coerentes. A checagem
da data usa `IS` com a data normalizada, de modo que um resultado SQL `NULL` não
permita uma entrada inválida.

A validação completa ocorre no domínio antes do INSERT; o SQL é uma segunda
barreira. Para textos, o CHECK SQL rejeita vazio e espaços ASCII, tabulações e
quebras de linha. O domínio também rejeita espaços Unicode por `strings.TrimSpace`.
Por isso, gravações da aplicação devem passar pelo repositório, mesmo existindo
restrições no banco. A aula não é normalizada silenciosamente.

## Operações disponíveis

`Registro` reúne `ID`, `Aula` e `DuracaoMinutos`, separando os dados persistidos
do domínio. A duração vem de `Aula.DuracaoMinutos()`; não é recebida do chamador.
O banco também confere se ela coincide com horários e intervalo.

- `Inserir(ctx, aula)`: valida, calcula e grava com parâmetros SQL; retorna o registro.
- `Buscar(ctx, id)`: retorna o registro ou `ErrNaoEncontrada`, verificável com `errors.Is`.
- `Listar(ctx)`: retorna todas as aulas, ordenadas por data, início e ID decrescentes.
  Um banco vazio retorna slice vazio, não `nil`.

A consulta usa o índice `(data DESC, inicio DESC, id DESC)`. Não há filtros,
paginação, edição, exclusão ou sincronização nesta etapa. Observações vazias são
permitidas. IDs são gerados pelo SQLite. As hipóteses de duração da etapa 4 e a
aprovação institucional pendente continuam em [dominio-aula.md](dominio-aula.md).

## Como estudar e verificar

Execute na raiz do projeto:

```bash
cd backend
export DATABASE_PATH="$PWD/aulas-estudo.db"
go run ./cmd/db migrar
go run ./cmd/db listar
go run ./cmd/db inserir-exemplo
go run ./cmd/db listar
go run ./cmd/db migrar
go run ./cmd/db listar
unset DATABASE_PATH
```

Em um arquivo novo, a primeira listagem deve ser `[]`. A inserção cria uma aula
fictícia de 08:00 a 10:00 com intervalo de 15 minutos, totalizando 105 minutos.
Cada execução é um processo diferente e fecha a conexão ao terminar. As listagens
seguintes devem preservar o mesmo ID e os mesmos dados, inclusive após migrar de
novo. Repetir `inserir-exemplo` insere outro registro: não é uma carga automática
nem uma operação idempotente. O JSON do comando serve para inspeção e não define
um contrato HTTP.

Para conferir a configuração padrão, em `backend/`, sem `DATABASE_PATH` no
ambiente, execute `go run ./cmd/db migrar`: o arquivo criado será `aulas.db`.
Os bancos locais e journals são ignorados pelo Git.

Testes automatizados não usam seu banco de estudo: cada caso cria arquivos em
`t.TempDir()`, removidos ao final.

```bash
CGO_ENABLED=0 go test -v ./internal/storage
go test ./...
go vet ./...
go mod verify
CGO_ENABLED=0 go build -o /tmp/aula-sync-db ./cmd/db
```

Os testes cobrem migração inicial e repetida, rollback de migração incompleta,
recusa de versão futura, arquivo persistente, reabertura, busca ausente,
ordenação, dados com aspas, validações do domínio, restrições SQL, limites de
data e duração, configuração padrão e falhas de abertura.

## Validação executada

Em 13/10/2026, passaram os testes de todos os pacotes com `CGO_ENABLED=0`,
`go vet ./...`, `go mod verify` e builds sem CGO de `cmd/db` e `cmd/api`.
O comando compilado foi executado em processos separados contra o novo arquivo
`backend/aulas-demo-etapa5.db`: migração, listagem vazia, inserção do exemplo,
listagem, nova migração e nova listagem. O registro manteve ID 1 e duração de
105 minutos. Esse banco contém apenas o exemplo fictício e está ignorado pelo Git.
Não foi modificado nenhum banco de desenvolvimento preexistente.
