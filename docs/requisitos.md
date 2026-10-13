# Requisitos — Aula Sync App

Status: rascunho para revisão com Maria Luzia, elaborado em 12/10/2026.
Base: [roadmap](../ROADMAP.md) e [roteiro de perguntas](perguntas-para-maria-luzia.md).
Ainda não foram registradas respostas ou aprovação da professora.

## Decisão sobre a primeira demonstração

Em 12/10/2026, o desenvolvedor decidiu construir o MVP local antes do primeiro
contato com Maria Luzia, para apresentar o fluxo e ajustar o produto a partir do
feedback. A descoberta permanece pendente, mas não bloqueia a preparação técnica.

A demonstração usará dados fictícios. Campos e regras adotados para o MVP deverão
ser registrados como hipóteses ao chegar à etapa de domínio, acompanhados de
exemplos de teste. Eles não representam regras institucionais aprovadas.
As hipóteses técnicas adotadas na etapa 4 e seus exemplos fictícios estão em
[domínio de aula](dominio-aula.md). A implementação não representa aprovação
das regras pela professora.

O objetivo é chegar ao cadastro, persistência, histórico e total de carga horária,
seguindo uma etapa por vez até o marco C. A integração Google fica para depois.

## História principal

Como professora, quero registrar uma aula uma única vez, consultar esse registro
no histórico e visualizar a carga horária calculada, para reduzir o preenchimento
repetido e facilitar a conferência das minhas aulas.

## Proposta para apresentar

O primeiro fluxo local previsto é preencher os dados de uma aula, salvar, consultar
o histórico e conferir a carga horária. O SQLite será a fonte principal dos dados.
A integração com os arquivos Google será desenvolvida depois do funcionamento local.

Este documento descreve o comportamento esperado; a aplicação ainda não oferece
esse fluxo. A implementação seguirá uma etapa do checklist por vez.

## Critérios de aceitação propostos

Estes critérios devem ser revisados com a professora. Campos, regras e resultados
esperados dependem das decisões pendentes.

| ID | Cenário | Resultado esperado |
| --- | --- | --- |
| CA01 | Aula preenchida com os campos obrigatórios e valores válidos acordados | Cadastro confirmado e registro disponível no histórico |
| CA02 | Campo obrigatório ausente ou valor que viola uma regra confirmada | Problema explicado e registro inválido não salvo |
| CA03 | Consulta de uma aula cadastrada | Dados correspondem ao registro salvo e duração corresponde ao exemplo aprovado |
| CA04 | Aplicação encerrada e iniciada novamente | Aula permanece disponível, persistida no SQLite |
| CA05 | Histórico filtrado por um período acordado | Lista contém as aulas do período e total corresponde a todo o conjunto filtrado |
| CA06 | Nenhuma aula corresponde ao filtro | Mensagem de ausência de registros e total zero |
| CA07 | Serviço Google indisponível | Cadastro e consulta locais continuam funcionando sem depender do Google |

## Decisões pendentes

| Tema | O que confirmar | Fonte esperada |
| --- | --- | --- |
| Processo atual | Onde a aula é anotada, quais dados são repetidos e quem os confere | Relato da professora e amostra anonimizada |
| Unidade do registro | Registro por aula, estudante, turma ou outro agrupamento | Perguntas 8 e 9 do roteiro |
| Campos | Nomes, tipos, obrigatoriedade e opções permitidas | Perguntas 6 a 11 e colunas da planilha |
| Duração | Horários, intervalos, hora-aula, arredondamento, fuso e passagem pela meia-noite | Perguntas 12 a 19 |
| Histórico e totais | Filtros, períodos e agrupamentos necessários | Perguntas 20 e 24 |
| Correções e cancelamentos | Quem pode alterar, limites e efeito nos totais | Perguntas 18 e 21 a 25 |
| Acesso | Quem cadastra, consulta e confere cada informação | Revisão com a responsável pelo processo |
| Correspondência com a planilha | Se cada coluna pode ser preenchida a partir dos dados registrados | Amostra anonimizada do caderno e da linha correspondente |

Nenhuma dessas decisões está confirmada. Não adotar automaticamente uma duração
de hora-aula, arredondamento ou política de intervalos.

## Exemplos para validar na conversa

Preencher os casos com informações fictícias ou anonimizadas, preservando as regras
do processo real. A professora deverá informar o resultado esperado antes de o
exemplo ser utilizado como referência para a implementação.

| Exemplo | Dados de entrada | Resultado esperado | Situação |
| --- | --- | --- | --- |
| Aula comum | A preencher | Duração e representação na planilha a confirmar | Pendente |
| Aula com intervalo | A preencher | Desconto do intervalo e duração a confirmar | Pendente |
| Registro inválido | A preencher | Motivo da rejeição a confirmar | Pendente |

## Como revisar este documento

1. Ler a história principal com Maria Luzia e verificar se representa sua necessidade.
2. Percorrer um registro anonimizado do caderno até a linha correspondente na planilha.
3. Registrar as respostas e resolver os campos e as regras da tabela de pendências.
4. Preencher os três exemplos e conferir os resultados manualmente com a professora.
5. Revisar cada critério de aceitação, registrar ajustes e identificar quem aprovou e quando.

Revisão pela professora: pendente. A escrita deste rascunho não conclui a etapa 0
nem confirma os demais itens do checklist.
