import { client } from './generated'
import { useState, useEffect } from 'react'

interface Config {
  APIEndpoint: string
}

const DEFAULT_API_URL = 'http://localhost:1234'

async function fetchConfig(): Promise<Config | null> {
  try {
    const response = await fetch('/_config')
    if (!response.ok) {
      throw new Error(`Failed to fetch config: ${response.statusText}`)
    }
    return await response.json()
  } catch (error) {
    console.warn('Config fetch failed, assuming standalone mode:', error)
    return null
  }
}

let apiUrl: string | null = null

export async function initializeApi(): Promise<string> {
  const config = await fetchConfig()
  apiUrl = config?.APIEndpoint || DEFAULT_API_URL

  client.setConfig({ baseUrl: apiUrl })

  console.log('API initialized with URL:', apiUrl)
  return apiUrl
}

export function useApiInitialization() {
  const [isInitialized, setIsInitialized] = useState(false)

  useEffect(() => {
    initializeApi().then(() => setIsInitialized(true))
  }, [])

  return isInitialized
}

export function useApiUrl() {
  return apiUrl
}
