import { OpenAPI } from './generated'
import { useState, useEffect } from 'react'

// This function will be used to initialize the API
export function initializeApi(apiUrl?: string) {
  OpenAPI.BASE =
    apiUrl ||
    process.env.NEXT_PUBLIC_BACALHAU_API_ADDRESS ||
    'http://localhost:1234'
  // log all env
  console.log('API initialized with URL:', OpenAPI.BASE)
}

export { OpenAPI }

export function useApiInitialization() {
  const [isInitialized, setIsInitialized] = useState(false)

  useEffect(() => {
    initializeApi()
    setIsInitialized(true)
  }, [])

  return isInitialized
}
