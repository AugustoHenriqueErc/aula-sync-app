<script lang="ts">
  import { onMount } from 'svelte'
  import { jsonRequest, duracao } from './api'

  let { apiUrl }: { apiUrl: string } = $props()
  type Config = {
    versao: number
    destino: null | { id: number; planilha_id: string; url: string }
  }
  type Mes = {
    periodo: string
    aba_esperada: string
    aulas: number
    canceladas: number
    duracao_ativa_minutos: number
  }
  const objeto = (v: unknown): v is Record<string, unknown> =>
    typeof v === 'object' && v !== null
  function isConfig(v: unknown): v is Config {
    return (
      objeto(v) &&
      Number.isSafeInteger(v.versao) &&
      Number(v.versao) >= 0 &&
      (v.destino === null ||
        (objeto(v.destino) &&
          Number.isSafeInteger(v.destino.id) &&
          Number(v.destino.id) > 0 &&
          typeof v.destino.planilha_id === 'string' &&
          v.destino.url ===
            `https://docs.google.com/spreadsheets/d/${v.destino.planilha_id}/edit` &&
          /^[A-Za-z0-9_-]+$/.test(v.destino.planilha_id)))
    )
  }
  function isMes(v: unknown): v is Mes {
    return (
      objeto(v) &&
      typeof v.periodo === 'string' &&
      /^\d{4}-(0[1-9]|1[0-2])$/.test(v.periodo) &&
      typeof v.aba_esperada === 'string' &&
      ['aulas', 'canceladas', 'duracao_ativa_minutos'].every(
        (k) => Number.isSafeInteger(v[k]) && Number(v[k]) >= 0,
      )
    )
  }
  let config = $state<Config | null>(null)
  let meses = $state<Mes[]>([])
  let link = $state('')
  let carregando = $state(false)
  let salvando = $state(false)
  let erro = $state('')
  let aviso = $state('')
  let controller: AbortController | undefined

  async function carregar() {
    if (carregando || salvando) return
    controller?.abort()
    const request = new AbortController()
    controller = request
    carregando = true
    erro = ''
    aviso = ''
    try {
      const data = await jsonRequest(
        `${apiUrl}/integracoes/google/planejamento`,
        { signal: request.signal },
      )
      if (
        !objeto(data) ||
        !isConfig(data.configuracao) ||
        !Array.isArray(data.meses) ||
        !data.meses.every(isMes) ||
        data.verificacao_remota !== 'nao_realizada'
      )
        throw new Error('Resposta inesperada ao carregar a configuração.')
      if (request.signal.aborted) return
      config = data.configuracao
      meses = data.meses
      link = config.destino?.url ?? ''
    } catch (error) {
      if (!request.signal.aborted)
        erro =
          error instanceof Error && error.name === 'Error'
            ? error.message
            : 'Não foi possível carregar a configuração. Verifique o backend e tente novamente.'
    } finally {
      if (!request.signal.aborted) carregando = false
    }
  }
  async function salvar(event: SubmitEvent) {
    event.preventDefault()
    if (!config || salvando || carregando) return
    const request = new AbortController()
    controller = request
    salvando = true
    erro = ''
    aviso = ''
    try {
      const data = await jsonRequest(
        `${apiUrl}/integracoes/google/configuracao`,
        {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ url: link, versao: config.versao }),
          signal: request.signal,
        },
      )
      if (!isConfig(data))
        throw new Error(
          'A resposta não confirmou a configuração. Recarregue para conferir antes de tentar novamente.',
        )
      if (request.signal.aborted) return
      config = data
      link = config.destino?.url ?? ''
      aviso =
        'Destino salvo localmente. Nenhuma aba foi criada e nenhum registro foi enviado.'
    } catch (error) {
      if (!request.signal.aborted)
        erro =
          error instanceof Error && error.name === 'Error'
            ? error.message
            : 'Não foi possível confirmar a configuração. O link foi preservado; recarregue para conferir antes de tentar novamente.'
    } finally {
      if (!request.signal.aborted) salvando = false
    }
  }
  onMount(() => {
    void carregar()
    return () => controller?.abort()
  })
</script>

<section aria-labelledby="google-title" aria-busy={carregando || salvando}>
  <h2 id="google-title">Integração com Google Sheets</h2>
  <p>
    Configure a planilha de teste. A criação de abas e o envio ao Google ainda
    não estão disponíveis.
  </p>
  <form onsubmit={salvar}>
    <fieldset disabled={carregando || salvando || !config}>
      <label for="google-url">Link da planilha de destino</label>
      <input
        id="google-url"
        name="url"
        type="url"
        required
        bind:value={link}
        placeholder="https://docs.google.com/spreadsheets/d/ID/edit"
        aria-describedby="google-help"
      />
      <p class="hint" id="google-help">
        O link será salvo neste aplicativo. Isso não confirma existência, acesso
        ou cabeçalhos na planilha.
      </p>
      <div class="actions">
        <button type="submit"
          >{salvando ? 'Salvando destino…' : 'Salvar destino'}</button
        >
      </div>
    </fieldset>
  </form>
  <div role="status" aria-live="polite">
    {#if carregando}<p>Carregando configuração…</p>{/if}
    {#if aviso}<p class="success">{aviso}</p>{/if}
    {#if erro}<p class="error">{erro}</p>{/if}
  </div>
  <button disabled={carregando || salvando} onclick={carregar}
    >Recarregar configuração e prévia</button
  >
  <p class="hint">
    Recarregar substitui o link digitado pela configuração salva e atualiza os
    meses abaixo.
  </p>
  {#if config}
    {#if config.destino}<p class="google-destino">
        Destino atual: <a
          href={config.destino.url}
          target="_blank"
          rel="noopener noreferrer">Abrir planilha configurada</a
        >
      </p>
    {:else}<p>Nenhuma planilha configurada.</p>{/if}
    <h3>Abas esperadas por mês</h3>
    <p>
      Prévia baseada nas aulas locais. As abas e os cabeçalhos ainda não foram
      verificados no Google.
    </p>
    {#if meses.length === 0}<p>
        Cadastre aulas para visualizar os meses correspondentes.
      </p>
    {:else}<ul class="aulas">
        {#each meses as mes (mes.periodo)}<li>
            <strong>{mes.aba_esperada}</strong>
            <p>
              {mes.aulas} aula(s), incluindo {mes.canceladas} cancelada(s) · carga
              horária ativa: {duracao(mes.duracao_ativa_minutos)}
            </p>
          </li>{/each}
      </ul>{/if}
  {/if}
</section>
