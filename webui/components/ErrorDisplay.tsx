import React from 'react'
import { AlertCircle } from 'lucide-react'
import { Alert, AlertTitle, AlertDescription } from '@/components/ui/alert'
import { ApiError } from '@/lib/api/errors'

interface ErrorDisplayProps {
  error: ApiError
}

export const ErrorDisplay: React.FC<ErrorDisplayProps> = ({ error }) => (
  <Alert variant="destructive" className="mb-4">
    <AlertCircle className="h-4 w-4" />
    <AlertTitle className="text-lg font-semibold mb-2">
      Error {error.Status}
    </AlertTitle>
    <AlertDescription>
      <div className="text-sm space-y-2">
        <p className="font-medium">{error.Message}</p>
        {error.Code && (
          <p>
            <span className="font-medium">Error Code:</span> {error.Code}
          </p>
        )}
        {error.Component && (
          <p>
            <span className="font-medium">Component:</span> {error.Component}
          </p>
        )}
        {error.RequestId && (
          <p>
            <span className="font-medium">Request ID:</span> {error.RequestId}
          </p>
        )}
      </div>
    </AlertDescription>
  </Alert>
)
