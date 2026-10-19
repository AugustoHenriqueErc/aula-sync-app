# Etapa 10 — Revisão do contrato e validação do ambiente

Status: revisão técnica do desenho realizada; validação da conta Google e
revisão dos cabeçalhos pela responsável pendentes. Este documento não comprova
execução remota nem encerra a etapa 10.

## Informações confirmadas pelo desenvolvedor

- O ambiente de teste usará uma conta Google pessoal.
- Ainda não existe projeto Apps Script para o aplicativo.
- Os cabeçalhos não foram revisados pela responsável; continuam como proposta.
- Foi solicitado acesso com a conta Google da pessoa. Falta esclarecer se cada
  pessoa conectará suas próprias planilhas ou se o destino será administrado
  por uma conta central. Essa escolha afeta a proposta de autenticação anterior.

Entrar com Google identifica a pessoa; acessar uma planilha exige autorização
adicional. O desenho anterior de Apps Script executado pela conta responsável
não passa a usar as permissões de outra pessoa apenas porque ela entrou no app.
O mecanismo definitivo será ajustado depois de esclarecer esse fluxo.

Mudanças de cabeçalhos antes do primeiro envio exigem atualizar o contrato e os
testes correspondentes. Depois de existirem dados remotos, exigem versionamento
do modelo e migração explícita, preservando linhas existentes. O SQLite mantém
o modelo local independente da apresentação na planilha.

## Revisão dos cenários

Simulação de mesa baseada no [contrato proposto](google-etapa10.md). Os resultados
abaixo são requisitos para os testes da etapa 11, não resultados de chamadas ao
Google. Use apenas aulas fictícias no ambiente descartável.

| Cenário | Resultado exigido pelo contrato |
| --- | --- |
| Planilha com apenas a aba inicial | Preservar a aba inicial; informar ausência da aba mensal sem criar nada durante a inspeção. |
| Aba mensal completamente vazia | Oferecer cabeçalhos; escrever somente após confirmação vinculada ao destino e período. |
| Fórmula que exibe texto vazio | Considerar conteúdo existente; não tratar a aba como vazia. |
| Cabeçalhos diferentes, extras ou com fórmulas | Apresentar incompatibilidade e preservar o conteúdo. |
| Recusa de preparação | Manter aula local e pendência; nenhuma escrita remota. |
| Falta de acesso | Informar falha de acesso, sem deduzir ausência de aba. |
| Duas confirmações para a mesma aba | Reinspecionar sob bloqueio; aproveitar a aba compatível já criada, sem duplicar nem sobrescrever. |
| Destino alterado após confirmação | Invalidar a confirmação; exigir nova inspeção do destino atual. |
| Aba renomeada ou substituída por outra com o mesmo nome | Comparar ID, nome, período e cabeçalhos; bloquear vínculo divergente. |
| Mesmo envio repetido após perda da resposta | Localizar a chave instalação + aula e comparar versão e conteúdo; confirmar a linha existente. |
| Duas chamadas concorrentes da mesma aula | Serializar a localização e escrita; uma única representação ativa por planilha. |
| Versão antiga chega depois de uma nova | Rejeitar a antiga; não sobrescrever a nova. |
| Mesma versão com conteúdo diferente ou chave duplicada | Retornar conflito; não escolher uma linha arbitrariamente. |
| Edição local durante envio | Guardar a confirmação da versão enviada; a versão local mais nova continua pendente. |
| Cancelamento | Atualizar situação e versão sem apagar a linha; excluir da carga horária ativa. |
| Outubro/2026 e outubro/2027 | Usar períodos e abas distintos, derivados da data civil da aula. |
| Mudança de outubro para novembro | Marcar a representação anterior como movida antes de ativar a nova; repetir deve retomar uma operação parcial. |
| Troca da planilha A pela B e retorno à A | Preservar referências por destino; comparar versão e período atuais antes de considerar sincronizado. |
| Timeout ou retorno HTML de login | Não marcar sucesso nem repetir automaticamente; conferir o resultado antes de reenviar. |

### Pontos esclarecidos na revisão

- A confirmação pertence à revisão da configuração, destino, período e ação.
  Ela não é uma autorização genérica para qualquer planilha.
- `request_id` correlaciona uma chamada e sua resposta. A chave instalação +
  aula fornece a identidade idempotente, inclusive em tentativas com outro
  `request_id`.
- A versão local continua sendo a referência após uma resposta atrasada.
- Detectar alteração manual exige comparar o conteúdo remoto com uma referência
  da última escrita confirmada. A implementação da etapa 11 deverá persistir
  essa referência (conteúdo ou assinatura canônica) e a versão do modelo usada
  no vínculo. As tabelas atuais ainda não armazenam esses dois elementos.
- Um bloqueio do Apps Script coordena execuções do script, mas não impede uma
  pessoa de editar a planilha. A etapa 11 deverá revalidar os dados imediatamente
  antes da escrita e explicitar esse limite; a conta de teste deve evitar edição
  manual durante os ensaios de sincronização.

## Roteiro para validar a conta de teste

1. Identificar a conta que executará o Apps Script e confirmar se é pessoal ou
   institucional. Usar essa mesma conta ao conferir a planilha e o projeto.
2. Na planilha descartável, conferir proprietário e papel de editor da conta
   executora. Registrar apenas o resultado; não copiar e-mails de terceiros ou
   credenciais para o repositório.
3. No projeto Apps Script independente, conferir as opções de implantação como
   Web App. O desenho proposto exige execução pela conta responsável e acesso
   do backend sem sessão de navegador. Se a conta não permitir essa modalidade,
   registrar a restrição e revisar a autenticação antes de implementar o adaptador.
4. Revisar o escopo `https://www.googleapis.com/auth/spreadsheets`, exigido por
   `SpreadsheetApp.openById`. O desenho atual não utiliza Gmail, Docs ou Drive.
   O consentimento da conta e a permissão sobre a planilha são verificações
   distintas. O escopo permite acesso às planilhas autorizadas para essa conta;
   não fica restrito automaticamente ao ID configurado no aplicativo.
5. Planejar `API_SECRET` nas Script Properties e
   `GOOGLE_APPS_SCRIPT_SECRET` no ambiente exclusivo do backend. Restringir quem
   pode editar o projeto; Script Properties não são um cofre isolado dos editores.
   Não enviar o segredo na conversa nem registrá-lo em logs, URL ou variáveis
   `VITE_`. Não há necessidade de gerar um segredo antes de existir um ambiente
   de teste definido.
6. Registrar as evidências na tabela abaixo. Uma opção disponível na interface
   não comprova uma chamada autenticada bem-sucedida. Essa comprovação exige
   um ensaio remoto identificado, ainda pendente.

O adaptador de inspeção/preparação/envio pertence à etapa 11. Se for necessário
um ensaio de implantação na etapa 10, seu escopo deve ficar restrito a comprovar
autenticação e autorização, sem enviar aulas nem preparar abas.

### Evidências do ambiente

| Verificação | Situação |
| --- | --- |
| Tipo da conta de teste | Conta pessoal confirmada; identidade executora depende do fluxo de conexão escolhido |
| Projeto Apps Script | Ainda não criado, conforme informado pelo desenvolvedor |
| Propriedade e permissão de edição da planilha | Pendente de confirmação na conta executora |
| Modalidade de implantação permitida pela conta | Pendente de verificação no Apps Script |
| Escopo necessário revisado tecnicamente | `spreadsheets`; consentimento real ainda pendente |
| Segredo armazenado nos dois ambientes | Pendente; nenhum segredo gerado nesta revisão |
| Chamada sem segredo e com segredo incorreto rejeitadas | Pendente de ensaio remoto |
| Chamada válida sem sessão de navegador | Pendente de ensaio remoto |
| Cabeçalhos aprovados pela responsável | Pendente de revisão |

## Revisão dos cabeçalhos com a responsável

Apresentar a tabela de colunas de [google-etapa10.md](google-etapa10.md) e uma
aula fictícia: 18/10/2026, 08:00–10:00, intervalo de 15 minutos, duração de
105 minutos. Confirmar nomes, ordem, formato de data/horário e o uso de minutos
como unidade armazenada. Esclarecer que canceladas e movidas não entram no total.
Verificar se a instituição aceita as colunas técnicas `chave_aula`, `versao`,
`situacao` e `updated_at` no mesmo modelo.

Registrar data, responsável pela revisão e ajustes solicitados. Sem essa
resposta, manter o modelo como proposta, mesmo que os testes locais passem.

## Referências oficiais consultadas

- [Web Apps: implantação, identidade executora e URL de teste](https://developers.google.com/apps-script/guides/web).
- [Autorização para serviços Google](https://developers.google.com/apps-script/guides/services/authorization).
- [Escopo de SpreadsheetApp.openById](https://developers.google.com/apps-script/reference/spreadsheet/spreadsheet-app#openById(String)).
- [Script Properties e seu compartilhamento dentro do projeto](https://developers.google.com/apps-script/guides/properties).
- [LockService: exclusão mútua entre execuções do script](https://developers.google.com/apps-script/reference/lock/lock-service).
