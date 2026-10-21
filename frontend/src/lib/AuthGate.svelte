<script lang="ts">
  import { onMount } from 'svelte'
  import App from '../App.svelte'

  const apiUrl = (import.meta.env.VITE_API_URL?.trim() || '/api').replace(
    /\/+$/,
    '',
  )
  type User = { sub: string; nome: string; email: string; foto?: string }
  let fotoFalhou = $state(false)
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
            typeof data.usuario?.email !== 'string' ||
            (data.usuario.foto !== undefined &&
              typeof data.usuario.foto !== 'string')))
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
    <div class="perfil">
      {#if usuario.foto?.startsWith('https://') && !fotoFalhou}
        <img
          class="avatar"
          src={usuario.foto}
          alt=""
          width="44"
          height="44"
          referrerpolicy="no-referrer"
          onerror={() => (fotoFalhou = true)}
        />
      {:else}
        <span class="avatar avatar-iniciais" aria-hidden="true"
          >{(usuario.nome || 'G').trim().slice(0, 1).toUpperCase()}</span
        >
      {/if}
      <strong class="perfil-nome">{usuario.nome || 'Conta Google'}</strong>
      <form method="POST" action={`${apiUrl}/auth/google/login`}>
        <button class="trocar-conta" type="submit">Trocar de conta</button>
      </form>
    </div>
    {#if !autorizado}<p role="status">
        Acesso ao Sheets não autorizado. Entre novamente para conceder acesso à
        planilha.
      </p>{/if}
  {:else}
    <img src="/aula-sync-icon.png" alt="" width="64" height="64" />
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
  .perfil {
    display: flex;
    align-items: center;
    gap: 12px;
    flex-wrap: wrap;
  }
  .avatar {
    width: 44px;
    height: 44px;
    flex: 0 0 44px;
    border-radius: 50%;
    object-fit: cover;
  }
  .avatar-iniciais {
    display: grid;
    place-items: center;
    background: #e0e7ff;
    color: #254fc4;
    font-weight: 700;
  }
  .perfil-nome {
    flex: 1;
    min-width: 100px;
  }
  .trocar-conta {
    background: #eef3ff;
    color: #254fc4;
    padding: 8px 12px;
    font-size: 0.9rem;
  }
  .trocar-conta:hover {
    background: #dce5f4;
  }
</style>
