<script lang="ts">
  import { onDestroy, onMount, tick, untrack } from 'svelte'

  import {
    isTurma,
    isAula,
    isConteudo,
    jsonRequest,
    type Turma,
    type Aula,
    type Conteudo,
  } from './api'
  let {
    apiUrl,
    turmas,
    turmasCarregando,
    turmaCriada,
    onSaved,
    editando = null,
    onCancel,
    onBusy,
  }: {
    apiUrl: string
    turmas: Turma[]
    turmasCarregando: boolean
    turmaCriada: (t: Turma) => void
    onSaved: (aula: Aula) => Promise<void>
    editando?: Aula | null
    onCancel: () => void
    onBusy: (value: boolean) => void
  } = $props()
  let novoNome = $state('')
  let criandoTurma = $state(false)
  let erroTurma = $state('')
  let turmaRequest: AbortController | undefined
  let conteudos = $state<Conteudo[]>([])
  let conteudosCarregando = $state(true)
  let conteudosErro = $state('')
  let novoConteudo = $state('')
  let criandoConteudo = $state(false)
  let erroConteudo = $state('')
  let cadastroConteudoIncerto = $state(false)
  let conteudosRequest: AbortController | undefined
  let conteudoRequest: AbortController | undefined
  const conteudoAntigo = $derived(
    editando && !conteudos.some((c) => c.nome === editando.conteudo)
      ? editando.conteudo
      : null,
  )

  async function carregarConteudos() {
    if (criandoConteudo || activeRequest) return
    conteudosRequest?.abort()
    const controller = new AbortController()
    conteudosRequest = controller
    conteudosCarregando = true
    conteudosErro = ''
    try {
      const data = await jsonRequest(`${apiUrl}/conteudos`, {
        signal: controller.signal,
      })
      if (
        !Array.isArray(data) ||
        !data.every(isConteudo) ||
        new Set(data.map((c) => c.id)).size !== data.length ||
        new Set(data.map((c) => c.nome)).size !== data.length
      )
        throw new Error('Resposta de conteúdos inválida.')
      if (controller.signal.aborted) return
      conteudos = data
      cadastroConteudoIncerto = false
      erroConteudo = ''
    } catch {
      if (!controller.signal.aborted)
        conteudosErro =
          'Não foi possível carregar os conteúdos. Tente atualizar a lista.'
    } finally {
      if (!controller.signal.aborted) conteudosCarregando = false
    }
  }

  async function criarConteudo() {
    if (
      criandoConteudo ||
      criandoTurma ||
      activeRequest ||
      conteudosCarregando ||
      conteudosErro ||
      cadastroConteudoIncerto ||
      !novoConteudo.trim()
    )
      return
    criandoConteudo = true
    onBusy(true)
    erroConteudo = ''
    const controller = new AbortController()
    conteudoRequest = controller
    try {
      const data = await jsonRequest(`${apiUrl}/conteudos`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ nome: novoConteudo }),
        signal: controller.signal,
      })
      if (!isConteudo(data)) throw new Error('Resposta de cadastro inválida.')
      if (controller.signal.aborted) return
      conteudos = [...conteudos.filter((c) => c.id !== data.id), data].sort(
        (a, b) => a.nome.localeCompare(b.nome),
      )
      values.conteudo = data.nome
      novoConteudo = ''
      errors.conteudo = undefined
    } catch (error) {
      if (!controller.signal.aborted) {
        cadastroConteudoIncerto = true
        const mensagem =
          error instanceof Error && error.name === 'Error'
            ? error.message
            : 'Não foi possível confirmar o cadastro do conteúdo.'
        erroConteudo = `${mensagem} Atualize os conteúdos e confira a lista antes de tentar cadastrar novamente.`
      }
    } finally {
      criandoConteudo = false
      onBusy(false)
      if (!controller.signal.aborted && !erroConteudo) {
        await tick()
        document.getElementById('conteudo')?.focus()
      }
    }
  }

  async function criarTurma() {
    if (criandoTurma || criandoConteudo || activeRequest) return
    criandoTurma = true
    onBusy(true)
    erroTurma = ''
    const controller = new AbortController()
    turmaRequest = controller
    try {
      const data = await jsonRequest(`${apiUrl}/turmas`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ nome: novoNome }),
        signal: controller.signal,
      })
      if (!isTurma(data)) throw new Error('Resposta de cadastro inválida.')
      if (controller.signal.aborted) return
      turmaCriada(data)
      values.turma = String(data.id)
      novoNome = ''
      errors.turma = undefined
    } catch (error) {
      if (!controller.signal.aborted)
        erroTurma =
          error instanceof Error && error.name === 'Error'
            ? error.message
            : 'Não foi possível confirmar o cadastro. Recarregue as turmas antes de tentar novamente.'
    } finally {
      criandoTurma = false
      onBusy(false)
    }
  }

  type Field =
    | 'data'
    | 'inicio'
    | 'fim'
    | 'turma'
    | 'conteudo'
    | 'observacoes'
    | 'intervalo_minutos'
  type FormValues = Record<Field, string>
  type SubmitState =
    | { status: 'idle' }
    | { status: 'sending' }
    | { status: 'error'; message: string }
    | { status: 'success'; duration: number }

  const emptyForm = (): FormValues => ({
    data: '',
    inicio: '',
    fim: '',
    turma: '',
    conteudo: '',
    observacoes: '',
    intervalo_minutos: '0',
  })
  let values = $state<FormValues>(emptyForm())
  let errors = $state<Partial<Record<Field, string>>>({})
  let submission = $state<SubmitState>({ status: 'idle' })
  $effect(() => {
    const aula = editando
    untrack(() => {
      values = aula
        ? {
            data: aula.data,
            inicio: aula.inicio,
            fim: aula.fim,
            turma: String(aula.turma_id),
            conteudo: aula.conteudo,
            observacoes: aula.observacoes,
            intervalo_minutos: String(aula.intervalo_minutos),
          }
        : emptyForm()
      errors = {}
      submission = { status: 'idle' }
    })
  })
  let form: HTMLFormElement
  let feedback: HTMLDivElement
  let activeRequest: AbortController | undefined

  const fields: Field[] = [
    'data',
    'inicio',
    'fim',
    'turma',
    'conteudo',
    'intervalo_minutos',
    'observacoes',
  ]

  function minutes(value: string): number | null {
    if (!/^([01]\d|2[0-3]):[0-5]\d$/.test(value)) return null
    const [hours, minutes] = value.split(':').map(Number)
    return hours * 60 + minutes
  }

  let preview = $derived.by(() => {
    const start = minutes(values.inicio)
    const end = minutes(values.fim)
    const interval =
      values.intervalo_minutos.trim() === ''
        ? NaN
        : Number(values.intervalo_minutos)
    if (
      start === null ||
      end === null ||
      end <= start ||
      !Number.isInteger(interval) ||
      interval < 0 ||
      interval >= end - start
    )
      return null
    return end - start - interval
  })

  function validate(): Partial<Record<Field, string>> {
    const result: Partial<Record<Field, string>> = {}
    const date = new Date(`${values.data}T00:00:00Z`)
    if (
      !/^\d{4}-\d{2}-\d{2}$/.test(values.data) ||
      values.data.startsWith('0000') ||
      Number.isNaN(date.getTime()) ||
      date.toISOString().slice(0, 10) !== values.data
    ) {
      result.data = 'Informe uma data válida.'
    }
    const start = minutes(values.inicio)
    const end = minutes(values.fim)
    if (start === null) result.inicio = 'Informe o horário de início.'
    if (end === null) result.fim = 'Informe o horário de término.'
    else if (start !== null && end <= start)
      result.fim = 'O término deve ser posterior ao início, no mesmo dia.'
    if (!values.turma.trim()) result.turma = 'Informe a turma ou o estudante.'
    if (
      !values.conteudo.trim() ||
      (!conteudos.some((c) => c.nome === values.conteudo) &&
        values.conteudo !== conteudoAntigo)
    )
      result.conteudo = 'Selecione um conteúdo para a aula.'
    const interval =
      values.intervalo_minutos.trim() === ''
        ? NaN
        : Number(values.intervalo_minutos)
    if (!Number.isSafeInteger(interval) || interval < 0)
      result.intervalo_minutos =
        'Informe um número inteiro de minutos, a partir de zero.'
    else if (
      start !== null &&
      end !== null &&
      end > start &&
      interval >= end - start
    )
      result.intervalo_minutos =
        'O intervalo deve ser menor que a duração da aula.'
    return result
  }

  async function focusError() {
    await tick()
    const first = fields.find((field) => errors[field])
    if (first) (form.elements.namedItem(first) as HTMLElement | null)?.focus()
    else feedback?.focus()
  }

  async function submit(event: SubmitEvent) {
    event.preventDefault()
    if (
      activeRequest ||
      submission.status === 'sending' ||
      criandoTurma ||
      criandoConteudo ||
      turmasCarregando ||
      conteudosCarregando ||
      conteudosErro
    )
      return
    errors = validate()
    if (Object.keys(errors).length > 0) {
      submission = {
        status: 'error',
        message: 'Revise os campos indicados antes de salvar.',
      }
      await focusError()
      return
    }

    const { turma, ...campos } = values
    const payload = {
      ...campos,
      ...(editando ? { versao: editando.versao } : {}),
      turma_id: Number(turma),
      intervalo_minutos: Number(values.intervalo_minutos),
    }
    const controller = new AbortController()
    activeRequest = controller
    submission = { status: 'sending' }
    onBusy(true)
    let timedOut = false
    const timeout = window.setTimeout(() => {
      timedOut = true
      controller.abort()
    }, 10000)

    try {
      const response = await fetch(
        editando ? `${apiUrl}/aulas/${editando.id}` : `${apiUrl}/aulas`,
        {
          method: editando ? 'PUT' : 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(payload),
          signal: controller.signal,
        },
      )
      const data: unknown = await response.json().catch(() => null)
      if (!response.ok) {
        if (
          response.status === 422 &&
          typeof data === 'object' &&
          data !== null &&
          'field' in data &&
          'error' in data &&
          typeof data.field === 'string' &&
          (fields.includes(data.field as Field) || data.field === 'turma_id') &&
          typeof data.error === 'string'
        ) {
          errors = {
            [data.field === 'turma_id' ? 'turma' : data.field]: data.error,
          }
          submission = {
            status: 'error',
            message: 'Revise o campo indicado antes de salvar.',
          }
        } else {
          submission = {
            status: 'error',
            message:
              response.status === 409
                ? 'Esta aula mudou desde que foi aberta. Seus campos foram preservados. Feche a edição e abra a aula novamente no histórico para conferir a versão atual.'
                : response.status === 413
                  ? 'O conteúdo enviado é muito grande. Reduza os textos e tente novamente.'
                  : 'Não foi possível salvar a aula. Os campos foram preservados. Verifique a conexão e tente novamente.',
          }
        }
        return
      }
      if (response.status !== (editando ? 200 : 201) || !isAula(data)) {
        submission = {
          status: 'error',
          message:
            'A resposta não confirmou o cadastro. Os campos foram preservados; confira se a aula foi salva antes de reenviar.',
        }
        return
      }
      submission = { status: 'success', duration: Number(data.duracao_minutos) }
      values = emptyForm()
      window.clearTimeout(timeout)
      await onSaved(data)
    } catch {
      if (controller.signal.aborted && !timedOut) return
      submission = {
        status: 'error',
        message: timedOut
          ? 'O envio demorou mais de 10 segundos. Os campos foram preservados; confira se a aula foi salva antes de reenviar.'
          : 'A conexão foi interrompida e não foi possível confirmar o cadastro. Os campos foram preservados; confira se a aula foi salva antes de reenviar.',
      }
    } finally {
      window.clearTimeout(timeout)
      activeRequest = undefined
      onBusy(false)
      if (!controller.signal.aborted || timedOut) {
        if (submission.status === 'error') await focusError()
        else {
          await tick()
          feedback?.focus()
        }
      }
    }
  }

  onMount(() => {
    void carregarConteudos()
  })
  onDestroy(() => {
    activeRequest?.abort()
    turmaRequest?.abort()
    conteudosRequest?.abort()
    conteudoRequest?.abort()
  })
</script>

<section aria-labelledby="form-title">
  <h2 id="form-title">{editando ? 'Editar aula' : 'Dados da aula'}</h2>
  <p id="form-help">
    Preencha os campos obrigatórios. Observações são opcionais.
  </p>
  <form
    bind:this={form}
    onsubmit={submit}
    novalidate
    aria-describedby="form-help"
    aria-busy={submission.status === 'sending' ||
      criandoTurma ||
      criandoConteudo}
  >
    <fieldset
      disabled={submission.status === 'sending' ||
        criandoTurma ||
        criandoConteudo}
    >
      <legend class="sr-only">Dados para registrar uma aula</legend>
      <div class="form-grid">
        <div class="field full-width">
          <label for="data">Data da aula</label>
          <input
            id="data"
            name="data"
            type="date"
            min="0001-01-01"
            max="9999-12-31"
            required
            bind:value={values.data}
            aria-invalid={!!errors.data}
            aria-describedby={errors.data ? 'data-error' : undefined}
          />
          {#if errors.data}<p class="field-error" id="data-error">
              {errors.data}
            </p>{/if}
        </div>
        <div class="field">
          <label for="inicio">Início</label>
          <input
            id="inicio"
            name="inicio"
            type="time"
            required
            step="60"
            bind:value={values.inicio}
            aria-invalid={!!errors.inicio}
            aria-describedby={errors.inicio ? 'inicio-error' : undefined}
          />
          {#if errors.inicio}<p class="field-error" id="inicio-error">
              {errors.inicio}
            </p>{/if}
        </div>
        <div class="field">
          <label for="fim">Término</label>
          <input
            id="fim"
            name="fim"
            type="time"
            required
            step="60"
            bind:value={values.fim}
            aria-invalid={!!errors.fim}
            aria-describedby={errors.fim ? 'fim-error' : undefined}
          />
          {#if errors.fim}<p class="field-error" id="fim-error">
              {errors.fim}
            </p>{/if}
        </div>
        <div class="field full-width">
          <label for="intervalo_minutos">Intervalo em minutos</label>
          <input
            id="intervalo_minutos"
            name="intervalo_minutos"
            type="number"
            min="0"
            step="1"
            required
            value={values.intervalo_minutos}
            oninput={(event) =>
              (values.intervalo_minutos = event.currentTarget.value)}
            aria-invalid={!!errors.intervalo_minutos}
            aria-describedby="interval-help interval-error"
          />
          <p class="hint" id="interval-help">Use 0 se não houve intervalo.</p>
          <p class="field-error" id="interval-error">
            {errors.intervalo_minutos ?? ''}
          </p>
        </div>
        <div class="field full-width">
          <label for="turma">Turma ou estudante</label>
          <select
            id="turma"
            name="turma"
            required
            bind:value={values.turma}
            disabled={turmasCarregando || criandoTurma}
            aria-invalid={!!errors.turma}
            aria-describedby={errors.turma ? 'turma-error' : undefined}
          >
            <option value="">Selecione uma turma ou estudante</option>
            {#each turmas as turma (turma.id)}<option value={String(turma.id)}
                >{turma.nome}</option
              >{/each}
          </select>
          <details>
            <summary>Criar turma ou estudante</summary>
            <label for="novo-nome">Nome</label>
            <input
              id="novo-nome"
              bind:value={novoNome}
              disabled={criandoTurma}
            />
            <button
              type="button"
              onclick={criarTurma}
              disabled={criandoTurma || !novoNome.trim()}
              >{criandoTurma ? 'Criando…' : 'Criar e selecionar'}</button
            >
            {#if erroTurma}<p class="error" role="alert">{erroTurma}</p>{/if}
          </details>
          {#if errors.turma}<p class="field-error" id="turma-error">
              {errors.turma}
            </p>{/if}
        </div>
        <div class="field full-width">
          <label for="conteudo">Conteúdo da aula</label>
          <select
            id="conteudo"
            name="conteudo"
            required
            bind:value={values.conteudo}
            disabled={conteudosCarregando || !!conteudosErro}
            aria-invalid={!!errors.conteudo}
            aria-describedby={`conteudo-help${errors.conteudo ? ' conteudo-error' : ''}${conteudosErro ? ' conteudos-error' : ''}`}
          >
            <option value="">Selecione um conteúdo</option>
            {#if conteudoAntigo !== null}<option value={conteudoAntigo}
                >{conteudoAntigo} (conteúdo original da aula)</option
              >{/if}
            {#each conteudos as conteudo (conteudo.id)}<option
                value={conteudo.nome}>{conteudo.nome}</option
              >{/each}
          </select>
          <p class="hint" id="conteudo-help">
            Selecione um conteúdo cadastrado ou cadastre um novo abaixo.
          </p>
          {#if values.conteudo}<p class="hint conteudo">
              {values.conteudo}
            </p>{/if}
          {#if conteudosCarregando}<p role="status">Carregando conteúdos…</p>
          {:else if conteudosErro}<p
              class="error"
              id="conteudos-error"
              role="alert"
            >
              {conteudosErro}
            </p>
          {:else if conteudos.length === 0}<p role="status">
              Nenhum conteúdo cadastrado. Cadastre o primeiro abaixo.
            </p>{/if}
          <button
            type="button"
            onclick={carregarConteudos}
            disabled={conteudosCarregando}>Atualizar conteúdos</button
          >
          <details id="cadastro-conteudo">
            <summary>Cadastrar conteúdo</summary>
            <label for="novo-conteudo">Nome do conteúdo</label>
            <textarea
              id="novo-conteudo"
              rows="3"
              bind:value={novoConteudo}
              aria-describedby={erroConteudo
                ? 'cadastro-conteudo-error'
                : undefined}></textarea>
            <button
              type="button"
              onclick={criarConteudo}
              disabled={conteudosCarregando ||
                !!conteudosErro ||
                cadastroConteudoIncerto ||
                !novoConteudo.trim()}
              >{criandoConteudo ? 'Criando…' : 'Criar e selecionar'}</button
            >
            {#if erroConteudo}<p
                class="error"
                id="cadastro-conteudo-error"
                role="alert"
              >
                {erroConteudo}
              </p>{/if}
          </details>
          {#if errors.conteudo}<p class="field-error" id="conteudo-error">
              {errors.conteudo}
            </p>{/if}
        </div>
        <div class="field full-width">
          <label for="observacoes">Observações (opcional)</label>
          <textarea
            id="observacoes"
            name="observacoes"
            rows="2"
            bind:value={values.observacoes}
            aria-invalid={!!errors.observacoes}
            aria-describedby={errors.observacoes
              ? 'observacoes-error'
              : undefined}></textarea>
          {#if errors.observacoes}<p class="field-error" id="observacoes-error">
              {errors.observacoes}
            </p>{/if}
        </div>
      </div>
      <div class="duration-preview" aria-live="polite" aria-atomic="true">
        <strong
          >Prévia de duração: {preview === null
            ? '—'
            : `${preview} minutos`}</strong
        >
        <p class="hint">
          {preview === null
            ? 'Informe horários e intervalo válidos para conferir a prévia.'
            : 'A duração será confirmada ao salvar a aula.'}
        </p>
      </div>
      <button
        type="submit"
        disabled={criandoTurma ||
          criandoConteudo ||
          turmasCarregando ||
          conteudosCarregando ||
          !!conteudosErro}
        >{submission.status === 'sending'
          ? 'Salvando…'
          : editando
            ? 'Salvar alterações'
            : 'Salvar aula'}</button
      >
      {#if editando}<button type="button" onclick={onCancel}
          >Fechar edição</button
        >{/if}
    </fieldset>
    <div
      bind:this={feedback}
      class="form-feedback"
      tabindex="-1"
      role="status"
      aria-live="polite"
      aria-atomic="true"
    >
      {#if submission.status === 'sending'}
        <p>Salvando aula…</p>
      {:else if submission.status === 'success'}
        <p class="success">
          Aula salva com sucesso. Duração registrada: {submission.duration} minutos.
        </p>
        <p>Você já pode preencher uma nova aula.</p>
      {:else if submission.status === 'error'}
        <p class="error">{submission.message}</p>
      {/if}
    </div>
  </form>
</section>
