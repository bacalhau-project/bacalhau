export function normalizeTimestamp(timestamp: number): number {
  // Timestamp is too small, must be in seconds, convert to milliseconds
  if (timestamp < 1e12) {
    return timestamp * 1e3
  }
  // Timestamp is in nanoseconds, convert to milliseconds
  if (timestamp > 1e15) {
    return Math.floor(timestamp / 1e6)
  }
  // Timestamp is already in milliseconds
  return timestamp
}

export function formatTime(
  timeString: string | undefined,
  includeSeconds: boolean = false
): string {
  if (!timeString) return 'N/A'
  const date = new Date(timeString)
  return formatTimestamp(date.getTime(), includeSeconds)
}

export function formatTimestamp(
  timestamp: number | undefined,
  includeSeconds: boolean = false
): string {
  if (!timestamp) return 'N/A'

  const date = new Date(normalizeTimestamp(timestamp))

  const year = date.getFullYear()
  const month = (date.getMonth() + 1).toString().padStart(2, '0')
  const day = date.getDate().toString().padStart(2, '0')
  const hours = date.getHours().toString().padStart(2, '0')
  const minutes = date.getMinutes().toString().padStart(2, '0')

  let formattedDate = `${year}-${month}-${day} ${hours}:${minutes}`

  if (includeSeconds) {
    const seconds = date.getSeconds().toString().padStart(2, '0')
    formattedDate += `:${seconds}`
  }

  return formattedDate
}

export function formatDuration(durationMs: number): string {
  const ms = durationMs % 1000
  const seconds = Math.floor(durationMs / 1000) % 60
  const minutes = Math.floor(durationMs / (1000 * 60)) % 60
  const hours = Math.floor(durationMs / (1000 * 60 * 60)) % 24
  const days = Math.floor(durationMs / (1000 * 60 * 60 * 24))

  if (days > 0) {
    return `${days}d ${hours}h`
  } else if (hours > 0) {
    return `${hours}h ${minutes}m`
  } else if (minutes > 0) {
    return `${minutes}m ${seconds}s`
  } else if (seconds > 0) {
    return `${seconds}s ${ms}ms`
  } else {
    return `${ms}ms`
  }
}
