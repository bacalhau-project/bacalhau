import { useState, useEffect, useCallback } from 'react'
import bluebird from 'bluebird'
import useApi from './useApi'
import { logger } from '../utils/debug'

export function useApiData<DataType = any>({
  url,
  defaultValue,
  active = true,
  reload = false,
  reloadInterval = 5000,
  jsonStringifyComparison = true,
  onChange = () => {},
}: {
  url: string,
  defaultValue: DataType,
  active?: boolean,
  reload?: boolean,
  reloadInterval?: number,
  // if this is set to false - we use JSON.stringify comparison
  // and don't update the state if equal
  jsonStringifyComparison?: boolean,
  onChange?: {
    (data: DataType): void,
  }
}): [DataType, {
  (): Promise<void>,
}] {

  const api = useApi()
  const [data, setData] = useState<DataType>(defaultValue)

  const fetchData = useCallback(async () => {
    const apiData = await api.get<DataType>(url)
    if (apiData === null) return
    // only update the state if the data is different
    // this prevents re-renders whilst loading data in a loop
    setData(currentValue => {
      if(!jsonStringifyComparison) return apiData
      const hasChanged = JSON.stringify(apiData) != JSON.stringify(currentValue)
      if(hasChanged && onChange) onChange(apiData)
      return hasChanged ?
        apiData :
        currentValue
    })
  }, [
    url,
  ])

  useEffect(() => {
    if(!active) return
    if(!reload) {
      fetchData()
      return  
    }
    let loading = true
    const doLoop = async () => {
      while(loading) {
        await fetchData()
        await bluebird.delay(reloadInterval)
      }
    }
    doLoop()
    return () => {
      loading = false
    }
  }, [
    active,
    url,
  ])

  useEffect(() => {
    logger('useApiData', url, data)
  }, [
    data,
  ])

  return [data, fetchData]
}

export default useApiData