import { useState, useCallback } from 'react'
import { ApiError, handleApiError } from '@/lib/api/errors'
import { useApi } from '@/app/providers/ApiProvider'

export function useApiOperation<T>() {
  const [data, setData] = useState<T | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<ApiError | null>(null)
  const { isInitialized } = useApi()

  const execute = useCallback(
    async (apiCall: () => Promise<T>) => {
      if (!isInitialized) return

      setIsLoading(true)
      setError(null)
      try {
        const result = await apiCall()
        setData(result)
        return result
      } catch (error) {
        const apiError = handleApiError(error)
        setError(apiError)
      } finally {
        setIsLoading(false)
      }
    },
    [isInitialized]
  )

  return { data, isLoading, error, execute }
}
