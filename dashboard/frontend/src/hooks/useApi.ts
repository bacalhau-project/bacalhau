import axios, { AxiosRequestConfig } from 'axios'
import { useContext, useCallback } from 'react'

import {
  SnackbarContext,
} from '../contexts/snackbar'

const API_MOUNT = ""

export interface IApiOptions {
  snackbar?: boolean,
}

export const getTokenHeaders = (token: string) => {
  return {
    Authorization: `Bearer ${token}`,
  }
}

export const extractErrorMessage = (error: any): string => {
  if(error.response && error.response.data && (error.response.data.message || error.response.data.error)) {
    return error.response.data.message || error.response.data.error
  }
  else {
    return error.toString()
  }
}

export const useApi = () => {

  const snackbar = useContext(SnackbarContext)

  const get = useCallback(async function<ResT = any>(url: string, axiosConfig?: AxiosRequestConfig, options?: IApiOptions): Promise<ResT | null> {
    try {
      const res = await axios.get<ResT>(`${API_MOUNT}${url}`, axiosConfig)
      return res.data
    } catch (e: any) {
      console.error(e)
      if(options?.snackbar !== false) snackbar.setSnackbar(extractErrorMessage(e), 'error')
      return null
    }
  }, [])

  const post = useCallback(async function<ReqT = any, ResT = any>(url: string, data: ReqT, axiosConfig?: AxiosRequestConfig, options?: IApiOptions): Promise<ResT | null> {
    try {
      const res = await axios.post<ResT>(`${API_MOUNT}${url}`, data, axiosConfig)
      return res.data
    } catch (e: any) {
      console.error(e)
      if(options?.snackbar !== false) snackbar.setSnackbar(extractErrorMessage(e), 'error')
      return null
    }
  }, [])

  const put = useCallback(async function<ReqT = any, ResT = any>(url: string, data: ReqT, axiosConfig?: AxiosRequestConfig, options?: IApiOptions): Promise<ResT | null> {
    try {
      const res = await axios.put<ResT>(`${API_MOUNT}${url}`, data, axiosConfig)
      return res.data
    } catch (e: any) {
      console.error(e)
      if(options?.snackbar !== false) snackbar.setSnackbar(extractErrorMessage(e), 'error')
      return null
    }
  }, [])

  const del = useCallback(async function<ResT = any>(url: string, axiosConfig?: AxiosRequestConfig, options?: IApiOptions): Promise<ResT | null> {
    try {
      const res = await axios.delete<ResT>(`${API_MOUNT}${url}`, axiosConfig)
      return res.data
    } catch (e: any) {
      console.error(e)
      if(options?.snackbar !== false) snackbar.setSnackbar(extractErrorMessage(e), 'error')
      return null
    }
  }, [])

  const setToken = useCallback(function(token: string) {
    axios.defaults.headers.common = token ? getTokenHeaders(token) : {}
  }, [])

  return {
    get,
    post,
    put,
    delete: del,
    setToken,
  }
}

export default useApi