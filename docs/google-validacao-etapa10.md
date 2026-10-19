# Etapa 10 — Revisão do contrato e validação do ambiente

Status: revisão técnica do desenho realizada; validação da conta Google e
revisão dos cabeçalhos pela responsável pendentes. Este documento não comprova
execução remota nem encerra a etapa 10.

## Informações confirmadas pelo desenvolvedor

- O ambiente de teste usará uma conta Google pessoal.
- Apps Script foi descartado: o backend Go acessará diretamente a API Sheets.
- O login Google determina a mesma conta usada para autorizar leituras e escritas.
- O usuário informa o link de uma planilha própria ou compartilhada com permissão
  de edição. Não é necessário ser proprietário.
- Os cabeçalhos não foram revisados pela responsável; continuam como proposta.

Decisão confirmada em 19/10/2026. O login usa OpenID Connect para identidade e
OAuth 2.0 para autorização. Credenciais e sessão devem pertencer à mesma conta;
o link não concede acesso. O contrato detalhado está em
[google-etapa10.md](google-etapa10.md#login-autorização-e-conta-executora).
Login e leitura direta de metadados implementados; testes automatizados cobrem
o fluxo com provedor simulado. Ensaio real segue pendente. Veja [login-google.md](login-google.md).

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
- Um bloqueio no backend coordena apenas operações dessa instância, mas não impede uma
  pessoa de editar a planilha. A etapa 11 deverá revalidar os dados imediatamente
  antes da escrita e explicitar esse limite; a conta de teste deve evitar edição
  manual durante os ensaios de sincronização.

## Roteiro para validar a conta de teste

1. Configurar projeto Google Cloud, habilitar Sheets API e preparar consentimento,
   usuário de teste, cliente OAuth Web e URI de retorno exata.
2. Implementar login/consentimento no backend e verificar identidade, sessão e
   credenciais da mesma conta. Testar rejeição de callback com `state` inválido.
3. Conferir permissão de edição da conta em uma planilha descartável, própria ou
   compartilhada. Registrar apenas o resultado, sem credenciais ou dados pessoais.
4. Revisar escopos concedidos e proteção dos tokens e segredo no backend.
5. Validar uma leitura autenticada via API Sheets com a conta da sessão. Essa
   leitura comprova acesso de leitura, não capacidade de escrita; não alterar
   dados para testar permissões nesta etapa.
6. Testar recusa de consentimento, sessão ausente, falta de acesso, troca de conta,
   logout e autorização revogada. Não reutilizar tokens ou confirmações da conta anterior.
7. Registrar evidências reais abaixo. A leitura anterior pelo conector não
   comprova autorização do aplicativo.

A etapa 10 implementa e valida login/autorização. Inspeção completa, criação de
abas, escrita e validação efetiva de gravação ficam na etapa 11. Falha de acesso,
inclusive conta com somente leitura ao tentar gravar, deve preservar aulas locais.

### Evidências do ambiente

| Verificação | Situação |
| --- | --- |
| Tipo da conta de teste | Conta pessoal confirmada; execução em nome do usuário logado |
| Projeto Google Cloud, Sheets API e cliente OAuth Web | Configuração pendente de comprovação |
| Login, sessão e vínculo com credenciais da mesma conta | Implementado em memória; testes automatizados aprovados; teste real pendente |
| Planilha própria ou compartilhada com permissão de edição | Pendente de conferência |
| Escopos e consentimento real | Pendentes de validação |
| Proteção de tokens e segredo OAuth | Tokens em memória e segredo no ambiente do backend; conferência do ambiente real pendente |
| Callback inválido, recusa e sessão ausente rejeitados | Testes automatizados aprovados; teste real pendente |
| Troca de conta, logout e revogação | Testes automatizados aprovados; teste real pendente |
| Leitura via Sheets com autorização do aplicativo | Pendente de ensaio remoto |
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

- [OAuth para aplicações web](https://developers.google.com/identity/protocols/oauth2/web-server).
- [OpenID Connect](https://developers.google.com/identity/openid-connect/openid-connect).
- [Escopos da API Sheets](https://developers.google.com/workspace/sheets/api/scopes).
