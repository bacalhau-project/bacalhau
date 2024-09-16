import { toast } from '@/hooks/use-toast'

export interface ApiError {
  Status: number
  Message: string
  RequestId?: string
  Code?: string
  Component?: string
}

export function handleApiError(error: unknown): ApiError {
  if (error instanceof Error) {
    try {
      const parsedError = JSON.parse(error.message)
      return {
        Status: parsedError.Status,
        Message: parsedError.Message || 'An unexpected error occurred',
        RequestId: parsedError.RequestId,
        Code: parsedError.Code,
        Component: parsedError.Component,
      }
    } catch (parseError) {
      // If parsing fails, it's not a JSON string
      return {
        Status: 500,
        Message: error.message,
      }
    }
  }

  // Fallback for unexpected error structures
  return {
    Status: 500,
    Message: 'An unexpected error occurred: ' + String(error),
  }
}

export function displayApiError(error: ApiError) {
  let description = error.Message
  if (error.Code) {
    description += `\nError Code: ${error.Code}`
  }
  if (error.Component) {
    description += `\nComponent: ${error.Component}`
  }
  if (error.RequestId) {
    description += `\nRequest ID: ${error.RequestId}`
  }

  toast({
    variant: 'destructive',
    title: `Error ${error.Status}`,
    description: description,
    duration: 7000,
  })
}

export function isNotFound(error: ApiError): boolean {
  return error.Status === 404 || error.Code === 'NotFound'
}

export function isBadRequest(error: ApiError): boolean {
  return error.Status === 400 || error.Code === 'BadRequest'
}

export function isInternalServerError(error: ApiError): boolean {
  return error.Status === 500
}
