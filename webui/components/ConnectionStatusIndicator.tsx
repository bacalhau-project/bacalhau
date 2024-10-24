'use client'

import React, { useEffect, useState } from 'react'
import { Wifi, WifiOff } from 'lucide-react'
import { useToast } from '@/hooks/use-toast'
import { useConnectionMonitor } from '@/hooks/useConnectionMonitor'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'

export function ConnectionStatusIndicator() {
  const { isOnline } = useConnectionMonitor()
  const { toast } = useToast()
  const [prevOnlineState, setPrevOnlineState] = useState<boolean | undefined>(
    undefined
  )

  useEffect(() => {
    if (isOnline === undefined) return

    if (!isOnline && prevOnlineState !== false) {
      // Connection lost
      toast({
        variant: 'destructive',
        title: 'Connection Lost',
        description: `You are currently offline. Please check your connection and that Bacalhau is still running.`,
        duration: Infinity,
      })
    } else if (isOnline && prevOnlineState === false) {
      // Reconnected
      toast({
        className: 'group border-green-500 bg-green-500 text-white',
        title: 'Connected',
        description: 'Your connection has been re-established.',
        duration: 3000,
        style: { color: 'white' },
      })
    }

    setPrevOnlineState(isOnline)
  }, [isOnline, prevOnlineState, toast])

  const getIconColor = () => {
    if (isOnline === undefined) return 'text-gray-500'
    return isOnline ? 'text-green-500' : 'text-red-500'
  }

  const tooltipContent =
    isOnline === undefined
      ? 'Checking connection...'
      : `${isOnline ? 'Connected successfully' : 'Failed to connect'}`

  return (
    <>
      <style jsx>{`
        @keyframes pulse {
          0%,
          100% {
            opacity: 1;
          }
          50% {
            opacity: 0.5;
          }
        }
        .animate-pulse {
          animation: pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite;
        }
      `}</style>
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>
            <div className="fixed bottom-4 right-4 p-2 rounded-full bg-background shadow-lg">
              {isOnline === undefined ? (
                <div className="animate-pulse">
                  <Wifi className="h-6 w-6 text-gray-500" />
                </div>
              ) : isOnline ? (
                <Wifi className={`h-6 w-6 ${getIconColor()}`} />
              ) : (
                <WifiOff className="h-6 w-6 text-red-500" />
              )}
            </div>
          </TooltipTrigger>
          <TooltipContent>
            <p>{tooltipContent}</p>
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
    </>
  )
}
