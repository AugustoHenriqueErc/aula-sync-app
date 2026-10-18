export type Turma = { id: number; nome: string }
export type Aula = {
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
export function isAula(a: unknown): a is Aula {
  return (
    object(a) &&
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
  const data: unknown = await response.json()
  if (!response.ok)
    throw new Error(
      object(data) && typeof data.error === 'string'
        ? data.error
        : 'Não foi possível concluir a solicitação.',
    )
  return data
}
