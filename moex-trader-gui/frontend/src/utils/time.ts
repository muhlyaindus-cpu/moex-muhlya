export function pad2(v: number): string {
  return String(v).padStart(2, '0')
}

export function parseClockToMinutes(raw: string): number {
  const m = /^(\d{1,2}):(\d{2})(?::\d{2})?$/.exec((raw || '').trim())
  if (!m) return -1
  const hh = Number(m[1])
  const mm = Number(m[2])
  if (!Number.isFinite(hh) || !Number.isFinite(mm)) return -1
  return hh * 60 + mm
}

export function shiftClock(raw: string, hoursDelta: number): string {
  const m = /^(\d{1,2}):(\d{2})(?::(\d{2}))?$/.exec((raw || '').trim())
  if (!m) return '--:--:--'
  const hh = Number(m[1])
  const mm = Number(m[2])
  const ss = Number(m[3] || '0')
  if (!Number.isFinite(hh) || !Number.isFinite(mm) || !Number.isFinite(ss)) return '--:--:--'
  const daySec = 24 * 60 * 60
  const base = hh * 3600 + mm * 60 + ss
  let next = (base + hoursDelta * 3600) % daySec
  if (next < 0) next += daySec
  const outH = Math.floor(next / 3600)
  const outM = Math.floor((next % 3600) / 60)
  const outS = next % 60
  return `${pad2(outH)}:${pad2(outM)}:${pad2(outS)}`
}

export function formatMarketClock(raw: string): string {
  const nsk = (raw || '').trim()
  if (!nsk) return '--:--:--'
  const msk = shiftClock(nsk, -4)
  return `${nsk} NSK (${msk} MSK)`
}

export function formatDuration(sec: number): string {
  const s = Math.max(0, Math.floor(sec))
  const h = Math.floor(s / 3600)
  const m = Math.floor((s % 3600) / 60)
  const r = s % 60
  return `${pad2(h)}:${pad2(m)}:${pad2(r)}`
}

export function secondsToNext(minsNow: number, targetHour: number, targetMin: number): number {
  if (minsNow < 0) return 0
  const nowSec = minsNow * 60
  const targetSec = (targetHour * 60 + targetMin) * 60
  let diff = targetSec - nowSec
  if (diff < 0) diff += 24 * 60 * 60
  return diff
}

export function formatDateTimeNskMsk(ts: string): string {
  if (!ts) return '-'
  const d = new Date(ts)
  if (Number.isNaN(d.getTime())) return ts
  const nsk = d.toLocaleString('ru-RU', {
    timeZone: 'Asia/Novosibirsk',
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  })
  const msk = d.toLocaleString('ru-RU', {
    timeZone: 'Europe/Moscow',
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  })
  return `${nsk} NSK (${msk} MSK)`
}

export function formatClockNskMsk(ts?: string): string {
  if (!ts) return '-'
  const d = new Date(ts)
  if (Number.isNaN(d.getTime())) return ts
  const nsk = d.toLocaleTimeString('ru-RU', { timeZone: 'Asia/Novosibirsk' })
  const msk = d.toLocaleTimeString('ru-RU', { timeZone: 'Europe/Moscow' })
  return `${nsk} NSK (${msk} MSK)`
}
