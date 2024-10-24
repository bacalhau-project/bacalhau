import { client } from './generated'
import { useState, useEffect } from 'react'

export function initializeApi(): void {
  client.setConfig({ baseUrl: "" })
}

export function useApiInitialization() {
  const [isInitialized, setIsInitialized] = useState(false)

  useEffect(() => {
    initializeApi()
    setIsInitialized(true)
  }, [])

  return isInitialized
}

