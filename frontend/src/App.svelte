<script lang="ts">
  import { onMount } from 'svelte'
  import AulaForm from './lib/AulaForm.svelte'
  import Historico from './lib/Historico.svelte'
  import { isTurma, jsonRequest, type Turma } from './lib/api'
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
  <p class="eyebrow">Aula Sync App</p>
  <h1>Registrar aula</h1>
  <p>Registre o conteúdo e os horários da aula em um só lugar.</p>

  {#if turmasCarregando}<p role="status">
      Carregando turmas e estudantes…
    </p>{/if}
  {#if turmasErro}<p class="error" role="alert">{turmasErro}</p>
  {/if}
  <button onclick={carregarTurmas} disabled={turmasCarregando}
    >Atualizar turmas</button
  >
  <AulaForm
    {apiUrl}
    {turmas}
    {turmasCarregando}
    {turmaCriada}
    onSaved={() => revisao++}
  />
  <Historico {apiUrl} {turmas} {revisao} />

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
      {connection.status === 'loading' ? 'Verificando…' : 'Verificar novamente'}
    </button>
  </section>
</main>
