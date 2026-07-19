import { client } from './generated/client.gen'

export function initializeApi(): void {
  client.setConfig({ baseUrl: '' })
}

initializeApi()

export function useApiInitialization() {
  return true
}
