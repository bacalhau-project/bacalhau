import React, { FC, createContext, useMemo, useState, useCallback } from 'react'

export type ISnackbarSeverity = 'error' | 'warning' | 'info' | 'success'

export interface ISnackbarData {
  message: string,
  severity: ISnackbarSeverity,
}

export interface ISnackbarContext {
  snackbar?: ISnackbarData,
  setSnackbar: {
    (message: string, severity?: ISnackbarSeverity): void,
  },
}

export const SnackbarContext = createContext<ISnackbarContext>({
  setSnackbar: () => {},
})

export const useSnackbarContext = (): ISnackbarContext => {
  const [ snackbar, setRawSnackbar ] = useState<ISnackbarData>()
  const setSnackbar = useCallback((message: string, severity?: ISnackbarSeverity) => {
    if(!message) {
      setRawSnackbar(undefined)
    } else {
      setRawSnackbar({
        message,
        severity: severity || 'info',
      })
    }
    
  }, [])
  const contextValue = useMemo<ISnackbarContext>(() => ({
    snackbar,
    setSnackbar,
  }), [
    snackbar,
    setSnackbar,
  ])
  return contextValue
}

export const SnackbarContextProvider: FC = ({ children }) => {
  const value = useSnackbarContext()
  return (
    <SnackbarContext.Provider value={ value }>
      { children }
    </SnackbarContext.Provider>
  )
}