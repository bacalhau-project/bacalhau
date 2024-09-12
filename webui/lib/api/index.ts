import { OpenAPI } from './generated'
import { useState, useEffect } from 'react'

interface Config {
  APIEndpoint: string
}

export async function initializeApi() {
  try {
    const response = await fetch('/_config')
    if (!response.ok) {
      throw new Error('Failed to fetch config')
    }
    const config: Config = await response.json()
    OpenAPI.BASE = config.APIEndpoint || 'http://localhost:1234'
    console.log('API initialized with URL:', OpenAPI.BASE)
  } catch (error) {
    console.error('Error initializing API:', error)
    OpenAPI.BASE = 'http://localhost:1234' // Fallback to default
  }
}

export { OpenAPI }

export function useApiInitialization() {
  const [isInitialized, setIsInitialized] = useState(false)

  useEffect(() => {
    initializeApi().then(() => setIsInitialized(true))
  }, [])

  return isInitialized
}
