import useLoading from './useLoading'
import useSnackbar from './useSnackbar'

type asyncFunction = {
  (): Promise<void>,
}

type asyncFunctionBoolan = {
  (): Promise<boolean>,
}

export const useLoadingErrorHandler = ({
  withSnackbar = true,
  withLoading = true,
}: {
  withSnackbar?: boolean,
  withLoading?: boolean,
} = {}) => {
  const loading = useLoading()
  const snackbar = useSnackbar()
  return (handler: asyncFunction): asyncFunctionBoolan => {
    return async (): Promise<boolean> => {
      let sawError = false
      if(withLoading) loading.setLoading(true)
      try {
        await handler()
      } catch(e: any) {
        sawError = true
        // if(e.response) console.error(e.response.body)
        // if(withSnackbar) snackbar.error(e.toString())
      }
      if(withLoading) loading.setLoading(false)
      return sawError
    }
  }
}

export default useLoadingErrorHandler