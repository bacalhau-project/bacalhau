import React, { useMemo } from 'react'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  apimodels_ListJobExecutionsResponse,
  models_JobStateType, models_SpecConfig,
  models_State_models_JobStateType,
  models_Task
} from '@/lib/api/generated'
import {
  getExecutionDesiredStateLabel,
  getExecutionStateLabel,
  getJobState,
  shortID,
} from '@/lib/api/utils'
import { formatTimestamp } from '@/lib/time'
import { Download } from 'lucide-react'
import * as Tooltip from '@radix-ui/react-tooltip'
import { toast } from '@/hooks/use-toast'
import { Card, CardContent } from '@/components/ui/card'

const SUPPORTED_PUBLISHERS = ['local', 's3'];
const COMPLETED_JOB_STATES = [
  models_JobStateType.JobStateTypeCompleted,
  models_JobStateType.JobStateTypeFailed,
  models_JobStateType.JobStateTypeStopped,
];

interface DownloadButtonProps {
  publisherType: string;
  publishedResult: models_SpecConfig | undefined
  isTaskCompleted: boolean;
  onDownload: (publishedResult: models_SpecConfig | undefined) => Promise<void>;
}

const DownloadButton: React.FC<DownloadButtonProps> = ({
  publisherType,
  publishedResult,
  isTaskCompleted,
  onDownload
}) => {
  const enabledClass = "p-2 text-black transition-colors duration-150 hover:text-blue-500 active:text-blue-700";
  const disabledClass = "p-2 text-gray-300";

  const isSupportedPublisher = SUPPORTED_PUBLISHERS.includes(publisherType);

  let disabledReason = "";
  if (!publisherType) {
    disabledReason = "No publisher specified"
  } else if (!isTaskCompleted) {
    disabledReason = "Task still running";
  } else if (!publishedResult?.Params) {
    disabledReason = "No results published";
  } else if (!isSupportedPublisher) {
    disabledReason = "Not supported for this publisher";
  }

  if (!disabledReason) {
    return (
      <button
        className={enabledClass}
        onClick={() => onDownload(publishedResult)}
      >
        <Download size={18} strokeWidth={2} />
      </button>
    )
  }

  return (
    <Tooltip.Provider>
      <Tooltip.Root>
        <Tooltip.Trigger asChild>
          <button className={disabledClass} disabled>
            <Download size={18} strokeWidth={2} />
          </button>
        </Tooltip.Trigger>
        <Tooltip.Portal>
          <Tooltip.Content
            className="px-3 py-2 text-sm text-white bg-gray-900 rounded-md shadow-lg"
            sideOffset={5}
          >
            {disabledReason}
            <Tooltip.Arrow className="fill-current text-gray-900" />
          </Tooltip.Content>
        </Tooltip.Portal>
      </Tooltip.Root>
    </Tooltip.Provider>
  );
};

/**
 * Shows a toast notification when no results are found for a job.
 */
const showNoResultsToast = () => {
  toast({
    variant: 'destructive',
    title: 'No results found for this job',
    description: 'You can check the logged output of the job in the logs tab.',
  });
};

/**
 * Checks if a URL is valid.
 */
const isValidUrl = (url: string): boolean => {
  try {
    new URL(url);
    return true;
  } catch {
    return false;
  }
};

/**
 * Opens a URL in a new window after validating it.
 * Shows a toast notification if the URL is invalid or the popup is blocked.
 * @param url The URL to open in a new window
 */
const safeWindowOpen = (url: string): void => {
  if (!isValidUrl(url)) {
    toast({
      variant: 'destructive',
      title: 'Invalid download URL',
      description: 'The download URL appears to be invalid.',
    });
    return;
  }

  const newWindow = window.open(url, "_blank");
  if (!newWindow) {
    toast({
      variant: 'destructive',
      title: 'Download blocked',
      description: 'Please allow popups for this site to download results.',
    });
  }
};

interface Downloader {
  download: (data: models_SpecConfig) => Promise<void>;
}

/**
 * Creates a downloader that opens a URL in a new window.
 * @param urlField The field in the spec config that contains the URL to download
 */
const createDownloader = (urlField: string): Downloader => ({
  async download(data: models_SpecConfig) {
    const params = data.Params as Record<string, any>;
    const url = params?.[urlField];
    if (url) {
      safeWindowOpen(url);
    } else {
      showNoResultsToast();
    }
  },
});

/**
 * Returns a downloader based on the publisher type.
 */
const getDownloader = (publisherType: string): Downloader => {
  switch (publisherType) {
    case "s3":
      return createDownloader("PreSignedURL");
    default:
      return createDownloader("URL");
  }
};

const JobExecutions = ({
  executions,
  tasks,
  state
}: {
  executions?: apimodels_ListJobExecutionsResponse,
  tasks: Array<models_Task>,
  state: models_State_models_JobStateType
}) => {
  const jobState = getJobState(state?.StateType);
  const isTaskCompleted = jobState ? COMPLETED_JOB_STATES.includes(jobState) : false;
  const publisherType = useMemo(() =>
    tasks[0]?.Publisher?.Type ?? "UNKNOWN",
    [tasks]
  );

  const onDownload = async (publishedResult: models_SpecConfig | undefined) => {
    if (!publishedResult) {
      showNoResultsToast();
      return;
    }

    const downloader = getDownloader(publisherType);
    await downloader.download(publishedResult);
  };

  return (
    <Card>
      <CardContent className="pt-6">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Created Time</TableHead>
              <TableHead>Modified Time</TableHead>
              <TableHead>ID</TableHead>
              <TableHead>Node ID</TableHead>
              <TableHead>State</TableHead>
              <TableHead>Desired State</TableHead>
              <TableHead>Results</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {executions?.Items?.map((execution) => (
              <TableRow key={execution.ID}>
                <TableCell>
                  {formatTimestamp(execution.CreateTime, true)}
                </TableCell>
                <TableCell>
                  {formatTimestamp(execution.ModifyTime, true)}
                </TableCell>
                <TableCell>{shortID(execution.ID)}</TableCell>
                <TableCell>{shortID(execution.NodeID)}</TableCell>
                <TableCell>
                  {getExecutionStateLabel(execution.ComputeState?.StateType)}
                </TableCell>
                <TableCell>
                  {getExecutionDesiredStateLabel(
                    execution.DesiredState?.StateType
                  )}
                </TableCell>
                <TableCell>
                  <DownloadButton
                    publisherType={publisherType}
                    publishedResult={execution.PublishedResult}
                    isTaskCompleted={isTaskCompleted}
                    onDownload={onDownload}
                  />
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  )
}

export default JobExecutions
