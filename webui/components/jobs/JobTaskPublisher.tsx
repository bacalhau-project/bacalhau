import { useCallback, useMemo } from "react";
import { apimodels_ListJobResultsResponse, models_JobStateType, models_State_models_JobStateType, models_Task, Orchestrator } from "@/lib/api/generated";
import { getJobState } from "@/lib/api/utils";
import { useApiOperation } from "@/hooks/useApiOperation";
import { toast } from "@/hooks/use-toast";
import { Download, Loader } from "lucide-react";
import * as Tooltip from "@radix-ui/react-tooltip";

const SUPPORTED_PUBLISHERS = ['local', 's3'];
const COMPLETED_JOB_STATES = [
  models_JobStateType.JobStateTypeCompleted,
  models_JobStateType.JobStateTypeFailed,
  models_JobStateType.JobStateTypeStopped,
];

interface JobTaskPublisherDisplayProps {
  jobId: string;
  tasks: Array<models_Task>;
  state?: models_State_models_JobStateType;
}

interface DownloadButtonProps {
  isDownloadable: boolean;
  isLoading: boolean;
  disabledReason?: string;
  onDownload: () => void;
}

const DownloadButton: React.FC<DownloadButtonProps> = ({ isDownloadable, isLoading, disabledReason, onDownload }) => {
  const enabledClass = "p-2 text-black transition-colors duration-150 hover:text-blue-500 active:text-blue-700";
  const disabledClass = "p-2 text-gray-300";
  const loadingClass = "p-2 text-black";

  if (isLoading) {
    return (
      <button className={loadingClass} disabled>
        <Loader size={18} strokeWidth={2} className="animate-spin" />
      </button>
    );
  }

  if (isDownloadable) {
    return (
      <button
        className={enabledClass}
        onClick={onDownload}
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

interface Downloader {
  download: (data: apimodels_ListJobResultsResponse) => Promise<void>;
}

const showNoResultsToast = () => {
  toast({
    variant: 'destructive',
    title: 'No results found for this job',
    description: 'You can check the logged output of the job in the logs tab.',
  });
};

const isValidUrl = (url: string): boolean => {
  try {
    new URL(url);
    return true;
  } catch {
    return false;
  }
};

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


const LocalDownloader: Downloader = {
  async download(data) {
    const params = data?.Items?.[0]?.Params as { URL?: string };
    if (params?.URL) {
      safeWindowOpen(params.URL);
    } else {
      showNoResultsToast();
    }
  },
};

const S3Downloader: Downloader = {
  async download(data) {
    const params = data?.Items?.[0]?.Params as { PreSignedURL?: string };
    if (params?.PreSignedURL) {
      safeWindowOpen(params.PreSignedURL);
    } else {
      showNoResultsToast();
    }
  },
};

const getDownloader = (publisherType: string): Downloader => {
  switch (publisherType) {
    case "s3":
      return S3Downloader;
    default:
      return LocalDownloader;
  }
};

const JobTaskPublisherDisplay: React.FC<JobTaskPublisherDisplayProps> = ({ jobId, tasks, state }) => {
  const { isLoading, error, execute, } = useApiOperation<apimodels_ListJobResultsResponse>()

  const downloadResults = useCallback(() => {
    return execute(() =>
      Orchestrator.jobResults({
        path: { id: jobId },
        throwOnError: true,
      }).then((response) => response.data)
    )
  }, [execute, jobId]);

  const publisherType = useMemo(() =>
    tasks[0]?.Publisher?.Type ?? "UNKNOWN",
    [tasks]
  );
  const jobState = getJobState(state?.StateType);
  const isSupportedPublisher = SUPPORTED_PUBLISHERS.includes(publisherType);
  const isTaskCompleted = jobState ? COMPLETED_JOB_STATES.includes(jobState) : false;
  const isDownloadable = isSupportedPublisher && isTaskCompleted;

  let disabledReason = "";
  if (!isSupportedPublisher) {
    disabledReason = "Not supported for this publisher";
  } else if (!isTaskCompleted) {
    disabledReason = "Task still running";
  }

  const onDownload = async () => {
    const resultsData = await downloadResults();
    if (error || !resultsData) return;

    const downloader = getDownloader(publisherType);
    await downloader.download(resultsData);
  };

  return (
    <div className="flex items-center space-x-2">
      {publisherType}
      <DownloadButton
        isDownloadable={isDownloadable}
        isLoading={isLoading}
        disabledReason={disabledReason}
        onDownload={onDownload}
      />
    </div>
  );
};

export default JobTaskPublisherDisplay;
