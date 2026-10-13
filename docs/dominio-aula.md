# Domínio de aula — hipóteses do MVP

Status: hipóteses técnicas para demonstração com dados fictícios, registradas em
13/10/2026 antes da implementação da etapa 4. Não são regras institucionais
aprovadas. A revisão com Maria Luzia e a etapa 0 continuam pendentes, conforme
[requisitos](requisitos.md).

## Campos e validação

| Campo Go | Representação | Hipótese |
| --- | --- | --- |
| `Data` | string `AAAA-MM-DD` | Obrigatória; data válida, ano de 0001 a 9999 |
| `Inicio` | string `HH:MM` | Obrigatório; de 00:00 a 23:59 |
| `Fim` | string `HH:MM` | Obrigatório; posterior ao início, no mesmo dia |
| `Turma` | string | Nome da turma ou identificação fictícia do estudante; obrigatório |
| `Conteudo` | string | Descrição do conteúdo; obrigatória |
| `Observacoes` | string | Opcional |
| `IntervaloMinutos` | int | Soma dos intervalos; zero significa sem intervalo |

Textos obrigatórios não podem ser vazios ou conter apenas espaços. A validação
não modifica a entrada. Datas e horários exigem formato exato, sem espaços,
segundos ou fuso embutido. Datas passadas e futuras são permitidas: não há regra
baseada no dia atual. IDs, datas de auditoria e sincronização não pertencem a
esta etapa.

## Cálculo

`duração líquida = minutos do término − minutos do início − intervalo`

O intervalo deve ser não negativo e menor que a duração bruta. Duração zero,
término igual ou anterior ao início e passagem pela meia-noite são rejeitados.
A menor duração válida é um minuto. O registro deve descrever uma aula em um
único dia; suporte a aulas entre dias diferentes depende da descoberta.

A unidade é minuto inteiro de relógio, sem arredondamento e sem conversão para
hora-aula de 45, 50 ou qualquer outra quantidade de minutos. O intervalo é
informado explicitamente; não há desconto automático nem modelagem dos horários
de cada pausa nesta hipótese.

Data e horários representam o calendário e o relógio locais informados pela
usuária. Não representam um instante UTC. A subtração usa minutos desde 00:00,
sem consultar `time.Local`, `time.Now` ou a base de fusos. Portanto, calcula a
diferença civil informada, não o tempo físico transcorrido em uma mudança de
horário de verão. Fuso institucional e tratamento desses eventos continuam
pendentes antes de converter horários em instantes ou integrar com o Google.

## Exemplos fictícios e resultados manuais

Todos usam turma `Turma de demonstração` e conteúdo `Leitura`, salvo indicação.
São referências técnicas propostas, não exemplos aprovados pela professora.

| Caso | Data | Início–fim | Intervalo | Resultado |
| --- | --- | --- | --- | --- |
| Aula comum | 2026-10-13 | 08:00–09:00 | 0 | 60 minutos |
| Aula com intervalo | 2026-10-13 | 08:00–10:00 | 15 | 120 − 15 = 105 minutos |
| Sem arredondamento | 2026-10-13 | 08:07–09:02 | 0 | 55 minutos |
| Duração mínima | 2026-10-13 | 08:00–08:01 | 0 | 1 minuto |
| Intervalo deixa um minuto | 2026-10-13 | 08:00–09:00 | 59 | 1 minuto |
| Limites do dia | 2026-10-13 | 00:00–23:59 | 0 | 1439 minutos |
| Data bissexta | 2024-02-29 | 08:00–09:00 | 0 | 60 minutos |
| Data inexistente | 2026-02-29 | 08:00–09:00 | 0 | Erro em Data |
| Meia-noite | 2026-10-13 | 23:00–01:00 | 0 | Erro em Fim |
| Início igual ao fim | 2026-10-13 | 08:00–08:00 | 0 | Erro em Fim |
| Intervalo negativo | 2026-10-13 | 08:00–09:00 | -1 | Erro em IntervaloMinutos |
| Intervalo ocupa toda a aula | 2026-10-13 | 08:00–09:00 | 60 | Erro em IntervaloMinutos |
| Conteúdo em branco | 2026-10-13 | 08:00–09:00 | 0 | Erro em Conteudo |

## Uso e limites desta etapa

O pacote `backend/internal/aula` contém `Aula`, `Validar()` e
`DuracaoMinutos() (int, error)`. A duração é calculada sob demanda para não manter
um campo calculado que possa ficar incoerente após alterar os horários. O cálculo
valida todo o registro; entradas inválidas retornam zero e erro, nunca uma duração
utilizável. `Validar` retorna o primeiro `*ErroValidacao`, com `Campo` e `Mensagem`,
na ordem Data, Inicio, Fim, Turma, Conteudo, relação entre horários e intervalo.
A representação persistida da etapa 5 é `storage.Registro`, que reúne ID, aula
e duração calculada. A etapa 6 define DTOs HTTP em `internal/httpapi/aulas.go`,
conforme o [contrato JSON v1](api-v1.md). Consulte também a
[persistência SQLite](persistencia-sqlite.md).

Para executar os testes de tabela, em `backend/`:

```bash
go test -v ./internal/aula
go test ./...
go vet ./...
```

A interface web continua exibindo o diagnóstico da etapa 3. A persistência está
disponível pelo repositório, pelo comando de estudo da etapa 5 e pelos endpoints
de cadastro e consulta da etapa 6. O formulário web será a etapa 7.
