<script lang="ts">
  import { untrack, onDestroy, tick } from 'svelte'
  import {
    jsonRequest,
    sincronizarAula,
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
    onBusy,
  }: {
    apiUrl: string
    turmas: Turma[]
    revisao: number
    onEdit: (aula: Aula) => void
    bloqueado: boolean
    onBusy: (value: boolean) => void
  } = $props()
  let inicial = $state('')
  let final = $state('')
  let turma = $state('')
  let situacao = $state('todas')
  let alterando = $state(false)
  let aviso = $state('')
  let sincronizando = $state(false)
  let lote = $state(false)
  let excluindo = $state(false)
  const ocupado = $derived(
    alterando || sincronizando || lote || excluindo || bloqueado,
  )
  function mensagemErro(error: unknown) {
    return error instanceof Error && error.name === 'Error'
      ? error.message
      : 'Não foi possível confirmar a operação. Confira o histórico antes de tentar novamente.'
  }
  // Busca todas as páginas antes de alterar: cancelar não desloca a paginação do lote.
  async function aulasFiltradas(signal: AbortSignal): Promise<Aula[]> {
    const aulas: Aula[] = []
    let total: number | undefined
    let configVersao: number | undefined
    for (let p = 1; ; p++) {
      const params = parametros(p)
      const data = await jsonRequest(`${apiUrl}/aulas?${params}`, { signal })
      if (!isHistorico(data) || data.pagina !== p)
        throw new Error('Não foi possível conferir todas as aulas do filtro.')
      if (total !== undefined && total !== data.total_registros)
        throw new Error(
          'O histórico mudou durante a consulta. Confira os filtros e tente novamente.',
        )
      total = data.total_registros
      for (const aula of data.aulas) {
        if (aulas.some((a) => a.id === aula.id))
          throw new Error(
            'O histórico mudou durante a consulta. Tente novamente.',
          )
        if (aula.google) {
          if (
            configVersao !== undefined &&
            configVersao !== aula.google.config_versao
          )
            throw new Error(
              'A planilha de destino mudou durante a consulta. Tente novamente.',
            )
          configVersao = aula.google.config_versao
        }
        aulas.push(aula)
      }
      if (p * data.por_pagina >= total) break
    }
    if (aulas.length !== total)
      throw new Error(
        'A lista de aulas mudou. Confira o histórico e tente novamente.',
      )
    return aulas
  }
  async function executarLote(acao: 'enviar' | 'cancelar') {
    if (ocupado || carregando) return
    lote = true
    onBusy(true)
    aviso = 'Conferindo as aulas dos filtros aplicados…'
    const controller = new AbortController()
    alteracaoRequest = controller
    let concluidas = 0
    let quantidade = 0
    try {
      const todas = await aulasFiltradas(controller.signal)
      const aulas = todas.filter((a) =>
        acao === 'cancelar'
          ? !a.cancelada
          : a.google?.estado !== 'sincronizado',
      )
      quantidade = aulas.length
      if (!quantidade) {
        aviso =
          acao === 'cancelar'
            ? 'Nenhuma aula ativa para cancelar neste filtro.'
            : 'Nenhuma aula com envio pendente neste filtro.'
        return
      }
      if (
        acao === 'enviar' &&
        aulas.some((a) => !a.google || a.google.estado === 'sem_destino')
      )
        throw new Error(
          'Configure uma planilha na aba Integração Google antes de enviar as aulas.',
        )
      const pergunta =
        acao === 'enviar'
          ? `Enviar ${quantidade} aula(s) pendente(s) dos filtros aplicados para a planilha configurada? Todas as páginas serão incluídas. Correções e cancelamentos também serão enviados.`
          : `Cancelar ${quantidade} aula(s) ativa(s) dos filtros aplicados, em todas as páginas? Elas serão preservadas e deixarão de contar na carga horária. O Google será atualizado quando você sincronizar novamente.`
      if (!window.confirm(pergunta)) {
        aviso = 'Nenhuma alteração realizada.'
        return
      }
      for (const aula of aulas) {
        aviso = `${acao === 'enviar' ? 'Enviando' : 'Cancelando'} ${concluidas + 1} de ${quantidade}…`
        if (acao === 'enviar') {
          const envio = await sincronizarAula(
            apiUrl,
            aula,
            aula.google!.config_versao,
            controller.signal,
          )
          if (envio === 'adiado') {
            aviso = `${concluidas} de ${quantidade} aula(s) concluída(s). Preparação recusada; as demais continuam pendentes.`
            return
          }
        } else {
          const data = await jsonRequest(
            `${apiUrl}/aulas/${aula.id}/situacao`,
            {
              method: 'PATCH',
              headers: { 'Content-Type': 'application/json' },
              body: JSON.stringify({ versao: aula.versao, cancelada: true }),
              signal: controller.signal,
            },
          )
          if (!isAula(data) || !data.cancelada)
            throw new Error(
              'Não foi possível confirmar o cancelamento desta aula.',
            )
        }
        concluidas++
      }
      aviso = `${concluidas} aula(s) ${acao === 'enviar' ? 'sincronizada(s).' : 'cancelada(s). Para atualizar a planilha, sincronize as aulas canceladas.'}`
    } catch (error) {
      if (!controller.signal.aborted)
        aviso = `${concluidas} de ${quantidade} aula(s) concluída(s). Lote interrompido. ${mensagemErro(error)}`
    } finally {
      if (!controller.signal.aborted) {
        lote = false
        onBusy(false)
        tentativa++
      }
    }
  }
  async function sincronizar(aula: Aula) {
    if (ocupado || !aula.google) return
    if (aula.google.exclusao_pendente) {
      aviso =
        'Esta aula tem uma exclusão aguardando confirmação. Use Excluir registro para concluir.'
      return
    }
    if (aula.google.estado === 'sincronizado') {
      aviso = 'Esta aula já está na planilha. Nenhum novo registro foi enviado.'
      return
    }
    if (
      !window.confirm(
        'Enviar esta aula para a planilha configurada? O envio também atualiza correções e cancelamentos anteriores nesse destino.',
      )
    )
      return
    sincronizando = true
    onBusy(true)
    aviso = ''
    try {
      const envio = await sincronizarAula(
        apiUrl,
        aula,
        aula.google.config_versao,
      )
      aviso =
        envio === 'adiado'
          ? 'Aula preservada. Envio pendente; nenhuma aba foi preparada.'
          : envio === 'ja_sincronizada'
            ? 'Esta aula já está na planilha. Nenhum novo registro foi enviado.'
            : aula.cancelada
              ? 'Cancelamento sincronizado.'
              : 'Aula sincronizada com a planilha configurada.'
    } catch (error) {
      aviso =
        error instanceof Error && error.name === 'Error'
          ? error.message
          : 'Envio não confirmado. Tente novamente para conferir o resultado.'
    } finally {
      sincronizando = false
      onBusy(false)
      tentativa++
    }
  }
  let alteracaoRequest: AbortController | undefined
  async function mudarSituacao(aula: Aula) {
    if (ocupado) return
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
    onBusy(true)
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
        onBusy(false)
        tentativa++
      }
    }
  }
  let dialogoExclusao: HTMLDialogElement
  let aulaExcluir = $state<Aula | null>(null)
  let removerGoogle = $state(false)
  let erroExclusao = $state('')
  async function abrirExclusao(aula: Aula) {
    if (ocupado) return
    aulaExcluir = aula
    removerGoogle = aula.google?.exclusao_google ?? false
    erroExclusao = ''
    await tick()
    dialogoExclusao.showModal()
    dialogoExclusao.querySelector('button')?.focus()
  }
  async function excluir(event: SubmitEvent) {
    event.preventDefault()
    if (!aulaExcluir || ocupado) return
    excluindo = true
    onBusy(true)
    erroExclusao = ''
    const request = new AbortController()
    alteracaoRequest = request
    try {
      const resposta = await jsonRequest(`${apiUrl}/aulas/${aulaExcluir.id}`, {
        method: 'DELETE',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          versao: aulaExcluir.versao,
          config_versao: aulaExcluir.google?.config_versao ?? 0,
          remover_google: removerGoogle,
        }),
        signal: request.signal,
      })
      if (resposta !== null)
        throw new Error(
          'Não foi possível confirmar a exclusão. Confira o histórico.',
        )
      aviso = removerGoogle
        ? 'Aula excluída definitivamente do aplicativo e da planilha configurada.'
        : 'Aula excluída definitivamente do aplicativo. Os registros no Google foram preservados.'
      dialogoExclusao.close()
      aulaExcluir = null
    } catch (error) {
      if (!request.signal.aborted) erroExclusao = mensagemErro(error)
    } finally {
      if (!request.signal.aborted) {
        excluindo = false
        onBusy(false)
        tentativa++
      }
    }
  }
  onDestroy(() => alteracaoRequest?.abort())
  let aplicados = $state({
    inicial: '',
    final: '',
    turma: '',
    situacao: 'todas',
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
    situacao = 'todas'
    erroFiltro = ''
    pagina = 1
    aplicados = { inicial: '', final: '', turma: '', situacao: 'todas' }
  }
  function parametros(p: number) {
    const params = new URLSearchParams({ pagina: String(p) })
    if (aplicados.inicial) params.set('data_inicial', aplicados.inicial)
    if (aplicados.final) params.set('data_final', aplicados.final)
    if (aplicados.turma) params.set('turma_id', aplicados.turma)
    params.set('situacao', aplicados.situacao)
    return params
  }
  $effect(() => {
    const params = parametros(pagina)
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
    <fieldset disabled={ocupado}>
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
    </fieldset>
  </form>
  <p class="hint">Aulas canceladas não entram na carga horária.</p>
  <div class="actions lote-actions" aria-label="Ações em lote">
    <button
      disabled={ocupado || carregando || !resultado?.total_registros}
      onclick={() => executarLote('enviar')}>Enviar todas as pendentes</button
    >
    <button
      class="button-danger"
      disabled={ocupado || carregando || !resultado?.total_registros}
      onclick={() => executarLote('cancelar')}>Cancelar todas as aulas</button
    >
  </div>
  <p class="hint">
    As ações em lote incluem todas as páginas dos filtros aplicados.
  </p>
  <p role="status" aria-live="polite">{aviso}</p>
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
          <li
            class="aula-card"
            data-aula-id={aula.id}
            class:envio-pendente={!aula.cancelada &&
              aula.google?.estado !== 'sincronizado'}
            class:sincronizado={!aula.cancelada &&
              aula.google?.estado === 'sincronizado'}
            class:cancelada={aula.cancelada}
          >
            <h3>{dataLocal(aula.data)} · {aula.turma}</h3>
            <p>{aula.inicio}–{aula.fim} · {duracao(aula.duracao_minutos)}</p>
            <p class="conteudo">{aula.conteudo}</p>
            {#if aula.cancelada}<p><strong>Cancelada</strong></p>{/if}
            {#if aula.updated_at}<p class="hint">
                Atualizada em {new Date(aula.updated_at).toLocaleString(
                  'pt-BR',
                )}
              </p>{/if}
            <p class="envio-status">
              {aula.cancelada
                ? aula.google?.estado === 'sincronizado'
                  ? 'Cancelamento sincronizado'
                  : 'Cancelamento pendente de sincronização'
                : aula.google?.estado === 'sincronizado'
                  ? 'Sincronizado'
                  : 'Envio pendente'}
            </p>
            {#if aula.google?.exclusao_pendente}<p class="error">
                Exclusão aguardando confirmação. Use Excluir registro para
                concluir com as mesmas opções.
              </p>{/if}
            {#if aula.google?.erro}<p class="error">{aula.google.erro}</p>{/if}
            {#if aula.google?.estado === 'sem_destino'}<p class="hint">
                Configure a planilha na aba Integração Google.
              </p>{/if}
            {#if aula.google?.estado === 'aguardando_preparacao'}<p
                class="hint"
              >
                Ao sincronizar, você poderá confirmar a preparação da aba mensal
                aqui.
              </p>{/if}
            <div class="actions">
              {#if aula.google && aula.google.estado !== 'sem_destino'}
                <button disabled={ocupado} onclick={() => sincronizar(aula)}>
                  {sincronizando
                    ? 'Enviando…'
                    : aula.google.estado === 'erro'
                      ? 'Tentar envio novamente'
                      : 'Sincronizar com Google'}
                </button>
              {/if}
              {#if !aula.cancelada}<button
                  disabled={ocupado}
                  onclick={() => onEdit(aula)}>Editar</button
                >{/if}
              <button disabled={ocupado} onclick={() => mudarSituacao(aula)}
                >{aula.cancelada ? 'Reativar aula' : 'Cancelar aula'}</button
              >
              {#if aula.cancelada}<button
                  class="button-danger"
                  disabled={ocupado}
                  onclick={() => abrirExclusao(aula)}>Excluir registro</button
                >{/if}
            </div>
          </li>
        {/each}
      </ul>
    {/if}
    <nav class="actions" aria-label="Páginas do histórico">
      <button disabled={ocupado || pagina <= 1} onclick={() => pagina--}
        >Anterior</button
      >
      <span
        >Página {resultado.pagina} de {Math.max(
          1,
          Math.ceil(resultado.total_registros / 20),
        )}</span
      >
      <button
        disabled={ocupado || pagina * 20 >= resultado.total_registros}
        onclick={() => pagina++}>Próxima</button
      >
    </nav>
  {/if}
</section>

<dialog
  bind:this={dialogoExclusao}
  aria-labelledby="excluir-title"
  oncancel={(event) => {
    if (excluindo) event.preventDefault()
  }}
>
  <h2 id="excluir-title">Excluir registro definitivamente?</h2>
  {#if aulaExcluir}
    <p>{dataLocal(aulaExcluir.data)} · {aulaExcluir.turma}</p>
    <p>
      A aula desaparecerá do histórico e não poderá ser recuperada. Você poderá
      cadastrá-la novamente.
    </p>
    <form onsubmit={excluir}>
      <fieldset disabled={excluindo}>
        {#if aulaExcluir.google?.destino_id}
          <label class="checkbox-label"
            ><input
              type="checkbox"
              disabled={!!aulaExcluir.google?.exclusao_pendente}
              bind:checked={removerGoogle}
            />Excluir também o registro da planilha configurada</label
          >
          <p class="hint">
            Inclui as representações desta aula em meses anteriores dessa
            planilha. Outros destinos são preservados. Sincronize o cancelamento
            antes de excluir uma aula já enviada.
          </p>
        {:else}<p>
            Sem planilha configurada: a exclusão será somente no aplicativo.
          </p>{/if}
        {#if erroExclusao}<p class="error" role="alert">{erroExclusao}</p>{/if}
        <div class="actions">
          <button type="button" onclick={() => dialogoExclusao.close()}
            >Voltar</button
          >
          <button class="button-danger" type="submit"
            >{excluindo ? 'Excluindo…' : 'Excluir definitivamente'}</button
          >
        </div>
      </fieldset>
    </form>
  {/if}
</dialog>
