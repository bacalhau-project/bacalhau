import React, { useState } from 'react'
import { Button } from '@/components/ui/button'
import { StopCircle } from 'lucide-react'
import { isTerminalJobState } from '@/lib/api/utils'
import { Orchestrator, models_Job } from '@/lib/api/generated'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog'

const JobActions = ({
  job,
  onJobUpdated,
}: {
  job: models_Job
  onJobUpdated: () => void
}) => {
  const [isStoppingJob, setIsStoppingJob] = useState(false)
  const [stopJobError, setStopJobError] = useState<string | null>(null)

  if (job.ID === undefined) {
    return (
      <Alert variant="destructive">
        <AlertTitle>Error</AlertTitle>
        <AlertDescription>
          Job ID is undefined. Cannot perform actions.
        </AlertDescription>
      </Alert>
    )
  }

  const handleStopJob = async () => {
    setIsStoppingJob(true)
    setStopJobError(null)
    try {
      await Orchestrator.stopJob({
        path: { id: job.ID! },
        query: { reason: 'User requested stop' },
        throwOnError: true,
      })
      onJobUpdated() // Trigger re-render of parent component
    } catch (error) {
      console.error('Error stopping job:', error)
      setStopJobError('Failed to stop the job. Please try again.')
    } finally {
      setIsStoppingJob(false)
    }
  }

  return (
    <div className="space-y-4">
      {stopJobError && (
        <Alert variant="destructive">
          <AlertTitle>Error</AlertTitle>
          <AlertDescription>{stopJobError}</AlertDescription>
        </Alert>
      )}
      <div className="flex space-x-2">
        {!isTerminalJobState(job.State?.StateType) && (
          <AlertDialog>
            <AlertDialogTrigger asChild>
              <Button variant="destructive">
                <StopCircle className="mr-2 h-4 w-4" />
                Stop Job
              </Button>
            </AlertDialogTrigger>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>
                  Are you sure you want to stop this job?
                </AlertDialogTitle>
                <AlertDialogDescription>
                  This action cannot be undone. The job will be stopped and
                  cannot be resumed.
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>Cancel</AlertDialogCancel>
                <AlertDialogAction
                  onClick={handleStopJob}
                  disabled={isStoppingJob}
                >
                  {isStoppingJob ? 'Stopping...' : 'Stop Job'}
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
        )}
      </div>
    </div>
  )
}

export default JobActions
