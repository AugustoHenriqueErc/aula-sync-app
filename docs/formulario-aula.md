# Formulário de aula — etapa 7

A página inicial agora permite cadastrar uma aula. Os campos seguem as
[hipóteses do MVP](dominio-aula.md), ainda pendentes de revisão com Maria Luzia.
O formulário usa o [contrato HTTP v1](api-v1.md) e mantém o diagnóstico de conexão
abaixo do cadastro. Histórico e filtros pertencem à etapa 8.

## Comportamento

- Data, início, término, turma/estudante e conteúdo são obrigatórios.
- Intervalo começa em zero; no formulário deve conter um inteiro não negativo.
  A API continua permitindo omitir esse campo, conforme o contrato.
- Observações são opcionais. Textos preenchidos são enviados sem normalização.
- A prévia desconta o intervalo em minutos, sem arredondamento. Não é enviada
  como duração à API. A confirmação usa `duracao_minutos` retornada pelo servidor.
- Erros locais e erros 422 da API aparecem junto ao campo, com `aria-invalid`,
  descrição associada e foco para correção. A validação do backend continua
  sendo a autoridade sobre os dados persistidos.
- Durante o envio, o `fieldset` desabilita os campos e o botão. Uma trava no
  handler também impede requisições simultâneas por cliques ou eventos repetidos.
- Depois de uma resposta 201 com ID e duração válidos, a confirmação recebe foco
  e o formulário volta a ficar vazio, com intervalo zero, para outra aula.
- Falhas preservam os valores. O envio tem limite de dez segundos e não é repetido
  automaticamente. Ao sair do componente, a requisição é cancelada no navegador.

Se a resposta se perder após o servidor gravar, o navegador não pode afirmar que
o cadastro falhou. Por isso, erros de rede, timeout ou confirmação incompatível
orientam a conferir se a aula foi salva antes de reenviar. Nesta etapa, a consulta
pode ser feita por `GET /aulas` ou pelo comando `cmd/db`; o histórico web ainda
não existe. A trava de envio não é idempotência entre requisições separadas:
repetir um POST válido pode criar outro registro.

## Organização para estudo

`frontend/src/lib/AulaForm.svelte` reúne estado, validação, prévia e envio do
formulário. `App.svelte` fornece a URL da API e mantém o diagnóstico existente.
Não foi adicionada biblioteca de formulários nem alterado o backend.

O estado usa `$state`, e `$derived` recalcula a prévia ao mudar horários ou
intervalo. A resposta HTTP é tratada como `unknown` até conferir sua estrutura.
Os inputs de data e hora são nativos: a apresentação acompanha a configuração
do navegador, enquanto os valores enviados seguem `AAAA-MM-DD` e `HH:MM`.

## Teste manual

Em dois terminais, a partir da raiz do projeto:

```bash
cd backend
DATABASE_PATH=aulas-formulario-estudo.db go run ./cmd/api
```

```bash
cd frontend
npm run dev -- --host 127.0.0.1 --port 5173 --strictPort
```

Abra http://127.0.0.1:5173 e siga o roteiro:

1. Clique em **Salvar aula** sem preencher: os campos obrigatórios devem ser
   indicados, o foco deve ir para a data e nenhuma requisição POST deve ocorrer.
2. Preencha data `13/10/2026`, início `08:00`, término `10:00`, intervalo `15`,
   turma `Turma fictícia` e conteúdo `Leitura de demonstração`. A prévia deve ser
   **105 minutos**. Observações podem ficar vazias.
3. Altere o término para `07:00`, ou o intervalo para `120`, e tente salvar:
   o campo correspondente deve receber erro e o formulário não deve ser limpo.
4. Corrija os valores. Percorra o formulário com Tab/Shift+Tab e acione o botão
   por Enter. Os campos devem ter labels e o foco deve permanecer visível.
5. Durante uma conexão lenta (simulada pelo DevTools), tente clicar duas vezes:
   deve haver apenas um POST em andamento. O botão mostra **Salvando…**.
6. Após salvar, confira a confirmação de **105 minutos** e o formulário limpo.
   Compare os valores com `curl http://localhost:3000/aulas` ou, em `backend/`,
   `DATABASE_PATH=aulas-formulario-estudo.db go run ./cmd/db listar`.
7. Preencha outra aula, desligue o backend e tente salvar. Os dados devem permanecer.
   Religue o backend e tente novamente. Evite reenviar um cadastro cuja resposta
   se perdeu sem antes conferir os registros existentes.
8. Reduza a largura para 375 px: campos e botões devem caber sem rolagem horizontal.

Para verificar o código, em `frontend/`:

```bash
npm run format:check
npm run check
npm run build
```

## Validação executada

Em 13/10/2026, formatação, checagem Svelte/TypeScript (zero erros e avisos) e build
passaram. O Chrome headless usou API na porta 33007, Vite na 5177 e banco de teste
em `/tmp`, sem alterar bancos existentes. Foram verificados campos obrigatórios,
horários inválidos, intervalo, prévia, Tab e Enter, cadastro real com 105 minutos,
correspondência com GET /aulas, foco e limpeza após sucesso, bloqueio de eventos
repetidos e preservação de valores após desligar/religar o backend.

O teste de erro 422 modificou somente a requisição interceptada e recebeu a
validação real do Go. Resposta incompatível e timeout foram simulados no Chrome;
ambos preservaram os dados sem reenvio automático. A captura completa em 375 px
foi revisada, sem rolagem horizontal. Os processos de teste foram encerrados.
