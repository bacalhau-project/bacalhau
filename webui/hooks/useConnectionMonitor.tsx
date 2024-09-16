'use client'

import { useState, useEffect, useCallback } from 'react'
import { useApi } from '@/app/providers/ApiProvider'
import { Ops } from '@/lib/api/generated'

export const useConnectionMonitor = (checkInterval = 5000) => {
  const { isInitialized, apiUrl } = useApi()
  const [isOnline, setIsOnline] = useState<boolean | undefined>(undefined)
  const [error, setError] = useState<string | null>(null)

  const checkConnection = useCallback(async () => {
    if (!isInitialized) {
      setIsOnline(undefined)
      setError('API not initialized')
      return
    }
    try {
      const response = await Ops.agentAlive<true>({ throwOnError: true })
      if (response.data.Status === 'OK') {
        setIsOnline(true)
        setError(null)
      } else {
        throw new Error('Unexpected response from agent')
      }
    } catch (err) {
      setIsOnline(false)
      setError(err instanceof Error ? err.message : 'An unknown error occurred')
    }
  }, [isInitialized])

  useEffect(() => {
    checkConnection() // Immediate check on mount
    const intervalId = setInterval(checkConnection, checkInterval)
    return () => clearInterval(intervalId)
  }, [checkConnection, checkInterval])

  return { isOnline, checkConnection, clientUrl: apiUrl, error }
}
