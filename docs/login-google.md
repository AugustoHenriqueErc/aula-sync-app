# Etapa 10 — Login Google e leitura de teste

Implementado: login OpenID Connect + OAuth 2.0 no backend Go, sessão vinculada
à mesma conta e leitura de metadados pela API Sheets. Escrita de aulas, criação
de abas e inspeção de cabeçalhos continuam na etapa 11. Validação com uma conta
Google real ainda pendente; testes automatizados não comprovam consentimento real.

## Como funciona

O navegador inicia o login por POST e segue para o Google. O backend verifica
`state`, troca o código usando PKCE e valida assinatura, emissor, audiência,
validade, `nonce`, `azp` quando presente e e-mail verificado do ID token. A conta
é identificada pelo `sub`. A sessão recebe um identificador aleatório em cookie
HttpOnly e SameSite=Lax (Secure em HTTPS). Tokens nunca são devolvidos ao frontend.

O access token da mesma troca é usado para ler a planilha salva. Uma conta sem
o escopo Sheets pode entrar, mas a leitura fica bloqueada até novo consentimento.
Somente a identidade não concede acesso à planilha. A planilha pode ser própria
ou compartilhada; escrita futura exige permissão de edição.

Sessões e tokens ficam exclusivamente na memória, por até uma hora ou até a
expiração dos tokens, o que ocorrer primeiro. Não há refresh token, acesso offline
ou persistência de credenciais neste incremento. Reiniciar o backend, expirar ou
sair exige novo login. Sair elimina a sessão e seu token no aplicativo, mas não
revoga o consentimento no Google. Uma resposta 401 do Sheets encerra a sessão;
403/404 indicam falha de acesso/configuração e nunca significam aba ausente.

Com OAuth configurado, aulas, turmas e configuração exigem sessão. Escritas HTTP
exigem Origin igual a APP_URL. Health e rotas de autenticação continuam públicos.
Sem nenhuma variável OAuth, o modo local de estudo permanece sem autenticação.
Configuração parcial falha na inicialização. Frontend e backend devem ser
acessados pela mesma origem pública, via proxy `/api`.

**Limite do protótipo:** aulas e destino ainda pertencem a uma única base local,
compartilhada entre as contas que entrarem nesta instalação. Não há isolamento
por usuário nem lista de usuários permitidos. Usar somente dados fictícios e
usuários de teste; revisar autorização dos dados locais antes do piloto.

## Configurar o teste real

1. No Google Cloud, criar/selecionar um projeto e habilitar **Google Sheets API**.
2. Configurar a tela de consentimento Google Auth Platform. Para conta pessoal,
   usar público externo em teste e adicionar a conta aos usuários de teste.
3. Criar um cliente OAuth do tipo **Aplicativo da Web**. Cadastrar exatamente
   `http://localhost:5173/api/auth/google/callback` como URI de redirecionamento.
4. Solicitar `openid email profile` e
   `https://www.googleapis.com/auth/spreadsheets`. O escopo de Sheets é amplo;
   não fica limitado ao link. O aplicativo limita a leitura ao destino salvo.
5. Exportar as quatro variáveis abaixo no terminal que inicia o backend. Não
   enviar o segredo na conversa, versioná-lo nem colocá-lo em `frontend/.env`.
   Para não digitar o segredo no histórico do shell, usar `read -s`:

```bash
cd backend
read -r -p 'Client ID: ' GOOGLE_CLIENT_ID
read -r -s -p 'Client secret: ' GOOGLE_CLIENT_SECRET
export GOOGLE_CLIENT_ID GOOGLE_CLIENT_SECRET
export APP_URL=http://localhost:5173
export GOOGLE_REDIRECT_URI=http://localhost:5173/api/auth/google/callback
go run ./cmd/api
```

Em outro terminal:

```bash
cd frontend
npm run dev -- --host localhost --port 5173 --strictPort
```

Abrir `http://localhost:5173`. Não alternar com `127.0.0.1`: a origem e o retorno
precisam coincidir com a configuração. O backend lê o ambiente do processo, não
carrega `.env` automaticamente. HTTPS é obrigatório fora de localhost/127.0.0.1.

## Como testar

1. Sem sessão, conferir a tela **Entrar com Google** e rejeição 401 das rotas de aulas.
2. Entrar com a conta de teste e autorizar Sheets; conferir nome/e-mail exibidos.
3. Colar e salvar o link de uma planilha descartável própria ou compartilhada.
4. Clicar **Verificar leitura com minha conta**. Deve exibir o título e explicar
   que escrita e cabeçalhos não foram verificados. Não deve criar abas ou linhas.
5. Testar link de arquivo inacessível: deve apresentar erro e preservar os dados locais.
6. Testar uma planilha com somente leitura: a leitura pode funcionar; isso não
   comprova que será possível gravar. Essa verificação pertence à etapa 11.
7. Sair/trocar de conta e entrar com outra conta de teste. A consulta seguinte
   deve usar as permissões da nova conta. O destino local permanece o mesmo,
   mas a leitura anterior não é exibida como autorização da conta atual.
8. Recusar consentimento; testar callback inválido; reiniciar backend; revogar
   autorização no Google e tentar ler. Exigir novo login quando a sessão/token
   deixar de ser válido. Não registrar tokens como evidência.

Verificações automatizadas, sem conta Google:

```bash
cd backend
go test -race ./...
go vet ./...
```

```bash
cd frontend
npm run check
npm run build
```

Da raiz, com Chrome instalado (ou `CHROME_BIN` apontando para ele):

```bash
node scripts/test-etapa9.cjs --auth
node scripts/test-etapa9.cjs --google
```

O teste `--auth` usa credenciais fictícias e verifica a tela, bloqueio de acesso e
callback inválido sem chamar o Google. Os testes Go usam tokens assinados e
verificador OIDC real com chave local: cobrem assinatura, audiência, emissor,
expiração, nonce, state, PKCE, repetição, escopo, conta executora, origem e logout.
O teste `--google` continua verificando cadastro e configuração no modo local.

## Rotas adicionadas

| Método e rota no backend | Comportamento |
| --- | --- |
| GET `/auth/session` | Estado da configuração e identidade da sessão; sem tokens |
| POST `/auth/google/login` | Inicia login, invalida sessão anterior e exige Origin |
| GET `/auth/google/callback` | Consome tentativa única e redireciona à aplicação |
| POST `/auth/logout` | Remove sessão/token e cookie; exige Origin |
| GET `/integracoes/google/acesso` | Lê ID/título do destino salvo com token da sessão |

No navegador, o proxy acrescenta `/api`. O resultado de acesso inclui a versão
da configuração, `leitura_confirmada: true` e `escrita_verificada: false`. Mudança
de destino durante a leitura resulta em conflito, sem confirmar o destino antigo.

Referências: [OAuth web](https://developers.google.com/identity/protocols/oauth2/web-server),
[OpenID Connect](https://developers.google.com/identity/openid-connect/openid-connect) e
[escopos Sheets](https://developers.google.com/workspace/sheets/api/scopes).
