import { FC } from 'react'

import {
  SnackbarContextProvider,
} from './snackbar'

import {
  LoadingContextProvider,
} from './loading'

import {
  RouterContextProvider,
} from './router'

import {
  UserContextProvider,
} from './user'

const AllContextProvider: FC = ({ children }) => {
  return (
    <SnackbarContextProvider>
      <LoadingContextProvider>
        <RouterContextProvider>
          <UserContextProvider>
            { children }
          </UserContextProvider>
        </RouterContextProvider>
      </LoadingContextProvider>
    </SnackbarContextProvider>
  )
}

export default AllContextProvider