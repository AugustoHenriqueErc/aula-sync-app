export type Turma = { id: number; nome: string }
export type Conteudo = { id: number; nome: string }
export type Aula = {
  google?: {
    estado: string
    destino_id: number
    config_versao: number
    periodo: string
    tentativas: number
    exclusao_pendente?: boolean
    exclusao_google?: boolean
    ultima_tentativa?: string
    erro?: string
  }
  id: number
  turma_id: number
  data: string
  inicio: string
  fim: string
  turma: string
  conteudo: string
  versao: number
  cancelada: boolean
  updated_at: string
  intervalo_minutos: number
  observacoes: string
  duracao_minutos: number
}
export type Historico = {
  aulas: Aula[]
  pagina: number
  por_pagina: number
  total_registros: number
  duracao_total_minutos: number
}
const object = (v: unknown): v is Record<string, unknown> =>
  typeof v === 'object' && v !== null
const inteiro = (v: unknown, min = 0): v is number =>
  Number.isSafeInteger(v) && Number(v) >= min
export function isTurma(v: unknown): v is Turma {
  return object(v) && inteiro(v.id, 1) && typeof v.nome === 'string'
}
export function isConteudo(v: unknown): v is Conteudo {
  return (
    object(v) &&
    inteiro(v.id, 1) &&
    typeof v.nome === 'string' &&
    v.nome.trim() !== ''
  )
}
export function isAula(a: unknown): a is Aula {
  return (
    object(a) &&
    (a.google === undefined ||
      (object(a.google) &&
        [
          'sem_destino',
          'aguardando_preparacao',
          'pendente',
          'sincronizado',
          'erro',
        ].includes(String(a.google.estado)) &&
        inteiro(a.google.destino_id) &&
        inteiro(a.google.config_versao) &&
        inteiro(a.google.tentativas) &&
        typeof a.google.periodo === 'string' &&
        (a.google.exclusao_pendente === undefined ||
          typeof a.google.exclusao_pendente === 'boolean') &&
        (a.google.exclusao_google === undefined ||
          typeof a.google.exclusao_google === 'boolean') &&
        (a.google.erro === undefined || typeof a.google.erro === 'string') &&
        (a.google.ultima_tentativa === undefined ||
          typeof a.google.ultima_tentativa === 'string'))) &&
    inteiro(a.id, 1) &&
    inteiro(a.turma_id, 1) &&
    inteiro(a.versao, 1) &&
    inteiro(a.intervalo_minutos) &&
    inteiro(a.duracao_minutos, 1) &&
    typeof a.cancelada === 'boolean' &&
    [
      'data',
      'inicio',
      'fim',
      'turma',
      'conteudo',
      'observacoes',
      'updated_at',
    ].every((k) => typeof a[k] === 'string')
  )
}
export function isHistorico(v: unknown): v is Historico {
  return (
    object(v) &&
    inteiro(v.pagina, 1) &&
    v.por_pagina === 20 &&
    inteiro(v.total_registros) &&
    inteiro(v.duracao_total_minutos) &&
    Array.isArray(v.aulas) &&
    v.aulas.length <= 20 &&
    v.aulas.every(isAula)
  )
}
export function duracao(minutos: number) {
  return `${Math.floor(minutos / 60)}h${String(minutos % 60).padStart(2, '0')}`
}
export async function jsonRequest(
  url: string,
  options: RequestInit = {},
): Promise<unknown> {
  const signal = options.signal
    ? AbortSignal.any([options.signal, AbortSignal.timeout(10000)])
    : AbortSignal.timeout(10000)
  const response = await fetch(url, { ...options, signal, cache: 'no-store' })
  if (response.status === 401)
    window.dispatchEvent(new Event('aula-auth-required'))
  if (response.ok && response.status === 204) return null
  const data: unknown = await response.json()
  if (!response.ok)
    throw new Error(
      object(data) && typeof data.error === 'string'
        ? data.error
        : 'Não foi possível concluir a solicitação.',
    )
  return data
}

export type ResultadoEnvio = 'sincronizado' | 'ja_sincronizada' | 'adiado'
export async function sincronizarAula(
  apiUrl: string,
  aula: Aula,
  configVersao: number,
  signal?: AbortSignal,
): Promise<ResultadoEnvio> {
  if (aula.google?.estado === 'sincronizado') return 'ja_sincronizada'
  const periodo = aula.data.slice(0, 7)
  const inspecao = await jsonRequest(
    `${apiUrl}/integracoes/google/inspecionar`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ config_versao: configVersao, periodo }),
      signal,
    },
  )
  if (
    !object(inspecao) ||
    inspecao.periodo !== periodo ||
    inspecao.config_versao !== configVersao ||
    typeof inspecao.estado !== 'string' ||
    typeof inspecao.mensagem !== 'string'
  )
    throw new Error('Não foi possível conferir a aba da aula. Tente novamente.')
  if (inspecao.estado === 'ausente' || inspecao.estado === 'vazia') {
    if (
      typeof inspecao.confirmacao !== 'string' ||
      typeof inspecao.aba !== 'string' ||
      typeof inspecao.planilha_id !== 'string' ||
      !/^[A-Za-z0-9_-]+$/.test(inspecao.planilha_id)
    )
      throw new Error('Confirmação de preparação inválida. Tente novamente.')
    const pergunta =
      inspecao.estado === 'ausente'
        ? `A aba ${inspecao.aba} não existe. Deseja criá-la com os cabeçalhos e enviar esta aula?`
        : `A aba ${inspecao.aba} está vazia. Deseja criar os cabeçalhos e enviar esta aula?`
    if (
      !window.confirm(
        `${pergunta}\nPlanilha: https://docs.google.com/spreadsheets/d/${inspecao.planilha_id}/edit`,
      )
    )
      return 'adiado'
    const preparada = await jsonRequest(
      `${apiUrl}/integracoes/google/preparar`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          config_versao: configVersao,
          periodo,
          confirmacao: inspecao.confirmacao,
        }),
        signal,
      },
    )
    if (
      !object(preparada) ||
      preparada.estado !== 'pronta' ||
      preparada.periodo !== periodo ||
      preparada.config_versao !== configVersao ||
      preparada.planilha_id !== inspecao.planilha_id
    )
      throw new Error('A preparação não foi confirmada. Tente novamente.')
  } else if (inspecao.estado !== 'pronta') throw new Error(inspecao.mensagem)
  const data = await jsonRequest(`${apiUrl}/aulas/${aula.id}/sincronizar`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ versao: aula.versao, config_versao: configVersao }),
    signal,
  })
  if (!object(data) || data.estado !== 'sincronizado')
    throw new Error(
      'Envio não confirmado. Confira o histórico antes de tentar novamente.',
    )
  return data.ja_sincronizada === true ? 'ja_sincronizada' : 'sincronizado'
}
