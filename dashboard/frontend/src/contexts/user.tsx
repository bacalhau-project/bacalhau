import React, { FC, createContext, useMemo, useState, useCallback, useEffect } from 'react'
import axios from 'axios'
import useApi from '../hooks/useApi'

import {
  TokenResponse,
  User,
} from '../types'

export interface IUserContext {
  user?: User,
  login: {
    (username: string, password: string): Promise<boolean>,
  },
  logout:{
    (): Promise<void>,
  },
  initialise: {
    (): Promise<void>,
  },
}

export const getHTTPTokenHeaders = (token: string): {
  Authorization: string,
} => ({
  Authorization: token ? `Bearer ${token}` : '',
})

export const setHTTPToken = (token: string) => {
  axios.defaults.headers.common = getHTTPTokenHeaders(token)
}

export const unsetHTTPToken = () => {
  axios.defaults.headers.common = getHTTPTokenHeaders('')
}

export const UserContext = createContext<IUserContext>({
  user: undefined,
  login: async () => {
    return true
  },
  logout: async () => {},
  initialise: async () => {},
})

export const useUserContext = (): IUserContext => {
  const api = useApi()
  const [ user, setUser ] = useState<User>()

  const logout = useCallback(async () => {
    unsetHTTPToken()
    localStorage.removeItem('token')
    setUser(undefined)
  }, [])

  const loadStatus = useCallback(async () => {
    const statusResult = await api.get<User>('/api/v1/admin/status')
    if(!statusResult) {
      await logout()
    }
    else {
      setUser(statusResult)
    }
  }, [
    logout,
  ])

  const initialise = useCallback(async () => {
    const token = localStorage.getItem('token')
    if(!token) return
    setHTTPToken(token)
    await loadStatus()
  }, [
    loadStatus,
  ])

  const login = useCallback(async (username: string, password: string): Promise<boolean> => {
    try {
      const tokenResult = await api.post('/api/v1/admin/login', {
        username,
        password,
      }) as TokenResponse
      if(!tokenResult) return false
      const token = tokenResult.token
      localStorage.setItem('token', token)
      setHTTPToken(token)
      await loadStatus()
      return true
    } catch(e: any) {
      return false
    }
  }, [
    loadStatus,
  ])

  const contextValue = useMemo<IUserContext>(() => ({
    user,
    login,
    logout,
    initialise,
  }), [
    user,
    login,
    logout,
    initialise,
  ])

  return contextValue
}

export const UserContextProvider: FC = ({ children }) => {
  const value = useUserContext()
  return (
    <UserContext.Provider value={ value }>
      { children }
    </UserContext.Provider>
  )
}