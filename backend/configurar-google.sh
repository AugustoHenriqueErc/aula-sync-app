#!/usr/bin/env bash

# Use source para manter as variáveis no terminal que iniciará o backend.
if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  printf 'Execute com: source ./configurar-google.sh\n' >&2
  exit 1
fi

printf 'Cole apenas os valores, sem rótulos como CLIENT ID: ou CLIENT SECRET:.\n'
if ! IFS= read -r -p 'Client ID (visível): ' GOOGLE_CLIENT_ID </dev/tty; then
  printf '\nNão foi possível ler o Client ID no terminal.\n' >&2
  return 1
fi
if [[ "$GOOGLE_CLIENT_ID" != ?*.apps.googleusercontent.com || "$GOOGLE_CLIENT_ID" == *[[:space:]]* ]]; then
  printf 'Client ID inválido: cole o ID completo terminado em .apps.googleusercontent.com, sem espaços.\n' >&2
  return 1
fi
if ! IFS= read -rs -p 'Client Secret (oculto; cole o valor e pressione Enter): ' GOOGLE_CLIENT_SECRET </dev/tty; then
  printf '\nNão foi possível ler o Client Secret no terminal.\n' >&2
  return 1
fi
printf '\n'
if [[ -z "$GOOGLE_CLIENT_SECRET" || "$GOOGLE_CLIENT_SECRET" == *[[:space:]]* ]]; then
  printf 'Client Secret vazio ou com espaços: execute novamente e cole somente o valor.\n' >&2
  return 1
fi

# Preserva URLs já configuradas; usa o túnel ngrok de desenvolvimento como padrão.
export GOOGLE_REDIRECT_URI="${GOOGLE_REDIRECT_URI:-https://transpire-travesty-untagged.ngrok-free.dev/api/auth/google/callback}"
export APP_URL="${APP_URL:-https://transpire-travesty-untagged.ngrok-free.dev}"
export GOOGLE_CLIENT_ID GOOGLE_CLIENT_SECRET

google_config_incompleta=0
for google_config_nome in GOOGLE_CLIENT_ID GOOGLE_CLIENT_SECRET GOOGLE_REDIRECT_URI APP_URL; do
  if [[ -n "${!google_config_nome}" ]]; then
    printf '%s: OK\n' "$google_config_nome"
  else
    printf '%s: VAZIA\n' "$google_config_nome"
    google_config_incompleta=1
  fi
done
unset google_config_nome

if [[ "$google_config_incompleta" == 1 ]]; then
  unset google_config_incompleta
  printf 'Configuração incompleta. Execute o script novamente e preencha os campos vazios.\n' >&2
  return 1
fi
unset google_config_incompleta

printf 'Origem da aplicação: %s\n' "$APP_URL"
printf 'Confira se esta URL está nos URIs de redirecionamento autorizados do Google OAuth:\n%s\n' "$GOOGLE_REDIRECT_URI"
printf 'Ao trocar o domínio, atualize APP_URL, GOOGLE_REDIRECT_URI e o cadastro no Google OAuth.\n'
printf 'Variáveis preenchidas. Agora execute neste terminal: go run ./cmd/api/\n'
printf 'Esta verificação não valida as credenciais junto ao Google.\n'
