import React from 'react'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'

interface TruncatedTextWithTooltipProps {
  text?: string
  maxLength?: number
}

const TruncatedTextWithTooltip: React.FC<TruncatedTextWithTooltipProps> = ({
  text,
  maxLength = 50,
}) => {
  if (!text) return null
  const shouldTruncate = text.length > maxLength
  const displayText = shouldTruncate ? `${text.slice(0, maxLength)}...` : text

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <span>{displayText}</span>
        </TooltipTrigger>
        {shouldTruncate && (
          <TooltipContent>
            <p className="max-w-xs whitespace-normal break-words">{text}</p>
          </TooltipContent>
        )}
      </Tooltip>
    </TooltipProvider>
  )
}

export default TruncatedTextWithTooltip
