# Integração Google — destino configurável e abas mensais

Status: comportamento inicial do MVP acordado com o desenvolvedor em 18/10/2026.
Planejamento das etapas 10 e 11; ainda não implementado. O modelo da escola e os
cabeçalhos serão revisados com Maria Luzia. Não representa aprovação institucional.

## Comportamento acordado

A aplicação recebe o link de uma planilha nas configurações, separado dos campos
da aula. O endereço não fica fixo no código. O MVP deve aceitar uma planilha
vazia, sem exigir abas mensais ou cabeçalhos previamente preparados. Não exige
que o arquivo tenha literalmente zero abas: abas iniciais ou alheias ao envio
serão preservadas. A aplicação prepara apenas os destinos necessários.

A data da aula determina o mês e o ano, nunca o relógio atual nem uma seleção
livre de aba. Padrão inicial de nome: `Registros aulas - MM/AAAA`.

| Data da aula | Período | Aba esperada |
| --- | --- | --- |
| 18/10/2026 | 2026-10 | Registros aulas - 10/2026 |
| 01/11/2026 | 2026-11 | Registros aulas - 11/2026 |
| 18/10/2027 | 2027-10 | Registros aulas - 10/2027 |

Uma aula de outubro de 2026 não pode ser enviada para novembro nem para outubro
de outro ano. O backend determina o destino; a integração confere a correspondência
antes da escrita. A técnica exata de vínculo e validação será definida na etapa 10;
o nome visível sozinho não encerra essa validação.

## Preparação do destino

Antes de enviar, a aplicação identifica a planilha, o período, a aba esperada e
se sua estrutura está pronta. O comportamento é:

| Condição | Ação esperada |
| --- | --- |
| Aba mensal ausente | Perguntar se o usuário deseja criá-la com os cabeçalhos do modelo |
| Usuário confirma a criação | Criar a aba e os cabeçalhos; conferir o resultado antes de enviar |
| Usuário recusa | Manter a aula local e informar a pendência de preparação; não criar a aba nem enviar |
| Aba mensal existente e completamente vazia | Oferecer a criação dos cabeçalhos, também mediante confirmação |
| Aba mensal existente com cabeçalhos compatíveis | Usar a aba correspondente ao período |
| Aba com conteúdo e cabeçalhos ausentes ou incompatíveis | Informar as diferenças e bloquear o envio; preservar o conteúdo existente |
| Planilha inacessível ou falta de permissão | Informar o problema; não interpretar a falha como aba ausente |

A simples configuração do link não autoriza criar abas, preencher cabeçalhos ou
sincronizar aulas. Cada confirmação deve identificar claramente a planilha e os
meses envolvidos. Em um envio com vários meses, apresentar os destinos ausentes
antes de pedir confirmação. O link identifica o arquivo; não substitui a autorização
de acesso do Apps Script.

## Troca de planilha

Guardar a configuração no SQLite. Ao trocar o destino, verificar as abas dos
meses dos registros que se pretende enviar. Se ainda não houver registros a
enviar, não criar abas mensais antecipadamente.

A troca não apaga dados locais nem move ou apaga linhas da planilha anterior.
Enviar aulas ao novo destino exige uma ação explícita. A aplicação não pode
mostrar uma aula como sincronizada no novo destino apenas porque foi enviada
a outro anteriormente.

O contrato deve associar a sincronização à aula, sua versão, ao período e ao
destino (planilha e aba). Preservar as referências anteriores necessárias para
rastrear os envios. Se o destino mudar durante uma confirmação ou uma requisição,
não reaproveitar a confirmação para escrever em outro arquivo. Conferir novamente
o destino antes de uma escrita após alterações de configuração ou estrutura.

## Separação entre cadastro e envio

Cadastro, edição, consulta e cálculo de duração continuam locais, com SQLite
como fonte principal. Falta de aba, recusa de criação, cabeçalhos incompatíveis
ou indisponibilidade do Google não impedem salvar uma aula localmente.
A interface deve distinguir pendência de preparação de envio confirmado.

Se uma aula já enviada mudar de mês por edição, será necessário reconciliar sua
representação no destino anterior e no novo. O procedimento exato, inclusive
recuperação de falhas parciais, permanece uma tarefa do contrato da etapa 10,
com implementação e testes na etapa 11. O mesmo vale para alterações manuais,
renomeação de abas e correções/cancelamentos após sincronizar.

## Distribuição no plano

### Etapa 10 — Preparação

- Definir a configuração editável da planilha e sua persistência no SQLite.
- Especificar período, nomes mensais e vínculo com identificadores de destino.
- Propor o esquema de colunas e as regras de compatibilidade de cabeçalhos.
- Definir os contratos de inspeção, confirmação de preparação e sincronização.
- Definir autenticação e permissões: o link não é uma credencial.
- Modelar estado por destino/versão, troca de planilha e edição que muda o mês.
- Preparar uma planilha vazia de teste com dados fictícios e revisar os cenários.

### Etapa 11 — Execução no Google

- Inspecionar a planilha e validar a aba e seus cabeçalhos.
- Pedir confirmação e criar abas/cabeçalhos somente quando autorizado.
- Enviar os registros para o período correto e registrar o resultado.
- Implementar repetição sem duplicatas, mudanças de destino/período e recuperação.
- Validar tudo em ambiente de teste antes de qualquer uso com dados reais.

Registrar este acordo não conclui a etapa 10 nem antecipa código da etapa 11.

## Cenários de aceitação planejados

1. Configurar uma planilha vazia sem editar o código.
2. Preparar uma aula de 18/10/2026: solicitar a aba de 10/2026, nunca a de outro mês.
3. Aceitar a criação: conferir aba e cabeçalhos antes de escrever a aula.
4. Recusar: conferir que a aula permanece no SQLite e nada foi criado no Google.
5. Usar uma aba mensal totalmente vazia: pedir autorização para os cabeçalhos.
6. Encontrar conteúdo sem cabeçalhos válidos: explicar o problema sem sobrescrever.
7. Repetir a preparação/envio, inclusive após perda de resposta: não duplicar abas
   ou aulas; definir a proteção contra chamadas concorrentes no contrato.
8. Trocar de planilha com aulas de outubro e novembro: verificar ambos os meses e
   não herdar indevidamente o estado de sincronização do destino anterior.
9. Trocar de ano: outubro de 2026 e outubro de 2027 devem ter destinos distintos.
10. Alterar o destino após uma confirmação ou renomear/modificar a aba: impedir
    escrita com validação desatualizada e orientar nova conferência.
11. Simular falta de acesso: preservar dados locais e não oferecer criação como
    se a planilha tivesse sido lida com sucesso.
12. Editar o mês de uma aula já enviada: validar a política de reconciliação
    definida na etapa 10, evitando contagem indevida nos dois meses.

## Pendências de definição

- Colunas, ordem, tipos, cabeçalhos e apresentação: proposta a comparar com a
  planilha padrão da escola e revisar com Maria Luzia.
- Critério preciso de compatibilidade (incluindo colunas extras e fórmulas).
- Estratégia de identificação confiável do período e reação a abas renomeadas.
- Autenticação, permissões e local de execução do Apps Script.
- Política de alterações manuais e reconciliação de mudança de mês/destino.

Não foram escolhidos valores finais para essas decisões apenas por registrar
o comportamento inicial.
