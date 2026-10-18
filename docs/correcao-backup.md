# Etapa 9 — Correção, cancelamento e backup

## Regras adotadas para o MVP

- Editar uma aula substitui seus dados e recalcula a duração no backend.
- Cancelar preserva o registro, mas retira sua duração da carga horária.
- Aulas canceladas podem ser consultadas e reativadas; precisam ser reativadas
  antes de editar. Não existe exclusão física pela API.
- O histórico usa `situacao=ativas` por padrão. `canceladas` e `todas` permitem
  consultar os registros preservados. A quantidade inclui todas as aulas do
  filtro; a duração soma somente as ativas, inclusive na opção Todas.
- Filtros de data/turma e tamanho de página continuam iguais. Se uma alteração
  esvazia a última página, a interface consulta a última página disponível.
- Motivo da alteração e identificação do responsável não foram definidos nos
  requisitos. O MVP não possui autenticação nem atribuição de autoria.

## Concorrência otimista e rastreabilidade

Cada aula tem `versao`, iniciando em 1. O formulário envia a versão que abriu.
O banco só altera se essa versão ainda for a atual; caso contrário, a API retorna
409. Os campos digitados permanecem no formulário. Fechar a edição recarrega o
histórico; abra novamente a aula para conferir a alteração feita pela outra aba.

Cada mudança efetiva incrementa a versão e grava `updated_at` em UTC (RFC3339Nano).
A transação também insere em `aula_alteracoes` o estado anterior completo, sua
versão, a operação (`editar`, `cancelar`, `reativar`) e a data da mudança. Se essa
inserção falhar, a atualização da aula é revertida. Não existe tela de auditoria
nesta etapa; a consulta é local por SQL:

```sql
SELECT aula_id, versao_anterior, operacao, registrada_em, estado_anterior
FROM aula_alteracoes
WHERE aula_id = 1
ORDER BY versao_anterior;
```

`estado_anterior` é o JSON da struct de persistência `Registro`, incluindo
`Aula`, `Versao`, `Cancelada`, `UpdatedAt`, `TurmaID` e `DuracaoMinutos`.
Os estados anteriores têm os mesmos dados pessoais do cadastro; permanecem no
SQLite e nos backups, não nos logs. Essa tabela não é uma trilha inviolável.

A migração 003 preserva aulas e turmas. Registros anteriores passam a ter versão
1 e situação ativa. Seu `updated_at` fica vazio porque o sistema não conhecia a
data anterior; não inventamos uma data histórica. Novos cadastros recebem a data
atual e alterações posteriores passam a registrar sua própria data.

## Logs

A API usa logs JSON com `slog`. Alterações registram evento, ID, versão e operação
ou situação; falhas internas registram evento e status. Não são registrados
nomes, conteúdo, observações, corpos HTTP, filtros nem detalhes internos do SQL.
A saída padrão do comando didático `cmd/db listar` continua mostrando os dados
solicitados explicitamente; isso é diferente dos logs operacionais da API.

## Criar um backup

O comando usa [VACUUM INTO do SQLite](https://www.sqlite.org/lang_vacuum.html),
que produz uma cópia consistente mesmo com o banco em uso. O destino deve ser
novo: arquivos existentes, inclusive vazios, são recusados. A pasta precisa
existir. O arquivo é criado com permissão 0600 e sincronizado antes do sucesso.

Em `backend/`, com o mesmo DATABASE_PATH usado pela API:

```bash
mkdir -p ../backups
DATABASE_PATH=aulas.db go run ./cmd/db backup ../backups/aulas-2026-10-18.db
```

A origem precisa existir, evitando criar acidentalmente um backup de banco vazio
por erro de caminho. O comando abre a origem pelo mesmo fluxo da aplicação e
aplica migrações pendentes; use a versão atual do programa. Se houver falha ou
interrupção, não utilize um arquivo de destino parcial. Escolha outro nome na
próxima tentativa e confira o resultado antes de considerar o backup válido.

## Restaurar sem sobrescrever a origem

1. Encerre a API com Ctrl+C e aguarde o término das requisições.
2. Escolha um caminho novo, que não exista. Em `backend/`:

```bash
cp -n ../backups/aulas-2026-10-18.db aulas-restauradas.db
DATABASE_PATH=aulas-restauradas.db go run ./cmd/db verificar
DATABASE_PATH=aulas-restauradas.db go run ./cmd/db listar
DATABASE_PATH=aulas-restauradas.db go run ./cmd/api
```

O `cp -n` preserva qualquer arquivo existente; por isso confirme previamente que
`aulas-restauradas.db` é um nome novo. `verificar` confere integridade SQLite e
chaves estrangeiras. Os comandos também aplicam migrações pendentes ao abrir.
Confira aulas, situações e totais no navegador antes de adotar o banco restaurado.
A origem continua disponível no caminho anterior. Restaurar usa o estado da data
do backup; alterações posteriores não aparecem nessa cópia.

## Como testar a implementação

Backend, em `backend/`:

```bash
go test ./...
go vet ./...
go build -o /tmp/aula-sync-backend ./cmd/api
go build -o /tmp/aula-sync-db ./cmd/db
```

Frontend, em `frontend/`:

```bash
npm run format:check
npm run check
npm run build
```

Teste de interface reproduzível, na raiz (Linux, Go, Node 22, dependências do
frontend instaladas e Google Chrome disponível):

```bash
node scripts/test-etapa9.cjs
```

Para outro executável Chromium, defina `CHROME_BIN=/caminho/do/navegador`.
O script escolhe portas locais livres, compila a API, inicia Vite e Chrome
headless, usa SQLite temporário e encerra os processos ao terminar. Imprime o
caminho dos artefatos temporários, incluindo uma captura de tela de 375px. Não
usa o banco de desenvolvimento. O Chrome de teste roda sem sandbox de navegador
nesse ambiente local; a execução não exige instalar bibliotecas de automação.

Roteiro manual:

1. Cadastre uma aula de 08:00 a 09:00, sem intervalo. Confira 1h00.
2. Clique Editar, altere o término para 10:00 e salve: confira 2h00.
3. Tente término 07:00: o erro deve preservar os campos e o registro anterior.
4. Abra a mesma aula para edição em duas abas. Salve na primeira e tente salvar
   na segunda: deve aparecer conflito, sem perder os campos da segunda aba nem
   sobrescrever a primeira alteração. Feche e reabra para recarregar.
5. Cancele e confirme: a aula desaparece do filtro Ativas. Em Canceladas ela
   continua visível, com total 0h00. Reative e volte a Ativas: total 2h00.
6. Com 21 aulas ativas, vá à página 2 e cancele a única aula: a página deve voltar
   para a primeira, com 20 aulas.
7. Desligue a API durante uma edição e tente salvar: os campos permanecem.
   Reinicie com o mesmo banco e tente novamente. Se o primeiro envio tiver sido
   confirmado no servidor, o controle de versão impedirá uma sobrescrita antiga.
8. Crie e restaure um backup seguindo o procedimento acima, sempre em outro
   arquivo. Confira também a tabela de alterações.

As hipóteses de correção e cancelamento ainda precisam ser revistas com Maria
Luzia. Esta etapa não inicia a integração Google.


## Evidências — 18/10/2026

- Suíte Go, `go vet` e builds da API e do comando de banco aprovados.
- Formatação, checagem Svelte/TypeScript (zero erros/avisos) e build aprovados.
- Teste reproduzível no Chrome passou com SQLite temporário: cadastro, edição,
  recálculo, rejeição de horário inválido, conflito e preservação dos campos,
  cancelamento/reativação, ajuste da última página, foco por Tab, largura 375px
  sem rolagem horizontal e recuperação após reinício da API.
- Backup executado em processo separado, copiado para outro arquivo e verificado:
  integridade, referências e igualdade dos 21 registros aprovadas. Destino
  existente e origem ausente foram rejeitados. Os testes Go também comprovaram
  preservação da trilha de alterações no backup.
- Bancos de desenvolvimento não foram usados nesses testes. Validação com a
  professora e revisão com leitor de tela continuam fora destas evidências.
