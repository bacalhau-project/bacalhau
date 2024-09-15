import { ConnectionStatusIndicator } from './ConnectionStatusIndicator'
import { Toaster } from '@/components/ui/toaster'

export function ConnectionStatus() {
  return (
    <>
      <ConnectionStatusIndicator />
      <Toaster />
    </>
  )
}
