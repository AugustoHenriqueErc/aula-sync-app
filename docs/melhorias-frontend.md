# Melhorias de experiência — etapa 11

Solicitadas pelo desenvolvedor após confirmar o envio de uma aula para o Google
pelo botão **Sincronizar com Google**. Mantêm o escopo do fluxo de aulas e Sheets.

## Implementado

- Perfil com foto circular da conta Google e nome. O e-mail permanece nos dados
  da sessão, sem exibição no perfil. Se a foto não carregar, aparece a inicial do nome.
- **Trocar de conta** inicia diretamente a seleção de conta do Google. A foto é
  obtida do token de identidade validado; reiniciar o backend e entrar novamente
  atualiza a sessão com esse campo.
- Depois de cadastrar uma aula, um popup oferece sincronizá-la. Recusar mantém
  o cadastro local. Aceitar consulta o destino atual e tenta o envio. Falha no
  Google não muda o sucesso do cadastro nem provoca um segundo cadastro.
- Histórico de aulas ativas com fundo laranja e **Envio pendente** ou fundo verde e **Sincronizado**; canceladas em vermelho.
  Pendência inclui primeira tentativa, erro, correção e cancelamento ainda não
  enviados. Erros e orientação para preparar o mês continuam disponíveis.
- **Enviar todas as pendentes** e **Cancelar todas as aulas** respeitam os filtros
  aplicados e incluem todas as páginas. A confirmação mostra a quantidade.
  Cancelamento afeta somente aulas ativas, preservando os registros locais.
- O lote busca todas as páginas antes de começar. Usa as versões de cada aula e
  da configuração, mostra progresso e para na primeira falha, informando o total
  confirmado. As operações anteriores continuam válidas; uma nova consulta permite
  conferir os resultados. A ação de cancelar não envia alterações ao Google:
  selecione **Canceladas** ou **Todas** e sincronize para refletir a mudança lá.
- A configuração e os meses são carregados ao abrir **Integração Google** e após
  salvar o destino. O rascunho do link é preservado ao navegar entre abas.
- Removidos o botão **Recarregar configuração e prévia**, o título **Abas esperadas
  por mês** e o texto de prévia solicitado. As ações de inspeção/preparação dos
  meses continuam disponíveis.

## Atualização aprovada em 21/10/2026

A verificação de leitura ao salvar o destino foi implementada, e o botão separado
foi removido. O histórico agora abre em **Todas**, mantém canceladas em vermelho
e oferece exclusão definitiva. A preparação pode ser confirmada pelo histórico.
Detalhes e roteiro em [ajustes da etapa 11](ajustes-integracao-etapa11.md).

### Pendência futura

Detectar dois cadastros diferentes de aulas com dados iguais. O aviso atual de
duplicidade refere-se ao reenvio da mesma aula já sincronizada, conforme aprovado.

## Como conferir

1. Reinicie o backend com OAuth e entre com Google: conferir foto, nome e **Trocar
   de conta**, sem e-mail no perfil. Testar também o retorno à seleção de conta.
2. Cadastre uma aula e recuse o convite: ela deve aparecer uma única vez, como
   **Envio pendente** em laranja. Cadastre outra e aceite: com mês preparado e
   permissão de escrita, deve ficar **Sincronizado** em verde.
3. Force falha de envio: a aula deve continuar salva; use o histórico para tentar
   novamente. Corrigir ou cancelar uma aula sincronizada deve voltar a pendência.
4. Com mais de 20 aulas fictícias, aplique filtros e confira os quantitativos nas
   confirmações de lote. Recuse primeiro e confira que nada foi alterado.
5. Confirme o envio de pendentes: registros já sincronizados ficam fora do lote.
   Em uma falha intermediária, conferir quantas aulas foram concluídas.
6. Cancele as ativas do filtro: confira total local e registros preservados. Use
   **Canceladas** ou **Todas** para sincronizar os cancelamentos depois.
7. Cadastre uma aula de outro mês; abra **Integração Google**. O mês deve aparecer
   automaticamente. Salve o link e confira a leitura automática. Navegue entre abas com um link em rascunho e confira sua preservação.

Verificação automatizada aprovada:

```bash
# Em backend/
go test ./internal/auth
go vet ./internal/auth

# Em frontend/
npm run check
npm run build
npm run format:check

# Na raiz, com Chrome instalado
node scripts/test-etapa9.cjs --ux
node scripts/test-etapa9.cjs --profile
```

Os testes de navegador usam SQLite temporário e respostas de sessão/Sheets
simuladas. Cobrem recusa, envio após cadastro, falha parcial, filtros, lotes em
várias páginas, preservação dos registros e tela móvel. O teste Go confirma a
foto recebida no token de identidade e exposta na sessão sem access token.
