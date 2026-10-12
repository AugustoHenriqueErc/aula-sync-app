# Checklist de implementação — Aula Sync App

Análise inicial: 12/10/2026.

Base: [ROADMAP.md](ROADMAP.md), [AGENTS.md](AGENTS.md), código e configurações existentes, [perguntas para Maria Luzia](docs/perguntas-para-maria-luzia.md) e [fluxo de estudo](workflow.mmd).

## Parecer de viabilidade

**Sim, o plano é tecnicamente viável para o escopo incremental descrito.** A separação entre frontend, API, regras de domínio e persistência permite construir e validar o fluxo local antes da integração. Não foi identificado, na leitura do repositório, impedimento estrutural que exija trocar a arquitetura proposta.

O parecer é baseado em inspeção estática: não foram executados builds, testes, servidores nem chamadas ao Google. A compatibilidade das versões declaradas e o funcionamento no ambiente precisam ser comprovados na etapa 1. Viabilidade não significa que as etapas já estejam implementadas, nem permite estimar prazo sem conhecer disponibilidade e requisitos finais.

### Condições e decisões pendentes

- **Regras reais:** ainda não há respostas registradas ao questionário. Campos, intervalos, hora-aula, arredondamento, cancelamentos e fuso precisam ser confirmados antes da modelagem da etapa 4.
- **Ambiente reproduzível:** conferir a versão `go 1.27.0` declarada no `backend/go.mod` e as dependências com o ambiente utilizado; definir também a versão de Node.js. Não alterar versões por suposição.
- **Integração Google:** a viabilidade operacional depende de permissões, modelos de arquivos, autenticação e limites do ambiente institucional. Comprovar esses pontos com arquivos de teste nas etapas 10 a 12.
- **Correções após sincronizar:** definir como edições e cancelamentos locais atualizam o Google, como detectar uma versão desatualizada e o que fazer se alguém editar a planilha manualmente. O SQLite continua sendo a fonte principal.
- **Vários destinos:** acompanhar Sheets, Drive e Docs separadamente, para uma falha parcial não obrigar a repetir operações já concluídas nem gerar duplicatas.
- **Piloto:** manter as etapas anteriores à autenticação restritas a desenvolvimento/testes com dados fictícios ou anonimizados. O controle de acesso deve existir antes do uso com dados reais.
- **Hospedagem:** confirmar volume e concorrência esperados e planejar armazenamento persistente, migrações e restauração do SQLite na etapa 13.

## Estado observado

| Área | Evidência no repositório | Situação |
| --- | --- | --- |
| Planejamento | `ROADMAP.md`, `AGENTS.md` e questionário | Documentados; requisitos ainda precisam de confirmação |
| Backend mínimo | `backend/main.go`: Fiber, `GET /` com texto e porta fixa `3000` | Código presente; execução não validada nesta análise |
| Frontend mínimo | `frontend/src/App.svelte`: mensagem estática | Código presente; ainda não consome a API |
| Ferramentas do frontend | `package.json`, `package-lock.json`, configurações Svelte/Vite/TypeScript | Scripts `dev`, `check` e `build` presentes; execução pendente |
| Documentação de execução | `README.md` da raiz vazio; README do frontend é do template | Pendente |
| Configuração | `.env.example` vazio | Pendente |
| Arquivos ignorados | `.gitgnore` da raiz vazio e com nome incorreto; `frontend/.gitignore` presente | Falta `.gitignore` adequado na raiz |
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
- [ ] Corrigir o arquivo `.gitgnore` para `.gitignore` e definir regras para segredos, banco local, backups, binários e artefatos gerados, preservando `.env.example`.
- [ ] Preencher `.env.example` com configurações previstas e valores fictícios; explicar como cada aplicação as carregará.
- [ ] Conferir versões e dependências declaradas e documentar as versões suportadas de Go, Node.js e npm.
- [ ] Preencher o README da raiz com instalação, diretórios de execução, portas e comandos de desenvolvimento.
- [ ] Definir comandos de formatação e verificações básicas para os dois projetos.
- [ ] Validar a instalação em ambiente limpo e iniciar frontend/backend separadamente.
- [ ] Executar `npm run check` e `npm run build` em `frontend/` e `go vet ./...` e `go build ./...` em `backend/`.
- [ ] Conferir formatação e verificar que segredos e artefatos locais não estão versionados.
- [ ] **Etapa 1 concluída:** outra pessoa consegue reproduzir a execução pelo README.

## Etapa 2 — Backend e primeiro endpoint

- [ ] Organizar o ponto de entrada e as rotas, criando apenas os diretórios necessários.
- [ ] Implementar `GET /health` com status e JSON estáveis.
- [ ] Configurar a porta por variável de ambiente.
- [ ] Implementar tratamento central de erros com respostas consistentes.
- [ ] Criar o primeiro teste HTTP automatizado.
- [ ] Validar `/health` com `curl` ou cliente HTTP e testar uma porta alternativa.
- [ ] Executar `go test ./...` e `go vet ./...` em `backend/`.
- [ ] **Etapa 2 concluída:** contrato de saúde documentado e teste aprovado.

## Etapa 3 — Frontend e consumo da API

- [ ] Configurar a URL da API no frontend e documentar o valor local.
- [ ] Definir proxy de desenvolvimento ou CORS compatível com as origens utilizadas.
- [ ] Criar página de diagnóstico consumindo `GET /health` de verdade.
- [ ] Exibir estados de carregamento, sucesso e erro compreensível.
- [ ] Validar o comportamento com a API ligada e desligada.
- [ ] Executar `npm run check` e `npm run build` em `frontend/`.
- [ ] **Etapa 3 concluída:** comunicação frontend/backend comprovada no navegador.

## Etapa 4 — Domínio de aula e carga horária

- [ ] Conferir se as regras necessárias da etapa 0 estão aprovadas antes de modelar.
- [ ] Definir `Aula`, separando entrada HTTP, domínio e representação persistida quando necessário.
- [ ] Implementar validações e cálculo de duração independentes de HTTP e banco.
- [ ] Aplicar as regras confirmadas de intervalos, arredondamento, fuso e início/término.
- [ ] Criar testes de tabela para casos comuns, limites e entradas inválidas.
- [ ] Comparar os resultados com os exemplos aprovados e verificar independência do relógio da máquina.
- [ ] **Etapa 4 concluída:** cálculo determinístico documentado por testes aprovados.

## Etapa 5 — Persistência com SQLite

- [ ] Escolher e documentar o driver SQLite e seus requisitos de compilação/execução.
- [ ] Configurar localização do banco e abertura/fechamento da conexão.
- [ ] Criar a primeira migração versionada com restrições coerentes com o domínio.
- [ ] Implementar inserção, busca por identificador e listagem de aulas.
- [ ] Testar migração em banco vazio e comportamento de uma segunda execução.
- [ ] Criar testes de integração com banco temporário para gravação, leitura e rejeição de dados inválidos.
- [ ] Confirmar persistência após reiniciar o processo com o banco de desenvolvimento.
- [ ] **Etapa 5 concluída:** persistência validada sem depender da API.

## Etapa 6 — API de cadastro e consulta

- [ ] Documentar e versionar o contrato JSON com exemplos de entrada, saída e erros.
- [ ] Implementar `POST /aulas`, usando validação e cálculo oficiais do domínio.
- [ ] Implementar `GET /aulas` com ordenação definida e contrato de paginação quando necessário.
- [ ] Implementar `GET /aulas/:id`.
- [ ] Mapear entrada inválida, registro inexistente e falha interna para respostas adequadas.
- [ ] Testar cadastro e recuperação, JSON inválido, campos inválidos e identificador inexistente.
- [ ] Comparar respostas com os registros no SQLite e executar testes unitários e de integração.
- [ ] **Etapa 6 concluída:** cliente HTTP cadastra e consulta aulas com duração correta.

## Etapa 7 — Formulário web

- [ ] Criar formulário com campos aprovados, labels e mensagens acessíveis.
- [ ] Exibir prévia de duração como auxílio, preservando a autoridade do cálculo do backend.
- [ ] Enviar dados para `POST /aulas` e apresentar confirmação e erros por campo.
- [ ] Bloquear novos envios enquanto a requisição estiver em andamento.
- [ ] Tratar falha de rede preservando os dados preenchidos para correção ou nova tentativa.
- [ ] Validar teclado, largura de celular, obrigatórios ausentes, horários inválidos e duplo clique.
- [ ] Conferir se os dados salvos correspondem ao formulário e executar checagem/build do frontend.
- [ ] **Etapa 7 concluída:** uma professora consegue cadastrar uma aula pelo navegador.

## Etapa 8 — Histórico e carga horária acumulada

- [ ] Exibir histórico com ordenação por data definida.
- [ ] Implementar filtros de período e demais filtros confirmados pelas usuárias.
- [ ] Calcular o total no backend para todo o conjunto filtrado, sem limitá-lo à página exibida.
- [ ] Exibir total, carregamento, lista vazia e falhas de consulta.
- [ ] Definir limites inclusivos/exclusivos de datas de forma consistente entre API, interface e banco.
- [ ] Comparar os totais com um conjunto conhecido calculado manualmente.
- [ ] Validar zero, um e vários resultados, mudança de mês e um volume maior de registros.
- [ ] **Etapa 8 concluída:** cadastro e histórico formam um produto local útil.

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
- [ ] Mapear campos para Sheets, Drive e Docs e criar arquivos descartáveis de teste.
- [ ] Definir onde o Apps Script será executado e documentar a decisão arquitetural.
- [ ] Definir contrato, autenticação, escopos mínimos e armazenamento de segredos no servidor.
- [ ] Definir identificador idempotente, versão do registro, timeouts e política de retentativas.
- [ ] Modelar estado, tentativas, data da última tentativa e erro no SQLite.
- [ ] Definir como edições/cancelamentos voltam a ficar pendentes e como tratar alterações manuais no Google.
- [ ] Prever estado por destino para recuperar falhas parciais nas etapas seguintes.
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

## Etapa 12 — Drive e Docs

- [ ] Confirmar template, pasta, nomeação, propriedade e permissões esperadas.
- [ ] Gerar ou atualizar documento a partir do modelo aprovado.
- [ ] Organizar o arquivo na pasta correta e persistir IDs e links necessários.
- [ ] Rastrear cada destino e recuperar falhas parciais sem refazer efeitos já concluídos.
- [ ] Conferir conteúdo, pasta, nome e permissões com o modelo institucional.
- [ ] Testar repetição, atualização e falha intermediária sem duplicação de documentos.
- [ ] **Etapa 12 concluída:** aula de teste chega aos destinos previstos com estado rastreável.

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

## Marcos

- [ ] **A — Comunicação web:** etapas 1 a 3 validadas.
- [ ] **B — Regra confiável:** etapa 4 validada com os exemplos da descoberta.
- [ ] **C — Produto local útil:** etapas 5 a 8 validadas.
- [ ] **D — Operação segura:** etapa 9 validada.
- [ ] **E — Integração controlada:** etapas 10 a 12 validadas.
- [ ] **F — Piloto real:** etapa 13 validada.

## Registro de validação

Ao concluir cada etapa, adicionar uma linha. Registrar comandos/cenários executados, resultados e referência à documentação ou aos testes; não incluir credenciais nem dados pessoais.

| Etapa | Data | Evidências e resultados | Pendências / decisão |
| --- | --- | --- | --- |
| Análise inicial | 12/10/2026 | Leitura do roadmap, instruções, questionário, código e configurações; sem execução da aplicação | Nenhuma etapa integralmente validada nesta análise |
| 0 — História e critérios de aceitação | 12/10/2026 | Documento `docs/requisitos.md` conferido contra o fluxo do roadmap; critérios propostos e roteiro de revisão registrados | Respostas, exemplos e aprovação da professora pendentes; etapa 0 permanece aberta |

**Próxima ação:** concluir a descoberta da etapa 0 com Maria Luzia. A estrutura da etapa 1 já foi iniciada, mas isso não substitui a confirmação dos requisitos nem a validação técnica do ambiente.
