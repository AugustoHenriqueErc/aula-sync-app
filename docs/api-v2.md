# Contrato HTTP v2 — histórico e turmas

> Base da etapa 8. Consulte as [extensões do contrato v3](api-v3.md) para
> metadados, edição, cancelamento e o filtro padrão de aulas ativas.

Contrato local da etapa 8. A versão é documental; os caminhos continuam sem
prefixo de versão. Substitui o contrato v1: POST /aulas recebe `turma_id` em vez
de texto livre, e GET /aulas retorna um objeto com página e totais.

## Cadastro de turmas/estudantes

`GET /turmas` retorna 200 e um array ordenado pelo nome normalizado:

```json
[{"id": 1, "nome": "Augusto"}]
```

Sem cadastros: `[]`. Não aceita parâmetros. O mesmo catálogo representa turmas
ou estudantes, sem cadastro pessoal adicional.

`POST /turmas`, com Content-Type application/json:

```json
{"nome": "  Augusto  "}
```

Retorna 201 com `{"id":1,"nome":"Augusto"}`. O nome exibido preserva a
capitalização, removendo espaços externos e reduzindo espaços internos repetidos.
A chave de comparação usa essa mesma limpeza e conversão para minúsculas no Go.
Acentos são preservados; não há remoção de acentos ou normalização Unicode NFC.
`Augusto` e `augusto` conflitam; `João` e `Joao` são distintos. Homônimos são
tratados como a mesma opção nesta hipótese do MVP.

O SQLite garante UNIQUE na chave normalizada, inclusive em pedidos simultâneos.
Duplicidade retorna 409 com `error`; nome vazio retorna 422 com `field: nome`.
JSON inválido, campos desconhecidos ou objetos concatenados retornam 400;
Content-Type incorreto retorna 415. A criação é explícita e independente da aula:
se o usuário desistir do formulário, a turma criada continua disponível.

## Cadastro de aula

`POST /aulas`, Content-Type application/json:

```json
{
  "data": "2026-10-13",
  "inicio": "08:00",
  "fim": "10:00",
  "turma_id": 1,
  "conteudo": "Leitura",
  "observacoes": "Exemplo fictício",
  "intervalo_minutos": 15
}
```

Use o ID retornado pela criação ou listagem de turmas. `turma_id` deve ser um
inteiro positivo existente. Ausente, nulo, zero ou inexistente retorna 422 com
`field: turma_id`. Tipo incompatível retorna 400. O antigo campo `turma` na
entrada é rejeitado como desconhecido. Os demais campos e regras de duração
permanecem os do domínio; observações e intervalo continuam opcionais.

Sucesso: 201, Location `/aulas/1`, com o registro completo:

```json
{
  "turma_id": 1,
  "id": 1,
  "data": "2026-10-13",
  "inicio": "08:00",
  "fim": "10:00",
  "turma": "Augusto",
  "conteudo": "Leitura",
  "observacoes": "Exemplo fictício",
  "intervalo_minutos": 15,
  "duracao_minutos": 105
}
```

`GET /aulas/:id` retorna o mesmo registro com 200, ou 404 se inexistente;
ID malformado retorna 400. Um POST repetido cria outra aula: esta etapa não
introduz idempotência de cadastro.

## Histórico

```http
GET /aulas?data_inicial=2026-10-01&data_final=2026-10-31&turma_id=1&pagina=1
```

Todos os parâmetros são opcionais. Ausência de filtros consulta todas as aulas;
página padrão 1, tamanho fixo 20. Não há parâmetro para alterar o tamanho.

- `data_inicial` e `data_final`: AAAA-MM-DD, ano 0001 a 9999, datas válidas.
- Ambos os extremos são inclusivos; limites podem ser usados separadamente.
- Início posterior ao fim é inválido.
- `turma_id`: inteiro positivo; um ID sem aulas resulta em conjunto vazio.
- `pagina`: inteiro positivo, limitado para evitar overflow no cálculo do offset.
- Parâmetros desconhecidos, vazios, repetidos ou inválidos retornam 400.

Resposta 200 (exemplo sem resultados):

```json
{
  "aulas": [],
  "pagina": 1,
  "por_pagina": 20,
  "total_registros": 0,
  "duracao_total_minutos": 0
}
```

Cada elemento de `aulas` tem o mesmo formato da consulta individual. A ordenação
é data, início e ID decrescentes. Quantidade e duração total correspondem ao
conjunto filtrado inteiro, nunca apenas à página. Uma página além do fim retorna
array vazio, mantendo os totais do filtro. Página e totais são lidos na mesma
transação. A interface mostra a duração em horas e minutos e volta à página 1
ao aplicar ou limpar filtros. Após cadastro, consulta novamente a página e os
filtros ativos; uma aula fora do filtro não aparece naquela consulta.

## Erros e saúde

O formato continua `{"error":"mensagem"}`, com `field` em erros de domínio.
Falhas internas retornam 500 genérico, sem detalhes SQL. O corpo tem limite de
64 KiB (413). GET /health continua verificando somente a disponibilidade HTTP.

## Exemplo de uso

Com a API ligada, crie uma turma e anote o ID retornado:

```bash
curl --include http://localhost:3000/turmas \
  -H 'Content-Type: application/json' --data '{"nome":"Augusto"}'
curl --include http://localhost:3000/turmas
```

Ajuste `turma_id` no corpo do exemplo para o ID obtido:

```bash
curl --include http://localhost:3000/aulas \
  -H 'Content-Type: application/json' --data-binary @docs/exemplos/aula.json
curl --include 'http://localhost:3000/aulas?data_inicial=2026-10-01&data_final=2026-10-31&pagina=1'
```

Comandos executados a partir da raiz. Testes e cenários de interface em
[historico-aulas.md](historico-aulas.md).
