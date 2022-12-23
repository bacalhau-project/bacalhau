import React, { FC, createContext } from 'react'
import useRoute from '../hooks/useRoute'
import {
  IRouteObject,
} from '../routes'

export const RouterContext = createContext<IRouteObject>({
  id: '',
  title: '',
  render: () => <></>,
  params: {},
})

export const useRouterContext = (): IRouteObject => {
  const route = useRoute()
  return route
}

export const RouterContextProvider: FC = ({ children }) => {
  const value = useRouterContext()
  return (
    <RouterContext.Provider value={ value }>
      { children }
    </RouterContext.Provider>
  )
}