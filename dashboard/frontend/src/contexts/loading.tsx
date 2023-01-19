import React, { FC, createContext, useMemo, useState } from 'react'

export interface ILoadingContext {
  loading: boolean,
  setLoading: {
    (val: boolean): void,
  },
}

export const LoadingContext = createContext<ILoadingContext>({
  loading: false,
  setLoading: () => {},
})

export const useLoadingContext = (): ILoadingContext => {
  const [ loading, setLoading ] = useState(false)
  const contextValue = useMemo<ILoadingContext>(() => ({
    loading,
    setLoading,
  }), [
    loading,
    setLoading,
  ])
  return contextValue
}

export const LoadingContextProvider: FC = ({ children }) => {
  const value = useLoadingContext()
  return (
    <LoadingContext.Provider value={ value }>
      { children }
    </LoadingContext.Provider>
  )
}