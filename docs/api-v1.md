# Contrato HTTP v1 — aulas

Contrato inicial da etapa 6, versionado neste arquivo e no Git. A versão não faz
parte da URL nesta etapa. Base local: `http://localhost:3000`. As regras de aula
são as [hipóteses do MVP](dominio-aula.md), ainda pendentes de revisão institucional.

## Cadastro: POST /aulas

Envie `Content-Type: application/json` (parâmetro `charset` é aceito), com um único
objeto JSON. Campos desconhecidos, incluindo `id` e `duracao_minutos`, são
rejeitados. O cliente não escolhe o ID nem informa a duração calculada.

```json
{
  "data": "2026-10-13",
  "inicio": "08:00",
  "fim": "10:00",
  "turma": "Turma fictícia",
  "conteudo": "Leitura de demonstração",
  "observacoes": "Exemplo da API",
  "intervalo_minutos": 15
}
```

`data`, `inicio`, `fim`, `turma` e `conteudo` são obrigatórios. Datas usam
`AAAA-MM-DD`, horários usam `HH:MM`. Campos obrigatórios ausentes ou nulos são
tratados como vazios e rejeitados pela validação. `observacoes` e
`intervalo_minutos` são opcionais: ausentes ou nulos assumem `""` e `0`.
O intervalo, quando informado, deve ser um número inteiro. Textos não são
normalizados silenciosamente. O limite de corpo da API é 64 KiB.

Sucesso: **201 Created**, `Location: /aulas/1` e o registro completo:

```json
{
  "id": 1,
  "data": "2026-10-13",
  "inicio": "08:00",
  "fim": "10:00",
  "turma": "Turma fictícia",
  "conteudo": "Leitura de demonstração",
  "observacoes": "Exemplo da API",
  "intervalo_minutos": 15,
  "duracao_minutos": 105
}
```

O ID do exemplo é ilustrativo. Cada POST válido cria outro registro, mesmo com
conteúdo idêntico. O repositório valida o domínio e calcula a duração antes do
INSERT; somente depois da gravação confirmada a API responde 201.

## Consulta: GET /aulas/:id

Retorna **200 OK** com o mesmo objeto do cadastro. O ID deve conter apenas
dígitos decimais, ser positivo e caber em inteiro de 64 bits. Um ID malformado
retorna 400; um ID válido que não existe retorna 404.

## Listagem: GET /aulas

Retorna **200 OK** com um array dos objetos acima. Banco vazio retorna `[]`.
A ordenação é data decrescente, início decrescente e ID decrescente como desempate.
A listagem inicial retorna todos os registros do MVP local, sem paginação ou
filtros. Parâmetros de consulta são rejeitados com 400 para não aparentar que
um filtro ou limite foi aplicado. Paginação será definida quando necessária;
filtros e totalizações pertencem à etapa de histórico.

## Erros

Todas as respostas de erro são JSON e preservam o campo `error` da API inicial.
Erros do domínio acrescentam `field`, com o nome do campo no JSON. É retornado
apenas o primeiro problema, seguindo a ordem de validação do domínio.

| Status | Situação | Exemplo de corpo |
| --- | --- | --- |
| 400 | JSON malformado, tipo incompatível, campo desconhecido, corpo vazio ou não objeto | `{"error":"Envie um único objeto JSON com os campos e tipos documentados"}` |
| 400 | ID inválido | `{"error":"O ID deve ser um inteiro positivo"}` |
| 400 | Parâmetro não suportado em GET /aulas | `{"error":"A listagem ainda não aceita parâmetros de consulta"}` |
| 404 | Aula inexistente | `{"error":"Aula não encontrada"}` |
| 413 | Corpo maior que o limite | Erro JSON de tamanho do Fiber |
| 415 | Content-Type ausente ou diferente de application/json | `{"error":"Use Content-Type: application/json"}` |
| 422 | Aula viola uma regra do domínio | `{"error":"o término deve ser posterior ao início, no mesmo dia","field":"fim"}` |
| 500 | Falha interna, inclusive SQLite | `{"error":"Erro interno do servidor"}` |

Mensagens internas de SQL, caminhos e detalhes da implementação não são enviados
em erros 500. Entradas rejeitadas não devem inserir registros.
`/health` mantém `200 {"status":"ok"}`; verifica a resposta da API, não monitora
continuamente o banco.

## Como testar

Em `backend/`, inicie com um banco de estudo:

```bash
DATABASE_PATH=aulas-api-estudo.db go run ./cmd/api
```

O servidor abre o SQLite e aplica as migrações ao iniciar. Se isso falhar, não
começa a atender requisições. Ctrl+C encerra o servidor e fecha a conexão.
Em outro terminal, a partir da raiz do projeto:

```bash
curl --include http://localhost:3000/aulas
curl --include -X POST http://localhost:3000/aulas \
  -H 'Content-Type: application/json' \
  --data-binary @docs/exemplos/aula.json
curl --include http://localhost:3000/aulas/1
curl --include http://localhost:3000/aulas
curl --include http://localhost:3000/aulas/999999
curl --include http://localhost:3000/aulas/abc
curl --include -X POST http://localhost:3000/aulas \
  -H 'Content-Type: application/json' --data '{'
curl --include -X POST http://localhost:3000/aulas \
  -H 'Content-Type: application/json' --data '{}'
```

Use o ID recebido no cadastro ao consultar. `{}` é JSON válido, mas resulta em
422 para `data`; `{` é JSON malformado e resulta em 400. Para comparar com o banco,
execute em `backend/`: `DATABASE_PATH=aulas-api-estudo.db go run ./cmd/db listar`.
Encerre e reinicie a API com o mesmo caminho: o registro deve permanecer.

Os testes automatizados usam HTTP em memória e SQLite em arquivos temporários:

```bash
cd backend
go test -v ./internal/httpapi
go test ./...
go vet ./...
```

O [formulário da etapa 7](formulario-aula.md) consome POST /aulas e mantém o
diagnóstico de conexão disponível. O histórico web será a etapa 8.
