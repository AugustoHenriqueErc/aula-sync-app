<script lang="ts">
  import { untrack } from 'svelte'
  import {
    jsonRequest,
    isHistorico,
    duracao,
    type Historico,
    type Turma,
  } from './api'
  let {
    apiUrl,
    turmas,
    revisao,
  }: { apiUrl: string; turmas: Turma[]; revisao: number } = $props()
  let inicial = $state('')
  let final = $state('')
  let turma = $state('')
  let aplicados = $state({ inicial: '', final: '', turma: '' })
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
    aplicados = { inicial, final, turma }
  }
  function limpar() {
    inicial = ''
    final = ''
    turma = ''
    erroFiltro = ''
    pagina = 1
    aplicados = { inicial: '', final: '', turma: '' }
  }
  $effect(() => {
    const params = new URLSearchParams({ pagina: String(pagina) })
    if (aplicados.inicial) params.set('data_inicial', aplicados.inicial)
    if (aplicados.final) params.set('data_final', aplicados.final)
    if (aplicados.turma) params.set('turma_id', aplicados.turma)
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
        if (!controller.signal.aborted) resultado = data
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
