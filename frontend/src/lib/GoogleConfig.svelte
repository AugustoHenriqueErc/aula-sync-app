<script lang="ts">
  import { onMount } from 'svelte'
  import { jsonRequest, duracao } from './api'

  let { apiUrl, onChange }: { apiUrl: string; onChange: () => void } = $props()
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
  let verificando = $state(false)
  let acesso = $state('')
  type Inspecao = {
    periodo: string
    aba: string
    estado: string
    mensagem: string
    confirmacao?: string
    config_versao: number
    planilha_id: string
  }
  let inspecoes = $state<Record<string, Inspecao>>({})
  async function operarMes(periodo: string, preparar = false) {
    if (!config?.destino || verificando || carregando || salvando) return
    const anterior = inspecoes[periodo]
    if (
      preparar &&
      (!anterior?.confirmacao ||
        !window.confirm(
          `Preparar a aba ${anterior.aba} na planilha ${config.destino.url}? Serão criados a aba ausente ou os cabeçalhos da aba vazia.`,
        ))
    )
      return
    verificando = true
    erro = ''
    try {
      const data = await jsonRequest(
        `${apiUrl}/integracoes/google/${preparar ? 'preparar' : 'inspecionar'}`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            periodo,
            config_versao: config.versao,
            ...(preparar ? { confirmacao: anterior.confirmacao } : {}),
          }),
        },
      )
      if (
        !objeto(data) ||
        data.periodo !== periodo ||
        data.config_versao !== config.versao ||
        data.planilha_id !== config.destino.planilha_id ||
        typeof data.aba !== 'string' ||
        typeof data.estado !== 'string' ||
        typeof data.mensagem !== 'string' ||
        (data.confirmacao !== undefined && typeof data.confirmacao !== 'string')
      )
        throw new Error('Resposta inesperada. Inspecione novamente.')
      inspecoes[periodo] = data as Inspecao
      onChange()
    } catch (error) {
      delete inspecoes[periodo]
      erro =
        error instanceof Error && error.name === 'Error'
          ? error.message
          : 'Não foi possível confirmar a operação. Inspecione novamente para conferir o resultado.'
    } finally {
      verificando = false
    }
  }
  let controller: AbortController | undefined

  async function carregar() {
    if (carregando || salvando || verificando) return
    controller?.abort()
    const request = new AbortController()
    controller = request
    carregando = true
    erro = ''
    aviso = ''
    acesso = ''
    inspecoes = {}
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
    if (!config || salvando || carregando || verificando) return
    const request = new AbortController()
    controller = request
    salvando = true
    erro = ''
    aviso = ''
    acesso = ''
    inspecoes = {}
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
      onChange()
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
  async function verificarAcesso() {
    if (!config?.destino || verificando) return
    verificando = true
    acesso = ''
    erro = ''
    try {
      const data = await jsonRequest(`${apiUrl}/integracoes/google/acesso`)
      if (
        !objeto(data) ||
        typeof data.titulo !== 'string' ||
        data.versao !== config.versao ||
        data.planilha_id !== config.destino.planilha_id ||
        data.leitura_confirmada !== true ||
        data.escrita_verificada !== false
      )
        throw new Error(
          'O destino mudou ou a resposta é inválida. Verifique novamente.',
        )
      acesso = `Leitura confirmada: ${data.titulo}. A permissão de escrita e os cabeçalhos ainda não foram verificados.`
    } catch (error) {
      erro =
        error instanceof Error
          ? error.message
          : 'Não foi possível consultar a planilha.'
    } finally {
      verificando = false
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
    Configure a planilha de teste e inspecione os meses. Após preparar as abas,
    envie cada aula pelo histórico. O cadastro local continua independente do
    Google.
  </p>
  <form onsubmit={salvar}>
    <fieldset disabled={carregando || salvando || verificando || !config}>
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
        Informe uma planilha própria ou compartilhada com permissão de edição
        para a conta conectada. Salvar o link não valida permissões.
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
  <button disabled={carregando || salvando || verificando} onclick={carregar}
    >Recarregar configuração e prévia</button
  >
  <p class="hint">
    Recarregar substitui o link digitado pela configuração salva e atualiza os
    meses abaixo.
  </p>
  <button
    disabled={!config?.destino || verificando || carregando || salvando}
    onclick={verificarAcesso}
  >
    {verificando ? 'Verificando acesso…' : 'Verificar leitura com minha conta'}
  </button>
  {#if acesso}<p role="status">{acesso}</p>{/if}
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
            <button
              disabled={!config.destino ||
                verificando ||
                salvando ||
                carregando}
              onclick={() => operarMes(mes.periodo)}
              >Inspecionar aba mensal</button
            >
            {#if inspecoes[mes.periodo]}
              <p role="status">{inspecoes[mes.periodo].mensagem}</p>
              {#if inspecoes[mes.periodo].confirmacao}
                <button
                  disabled={verificando || salvando || carregando}
                  onclick={() => operarMes(mes.periodo, true)}
                  >Confirmar preparação de {mes.aba_esperada}</button
                >
                <button
                  disabled={verificando}
                  onclick={() => {
                    delete inspecoes[mes.periodo]
                  }}>Agora não</button
                >
              {/if}
            {/if}
          </li>{/each}
      </ul>{/if}
  {/if}
</section>
