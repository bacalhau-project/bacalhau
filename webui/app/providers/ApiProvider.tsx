'use client'

import { ReactNode, createContext, useContext } from 'react'
import { useApiInitialization } from '@/lib/api'

interface ApiContextType {
  isInitialized: boolean
}

const ApiContext = createContext<ApiContextType>({
  isInitialized: false,
})

export function ApiProvider({ children }: { children: ReactNode }) {
  const isInitialized = useApiInitialization()

  return (
    <ApiContext.Provider value={{ isInitialized }}>
      {children}
    </ApiContext.Provider>
  )
}

export function useApi() {
  return useContext(ApiContext)
}
