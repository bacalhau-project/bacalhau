import { useEffect, useCallback } from 'react'

export function useRefreshContent(
  pageName: string,
  refreshFunction: () => void
) {
  const handleRefresh = useCallback(
    (event: CustomEvent) => {
      if (event.detail.page === pageName) {
        refreshFunction()
      }
    },
    [pageName, refreshFunction]
  )

  useEffect(() => {
    window.addEventListener('refreshContent', handleRefresh as EventListener)

    return () => {
      window.removeEventListener(
        'refreshContent',
        handleRefresh as EventListener
      )
    }
  }, [handleRefresh])
}
