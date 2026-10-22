# Etapa 12 — Relatório de aulas em PDF

O histórico oferece **Exportar PDF**. O relatório usa os filtros já aplicados
pelo botão **Filtrar**, inclui todas as páginas e consulta somente o SQLite.
Editar os campos sem aplicar o filtro não altera a exportação. O backend lê
registros e totais na mesma transação, com as mesmas regras do histórico.

## Modelo inicial proposto

PDF A4 com título, período (inclusive limites abertos), turma/estudante, situação,
quantidade e carga horária total. Cada aula apresenta identificador, data,
turma/estudante, início, término, intervalo, duração, conteúdo e observações.
Textos têm quebra de linha e continuação em outras páginas; as páginas são numeradas.
O arquivo usa fontes Go incorporadas, com suporte a acentos portugueses.

O relatório mostra a versão atual de cada aula corrigida. Canceladas aparecem
identificadas quando incluídas pelo filtro e não somam na carga horária.
Aulas excluídas definitivamente não aparecem. Sem resultados, gera um PDF com
mensagem explícita e total `0h00`. Não consulta o estado da sincronização.

Este modelo é uma proposta para Maria Luiza. A revisão institucional e os ajustes
solicitados pela cliente continuam pendentes; não há editor de relatórios.

## Contrato HTTP

`GET /aulas/relatorio.pdf` (pelo Vite: `/api/aulas/relatorio.pdf`).

Aceita os mesmos parâmetros de `GET /aulas`: `data_inicial`, `data_final`,
`turma_id`, `situacao`. O parâmetro `pagina`, quando informado e válido, é
ignorado na seleção: o PDF sempre inclui o conjunto completo. Assim como na API
do histórico, a situação omitida significa `ativas`; a interface envia `todas`
por padrão. Datas são inclusivas e a ordenação é data, início e ID decrescentes.

Sucesso: `200 application/pdf`, `Content-Disposition: attachment` com nome
`relatorio-aulas.pdf` e `Cache-Control: no-store`. Parâmetros inválidos: `400`;
sessão ausente com OAuth habilitado: `401`; falha interna: `500`, em JSON.
A proteção de sessão é a mesma das demais rotas de aulas. O PDF não exige
permissão de Sheets nem conexão com o Google; uma sessão local válida continua
necessária quando o OAuth está habilitado.

A interface bloqueia novos pedidos durante a geração, exibe a falha sem perder
os filtros e permite tentar novamente. O download tem limite de espera de 60s.
A geração do arquivo ocorre em memória; volumes muito grandes devem ser
avaliados antes do piloto.

## Como testar manualmente

1. Inicie backend e frontend pelos comandos do README e entre no aplicativo.
2. Cadastre aulas fictícias de 60, 90 e 45 minutos. No histórico, aplique filtros
   que incluam somente essas aulas, exporte e confira o total de **3h15**.
3. Compare datas, nomes, horários, intervalos, conteúdo e observações. Teste
   acentos, várias linhas e texto extenso; abra o arquivo em um leitor de PDF.
4. Teste somente uma aula e um filtro sem resultados. O último gera relatório
   vazio com total zero. Teste também os limites inicial/final e turma.
5. Com mais de 20 aulas, exporte da segunda página e confira todas as aulas.
6. Edite uma aula e cancele outra. Exporte com situações Todas, Ativas e
   Canceladas e compare com o histórico; canceladas não somam no total.
7. Altere um campo de filtro sem clicar em Filtrar: o PDF deve continuar usando
   os filtros aplicados anteriormente.
8. Bloqueie acesso ao Google mantendo o backend local e a sessão ativos: o PDF
   deve continuar disponível. Pare o backend e exporte: deve apresentar falha;
   reinicie, entre novamente se necessário e tente outra vez.

## Validação automatizada

No backend:

```bash
go test -p 1 ./...
go vet ./...
go build ./...
```

Os testes `TestRelatorioPDF*` geram arquivos com SQLite temporário e conferem
o texto com `pdftotext` (Poppler). Sem essa ferramenta, a conferência de texto
é pulada após verificar a resposta PDF. Para guardar exemplos fictícios:

```bash
AULA_PDF_TEST_DIR=/tmp/aula-sync-pdf go test ./internal/httpapi -run TestRelatorioPDF -count=1
```

No frontend: `npm run check`, `npm run build` e `npm run format:check`.
Na raiz: `node scripts/test-etapa12.cjs` usa Chrome, Node 22, Go e `pdftotext`,
com portas e banco temporários. Verifica download real, exportação da segunda
página, filtros aplicados, conjunto vazio, Google bloqueado, falha de rede e
recuperação, além de largura de 375px. Não usa credenciais reais.
