# Ajustes antes do deploy — plano

Status: plano de 29/10/2026. Etapa A implementada, com testes automáticos e
confirmação do desenvolvedor no celular real. Etapa B implementada, com testes
Go aprovados. Etapa C implementada, com checks e testes de navegador aprovados;
validação do dropdown no celular real pendente.
Executar uma etapa por vez, na ordem abaixo. Deploy não faz parte deste trabalho.

O desenvolvedor confirmou login Google e cadastro pelo celular usando ngrok.
Essa confirmação cobre esse fluxo manual, sem concluir as demais validações.
Solicitou corrigir o hover das abas, remover o diagnóstico no rodapé e cadastrar
conteúdos reutilizáveis em um dropdown como turma/estudante.

## A — Abas no celular e retirada do diagnóstico

O CSS anterior continha `:hover` global e nas abas sem distinguir dispositivos de toque.
Navegadores móveis podem manter esse estado após tocar. Seleção e foco devem ser
tratados separadamente: `aria-selected` indica a aba ativa e `:focus-visible`
mantém a orientação para quem navega pelo teclado.

- Reproduzir o problema e conferir a interação entre os estilos globais e locais.
- Restringir os efeitos de hover envolvidos a `@media (hover: hover) and
  (pointer: fine)` e garantir prioridade visual da aba selecionada.
- Preservar Tab, setas, Home/End e os atributos de acessibilidade.
- Remover o bloco **Conexão com o backend**, seu botão, estilos exclusivos,
  estado, função de verificação e chamada automática a `/api/health`.
- Preservar `apiUrl`, carregamento das turmas e erros das operações reais.
  Manter `/health` disponível no backend para diagnóstico pelo terminal.
- Atualizar o README sobre a retirada do diagnóstico visual.

Como testar:

1. No celular real via ngrok, alternar repetidamente entre as três abas e tocar
   fora delas: somente a ativa mantém a seleção, sem hover residual nas anteriores.
2. No PC, conferir mouse, Tab, setas, Home/End e indicação de foco.
3. Trocar de aba com formulário parcial e conferir os dados preservados.
4. Conferir ausência do diagnóstico e da chamada automática a `/api/health`.
5. Com API indisponível, conferir os erros de consulta e cadastro nas operações.

Em `frontend/`: `npm run format:check`, `npm run check` e `npm run build`.
Emulação móvel complementa, mas não substitui, a validação no celular real.

### Resultado da implementação da etapa A

- Hover global dos botões e hover das abas restritos à capacidade de mouse.
  Nas abas, o hover local se aplica apenas às não selecionadas; a seleção
  permanece azul mesmo com o ponteiro sobre ela. Foco por teclado preservado.
- Diagnóstico, botão, requisição automática, estado e estilos exclusivos removidos.
  `/health` permanece no backend e os erros das operações reais continuam ativos.
- `npm run format:check`, `npm run check` (zero erros e avisos) e `npm run build`
  aprovados.
- `node scripts/test-etapa9.cjs` aprovado com Chrome e SQLite temporário: abas,
  teclado, rascunhos, cadastro, edição, conflitos, paginação e falha/recuperação.
- Verificação pontual em `/tmp/aula-sync-interface-check.cjs` aprovada, derivada
  do roteiro existente e sem novo arquivo de teste no repositório: cores com
  mouse e hover forçado, seleção preservada, toque emulado a 375px, aba anterior
  sem hover residual, ausência do diagnóstico e da requisição `/api/health`.
  O Chrome headless foi configurado explicitamente com capacidade de mouse
  antes de alternar para toque; sua configuração padrão não tinha ponteiro.
- Desenvolvedor confirmou que testou no celular e que o comportamento ficou
  correto, antes de autorizar a implementação da etapa B.

## B — Catálogo de conteúdos no backend

Proposta: catálogo global neste SQLite, independente de turma. Cada aula continua
armazenando seu texto em `conteudo`; selecionar uma opção copia esse texto para a
aula. Isso preserva o contrato das aulas, histórico, auditoria, PDF e sincronização.
O catálogo fica no banco, sem depender de Google ou armazenamento do navegador.

- Adicionar migração 007 (conferir a versão vigente antes de implementar), com
  tabela `conteudos`: ID, nome e chave normalizada única.
- Normalizar a chave como turma/estudante: espaços equivalentes e diferenças de
  maiúsculas/minúsculas não geram duplicatas. Preservar o texto de apresentação,
  inclusive quebras de linha internas; rejeitar nomes compostos apenas de espaços.
- Popular o catálogo com conteúdos não vazios das aulas existentes, inclusive
  canceladas. Agrupar pela chave normalizada e escolher o primeiro texto por ID
  da aula. Não reescrever aulas, versões, snapshots ou estados de sincronização.
- Aplicar em transação; reabrir o banco não pode duplicar conteúdos.
- Criar `GET /conteudos`, lista ordenada de objetos `{id, nome}`, e
  `POST /conteudos`, recebendo `{nome}` e retornando o item criado com HTTP 201.
- Seguir o padrão de turmas: 422 para vazio, 409 para duplicidade, validação de
  JSON/Content-Type e mesma proteção de sessão/origem quando OAuth estiver ativo.
- Documentar as rotas e a migração.

Como testar, com SQLite temporário:

- Cadastro, ordenação, persistência após reabrir e rejeição de vazios/duplicatas.
- Migração v6 com conteúdos repetidos e multilinha: catálogo preenchido, dados
  anteriores preservados, migração sem reaplicação. Testar rollback de falha.
- Rotas com JSON inválido, duplicidade, ausência de sessão e origem inválida,
  conforme o modo de autenticação dos testes existentes.
- Fazer backup/cópia antes de validar a migração com o banco de desenvolvimento.

Em `backend/`: `go test ./...`, `go vet ./...` e
`go build -o /tmp/aula-sync-backend ./cmd/api`.

### Resultado da implementação da etapa B

- Migração 007 e catálogo implementados, com nomes únicos normalizados e
  preservação do texto multilinha. Importação de conteúdos existentes, inclusive
  de aulas canceladas, sem modificar as aulas ou seus vínculos.
- GET/POST `/conteudos` implementados sob a mesma proteção das rotas de turmas.
- Suíte `go test ./...` aprovada com bancos temporários: persistência,
  concorrência, migração v6 → v7, rollback/retomada, preservação de auditoria e
  sincronização, contrato HTTP, acesso sem sessão e política de origem.
- `go vet ./...`, build de `cmd/api`, formatação Go e `git diff --check`
  aprovados.
- Contrato, regras e comandos de teste isolado em
  [catálogo de conteúdos](catalogo-conteudos.md).
- Banco de desenvolvimento preservado; testes não aplicaram migração nele.
  O formulário foi integrado na etapa C descrita abaixo.

## C — Dropdown no formulário

- Carregar `/conteudos` e validar a resposta, com tipos simples em `api.ts`,
  seguindo os padrões de turmas.
- Substituir o textarea principal por `select` nativo com rótulo, opção inicial
  e mensagens acessíveis. Manter um conteúdo por aula.
- Oferecer **Cadastrar conteúdo**, com textarea e **Criar e selecionar** como
  no fluxo de turma/estudante. Atualizar a lista e selecionar o criado sem recarga
  da página nem perda dos demais campos.
- Continuar enviando o texto escolhido em `conteudo` no cadastro e na edição.
- Ao editar, selecionar por correspondência exata. Se o texto antigo estiver
  ausente do catálogo, mostrar opção temporária com esse texto, sem substituição
  nem cadastro automático.
- Preservar seleção entre abas e o comportamento atual de limpeza após salvar.
  Durante criação/carregamento, impedir envio inconsistente.
- Tratar lista vazia, carregamento, duplicidade e erros de rede; oferecer nova
  tentativa de carregar sem apagar o formulário. Se a criação perder a resposta,
  orientar a conferir/recarregar a lista antes de repetir o cadastro.
- Atualizar a documentação do formulário e os roteiros de validação.

Como testar:

1. Criar conteúdo, salvar aula e reutilizar em outra pelo celular; reiniciar e
   conferir persistência do catálogo.
2. Testar lista vazia, multilinha, duplicidade e falhas ao carregar/criar;
   preservar o formulário e impedir aula sem conteúdo.
3. Editar aula antiga, inclusive com texto ausente do catálogo, sem alterar
   involuntariamente o conteúdo; conferir conflitos de versão existentes.
4. Conferir histórico, PDF e envio à planilha de teste com o conteúdo escolhido.
5. Validar select, rótulos e foco no celular e no teclado do PC.

Executar os checks do frontend da etapa A e os testes de integração afetados.
Acrescentar cobertura de navegador para criação/reutilização/edição e recuperação
de falhas quando necessário, usando dados fictícios e banco temporário.

### Resultado da implementação da etapa C

- `Conteudo` e `isConteudo` adicionados em `api.ts`. O formulário carrega e
  valida a lista (inclusive IDs e nomes duplicados) e cancela requisições ao sair.
- Select nativo com rótulo, placeholder, mensagens acessíveis e prévia textual
  que preserva quebras de linha. **Cadastrar conteúdo** oferece textarea e
  **Criar e selecionar**, selecionando a opção criada e levando o foco ao select.
- Cadastro/edição continuam enviando texto em `conteudo`. Texto antigo ausente
  do catálogo aparece como **conteúdo original da aula**, sem cadastro automático.
- Carregamento/falha da lista bloqueiam seleção e envio inconsistente; o botão
  **Atualizar conteúdos** recupera a lista sem apagar o formulário. Durante
  criação, os campos e abas ficam bloqueados. Uma falha de criação exige atualizar
  a lista antes de repetir, inclusive quando o servidor gravou e a resposta se perdeu.
- Formatação, `npm run check` (zero erros/avisos), build e diff aprovados.
- `node scripts/test-etapa9.cjs --conteudos` aprovado: carregamento/lista vazia,
  conteúdo obrigatório, criação multilinha, seleção/foco, clique duplo, duplicata,
  resposta perdida, recuperação, JSON inválido, reutilização em duas aulas,
  edição antiga, texto no PDF, persistência após recarga e viewport de toque 375px.
- `node scripts/test-etapa9.cjs --ux` aprovado: regressão das abas, formulário,
  histórico, conflitos, recuperação e Google Sheets simulado. O roteiro agora
  cadastra as opções necessárias antes de selecioná-las no formulário.
- Testes com SQLite temporário, sem alterar o banco de desenvolvimento ou chamar
  o Google real. Pendente: conferir criação e reutilização pelo celular real,
  seguindo [formulário de aula](formulario-aula.md).

## Limites e acompanhamento

Não incluir edição/exclusão do catálogo, múltiplos conteúdos por aula, vínculo
com turma, novas colunas no Google ou deploy. Evitar abstração genérica de
cadastros; seguir os padrões simples existentes.

Concluir e validar A antes de B; concluir e validar B antes de C. Registrar os
resultados reais no checklist, distinguindo testes automáticos de validação manual.
