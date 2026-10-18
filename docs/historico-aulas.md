# Etapa 8 — Histórico e carga horária

## Decisões

- Intervalos de datas fechados, com limites opcionais e inclusivos.
- Combobox nativo (`select`) para turma/estudante; criação explícita no formulário.
- Nome único por espaços normalizados e minúsculas, preservando acentos.
- Botões Filtrar/Limpar e páginas fixas de 20 registros.
- Total e quantidade calculados pelo backend sobre todo o conjunto filtrado.
- Respostas antigas são canceladas ao trocar a consulta; timeout de 10 segundos.
- O cadastro confirmado atualiza a consulta sem remover os filtros ativos.
- Datas civis são formatadas sem conversão para UTC no histórico.

Na etapa 8 não havia edição. A etapa 9 adicionou edição e cancelamento
reversível, mantendo o histórico padrão somente com aulas ativas. Veja
[correção e backup](correcao-backup.md). Google e exportação seguem futuros.

## Banco e migração

A migração 002 cria `turmas` e reconstrói `aulas` com `turma_id` obrigatório e
chave estrangeira. Preserva IDs e todos os campos originais. O texto `turma`
permanece como registro do nome usado na aula; novas aulas recebem o nome do
catálogo. Nomes equivalentes compartilham um ID, com nome de exibição derivado
da primeira aula por ID. O procedimento inteiro é transacional, incluindo a
versão do esquema. Uma falha reverte a migração. `foreign_keys` é habilitado em
cada conexão. O comando didático `cmd/db inserir-exemplo` reutiliza ou cria a
turma pelo nome; a API pública exige cadastro prévio e ID.

## Testes automatizados

Em `backend/`:

```bash
go test ./...
go vet ./...
go build -o /tmp/aula-sync-backend ./cmd/api
```

A suíte cobre migração v1→v2 e reexecução, preservação e agrupamento dos nomes,
rollback, chave estrangeira, duplicidade concorrente, filtros e limites de mês,
60+90+45=195 minutos, 21 registros em páginas de 20+1, página além do fim,
parâmetros inválidos e contratos HTTP.

Em `frontend/`:

```bash
npm run format:check
npm run check
npm run build
```

## Teste manual reproduzível

1. Inicie o backend com um banco fictício separado:
   `DATABASE_PATH=aulas-etapa8-estudo.db go run ./cmd/api`, em `backend/`.
2. Em `frontend/`, execute `npm run dev` e abra a URL informada.
3. Crie `Augusto`. Tente criar ` augusto `: deve informar duplicidade.
4. Selecione Augusto e cadastre aulas: 30/09/2026 08:00–09:00,
   01/10/2026 08:00–09:30 e 31/10/2026 08:00–08:45, todas sem intervalo.
5. O histórico deve mostrar 3 aulas, mais recente primeiro, total **3h15**.
6. Filtre 30/09 a 31/10: continua **3h15**, incluindo os extremos.
   Filtre 01/10 a 31/10: **2h15**. Datas iguais incluem apenas aquele dia.
7. Teste cada limite sozinho, uma turma diferente, período sem aulas e período
   invertido. Limpar restaura todos os registros e a primeira página.
8. Com filtro ativo, cadastre aulas dentro e fora do período; confirme que
   a lista e o total respeitam o filtro, mantendo a confirmação do cadastro.
9. Cadastre mais registros fictícios até totalizar 21. Confira 20 na página 1,
   um na página 2 e o mesmo total de carga horária nas duas páginas.
10. Desligue o backend e aplique o filtro: deve aparecer erro e nova tentativa.
    Reinicie com o mesmo banco e tente novamente; registros devem permanecer.
11. Verifique a interface em 375px e navegue com Tab/Enter, inclusive seleção de
    turma, criação, filtros e paginação. A página não deve ter rolagem horizontal.
12. No DevTools, use rede lenta para observar carregamento. Troque filtros
    rapidamente: a última consulta aplicada deve prevalecer.

O botão Atualizar turmas permite recarregar cadastros criados em outra aba e
recuperar a lista após falha de conexão. A lista de opções ainda é integral,
adequada ao MVP local. Regras institucionais continuam pendentes de revisão.


## Evidências da validação — 17/10/2026

Testes Go, vet e build aprovados. Frontend com checagem Svelte/TypeScript sem
erros ou avisos e build aprovado. Chrome headless, API e Vite reais com SQLite
temporário validaram criação/duplicidade de nome, cadastro, total 3h15,
extremos inclusivos, subtotal 2h15, cadastro fora do filtro, estado vazio,
21 registros em páginas de 20+1, total global constante, falha de conexão,
recuperação após reiniciar a API com o mesmo banco e foco por Tab. Captura de 375px revisada, sem rolagem horizontal. Isso não
substitui uma revisão com leitora de tela ou a validação com a professora.
