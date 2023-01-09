import { useContext } from 'react'

import {
  LoadingContext,
} from '../contexts/loading'

export const useLoading = () => {
  const loading = useContext(LoadingContext)
  return loading
}

export default useLoading