'use client'

import { useState, useEffect, useCallback } from 'react'
import { useApi } from '@/app/providers/ApiProvider'
import { Ops } from '@/lib/api/generated'

export const useConnectionMonitor = (checkInterval = 5000) => {
  const { isInitialized } = useApi()
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
    const initialCheckId = window.setTimeout(checkConnection, 0)
    const intervalId = setInterval(checkConnection, checkInterval)
    return () => {
      window.clearTimeout(initialCheckId)
      clearInterval(intervalId)
    }
  }, [checkConnection, checkInterval])

  return { isOnline, checkConnection, error }
}
