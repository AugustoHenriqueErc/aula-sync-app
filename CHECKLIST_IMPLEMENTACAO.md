# Checklist de implementação — Aula Sync App

Análise inicial: 12/10/2026.

Base: [ROADMAP.md](ROADMAP.md), [AGENTS.md](AGENTS.md), código e configurações existentes, [perguntas para Maria Luzia](docs/perguntas-para-maria-luzia.md) e [fluxo de estudo](workflow.mmd).

## Parecer de viabilidade

**Sim, o plano é tecnicamente viável para o escopo incremental descrito.** A separação entre frontend, API, regras de domínio e persistência permite construir e validar o fluxo local antes da integração. Não foi identificado, na leitura do repositório, impedimento estrutural que exija trocar a arquitetura proposta.

O parecer inicial foi baseado em inspeção estática. A etapa 1 foi posteriormente validada com builds, verificações e respostas HTTP locais, conforme o registro abaixo; não foram feitas chamadas ao Google. Viabilidade não significa que as etapas já estejam implementadas, nem permite estimar prazo sem conhecer disponibilidade e requisitos finais.

### Condições e decisões pendentes

- **Regras reais:** ainda não há respostas registradas ao questionário. Para o MVP demonstrativo, campos e regras poderão ser definidos como hipóteses explícitas na etapa 4; a confirmação institucional ocorrerá após a apresentação, conforme a decisão em `docs/requisitos.md`.
- **Ambiente reproduzível:** Go 1.27.0, Node.js 22.23.0 e npm 10.9.8 validados na etapa 1. Versões e comandos documentados no README.
- **Integração Google:** a viabilidade operacional depende de permissões, autenticação e limites do ambiente institucional. Comprovar esses pontos com a planilha de teste nas etapas 10 e 11.
- **Correções após sincronizar:** definir como edições e cancelamentos locais atualizam o Google, como detectar uma versão desatualizada e o que fazer se alguém editar a planilha manualmente. O SQLite continua sendo a fonte principal.
- **Relatório PDF:** gerar a partir dos dados locais e dos filtros do histórico, independentemente do Google. O modelo inicial é uma proposta a revisar com Maria Luiza.
- **Destinos futuros:** Drive e Docs ficam para reavaliação após o PDF; se adotados, acompanhar cada destino separadamente para recuperar falhas parciais sem duplicações.
- **Piloto:** manter as etapas anteriores à autenticação restritas a desenvolvimento/testes com dados fictícios ou anonimizados. O controle de acesso deve existir antes do uso com dados reais.
- **Hospedagem:** confirmar volume e concorrência esperados e planejar armazenamento persistente, migrações e restauração do SQLite na etapa 13.

## Estado observado na análise inicial (12/10/2026)

Retrato histórico da análise inicial. O progresso posterior está registrado nas
etapas e no registro de validação abaixo; esta tabela não representa o estado atual.

| Área | Evidência no repositório | Situação |
| --- | --- | --- |
| Planejamento | `ROADMAP.md`, `AGENTS.md` e questionário | Documentados; requisitos ainda precisam de confirmação |
| Backend mínimo | `backend/main.go`: Fiber, `GET /` com texto e porta fixa `3000` | Build, análise estática e resposta HTTP validados na etapa 1 |
| Frontend mínimo | `frontend/src/App.svelte`: mensagem estática | Código presente; ainda não consome a API |
| Ferramentas do frontend | `package.json`, `package-lock.json`, configurações Svelte/Vite/TypeScript | Scripts de formatação, checagem e build validados na etapa 1 |
| Documentação de execução | README da raiz com comandos e limitações atuais | Comandos reproduzidos em cópia limpa do projeto |
| Configuração | `.env.example` com valores previstos e carregamento explicado no README | Modelo documentado; consumo nas etapas 2 e 3 |
| Arquivos ignorados | `.gitignore` na raiz e no frontend | Regras conferidas com `git check-ignore` |
| Saúde e comunicação | Não há `GET /health` nem chamada `fetch` no componente principal | Pendente |
| Domínio e persistência | Não foram encontrados modelo de aula, driver SQLite ou migrações | Pendente |
| Testes da aplicação | Não foram encontrados arquivos de teste da aplicação | Pendente |
| Google e acesso | Não foram encontradas implementações de sincronização ou autenticação | Pendente |

## Como acompanhar

- `[x]` indica um artefato cuja existência foi conferida ou uma tarefa posteriormente concluída e validada.
- `[ ]` indica trabalho ou validação pendente. A existência de um arquivo não comprova sua execução.
- Trabalhar em uma etapa por vez, conforme `AGENTS.md`; este documento organiza o trabalho, sem autorizar antecipação de funcionalidades.
- Só marcar **Etapa concluída** depois de executar a validação indicada e registrar a evidência.
- Consultar o `ROADMAP.md` para os tópicos de estudo de cada etapa.

**Decisão de execução (12/10/2026):** construir o MVP local para o primeiro contato
com Maria Luzia, usando dados fictícios e hipóteses documentadas. A etapa 0 permanece
pendente de revisão; a implementação técnica prossegue pela etapa 1, uma etapa por
vez. Esta decisão substitui a exigência de validação prévia para a demonstração,
sem representar aprovação das regras pela professora.

**Revisão do plano (13/10/2026):** as etapas técnicas 1 a 7 estão registradas como
concluídas. A próxima implementação é somente a etapa 8. Concluir histórico,
totais, correções e backup (8 e 9), depois preparar e integrar Google Sheets
(10 e 11) e exportar o relatório PDF (12). A etapa 13 continua dedicada ao acesso
protegido e ao piloto. Drive/Docs passam a evolução futura. O relatório inicial
terá modelo fixo, com campos e apresentação ajustados posteriormente conforme o
retorno de Maria Luiza; sua aprovação continua pendente. Esta revisão altera
somente o planejamento, sem concluir tarefas de implementação.

## Etapa 0 — Descoberta do processo real

- [x] Preparar o roteiro de perguntas para Maria Luzia em `docs/perguntas-para-maria-luzia.md`.
- [ ] Registrar as respostas da professora e as dúvidas ainda abertas.
- [ ] Documentar uma amostra anonimizada do caderno e da linha correspondente na planilha.
- [ ] Desenhar o fluxo de trabalho atual e o desejado; `workflow.mmd` representa uma sequência de estudo e não substitui esse levantamento.
- [ ] Definir campos obrigatórios/opcionais, quem registra e quem consulta cada dado.
- [ ] Confirmar cálculo de duração, hora-aula, intervalos, arredondamento, fuso e passagem pela meia-noite.
- [ ] Confirmar regras de correção, cancelamento, filtros e totalização.
- [x] Escrever a história principal e critérios de aceitação em um documento de requisitos: [rascunho para revisão](docs/requisitos.md); aprovação da professora pendente.
- [ ] Validar três exemplos com resultados esperados: aula comum, aula com intervalo e registro inválido.
- [ ] Conferir se todas as informações exigidas pela planilha podem ser derivadas do registro.
- [ ] Obter a revisão dos requisitos por Maria Luzia ou outra professora responsável.
- [ ] **Etapa 0 concluída:** requisitos e exemplos revisados, com pendências registradas.

## Etapa 1 — Preparação do repositório e ferramentas

- [x] Criar a base do frontend com Svelte, Vite e TypeScript.
- [x] Criar a base do backend com Go e Fiber.
- [x] Declarar scripts de desenvolvimento, checagem e build do frontend.
- [x] Corrigir o arquivo `.gitgnore` para `.gitignore` e definir regras para segredos, banco local, backups, binários e artefatos gerados, preservando `.env.example`.
- [x] Preencher `.env.example` com configurações previstas e valores fictícios; explicar como cada aplicação as carregará.
- [x] Conferir versões e dependências declaradas e documentar as versões suportadas de Go, Node.js e npm.
- [x] Preencher o README da raiz com instalação, diretórios de execução, portas e comandos de desenvolvimento.
- [x] Definir comandos de formatação e verificações básicas para os dois projetos.
- [x] Validar a instalação em ambiente limpo e iniciar frontend/backend separadamente.
- [x] Executar `npm run check` e `npm run build` em `frontend/` e `go vet ./...` e `go build ./...` em `backend/`.
- [x] Conferir formatação e verificar que segredos e artefatos locais não estão versionados.
- [x] **Etapa 1 concluída:** outra pessoa consegue reproduzir a execução pelo README.

## Etapa 2 — Backend e primeiro endpoint

- [x] Organizar o ponto de entrada e as rotas, criando apenas os diretórios necessários.
- [x] Implementar `GET /health` com status e JSON estáveis.
- [x] Configurar a porta por variável de ambiente.
- [x] Implementar tratamento central de erros com respostas consistentes.
- [x] Criar o primeiro teste HTTP automatizado.
- [x] Validar `/health` com `curl` ou cliente HTTP e testar uma porta alternativa.
- [x] Executar `go test ./...` e `go vet ./...` em `backend/`.
- [x] **Etapa 2 concluída:** contrato de saúde documentado e teste aprovado.

## Etapa 3 — Frontend e consumo da API

- [x] Configurar a URL da API no frontend e documentar o valor local.
- [x] Definir proxy de desenvolvimento ou CORS compatível com as origens utilizadas.
- [x] Criar página de diagnóstico consumindo `GET /health` de verdade.
- [x] Exibir estados de carregamento, sucesso e erro compreensível.
- [x] Validar o comportamento com a API ligada e desligada.
- [x] Executar `npm run check` e `npm run build` em `frontend/`.
- [x] **Etapa 3 concluída:** comunicação frontend/backend comprovada no navegador.

## Etapa 4 — Domínio de aula e carga horária

- [x] Documentar as hipóteses de campos e cálculo do MVP e os exemplos esperados antes de modelar; manter a aprovação da professora pendente até a apresentação.
- [x] Definir `Aula`, separando entrada HTTP, domínio e representação persistida quando necessário.
- [x] Implementar validações e cálculo de duração independentes de HTTP e banco.
- [x] Aplicar as hipóteses documentadas do MVP para intervalos, ausência de arredondamento, horários civis e início/término; confirmação institucional pendente.
- [x] Criar testes de tabela para casos comuns, limites e entradas inválidas.
- [x] Comparar os resultados com os exemplos fictícios documentados e verificar independência do relógio e fuso da máquina; exemplos aprovados pela professora continuam pendentes.
- [x] **Etapa 4 concluída tecnicamente para o MVP:** cálculo determinístico documentado por testes aprovados. A validação institucional e o marco B permanecem pendentes.

## Etapa 5 — Persistência com SQLite

- [x] Escolher e documentar o driver SQLite e seus requisitos de compilação/execução.
- [x] Configurar localização do banco e abertura/fechamento da conexão.
- [x] Criar a primeira migração versionada com restrições coerentes com o domínio.
- [x] Implementar inserção, busca por identificador e listagem de aulas.
- [x] Testar migração em banco vazio e comportamento de uma segunda execução.
- [x] Criar testes de integração com banco temporário para gravação, leitura e rejeição de dados inválidos.
- [x] Confirmar persistência após reiniciar o processo com o banco de desenvolvimento.
- [x] **Etapa 5 concluída:** persistência validada sem depender da API.

## Etapa 6 — API de cadastro e consulta

- [x] Documentar e versionar o contrato JSON com exemplos de entrada, saída e erros.
- [x] Implementar `POST /aulas`, usando validação e cálculo oficiais do domínio.
- [x] Implementar `GET /aulas` com ordenação definida e contrato de paginação quando necessário.
- [x] Implementar `GET /aulas/:id`.
- [x] Mapear entrada inválida, registro inexistente e falha interna para respostas adequadas.
- [x] Testar cadastro e recuperação, JSON inválido, campos inválidos e identificador inexistente.
- [x] Comparar respostas com os registros no SQLite e executar testes unitários e de integração.
- [x] **Etapa 6 concluída:** cliente HTTP cadastra e consulta aulas com duração correta.

## Etapa 7 — Formulário web

- [x] Criar formulário com os campos das hipóteses documentadas do MVP, labels e mensagens acessíveis; revisão institucional ainda pendente.
- [x] Exibir prévia de duração como auxílio, preservando a autoridade do cálculo do backend.
- [x] Enviar dados para `POST /aulas` e apresentar confirmação e erros por campo.
- [x] Bloquear novos envios enquanto a requisição estiver em andamento.
- [x] Tratar falha de rede preservando os dados preenchidos para correção ou nova tentativa.
- [x] Validar teclado, largura de celular, obrigatórios ausentes, horários inválidos e duplo clique.
- [x] Conferir se os dados salvos correspondem ao formulário e executar checagem/build do frontend.
- [x] **Etapa 7 concluída:** uma professora consegue cadastrar uma aula pelo navegador.

## Etapa 8 — Histórico e carga horária acumulada

- [x] Exibir histórico com as aulas mais recentes primeiro, incluindo data, turma/estudante, horários, conteúdo e duração.
- [x] Implementar filtros de data inicial, data final e turma/estudante como hipóteses do MVP, a revisar com Maria Luiza.
- [x] Ampliar `GET /aulas` para aceitar os filtros documentados.
- [x] Calcular o total no backend para todo o conjunto filtrado, sem limitá-lo à página exibida.
- [x] Exibir total em horas e minutos, carregamento, lista vazia e falhas de consulta.
- [x] Atualizar o histórico e o total após cadastrar uma aula, respeitando os filtros ativos.
- [x] Definir limites inclusivos/exclusivos de datas de forma consistente entre API, interface e banco.
- [x] Conferir aulas fictícias de 60, 90 e 45 minutos: total de 3h15 e subtotais coerentes com os filtros.
- [x] Validar zero, um e vários resultados, mudança de mês e um volume maior de registros.
- [x] Criar e selecionar turma/estudante com nome único normalizado, migração preservando as aulas e referência por ID.
- [x] Paginar em 20 aulas, com Anterior/Próxima e contrato HTTP v2 documentado.
- [x] **Etapa 8 concluída:** cadastro e histórico formam um produto local útil.

## Etapa 9 — Correção, regressões e backup

- [ ] Definir contrato de atualização e implementar edição com recálculo da duração.
- [ ] Definir política de arquivamento/cancelamento e seu efeito sobre filtros e totais.
- [ ] Registrar `updated_at` e os demais dados de rastreabilidade exigidos pelos requisitos.
- [ ] Definir o comportamento de edições concorrentes, se houver esse cenário de uso.
- [ ] Cobrir o fluxo principal com testes e registrar logs sem dados sensíveis.
- [ ] Documentar criação e restauração de backup consistente do SQLite.
- [ ] Validar edição de horários, novo total, cancelamento, registro inexistente e dados inválidos.
- [ ] Executar a suíte de testes e restaurar um backup em banco separado.
- [ ] **Etapa 9 concluída:** correções e recuperação local comprovadas.

## Etapa 10 — Preparação da integração Google

- [ ] Confirmar autorização, responsáveis, propriedade dos arquivos e ambiente Google de teste.
- [ ] Mapear campos para Google Sheets e criar uma planilha descartável de teste.
- [ ] Definir onde o Apps Script será executado e documentar a decisão arquitetural.
- [ ] Definir contrato, autenticação, escopos mínimos e armazenamento de segredos no servidor.
- [ ] Definir identificador idempotente, versão do registro, timeouts e política de retentativas.
- [ ] Modelar estado, tentativas, data da última tentativa e erro no SQLite.
- [ ] Definir como edições/cancelamentos voltam a ficar pendentes e como tratar alterações manuais no Google.
- [ ] Revisar mapeamento com usuária e responsável pelos arquivos; verificar ausência de segredos no frontend, Git e logs.
- [ ] Revisar os cenários de repetição e concorrência para evitar duplicação.
- [ ] **Etapa 10 concluída:** contrato e ambiente de teste preparados, sem sincronização de dados reais.

## Etapa 11 — Sincronização com Sheets

- [ ] Implementar o Apps Script mínimo conforme o contrato e validar autenticação.
- [ ] Sincronizar uma aula com a planilha de teste sem condicionar o cadastro local ao Google.
- [ ] Persistir identificador externo, estado, tentativas, última tentativa e mensagem de erro.
- [ ] Implementar nova tentativa sobre a mesma aula e exibir estado no histórico.
- [ ] Implementar atualização de uma aula já sincronizada conforme a política definida.
- [ ] Validar repetição, chamadas concorrentes e perda de resposta após gravação, sem criar duplicatas.
- [ ] Forçar falha, conferir preservação local, corrigir a configuração e confirmar recuperação.
- [ ] Conferir colunas, valores e comportamento de correções/cancelamentos na planilha.
- [ ] **Etapa 11 concluída:** sincronização com Sheets recuperável e sem duplicações nos cenários testados.

## Etapa 12 — Relatório de aulas em PDF

Executar após a etapa 11 conforme a sequência acordada. A geração depende das
consultas e regras locais das etapas 8 e 9, sem dependência técnica do Google.

- [ ] Oferecer exportação em PDF usando os filtros ativos do histórico.
- [ ] Consultar pelo backend os registros do SQLite e o total de todo o conjunto filtrado, independentemente da paginação da tela.
- [ ] Criar modelo fixo inicial com período, data, turma/estudante, conteúdo, horários, duração e carga horária total.
- [ ] Aplicar as mesmas regras do histórico para correções e cancelamentos.
- [ ] Definir o comportamento sem resultados e apresentar falhas de exportação.
- [ ] Comparar aulas e total do PDF com o histórico para os mesmos filtros; conferir o exemplo de 3h15.
- [ ] Validar zero, um e vários registros, múltiplas páginas, textos longos, acentos e legibilidade em leitor de PDF.
- [ ] Validar a exportação com o serviço Google indisponível.
- [ ] Documentar o modelo inicial como proposta e registrar os ajustes de campos e apresentação após retorno de Maria Luiza, sem criar editor de relatórios.
- [ ] **Etapa 12 concluída tecnicamente:** PDF legível e coerente com o histórico, gerado a partir dos dados locais; registrar separadamente a situação da revisão pela cliente.

## Etapa 13 — Autenticação, publicação e piloto

- [ ] Confirmar identidade, permissões por usuária e acesso aos registros antes do piloto.
- [ ] Implementar autenticação e autorização no backend.
- [ ] Definir as proteções de sessão, cookies, CSRF e CORS aplicáveis à solução escolhida.
- [ ] Configurar produção, HTTPS, segredos e armazenamento persistente do SQLite.
- [ ] Documentar e validar migrações, backup, restauração e recuperação de implantação.
- [ ] Revisar minimização, acesso e retenção de dados com o responsável institucional.
- [ ] Testar acesso permitido e negado, incluindo tentativas de consultar/alterar dados de outra usuária sem permissão.
- [ ] Restaurar backup em ambiente separado e conferir os registros.
- [ ] Publicar piloto para poucas usuárias após validar os controles de acesso.
- [ ] Documentar suporte, monitoramento e retorno seguro ao processo manual.
- [ ] Acompanhar aulas reais, comparar resultados com o processo anterior e coletar feedback.
- [ ] **Etapa 13 concluída:** piloto validado antes de ampliar o uso.

## Evolução futura — Drive e Docs

Reavaliar após o PDF e o retorno de Maria Luiza, sem bloquear a etapa 13.

- [ ] Confirmar necessidade, template, pasta, nomeação, propriedade e permissões.
- [ ] Gerar ou atualizar documentos e organizar arquivos no Drive.
- [ ] Persistir IDs e links e acompanhar o estado de cada destino.
- [ ] Validar conteúdo, permissões, repetição e recuperação de falhas parciais sem duplicações.

## Marcos

- [x] **A — Comunicação web:** etapas 1 a 3 validadas.
- [ ] **B — Regra confiável:** etapa 4 validada com os exemplos da descoberta.
- [x] **C — Produto local útil:** etapas 5 a 8 validadas.
- [ ] **D — Operação segura:** etapa 9 validada.
- [ ] **E — Integração e relatório:** Sheets recuperável e sem duplicações, PDF coerente com o histórico; etapas 10 a 12 validadas.
- [ ] **F — Piloto real:** etapa 13 validada.

## Registro de validação

Ao concluir cada etapa, adicionar uma linha. Registrar comandos/cenários executados, resultados e referência à documentação ou aos testes; não incluir credenciais nem dados pessoais.

| Etapa | Data | Evidências e resultados | Pendências / decisão |
| --- | --- | --- | --- |
| Análise inicial | 12/10/2026 | Leitura do roadmap, instruções, questionário, código e configurações; sem execução da aplicação | Nenhuma etapa integralmente validada nesta análise |
| 0 — História e critérios de aceitação | 12/10/2026 | Documento `docs/requisitos.md` conferido contra o fluxo do roadmap; critérios propostos e roteiro de revisão registrados | Respostas, exemplos e aprovação da professora pendentes; etapa 0 permanece aberta |
| 1 — Preparação inicial | 12/10/2026 | Regras do `.gitignore` verificadas; `.env.example` e README preenchidos; Node.js 22.23.0 e npm 10.9.8 disponíveis | Frontend: `npm ci`, `npm run check` (0 erros/avisos) e `npm run build` aprovados; npm reportou 2 vulnerabilidades altas, a investigar. Go não encontrado; execução dos servidores, validação do backend e conclusão da etapa pendentes |
| 1 — Conclusão | 12/10/2026 | Go 1.27.0; `gofmt`, `go mod verify`, `go vet`, build; cópia limpa com `npm ci`, `format:check`, `check` e `build`; frontend e backend responderam HTTP 200; npm sem vulnerabilidades; revisão de arquivos versionados sem achados | Etapa 1 concluída no Linux; cache Go reutilizado; sem revisão visual. Mensagens em `docs/commits-etapa-1.md` |
| 2 — Backend e primeiro endpoint | 12/10/2026 | `go test ./...`, `go vet ./...` e build aprovados; `/health` validado com HTTP 200 e JSON nas portas 3000 e 3001 via curl; porta inválida rejeitada; testes de erros 400, 404, 500 e 503; contrato e comandos no README | Etapa 2 concluída; processos de teste encerrados |
| 3 — Frontend e consumo da API | 13/10/2026 | Formatação, check (0 erros/avisos) e build aprovados; Go test/vet/build aprovados; Chrome headless validou proxy em porta alternativa, carregamento, sucesso, backend desligado e recuperação; JSON incompatível e timeout simulados; captura de 375px revisada | Etapa 3 e marco A concluídos; comandos e configuração no README; processos de teste encerrados |
| 4 — Domínio de aula | 13/10/2026 | Hipóteses e exemplos em `docs/dominio-aula.md`; testes de tabela aprovados, 100% de cobertura de instruções em `internal/aula`; suíte Go, vet e build aprovados; cálculo sem relógio atual e testado em diferentes fusos locais | Conclusão técnica para o MVP; regras institucionais, etapa 0 e marco B ainda pendentes de revisão com Maria Luzia |
| 5 — Persistência SQLite | 13/10/2026 | Driver modernc.org/sqlite v1.58.0; testes com CGO_ENABLED=0, vet, mod verify e builds de cmd/db e cmd/api aprovados; migração inicial, repetição, rollback, versão futura, restrições, inserção/busca/listagem e reabertura testados; comandos em processos separados preservaram ID e 105 minutos | Etapa 5 concluída; banco fictício `backend/aulas-demo-etapa5.db` ignorado pelo Git; API será ligada ao repositório na etapa 6; regras institucionais ainda pendentes |
| 6 — API de cadastro e consulta | 13/10/2026 | Contrato `docs/api-v1.md`; POST /aulas e GET /aulas e /aulas/:id; testes HTTP com SQLite temporário, suíte sem CGO, vet e build aprovados; HTTP real na porta 33006, comparação SQL, persistência após reinício e encerramento por SIGINT/SIGTERM validados | Etapa 6 concluída; listagem integral sem paginação ou filtros; frontend continua diagnóstico; alterações locais de documentação da etapa 5 preservadas |
| 7 — Formulário web | 13/10/2026 | Formatação, check (0 erros/avisos) e build aprovados; Chrome com API e banco de teste validou cadastro, dados persistidos via GET, prévia, obrigatórios, horários/intervalo, erros 422, Tab/Enter, bloqueio de envio repetido, backend desligado e recuperação; resposta incompatível e timeout simulados; captura de 375px revisada | Etapa 7 concluída tecnicamente para o MVP; roteiro em `docs/formulario-aula.md`; hipóteses ainda pendentes de revisão com a professora |
| 8 — Histórico e carga horária | 17/10/2026 | Migração v2, catálogo único por nome normalizado, filtros inclusivos, paginação de 20, totais globais e contrato HTTP v2; testes Go, vet e build aprovados; frontend check sem erros/avisos e build; Chrome validou cadastro, duplicidade, filtros, atualização, 3h15, 21 registros em duas páginas, falha e recuperação de conexão; captura 375px sem rolagem horizontal | Conclusão técnica do MVP e marco C; regras institucionais e etapa 0 seguem pendentes; roteiro em `docs/historico-aulas.md` |

**Próxima ação:** discutir a etapa 9 (correção de registros, regressões e backup), sem iniciar sua implementação automaticamente. Validar as hipóteses com Maria Luzia após a demonstração; a etapa 0 continua pendente.
