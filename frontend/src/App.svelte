<script lang="ts">
  import { onMount, tick } from 'svelte'
  import AulaForm from './lib/AulaForm.svelte'
  import Historico from './lib/Historico.svelte'
  import GoogleConfig from './lib/GoogleConfig.svelte'
  import { isTurma, jsonRequest, type Turma, type Aula } from './lib/api'
  const abas = [
    { id: 'cadastro', titulo: 'Registrar aula' },
    { id: 'historico', titulo: 'Histórico' },
    { id: 'google', titulo: 'Integração Google' },
  ] as const
  type Aba = (typeof abas)[number]['id']
  let abaAtiva = $state<Aba>('cadastro')
  function navegarAbas(event: KeyboardEvent, indice: number) {
    if (salvando) return
    let proxima: number
    switch (event.key) {
      case 'ArrowRight':
        proxima = (indice + 1) % abas.length
        break
      case 'ArrowLeft':
        proxima = (indice + abas.length - 1) % abas.length
        break
      case 'Home':
        proxima = 0
        break
      case 'End':
        proxima = abas.length - 1
        break
      default:
        return
    }
    event.preventDefault()
    abaAtiva = abas[proxima].id
    document.getElementById(`aba-${abaAtiva}`)?.focus()
  }
  let salvando = $state(false)
  let avisoEdicao = $state('')
  let editando = $state<Aula | null>(null)
  async function editar(aula: Aula) {
    if (salvando) return
    avisoEdicao = ''
    editando = aula
    abaAtiva = 'cadastro'
    await tick()
    document
      .getElementById('form-title')
      ?.scrollIntoView({ behavior: 'smooth', block: 'start' })
    document.getElementById('data')?.focus({ preventScroll: true })
  }
  function salvo() {
    avisoEdicao = editando
      ? 'Alterações salvas. O histórico foi atualizado.'
      : ''
    editando = null
    revisao++
  }
  let turmas = $state<Turma[]>([])
  let turmasErro = $state('')
  let turmasCarregando = $state(true)
  let revisao = $state(0)
  let turmasRequest: AbortController | undefined
  async function carregarTurmas() {
    turmasRequest?.abort()
    const controller = new AbortController()
    turmasRequest = controller
    turmasCarregando = true
    turmasErro = ''
    try {
      const data = await jsonRequest(`${apiUrl}/turmas`, {
        signal: controller.signal,
      })
      if (!Array.isArray(data) || !data.every(isTurma))
        throw new Error('Resposta de turmas inválida.')
      if (!controller.signal.aborted) turmas = data
    } catch {
      if (!controller.signal.aborted)
        turmasErro = 'Não foi possível carregar turmas e estudantes.'
    } finally {
      if (!controller.signal.aborted) turmasCarregando = false
    }
  }
  function turmaCriada(turma: Turma) {
    turmasRequest?.abort()
    turmasCarregando = false
    turmasErro = ''
    turmas = [...turmas.filter((t) => t.id !== turma.id), turma].sort((a, b) =>
      a.nome.localeCompare(b.nome),
    )
  }

  type ConnectionState =
    | { status: 'loading' }
    | { status: 'success' }
    | { status: 'error'; message: string }

  const apiUrl = (import.meta.env.VITE_API_URL?.trim() || '/api').replace(
    /\/+$/,
    '',
  )
  let connection = $state<ConnectionState>({ status: 'loading' })
  let activeRequest: AbortController | undefined

  async function checkConnection() {
    if (activeRequest) return

    const controller = new AbortController()
    activeRequest = controller
    connection = { status: 'loading' }
    let timedOut = false
    const timeout = window.setTimeout(() => {
      timedOut = true
      controller.abort()
    }, 5000)

    try {
      const response = await fetch(`${apiUrl}/health`, {
        signal: controller.signal,
        cache: 'no-store',
      })
      if (!response.ok) {
        throw new Error(
          'Não foi possível consultar o backend. Verifique se ele está em execução e tente novamente.',
        )
      }

      // O tipo unknown exige conferir os dados externos antes de utilizá-los.
      const data: unknown = await response.json()
      if (
        typeof data !== 'object' ||
        data === null ||
        !('status' in data) ||
        data.status !== 'ok'
      ) {
        throw new Error(
          'O backend enviou uma resposta inesperada. Verifique a configuração da API.',
        )
      }
      connection = { status: 'success' }
    } catch (error: unknown) {
      if (controller.signal.aborted && !timedOut) return
      let message =
        'Não foi possível conectar ao backend. Verifique se ele está em execução e tente novamente.'
      if (timedOut) {
        message =
          'O backend demorou mais de 5 segundos para responder. Tente novamente.'
      } else if (error instanceof SyntaxError) {
        message =
          'O backend enviou uma resposta inesperada. Verifique a configuração da API.'
      } else if (error instanceof Error && !(error instanceof TypeError)) {
        message = error.message
      }
      connection = { status: 'error', message }
    } finally {
      window.clearTimeout(timeout)
      activeRequest = undefined
    }
  }

  onMount(() => {
    void carregarTurmas()
    void checkConnection()
    return () => {
      activeRequest?.abort()
      turmasRequest?.abort()
    }
  })
</script>

<main>
  <p class="eyebrow app-brand">
    <img src="/aula-sync-icon.png" alt="" width="48" height="48" />
    <span>Aula Sync App</span>
  </p>
  <h1>Suas aulas</h1>
  <p>Registre aulas, consulte o histórico e configure sua planilha.</p>

  <div class="abas" role="tablist" aria-label="Áreas da aplicação">
    {#each abas as aba, indice}
      <button
        type="button"
        role="tab"
        id={`aba-${aba.id}`}
        aria-controls={`painel-${aba.id}`}
        aria-selected={abaAtiva === aba.id}
        tabindex={abaAtiva === aba.id ? 0 : -1}
        disabled={salvando}
        onclick={() => (abaAtiva = aba.id)}
        onkeydown={(event) => navegarAbas(event, indice)}
        >{aba.titulo}{aba.id === 'cadastro' && editando
          ? ' (edição)'
          : ''}</button
      >
    {/each}
  </div>

  <div
    id="painel-cadastro"
    role="tabpanel"
    aria-labelledby="aba-cadastro"
    hidden={abaAtiva !== 'cadastro'}
    tabindex="0"
  >
    {#if turmasCarregando}<p role="status">
        Carregando turmas e estudantes…
      </p>{/if}
    {#if turmasErro}<p class="error" role="alert">{turmasErro}</p>
    {/if}
    <button onclick={carregarTurmas} disabled={turmasCarregando}
      >Atualizar turmas</button
    >
    <p role="status">{avisoEdicao}</p>
    <AulaForm
      {apiUrl}
      {turmas}
      {turmasCarregando}
      {turmaCriada}
      onSaved={salvo}
      onBusy={(value) => (salvando = value)}
      {editando}
      onCancel={() => {
        editando = null
        revisao++
      }}
    />
  </div>
  <div
    id="painel-historico"
    role="tabpanel"
    aria-labelledby="aba-historico"
    hidden={abaAtiva !== 'historico'}
    tabindex="0"
  >
    <Historico
      {apiUrl}
      {turmas}
      {revisao}
      onEdit={editar}
      bloqueado={salvando}
    />
  </div>
  <div
    id="painel-google"
    role="tabpanel"
    aria-labelledby="aba-google"
    hidden={abaAtiva !== 'google'}
    tabindex="0"
  >
    <GoogleConfig {apiUrl} onChange={() => revisao++} />
  </div>

  <details class="diagnostico">
    <summary
      >Conexão com o backend{connection.status === 'error'
        ? ' — indisponível'
        : ''}</summary
    >
    <section aria-labelledby="connection-title">
      <h2 id="connection-title">Conexão com o backend</h2>
      <div class="status" role="status" aria-live="polite" aria-atomic="true">
        {#if connection.status === 'loading'}
          <p>Verificando conexão…</p>
        {:else if connection.status === 'success'}
          <p class="success">Backend conectado.</p>
          <p>A API respondeu e está disponível.</p>
        {:else}
          <p class="error">Falha na conexão.</p>
          <p>{connection.message}</p>
        {/if}
      </div>
      <button
        onclick={checkConnection}
        disabled={connection.status === 'loading'}
      >
        {connection.status === 'loading'
          ? 'Verificando…'
          : 'Verificar novamente'}
      </button>
    </section>
  </details>
</main>

<style>
  .abas {
    display: flex;
    gap: 8px;
    margin: 24px 0;
    padding: 4px;
    border: 1px solid #d5dce5;
    border-radius: 10px;
    background: #e9eef6;
  }
  .abas button {
    flex: 1;
    min-width: 0;
    padding: 12px 8px;
    font-size: 0.9rem;
    color: #233047;
    background: transparent;
  }
  .abas button:hover:not(:disabled) {
    background: #dce5f4;
  }
  .abas button[aria-selected='true'] {
    color: #fff;
    background: #254fc4;
    box-shadow: 0 2px 4px #23304726;
  }
  .abas button:disabled {
    opacity: 0.65;
  }
  [role='tabpanel']:focus-visible {
    outline: 3px solid #254fc4;
    outline-offset: 4px;
  }
  .diagnostico {
    margin-top: 32px;
  }
  .diagnostico summary {
    color: #475569;
  }
</style>
