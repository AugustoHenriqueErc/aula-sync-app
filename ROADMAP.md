# Roadmap — Aula Sync App

## 1. Visão do produto

O Aula Sync App deve permitir que professoras registrem uma aula uma única vez,
consultem o histórico e obtenham automaticamente a carga horária. Em uma fase
posterior, os mesmos dados serão enviados aos documentos usados pela instituição
no Google Sheets, Google Drive e Google Docs.

O projeto também tem um objetivo didático: cada incremento deve ser pequeno o
suficiente para estudar uma tecnologia, aplicá-la e validar o resultado antes de
avançar.

### Primeiro fluxo de valor

1. A professora preenche os dados da aula em um formulário web.
2. O backend valida e salva o registro no SQLite.
3. A aplicação calcula a duração da aula.
4. A professora consulta o registro no histórico.
5. Mais tarde, uma sincronização envia o registro ao ecossistema Google sem
   exigir novo preenchimento.

## 2. Escopo inicial

### Dentro do escopo

- Cadastro, consulta e correção de aulas.
- Cálculo de duração e carga horária acumulada.
- Persistência local em SQLite.
- Histórico com filtros básicos.
- Integração incremental com Google Sheets, Drive e Docs.
- Indicação e rastreamento do estado de sincronização.

### Fora do primeiro ciclo

- Aplicativo móvel nativo.
- Funcionamento offline completo no navegador.
- Sincronização em tempo real.
- Arquitetura de microsserviços.
- Permissões institucionais complexas.
- Relatórios avançados e dashboards analíticos.

Esses itens podem ser reavaliados depois que o fluxo principal estiver sendo
usado e houver necessidades reais observadas.

## 3. Arquitetura inicial proposta

Um **monólito modular** em um único repositório é suficiente para começar:

```text
aula-sync-app/
├── frontend/             # Svelte + Vite + TypeScript
├── backend/              # Go + Fiber
│   ├── cmd/api/          # ponto de entrada da API
│   ├── internal/aula/    # regras e casos de uso de aulas
│   ├── internal/http/    # rotas, handlers e DTOs
│   ├── internal/storage/ # acesso ao SQLite
│   └── migrations/       # evolução versionada do banco
├── integrations/
│   └── google-apps-script/ # scripts adicionados apenas na fase Google
└── docs/                 # decisões e contratos relevantes
```

A estrutura é uma direção, não uma obrigação de criar todas as pastas no início.
Cada diretório deve surgir apenas quando a etapa correspondente for iniciada.

### Responsabilidades

- **Frontend:** formulário, histórico, feedback de validação e estado de
  sincronização. Não deve conter a regra oficial de carga horária.
- **API Go:** valida entradas, executa casos de uso e aplica as regras do domínio.
- **SQLite:** fonte principal dos dados da aplicação e do estado de sincronização.
- **Google Apps Script:** adaptador de integração com os arquivos Google. Não é a
  fonte principal nem deve ser necessário para registrar uma aula.

### Fluxo futuro de sincronização

```text
Navegador -> API Go -> SQLite
                 |
                 +-> solicitação de sincronização -> Apps Script -> Google
```

A sincronização deve guardar, no mínimo, estado (`pendente`, `sincronizado` ou
`erro`), data da última tentativa e mensagem de erro. Assim, uma falha do Google
não causa perda do registro e pode ser corrigida ou tentada novamente.

### Modelo conceitual inicial

O formato exato será decidido ao implementar, após comparar o caderno e a
planilha reais. Como hipótese inicial, uma aula pode ter:

- identificador;
- data;
- horário de início e término;
- turma ou estudante;
- conteúdo/descrição;
- observações opcionais;
- duração calculada;
- datas de criação e atualização;
- estado de sincronização.

Não se deve fechar o esquema antes de levantar os campos reais, regras de
arredondamento, intervalos, fusos horários e formato exigido pela instituição.

## 4. Decisões técnicas iniciais

1. **SQLite é a fonte de verdade.** O sistema continua útil sem acesso ao Google
   e não depende da disponibilidade de uma planilha.
2. **A carga horária é calculada no backend.** O frontend pode exibir uma prévia,
   mas o valor persistido deve seguir uma única regra oficial.
3. **API HTTP com JSON e endpoints simples.** REST é suficiente para os primeiros
   casos de uso e fácil de observar com navegador, `curl` ou cliente HTTP.
4. **Migrações versionadas desde o primeiro esquema.** Mesmo no estudo, isso
   evita bancos locais incompatíveis e ensina evolução segura de dados.
5. **Integração desacoplada do cadastro.** Salvar uma aula não deve falhar porque
   um serviço Google está indisponível.
6. **Interfaces surgem nas fronteiras necessárias.** Evitar uma arquitetura com
   muitas camadas vazias; separar domínio, HTTP e persistência já oferece uma boa
   base didática.
7. **Datas e horas exigem regra explícita.** Persistir datas em formato não
   ambíguo e definir o fuso da instituição antes de integrar com o Google.
8. **Autenticação é adiada, não ignorada.** O protótipo local vem primeiro, mas
   nenhuma publicação com dados reais deve ocorrer sem controle de acesso,
   segredos protegidos e revisão de privacidade.

## 5. Roadmap incremental

Cada etapa deve terminar com algo observável e validado. Não iniciar a seguinte
enquanto o resultado atual não estiver compreendido e funcionando.

### Etapa 0 — Descoberta do processo real

**Estudar**

- Técnicas básicas de levantamento de requisitos.
- Histórias de usuário e critérios de aceitação.
- Noções de privacidade e minimização de dados pessoais.

**Implementar**

- Nenhum código.
- Documentar uma amostra anonimizada do caderno e da planilha.
- Listar campos obrigatórios, regras de cálculo, correções permitidas e quem usa
  os dados.
- Desenhar o fluxo atual e o fluxo desejado.

**Resultado esperado**

Um pequeno documento de requisitos com uma história principal, exemplos de
aulas e critérios objetivos para considerar um registro correto.

**Como validar**

- Simular pelo menos três casos: aula comum, aula atravessando um intervalo e
  dado inválido.
- Confirmar o documento com uma professora usuária.
- Verificar se todos os campos da planilha final podem ser derivados do registro.

**Pré-requisitos**

- Entendimento do problema; não exige conhecimento de programação.

### Etapa 1 — Preparação do repositório e ferramentas

**Estudar**

- Estrutura básica de um repositório Git.
- Comandos essenciais de Go, Node.js e npm.
- Variáveis de ambiente e diferença entre configuração e segredo.
- Papel de formatadores, linters e arquivos ignorados pelo Git.

**Implementar**

- Criar apenas os projetos mínimos de `frontend` e `backend`.
- Definir versões suportadas e comandos de desenvolvimento.
- Adicionar `.gitignore`, exemplo de variáveis de ambiente e instruções no
  `README.md`.
- Configurar formatação e verificações básicas, sem regra de negócio.

**Resultado esperado**

Frontend e backend iniciam separadamente e exibem respostas mínimas de saúde,
com comandos documentados e reproduzíveis.

**Como validar**

- Clonar em ambiente limpo ou remover dependências geradas e seguir o README.
- Executar formatação, lint/verificações e build de ambos os projetos.
- Confirmar que nenhum segredo ou artefato local aparece no Git.

**Pré-requisitos**

- Terminal, Git básico e instalação de Go e Node.js.

### Etapa 2 — Fundamentos do backend e primeiro endpoint

**Estudar**

- Sintaxe essencial de Go: pacotes, structs, funções, erros e interfaces.
- HTTP: métodos, códigos de status, cabeçalhos e JSON.
- Rotas, handlers e middleware no Fiber.
- Testes com `testing` e `httptest`.

**Implementar**

- Organizar o ponto de entrada e o roteamento da API.
- Criar `GET /health` com resposta JSON estável.
- Adicionar tratamento central de erros e configuração de porta.
- Criar o primeiro teste HTTP automatizado.

**Resultado esperado**

A API inicia, responde que está saudável e possui ao menos um teste executável.

**Como validar**

- Fazer uma requisição a `/health` e conferir status e JSON.
- Executar `go test ./...` e `go vet ./...`.
- Alterar temporariamente a porta via ambiente para entender a configuração.

**Pré-requisitos**

- Etapa 1.
- Programação básica: variáveis, funções, condicionais e testes.

### Etapa 3 — Fundamentos do frontend e consumo da API

**Estudar**

- Componentes e reatividade do Svelte.
- TypeScript: tipos, interfaces, unions e tratamento de `unknown`.
- Ciclo de requisição com `fetch`, CORS e estados de interface.
- CSS básico; Tailwind permanece opcional nesta etapa.

**Implementar**

- Criar uma página simples de diagnóstico.
- Consultar `GET /health`.
- Exibir os estados carregando, sucesso e erro.
- Definir configuração local da URL da API.

**Resultado esperado**

O navegador informa visualmente se o backend está acessível, sem dados fixos
substituindo a requisição real.

**Como validar**

- Abrir a página com a API ligada e verificar o sucesso.
- Desligar a API e verificar uma mensagem de erro compreensível.
- Executar checagem TypeScript e build do frontend.

**Pré-requisitos**

- Etapas 1 e 2.
- HTML, CSS e JavaScript básicos.

### Etapa 4 — Domínio de aula e regra de carga horária

**Estudar**

- Modelagem de domínio simples e invariantes.
- Representação de datas, horários, duração e fuso em Go.
- Testes de tabela e casos de fronteira.
- Diferença entre dados recebidos, modelo de domínio e dados persistidos.

**Implementar**

- Definir o tipo `Aula` a partir da descoberta da etapa 0.
- Implementar validações puras e cálculo de duração.
- Definir regras para término anterior ao início, intervalos e arredondamento.
- Criar testes unitários para exemplos reais e limites.

**Resultado esperado**

Uma regra de carga horária independente de HTTP e banco, documentada por testes.

**Como validar**

- Executar testes com casos normais, limites e inválidos.
- Comparar manualmente os resultados com exemplos aprovados na etapa 0.
- Revisar se o cálculo é determinístico e não depende do relógio do computador.

**Pré-requisitos**

- Etapa 2.
- Requisitos de horários e intervalos definidos na etapa 0.

### Etapa 5 — Persistência com SQLite

**Estudar**

- SQL básico: tabelas, tipos, chaves, índices, `INSERT` e `SELECT`.
- Transações, restrições e migrações.
- Uso de SQLite a partir de Go e diferenças entre drivers.
- Testes de integração com banco temporário.

**Implementar**

- Escolher e documentar um driver SQLite adequado ao ambiente.
- Criar a primeira migração da tabela de aulas.
- Implementar repositório para inserir, buscar e listar aulas.
- Adicionar restrições coerentes com o domínio.

**Resultado esperado**

Aulas de teste podem ser gravadas e lidas do SQLite sem depender da API.

**Como validar**

- Executar migrações sobre um banco vazio.
- Rodar testes de integração em banco temporário.
- Reiniciar o processo e comprovar que dados do banco de desenvolvimento
  persistem.
- Testar que registros inválidos são rejeitados também nas fronteiras adequadas.

**Pré-requisitos**

- Etapa 4.
- Noções de arquivos e linha de comando.

### Etapa 6 — API de cadastro e consulta de aulas

**Estudar**

- Contratos de API, DTOs e validação de entrada.
- Semântica de `POST`, `GET`, `PUT`/`PATCH` e códigos HTTP.
- Paginação básica e respostas de erro consistentes.
- Testes de integração HTTP.

**Implementar**

- `POST /aulas` para cadastrar.
- `GET /aulas` para listar inicialmente em ordem definida.
- `GET /aulas/:id` para consultar uma aula.
- Mapear erros de validação, ausência e falha interna para respostas seguras.
- Publicar e versionar um exemplo do contrato JSON.

**Resultado esperado**

Um cliente HTTP consegue cadastrar uma aula válida e recuperá-la com a duração
calculada; entradas inválidas recebem erros úteis.

**Como validar**

- Executar uma coleção curta de requisições felizes e inválidas.
- Confirmar os registros diretamente no SQLite durante o estudo.
- Rodar testes unitários e de integração, incluindo ID inexistente.

**Pré-requisitos**

- Etapas 2, 4 e 5.

### Etapa 7 — Formulário web de cadastro

**Estudar**

- Formulários acessíveis, labels, foco e mensagens de erro.
- Estado de formulário e eventos no Svelte.
- Validação no cliente versus validação autoritativa no servidor.
- Noções de design responsivo.

**Implementar**

- Criar formulário com os campos definidos na etapa 0.
- Mostrar prévia de duração apenas como auxílio.
- Enviar para `POST /aulas` e exibir sucesso ou erros por campo.
- Impedir envios duplicados enquanto a requisição está em andamento.

**Resultado esperado**

Uma professora consegue registrar uma aula pelo navegador e recebe confirmação
clara sem precisar usar ferramentas técnicas.

**Como validar**

- Testar navegação apenas por teclado e em largura de celular.
- Enviar dados válidos, obrigatórios ausentes e horários inválidos.
- Conferir que duplo clique não cria dois registros.
- Comparar a aula salva com os dados preenchidos.

**Pré-requisitos**

- Etapas 3 e 6.
- Fundamentos de formulários HTML.

### Etapa 8 — Histórico e carga horária acumulada

**Estudar**

- Listas, renderização condicional e composição de componentes Svelte.
- Consultas SQL com filtros, ordenação, soma e paginação.
- Parâmetros de URL e estados vazios.
- Diferenças de interpretação de datas entre cliente, servidor e banco.

**Implementar**

- Exibir histórico ordenado por data.
- Adicionar filtros mínimos definidos pelas usuárias, como período e turma.
- Exibir carga horária total do conjunto filtrado.
- Incluir estados de carregamento, vazio e falha.

**Resultado esperado**

A professora localiza aulas passadas e vê a carga horária total coerente com o
filtro aplicado.

**Como validar**

- Preparar um conjunto conhecido e calcular o total manualmente.
- Testar filtros com zero, um e vários resultados.
- Testar mudança de mês e limites de datas.
- Verificar o comportamento com uma quantidade maior de registros de teste.

**Pré-requisitos**

- Etapas 6 e 7.
- SQL de consulta e conceitos de estado no frontend.

### Etapa 9 — Correção de registros e proteção contra regressões

**Estudar**

- Atualizações parciais, concorrência otimista e trilha de alterações.
- Pirâmide de testes e testes de interface.
- Estratégias de backup do SQLite.
- Princípios básicos de observabilidade e logs estruturados.

**Implementar**

- Permitir editar uma aula e recalcular sua duração.
- Definir uma política segura para exclusão: inicialmente, preferir arquivamento
  ou cancelamento em vez de remoção definitiva.
- Registrar `updated_at` e, se o requisito pedir, motivo da alteração.
- Cobrir o fluxo central com testes e adicionar logs sem dados sensíveis.
- Documentar criação e restauração de backup local.

**Resultado esperado**

Erros de digitação podem ser corrigidos sem perder rastreabilidade e sem quebrar
os cálculos do histórico.

**Como validar**

- Editar horário e confirmar novo total.
- Tentar atualizar um registro inexistente ou com dados inválidos.
- Executar toda a suíte de testes.
- Restaurar uma cópia de teste do banco seguindo a documentação.

**Pré-requisitos**

- Etapa 8.
- Testes automatizados e transações SQL básicas.

### Etapa 10 — Preparação segura da integração Google

**Estudar**

- Google Apps Script, permissões e escopos OAuth.
- Web Apps do Apps Script e modelos de autenticação possíveis.
- Gestão de segredos, princípio do menor privilégio e proteção de dados.
- Idempotência, timeouts, retentativas e limites de serviço.

**Implementar**

- Mapear exatamente qual dado vai para Sheets, Drive e Docs.
- Criar uma planilha e documentos exclusivamente de teste.
- Definir contrato entre API e Apps Script, autenticação e identificador
  idempotente, sem ainda ativar sincronização de dados reais.
- Modelar no SQLite o estado e as tentativas de sincronização.
- Registrar uma decisão arquitetural sobre onde e como a integração será
  executada.

**Resultado esperado**

Existe um contrato revisado e um ambiente Google descartável, com riscos e
permissões compreendidos antes de enviar qualquer dado real.

**Como validar**

- Revisar os escopos e garantir que sejam mínimos.
- Demonstrar que segredos não estão no frontend, Git ou logs.
- Simular chamadas repetidas e confirmar pelo desenho que não duplicarão aulas.
- Fazer revisão do mapeamento com uma usuária e com o responsável pelos arquivos.

**Pré-requisitos**

- Etapa 9.
- Conta/ambiente Google de teste e autorização institucional apropriada.

### Etapa 11 — Primeiro corte: sincronização com Google Sheets

**Estudar**

- APIs de planilha no Apps Script.
- Requisições autenticadas entre serviços.
- Identificadores externos, idempotência e tratamento de falhas parciais.
- Filas conceituais e processamento assíncrono, ainda que a primeira solução seja
  simples.

**Implementar**

- Criar um Apps Script mínimo que receba o contrato validado.
- Sincronizar uma aula com uma planilha de teste.
- Gravar identificador externo, estado, tentativas e última mensagem de erro.
- Oferecer ação de nova tentativa sem recriar a aula.
- Exibir o estado de sincronização no histórico.

**Resultado esperado**

Uma aula salva localmente aparece uma única vez na planilha de teste; uma falha
mantém o registro local e pode ser tentada novamente.

**Como validar**

- Sincronizar a mesma aula repetidamente e procurar duplicatas.
- Forçar indisponibilidade ou configuração inválida e conferir o estado de erro.
- Corrigir a configuração, repetir e confirmar a transição para sincronizado.
- Comparar cada coluna da planilha com o contrato.

**Pré-requisitos**

- Etapa 10.
- Conhecimento básico de JavaScript e segurança de requisições.

### Etapa 12 — Google Drive e Google Docs

**Estudar**

- APIs de Drive e Docs no Apps Script.
- Templates, permissões de arquivos e propriedade de documentos.
- Consistência entre múltiplos destinos e compensação de falhas.
- Auditoria e limites de execução do Apps Script.

**Implementar**

- Gerar ou atualizar um documento a partir de um template aprovado.
- Organizar o arquivo na pasta correta do Drive.
- Armazenar IDs e links externos necessários, sem expor permissões indevidas.
- Definir comportamento quando Sheets funciona, mas Drive ou Docs falha.

**Resultado esperado**

Uma aula de teste alimenta a planilha e produz/atualiza o documento esperado na
pasta correta, com estado de cada destino rastreável.

**Como validar**

- Comparar o documento produzido com o modelo institucional.
- Conferir pasta, nome, conteúdo e permissões.
- Forçar uma falha intermediária e verificar recuperação sem duplicação.
- Testar repetição da operação.

**Pré-requisitos**

- Etapa 11.
- Templates e política de compartilhamento definidos pela instituição.

### Etapa 13 — Autenticação, publicação e operação piloto

**Estudar**

- Cookies seguros, sessões, CSRF, CORS e HTTPS.
- Opções de identidade adequadas à instituição.
- Build e implantação de Svelte, Go e SQLite.
- Backups, logs, monitoramento e resposta a incidentes.
- LGPD aplicada aos dados tratados pelo sistema.

**Implementar**

- Definir e implementar autenticação apropriada antes de dados reais.
- Restringir acesso e ações por usuário conforme requisitos confirmados.
- Criar configuração de produção e processo de migração/backup.
- Publicar um ambiente piloto com poucas usuárias.
- Documentar recuperação, suporte e retorno seguro à planilha manual.

**Resultado esperado**

O fluxo completo pode ser usado em piloto controlado, com acesso protegido,
backup verificável e caminho de recuperação.

**Como validar**

- Testar acesso autorizado e bloqueio de acesso não autorizado.
- Executar checklist de segurança e privacidade.
- Restaurar backup em ambiente separado.
- Acompanhar aulas reais do piloto e comparar resultados com o processo anterior.
- Coletar feedback das professoras antes de ampliar o uso.

**Pré-requisitos**

- Etapas 9 a 12.
- Ambiente de hospedagem, domínio/HTTPS e decisões institucionais de identidade e
  privacidade.

## 6. Marcos de aprendizado

- **Marco A — Comunicação web:** etapas 1 a 3; frontend e backend conversam.
- **Marco B — Regra confiável:** etapa 4; carga horária é compreendida e testada.
- **Marco C — Produto local útil:** etapas 5 a 8; cadastro e histórico funcionam.
- **Marco D — Operação segura:** etapa 9; correção, testes e backup existem.
- **Marco E — Integração controlada:** etapas 10 a 12; dados chegam ao Google sem
  duplicação e com falhas rastreáveis.
- **Marco F — Piloto real:** etapa 13; acesso, implantação e operação foram
  validados.

## 7. Critério geral de conclusão de uma etapa

Uma etapa está concluída quando:

- o resultado esperado funciona de forma reproduzível;
- a validação descrita foi executada;
- testes relevantes passam;
- comandos e decisões novas estão documentados;
- não há segredos versionados;
- a pessoa que estuda consegue explicar o que foi construído e por quê;
- as dúvidas descobertas foram registradas antes de avançar.

## 8. Próxima ação recomendada

Iniciar somente a **Etapa 0**. O melhor próximo artefato é uma descrição
anonimizada de um registro real do caderno e da linha correspondente na planilha.
Isso permitirá fechar o modelo de aula e as regras de carga horária sem adivinhar
campos que depois precisariam ser refeitos.
