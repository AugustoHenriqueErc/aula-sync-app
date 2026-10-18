# Contrato HTTP v3 — Correção de aulas

Versão documental da etapa 9; as URLs continuam sem prefixo de versão.
Mantém os campos e endpoints do [contrato v2](api-v2.md), com as extensões abaixo.

## Metadados na resposta de uma aula

POST /aulas, GET /aulas/:id, GET /aulas, PUT /aulas/:id e PATCH de situação
retornam também:

```json
{
  "versao": 1,
  "cancelada": false,
  "updated_at": "2026-10-18T14:00:00Z"
}
```

Esses campos acompanham todos os dados já existentes. `updated_at` usa UTC e
pode ter frações de segundo; é vazio em aulas migradas que ainda não foram
alteradas. Novos cadastros recebem versão 1, situação ativa e data atual.
O POST continua recebendo apenas os campos de cadastro definidos no v2.

## Editar: PUT /aulas/:id

Envie a representação completa dos campos editáveis e a versão lida:

```json
{
  "versao": 1,
  "data": "2026-10-18",
  "inicio": "08:00",
  "fim": "10:00",
  "turma_id": 1,
  "conteudo": "Leitura revisada",
  "observacoes": "",
  "intervalo_minutos": 0
}
```

PUT substitui os dados editáveis. Observações e intervalo omitidos assumem vazio
e zero, como no cadastro; não é atualização parcial. Não envie ID, nome da turma,
duração, cancelada ou updated_at. A duração é recalculada no servidor.

Sucesso: 200 com a aula completa, versão incrementada e updated_at atualizado.
Uma aula cancelada precisa ser reativada antes de editar.

## Cancelar ou reativar: PATCH /aulas/:id/situacao

```json
{"versao": 2, "cancelada": true}
```

`false` reativa. `cancelada` precisa ser booleano explícito, não nulo.
Sucesso: 200 com a aula completa. Uma mudança incrementa a versão e registra o
estado anterior. Repetir a situação já existente com a versão atual retorna o
registro sem nova alteração; enviar uma versão antiga retorna conflito.
Não existe endpoint DELETE.

## Consulta por situação

```http
GET /aulas?situacao=ativas&pagina=1
```

`situacao` aceita `ativas` (padrão), `canceladas` ou `todas`. Os filtros inclusivos
de datas, turma_id e páginas de 20 continuam iguais. `total_registros` conta o
conjunto filtrado inteiro. `duracao_total_minutos` soma apenas as aulas ativas
desse conjunto: em Canceladas será zero; em Todas as canceladas não somam.
GET /aulas/:id consulta também uma aula cancelada.

## Erros das alterações

| Status | Situação |
| --- | --- |
| 400 | ID ou JSON inválido, campos desconhecidos, versão ausente/não positiva, situação inválida |
| 404 | Aula inexistente |
| 409 | Versão antiga ou tentativa de editar aula cancelada |
| 413 | Corpo maior que 64 KiB |
| 415 | Content-Type diferente de application/json |
| 422 | Dados de aula inválidos ou turma inexistente; inclui field |
| 500 | Falha interna; mensagem genérica |

Conflito usa `{"error":"mensagem"}`. A aplicação deve recarregar a aula para
conferir a versão atual; não deve reenviar automaticamente com a nova versão.
Cada alteração e a gravação do estado anterior são atômicas. A política de
rastreabilidade, testes, logs e backup está em [correcao-backup.md](correcao-backup.md).
