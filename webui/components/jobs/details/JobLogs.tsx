import React, { useState, useEffect, useRef, useCallback } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import {
  AlertCircle,
  Play,
  RefreshCcw,
  StopCircle,
  CheckCircle,
} from 'lucide-react'
import { useApi } from '@/app/providers/ApiProvider'
import { client } from '@/lib/api/generated'

interface LogEntry {
  type: number
  content: string
}

const JobLogs = ({ jobId }: { jobId: string | undefined }) => {
  const [logs, setLogs] = useState<LogEntry[]>([])
  const [error, setError] = useState<string | null>(null)
  const [isStreaming, setIsStreaming] = useState<boolean>(false)
  const [isStreamEnded, setIsStreamEnded] = useState<boolean>(false)
  const { isInitialized } = useApi()
  const logContainerRef = useRef<HTMLDivElement>(null)
  const wsRef = useRef<WebSocket | null>(null)

  const scrollToBottom = () => {
    if (logContainerRef.current) {
      logContainerRef.current.scrollTop = logContainerRef.current.scrollHeight
    }
  }

  const connectWebSocket = useCallback(() => {
    if (!jobId || !isInitialized) return

    if (wsRef.current) {
      console.log('WebSocket connection already exists')
      return
    }

    setLogs([]) // Clear existing logs when starting a new stream
    setError(null)
    setIsStreamEnded(false)

    const baseUrl = client.getConfig().baseUrl
    if (!baseUrl) {
      console.error('Base URL is not set')
      setError(
        'Failed to connect to log stream. Client not configured properly.'
      )
      return
    }
    const wsUrl = `${baseUrl.replace(/^http/, 'ws')}/api/v1/orchestrator/jobs/${jobId}/logs?follow=true`
    console.log('Attempting to connect to:', wsUrl)

    const ws = new WebSocket(wsUrl)

    ws.onopen = () => {
      console.log('WebSocket connected')
      setIsStreaming(true)
    }

    ws.onmessage = (event) => {
      console.log('Received message:', event.data)
      try {
        const message = JSON.parse(event.data)
        if (message.value && message.value.Line) {
          setLogs((prevLogs) => [
            ...prevLogs,
            { type: message.value.Type, content: message.value.Line },
          ])
          setTimeout(scrollToBottom, 0)
        } else if (message.Err) {
          setError(`Server error: ${message.Err}`)
        }
      } catch (err) {
        console.error('Error parsing message:', err)
        setLogs((prevLogs) => [...prevLogs, { type: 1, content: event.data }])
        setTimeout(scrollToBottom, 0)
      }
    }

    ws.onerror = (event) => {
      console.error('WebSocket error:', event)
      setError('Failed to connect to log stream. Please try again.')
    }

    ws.onclose = (event) => {
      console.log('WebSocket disconnected:', event)
      setIsStreaming(false)
      setIsStreamEnded(true)
      wsRef.current = null
      if (event.code !== 1000) {
        setError(
          `Connection closed unexpectedly (${event.code}). Please try reconnecting.`
        )
      }
    }

    wsRef.current = ws
  }, [jobId, isInitialized])

  const disconnectWebSocket = useCallback(() => {
    if (wsRef.current) {
      console.log('Closing WebSocket connection')
      wsRef.current.close()
      wsRef.current = null
      setIsStreaming(false)
      setIsStreamEnded(true)
    }
  }, [])

  useEffect(() => {
    return () => {
      disconnectWebSocket()
    }
  }, [disconnectWebSocket])

  const handleStartStreaming = () => {
    if (!isStreaming) {
      connectWebSocket()
    }
  }

  const handleStopStreaming = () => {
    disconnectWebSocket()
  }

  if (!jobId || !isInitialized) {
    return null
  }

  return (
    <Card className="bg-gray-900 text-green-400 h-full">
      <CardHeader className="p-5">
        <CardTitle className="flex justify-between items-center">
          <span>
            {logs.length > 0 ? (
              <span />
            ) : isStreaming ? (
              <div className="flex items-center">
                <span>Connected. Waiting for logs...</span>
              </div>
            ) : (
              <div className="flex items-center">
                <span>Click &apos;Start Streaming&apos; to fetch logs.</span>
              </div>
            )}
          </span>
          {isStreaming ? (
            <Button
              onClick={handleStopStreaming}
              variant="destructive"
              size="sm"
            >
              <StopCircle className="mr-2 h-4 w-4" />
              Stop Streaming
            </Button>
          ) : (
            <Button
              onClick={handleStartStreaming}
              variant="ghost"
              className="bg-green-400 text-gray-900"
              size="sm"
            >
              <Play className="mr-2 h-4 w-4" />
              {logs.length > 0 ? 'Restart Streaming' : 'Start Streaming'}
            </Button>
          )}
        </CardTitle>
      </CardHeader>
      <CardContent>
        {error ? (
          <div className="flex items-center text-red-500 mb-4">
            <AlertCircle className="mr-2 h-4 w-4" />
            <span>{error}</span>
            <Button
              onClick={handleStartStreaming}
              variant="ghost"
              size="sm"
              className="ml-2"
            >
              <RefreshCcw className="mr-2 h-4 w-4" />
              Reconnect
            </Button>
          </div>
        ) : null}
        <div
          ref={logContainerRef}
          className="h-[400px] overflow-y-auto font-mono text-sm bg-gray-900 p-4 rounded"
        >
          {logs.map((log, index) => (
            <div
              key={index}
              className={`whitespace-pre-wrap break-words ${
                log.type === 2 ? 'text-red-400' : 'text-green-400'
              }`}
            >
              {log.content}
            </div>
          ))}
          {isStreamEnded && (
            <div className="flex text-yellow-400 mt-4">
              <CheckCircle className="mr-2 h-4 w-4" />
              <span>End of log stream</span>
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  )
}

export default JobLogs
