'use client'

import {
  ReactNode,
  createContext,
  useContext,
  useState,
  useEffect,
} from 'react'
import { useApiInitialization, useApiUrl } from '@/lib/api'

interface ApiContextType {
  isInitialized: boolean
  apiUrl: string | null
}

const ApiContext = createContext<ApiContextType>({
  isInitialized: false,
  apiUrl: null,
})

export function ApiProvider({ children }: { children: ReactNode }) {
  const isInitialized = useApiInitialization()
  const apiUrl = useApiUrl()

  return (
    <ApiContext.Provider value={{ isInitialized, apiUrl }}>
      {children}
    </ApiContext.Provider>
  )
}

export function useApi() {
  return useContext(ApiContext)
}
