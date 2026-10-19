<script lang="ts">
  import { onMount } from 'svelte'
  import App from '../App.svelte'

  const apiUrl = (import.meta.env.VITE_API_URL?.trim() || '/api').replace(
    /\/+$/,
    '',
  )
  type User = { sub: string; nome: string; email: string }
  let usuario = $state<User | null>(null)
  let configurado = $state(true)
  let autorizado = $state(false)
  let carregando = $state(true)
  let erro = $state('')
  let aviso = $state('')
  let controller: AbortController | undefined

  async function carregar() {
    controller?.abort()
    const request = new AbortController()
    controller = request
    carregando = true
    erro = ''
    try {
      const response = await fetch(`${apiUrl}/auth/session`, {
        cache: 'no-store',
        signal: AbortSignal.any([request.signal, AbortSignal.timeout(10000)]),
      })
      const data = await response.json()
      if (
        !response.ok ||
        typeof data?.configurado !== 'boolean' ||
        typeof data?.sheets_autorizado !== 'boolean' ||
        (data.usuario !== null &&
          (typeof data.usuario?.sub !== 'string' ||
            typeof data.usuario?.nome !== 'string' ||
            typeof data.usuario?.email !== 'string'))
      )
        throw new Error('Resposta inesperada ao consultar a sessão.')
      if (request.signal.aborted) return
      configurado = data.configurado
      usuario = data.usuario
      autorizado = data.sheets_autorizado
    } catch {
      if (!request.signal.aborted)
        erro =
          'Não foi possível verificar a sessão. Confira o backend e tente novamente.'
    } finally {
      if (!request.signal.aborted) carregando = false
    }
  }
  async function sair() {
    erro = ''
    try {
      const response = await fetch(`${apiUrl}/auth/logout`, {
        method: 'POST',
        signal: AbortSignal.timeout(10000),
      })
      if (!response.ok) throw new Error()
      usuario = null
      autorizado = false
      aviso = 'Sessão encerrada. Para trocar de conta, entre novamente.'
    } catch {
      erro = 'Não foi possível confirmar a saída. Tente novamente.'
    }
  }
  onMount(() => {
    const url = new URL(window.location.href)
    if (url.searchParams.has('auth_error')) {
      aviso =
        url.searchParams.get('auth_error') === 'consentimento_recusado'
          ? 'O acesso não foi autorizado. Você pode tentar entrar novamente.'
          : 'Não foi possível concluir o login. Inicie uma nova tentativa.'
      url.searchParams.delete('auth_error')
      window.history.replaceState(null, '', url)
    }
    const expirada = () => {
      usuario = null
      aviso = 'Sessão expirada ou autorização revogada. Entre novamente.'
      void carregar()
    }
    window.addEventListener('aula-auth-required', expirada)
    void carregar()
    return () => {
      controller?.abort()
      window.removeEventListener('aula-auth-required', expirada)
    }
  })
</script>

<section aria-label="Conta Google">
  {#if carregando}
    <p role="status">Verificando sessão…</p>
  {:else if erro}
    <p role="alert" class="error">{erro}</p>
    <button onclick={carregar}>Verificar sessão novamente</button>
  {:else if !configurado}
    <p class="hint">
      Modo local de estudo. O login Google ainda não foi configurado.
    </p>
  {:else if usuario}
    <p>
      Conta conectada: <strong>{usuario.nome || usuario.email}</strong>
      ({usuario.email})
    </p>
    {#if !autorizado}<p role="status">
        Acesso ao Sheets não autorizado. Entre novamente para conceder acesso à
        planilha.
      </p>{/if}
    <button onclick={sair}>Sair / trocar conta</button>
  {:else}
    <h1>Entrar no Aula Sync</h1>
    <p>
      Use a conta Google que tem permissão de edição na planilha de destino.
    </p>
    <form method="POST" action={`${apiUrl}/auth/google/login`}>
      <button type="submit">Entrar com Google</button>
    </form>
  {/if}
  {#if aviso}<p role="status">{aviso}</p>{/if}
</section>

{#if !carregando && !erro && (!configurado || usuario)}
  <App />
{/if}

<style>
  section {
    width: calc(100% - 32px);
    max-width: 672px;
    margin: 32px auto 0;
    overflow-wrap: anywhere;
  }
</style>
