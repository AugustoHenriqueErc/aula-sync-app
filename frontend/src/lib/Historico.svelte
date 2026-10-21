<script lang="ts">
  import { untrack, onDestroy } from 'svelte'
  import {
    jsonRequest,
    isHistorico,
    isAula,
    type Aula,
    duracao,
    type Historico,
    type Turma,
  } from './api'
  let {
    apiUrl,
    turmas,
    revisao,
    onEdit,
    bloqueado,
  }: {
    apiUrl: string
    turmas: Turma[]
    revisao: number
    onEdit: (aula: Aula) => void
    bloqueado: boolean
  } = $props()
  let inicial = $state('')
  let final = $state('')
  let turma = $state('')
  let situacao = $state('ativas')
  let alterando = $state(false)
  let aviso = $state('')
  let sincronizando = $state(false)
  const estadosGoogle: Record<string, string> = {
    sem_destino: 'Configure uma planilha na aba Integração Google',
    aguardando_preparacao:
      'Inspecione e prepare a aba mensal na Integração Google',
    pendente: 'Envio pendente',
    sincronizado: 'Último envio confirmado',
    erro: 'Falha no envio',
  }
  async function sincronizar(aula: Aula) {
    if (sincronizando || !aula.google) return
    if (
      !window.confirm(
        'Enviar esta aula para a planilha configurada? O envio também atualiza correções e cancelamentos anteriores nesse destino.',
      )
    )
      return
    sincronizando = true
    aviso = ''
    try {
      const data = await jsonRequest(`${apiUrl}/aulas/${aula.id}/sincronizar`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          versao: aula.versao,
          config_versao: aula.google.config_versao,
        }),
      })
      if (
        typeof data !== 'object' ||
        data === null ||
        !('estado' in data) ||
        data.estado !== 'sincronizado'
      )
        throw new Error(
          'Envio não confirmado. Atualize o histórico antes de tentar novamente.',
        )
      aviso = 'Aula sincronizada com a planilha configurada.'
    } catch (error) {
      aviso =
        error instanceof Error && error.name === 'Error'
          ? error.message
          : 'Envio não confirmado. Tente novamente para conferir o resultado.'
    } finally {
      sincronizando = false
      tentativa++
    }
  }
  let alteracaoRequest: AbortController | undefined
  async function mudarSituacao(aula: Aula) {
    if (alterando) return
    const cancelar = !aula.cancelada
    if (
      !window.confirm(
        cancelar
          ? 'Cancelar esta aula? Ela será preservada, mas deixará de contar na carga horária.'
          : 'Reativar esta aula e incluí-la novamente na carga horária?',
      )
    )
      return
    alterando = true
    aviso = ''
    const controller = new AbortController()
    alteracaoRequest = controller
    try {
      const data = await jsonRequest(`${apiUrl}/aulas/${aula.id}/situacao`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ versao: aula.versao, cancelada: cancelar }),
        signal: controller.signal,
      })
      if (!isAula(data))
        throw new Error(
          'Não foi possível confirmar a alteração. Atualize o histórico antes de tentar novamente.',
        )
      if (!controller.signal.aborted)
        aviso = cancelar ? 'Aula cancelada.' : 'Aula reativada.'
    } catch (error) {
      if (!controller.signal.aborted)
        aviso =
          error instanceof Error && error.name === 'Error'
            ? error.message
            : 'Não foi possível confirmar a alteração. Confira o histórico antes de tentar novamente.'
    } finally {
      if (!controller.signal.aborted) {
        alterando = false
        tentativa++
      }
    }
  }
  onDestroy(() => alteracaoRequest?.abort())
  let aplicados = $state({
    inicial: '',
    final: '',
    turma: '',
    situacao: 'ativas',
  })
  let pagina = $state(1)
  let tentativa = $state(0)
  let carregando = $state(true)
  let erro = $state('')
  let erroFiltro = $state('')
  let resultado = $state<Historico | null>(null)
  function filtrar(event: SubmitEvent) {
    event.preventDefault()
    if (inicial && final && inicial > final) {
      erroFiltro = 'A data inicial deve ser anterior ou igual à final.'
      return
    }
    erroFiltro = ''
    pagina = 1
    aplicados = { inicial, final, turma, situacao }
  }
  function limpar() {
    inicial = ''
    final = ''
    turma = ''
    situacao = 'ativas'
    erroFiltro = ''
    pagina = 1
    aplicados = { inicial: '', final: '', turma: '', situacao: 'ativas' }
  }
  $effect(() => {
    const params = new URLSearchParams({ pagina: String(pagina) })
    if (aplicados.inicial) params.set('data_inicial', aplicados.inicial)
    if (aplicados.final) params.set('data_final', aplicados.final)
    if (aplicados.turma) params.set('turma_id', aplicados.turma)
    params.set('situacao', aplicados.situacao)
    const url = `${apiUrl}/aulas?${params}`
    void revisao
    void tentativa
    const controller = new AbortController()
    untrack(() => {
      carregando = true
      erro = ''
      resultado = null
    })
    void (async () => {
      try {
        const data = await jsonRequest(url, { signal: controller.signal })
        if (!isHistorico(data))
          throw new Error('O servidor enviou uma resposta inesperada.')
        if (!controller.signal.aborted) {
          const ultima = Math.max(1, Math.ceil(data.total_registros / 20))
          if (data.pagina > ultima) {
            pagina = ultima
            return
          }
          resultado = data
        }
      } catch (error) {
        if (!controller.signal.aborted)
          erro =
            error instanceof Error && error.name === 'Error'
              ? error.message
              : 'Não foi possível consultar as aulas. Verifique a conexão e tente novamente.'
      } finally {
        if (!controller.signal.aborted) carregando = false
      }
    })()
    return () => controller.abort()
  })
  function dataLocal(data: string) {
    return data.split('-').reverse().join('/')
  }
</script>

<section aria-labelledby="historico-title" aria-busy={carregando}>
  <h2 id="historico-title">Histórico de aulas</h2>
  <form onsubmit={filtrar}>
    <div class="form-grid">
      <div class="field">
        <label for="data-inicial">Data inicial</label><input
          id="data-inicial"
          type="date"
          min="0001-01-01"
          max="9999-12-31"
          bind:value={inicial}
        />
      </div>
      <div class="field">
        <label for="data-final">Data final</label><input
          id="data-final"
          type="date"
          min="0001-01-01"
          max="9999-12-31"
          bind:value={final}
        />
      </div>
      <div class="field full-width">
        <label for="filtro-turma">Turma ou estudante</label><select
          id="filtro-turma"
          bind:value={turma}
          ><option value="">Todas as turmas e estudantes</option
          >{#each turmas as t (t.id)}<option value={String(t.id)}
              >{t.nome}</option
            >{/each}</select
        >
      </div>
      <div class="field full-width">
        <label for="situacao">Situação</label><select
          id="situacao"
          bind:value={situacao}
          ><option value="ativas">Ativas</option><option value="canceladas"
            >Canceladas</option
          ><option value="todas">Todas</option></select
        >
      </div>
    </div>
    <p class="hint">O período inclui as datas inicial e final.</p>
    {#if erroFiltro}<p class="error" role="alert">{erroFiltro}</p>{/if}
    <div class="actions">
      <button type="submit">Filtrar</button><button
        type="button"
        onclick={limpar}>Limpar</button
      >
    </div>
  </form>
  <p class="hint">Aulas canceladas não entram na carga horária.</p>
  <p role="status">{aviso}</p>
  <div role="status" aria-live="polite">
    {#if carregando}<p>Carregando histórico…</p>
    {:else if erro}<p class="error">{erro}</p>
    {:else if resultado}<p>
        <strong
          >Carga horária: {duracao(resultado.duracao_total_minutos)}</strong
        >
        · {resultado.total_registros} aula(s) no filtro
      </p>{/if}
  </div>
  {#if erro}<button onclick={() => tentativa++}>Tentar novamente</button>{/if}
  {#if !carregando && resultado}
    {#if resultado.aulas.length === 0}<p>
        {resultado.total_registros === 0
          ? 'Nenhuma aula encontrada para os filtros aplicados.'
          : 'Não há aulas nesta página.'}
      </p>
    {:else}
      <ul class="aulas">
        {#each resultado.aulas as aula (aula.id)}
          <li>
            <h3>{dataLocal(aula.data)} · {aula.turma}</h3>
            <p>{aula.inicio}–{aula.fim} · {duracao(aula.duracao_minutos)}</p>
            <p class="conteudo">{aula.conteudo}</p>
            {#if aula.cancelada}<p><strong>Cancelada</strong></p>{/if}
            {#if aula.updated_at}<p class="hint">
                Atualizada em {new Date(aula.updated_at).toLocaleString(
                  'pt-BR',
                )}
              </p>{/if}
            {#if aula.google}
              <p>
                Google Sheets: {estadosGoogle[aula.google.estado] ??
                  'Estado desconhecido'}
              </p>
              {#if aula.google.erro}<p class="error">{aula.google.erro}</p>{/if}
              {#if aula.google.ultima_tentativa}<p class="hint">
                  Última tentativa: {new Date(
                    aula.google.ultima_tentativa,
                  ).toLocaleString('pt-BR')}
                </p>{/if}
            {/if}
            <div class="actions">
              {#if aula.google && aula.google.estado !== 'sem_destino'}
                <button
                  disabled={sincronizando || alterando || bloqueado}
                  onclick={() => sincronizar(aula)}
                >
                  {sincronizando
                    ? 'Enviando…'
                    : aula.google.estado === 'erro'
                      ? 'Tentar envio novamente'
                      : 'Sincronizar com Google'}
                </button>
              {/if}
              {#if !aula.cancelada}<button
                  disabled={alterando || bloqueado || sincronizando}
                  onclick={() => onEdit(aula)}>Editar</button
                >{/if}
              <button
                disabled={alterando || bloqueado || sincronizando}
                onclick={() => mudarSituacao(aula)}
                >{aula.cancelada ? 'Reativar aula' : 'Cancelar aula'}</button
              >
            </div>
          </li>
        {/each}
      </ul>
    {/if}
    <nav class="actions" aria-label="Páginas do histórico">
      <button disabled={pagina <= 1} onclick={() => pagina--}>Anterior</button>
      <span
        >Página {resultado.pagina} de {Math.max(
          1,
          Math.ceil(resultado.total_registros / 20),
        )}</span
      >
      <button
        disabled={pagina * 20 >= resultado.total_registros}
        onclick={() => pagina++}>Próxima</button
      >
    </nav>
  {/if}
</section>
