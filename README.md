# Aula Sync App

Projeto de estudo para registrar aulas e consultar a carga horária. Frontend em
Svelte, Vite e TypeScript; backend em Go com Fiber. O SQLite é a fonte principal
dos registros persistidos localmente.

## Estado atual e MVP

O backend possui um endpoint de saúde JSON, porta configurável e testes HTTP.
O frontend consulta a API e exibe carregamento, sucesso ou erro, com nova tentativa.
O pacote Go `internal/aula` valida aulas e calcula a duração líquida em minutos,
conforme as [hipóteses do MVP](docs/dominio-aula.md), ainda pendentes de revisão
com a professora. A persistência SQLite permite inserir, buscar e listar aulas
pelo repositório, pelo comando de estudo e pela API HTTP. Os endpoints de cadastro
e consulta estão descritos no [contrato v3](docs/api-v3.md). O formulário web
permite cadastrar aulas com prévia, validação e confirmação. O histórico no
navegador oferece filtros inclusivos de datas, seleção de turma/estudante,
páginas de 20 aulas e carga horária de todo o conjunto filtrado. O formulário
permite criar e selecionar turmas com nomes únicos após normalização.
A etapa 9 permite editar aulas, recalcular a duração, cancelar e reativar
registros, com controle de versão e trilha de alterações no SQLite.
O objetivo é construir um MVP local com dados fictícios para apresentar a Maria
Luzia e ajustar as hipóteses após o primeiro contato. Consulte os
[requisitos](docs/requisitos.md) e o [checklist](CHECKLIST_IMPLEMENTACAO.md).
A etapa 10 oferece configuração local, login Google e leitura de metadados;
o desenvolvedor confirmou login e leitura real em 20/10/2026. A etapa 11
implementa inspeção, preparação de abas mensais mediante confirmação e envio
por aula pelo histórico, com recuperação de falhas e proteção contra duplicações
nos cenários testados. A escrita real e a revisão institucional dos cabeçalhos
permanecem pendentes. Veja o [roteiro da etapa 11](docs/google-etapa11.md).

## Navegação por abas

A interface separa **Registrar aula**, **Histórico** e **Integração Google**.
Trocar de aba preserva o formulário, os filtros e o link digitado enquanto a
aplicação permanece aberta. Editar pelo histórico abre a aba de cadastro e leva
o foco à data da aula. A conta conectada permanece no topo; o diagnóstico da
conexão fica recolhido abaixo do painel ativo.

Para testar, alterne entre as abas com um formulário parcialmente preenchido.
Use Tab para chegar à aba selecionada e setas esquerda/direita, Home e End para
navegar. Confira também editar/salvar pelo histórico e o layout em celular.
A regressão automatizada pode ser executada da raiz com
`node scripts/test-etapa9.cjs --google` (Chrome necessário).

## Ferramentas

Ambiente validado em Linux, com instalação em uma cópia limpa do projeto:

- Node.js 22.23.0 e npm 10.9.8. A versão de Node está em `.nvmrc`;
  com nvm instalado, execute `nvm install` e `nvm use` na raiz.
- Go 1.27.0, conforme `backend/go.mod`. A instalação local 1.22.2 baixou
  automaticamente essa toolchain ao executar Go dentro de `backend/`.
  Esse primeiro uso requer acesso à rede; com `GOTOOLCHAIN=local`, é necessário
  já ter Go 1.27.0 instalado.

O frontend declara Node 22 a partir de 22.23.0 e npm 10 a partir de 10.9.8;
estas são as versões exatas testadas. Preserve `frontend/package-lock.json`
e `backend/go.sum` para reproduzir as dependências.

## Executar o frontend

Em um terminal, a partir da raiz do repositório:

```bash
cd frontend
npm ci
npm run dev -- --host 127.0.0.1 --port 5173 --strictPort
```

Abra http://127.0.0.1:5173 para preencher o formulário **Registrar aula**.
Com a API ligada, o cadastro grava no SQLite e confirma a duração calculada.
O diagnóstico abaixo do formulário mostra se o backend está acessível.

## Executar o backend

Com Go instalado, em outro terminal, a partir da raiz:

```bash
cd backend
go version
go mod download
go run ./cmd/api
```

A porta padrão é 3000. Ao iniciar, a API abre o SQLite e aplica as migrações.
O banco padrão é `aulas.db` no diretório de execução; use `DATABASE_PATH` para
escolher outro arquivo. A pasta deve existir. Em outro terminal:

```bash
curl --fail --include http://localhost:3000/health
```

A resposta esperada é HTTP 200, `Content-Type: application/json` e
`{"status":"ok"}`. Esse endpoint confirma que a API responde; ainda não verifica
banco de dados ou serviços externos. A rota `/` mantém a mensagem inicial.
Encerre cada servidor com Ctrl+C.

Para testar outra porta, execute em `backend/`:

```bash
PORT=3001 go run ./cmd/api
```

Em outro terminal, execute `curl --fail --include http://localhost:3001/health`.
Para conferir o erro 404 em JSON, use `curl --include http://localhost:3001/inexistente`;
a resposta terá `{"error":"Not Found"}`.

As rotas ficam em `backend/internal/httpapi/app.go`; o ponto de entrada em
`backend/cmd/api/main.go` lê a configuração e inicia o servidor. O tratamento
central transforma erros retornados pelos handlers em `{"error":"mensagem"}`,
preservando o status HTTP. Erros de validação de aula retornam 422 e também
incluem `field`, com o nome do campo JSON. Erros 5xx usam a mensagem genérica
`Erro interno do servidor` para não expor detalhes internos.

## Configurações de ambiente

O arquivo `.env.example` documenta os valores locais. O backend lê `PORT` do
ambiente do processo; quando ausente ou vazia, usa 3000. Valores que não sejam
inteiros entre 1 e 65535 encerram a inicialização com uma mensagem de erro.
Go não carrega arquivos `.env` automaticamente: use `PORT=3001 go run ./cmd/api`
ou `export PORT=3001` antes de executar. Copiar `.env.example` não carrega a configuração.

`cmd/api` e `cmd/db` leem `DATABASE_PATH`, com padrão `aulas.db` relativo ao
local de execução. Exemplo em `backend/`:
`DATABASE_PATH=aulas-api-estudo.db go run ./cmd/api`.
A API recusa iniciar se o banco não puder ser aberto ou migrado. Ctrl+C ou
SIGTERM solicita o encerramento HTTP, com espera de até cinco segundos pelas
requisições em andamento, antes de fechar o banco.

O frontend funciona com os padrões abaixo sem criar arquivos de configuração.
Para personalizar, crie `frontend/.env.local` com:

```dotenv
VITE_API_URL=/api
API_PROXY_TARGET=http://127.0.0.1:3000
```

`VITE_API_URL` é a base usada pelo `fetch` no navegador. No desenvolvimento,
`/api/health` chega ao Vite, que remove `/api` e encaminha `/health` para
`API_PROXY_TARGET`. O navegador usa a mesma origem do frontend, evitando a
necessidade de CORS entre as portas locais. Os dois servidores continuam separados.

Se iniciar o Go com `PORT=3001`, altere `API_PROXY_TARGET` para
`http://127.0.0.1:3001`. Reinicie o Vite após alterar o arquivo. O `.env.example`
da raiz serve como referência e não é carregado pelo frontend automaticamente.
Variáveis com prefixo `VITE_` são públicas e nunca devem conter segredos.

Mantenha `VITE_API_URL=/api` para usar o proxy configurado. Uma URL absoluta
apontando diretamente para outra origem exigiria configurar CORS no backend.
O proxy atende ao desenvolvimento com `npm run dev`; o build gera arquivos
estáticos e a hospedagem futura precisará encaminhar `/api` ao backend.

## Verificações

No diretório `frontend/`:

```bash
npm run format:check
npm run check
npm run build
```

Para aplicar a formatação do frontend, execute `npm run format`.

No diretório `backend/`:

```bash
gofmt -l .
go mod verify
go test ./...
go vet ./...
go build -o /tmp/aula-sync-backend ./cmd/api
```

`gofmt -l` não deve listar arquivos; para aplicar a formatação, use
`gofmt -w .`. O build em `/tmp` evita deixar um binário na árvore do projeto.

Para conferir as regras de arquivos ignorados, na raiz:

```bash
git check-ignore -v .env backend/aulas.db backend/aulas.db-wal backups/aulas.sql backend/bin/server frontend/dist/index.html
git check-ignore .env.example frontend/package-lock.json backend/go.sum
git diff --check
```

O primeiro comando deve listar as regras correspondentes. O segundo deve ficar
sem saída e retornar código 1: esses arquivos devem continuar versionáveis.
O terceiro deve ficar sem saída e retornar código 0.
O `.gitignore` não remove do histórico arquivos que já estejam versionados.

## Validações por etapa

Os roteiros de teste e registros das etapas 1 a 11 estão em
[Validações das etapas de implementação](docs/validacoes-implementacao.md).

## Exportação de aulas em PDF

No **Histórico**, aplique os filtros e clique em **Exportar PDF**. O arquivo
inclui todas as páginas e o total das aulas ativas, usando os dados locais.
Instruções, modelo proposto e testes: [relatório PDF](docs/relatorio-pdf.md).
