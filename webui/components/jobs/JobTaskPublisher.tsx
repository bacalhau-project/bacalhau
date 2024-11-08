import { useCallback } from "react";
import { apimodels_ListJobResultsResponse, models_JobStateType, models_State_models_JobStateType, models_Task, Orchestrator } from "@/lib/api/generated";
import { getJobState } from "@/lib/api/utils";
import { useApiOperation } from "@/hooks/useApiOperation";
import { toast } from "@/hooks/use-toast";
import { Download, Loader } from "lucide-react";
import * as Tooltip from "@radix-ui/react-tooltip";

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
  if (isLoading) {
    return (
      <button className="p-2 text-black" disabled>
        <Loader size={18} strokeWidth={2} className="animate-spin" />
      </button>
    );
  }

  if (isDownloadable) {
    return (
      <button
        className="p-2 text-black transition-colors duration-150 hover:text-blue-500 active:text-blue-700"
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
          <button className="p-2 text-gray-300" disabled>
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


const LocalDownloader: Downloader = {
  async download(data) {
    const params = data?.Items?.[0]?.Params as { URL?: string };
    if (params?.URL) {
      window.open(params.URL, "_blank");
    } else {
      showNoResultsToast();
    }
  },
};

const S3Downloader: Downloader = {
  async download(data) {
    const params = data?.Items?.[0]?.Params as { PreSignedURL?: string };
    if (params?.PreSignedURL) {
      window.open(params.PreSignedURL, "_blank");
    } else {
      showNoResultsToast();
    }
  },
};

const getDownloader = (publisherType: string): Downloader => {
  switch (publisherType) {
    case "s3":
      return S3Downloader;
    case "local":
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

  const publisherType = tasks[0]?.Publisher?.Type ?? "UNKNOWN";
  const jobState = getJobState(state?.StateType);
  const isSupportedPublisher = ["local", "s3"].includes(publisherType);
  const isTaskCompleted = jobState
    ? [
      models_JobStateType.JobStateTypeCompleted,
      models_JobStateType.JobStateTypeFailed,
      models_JobStateType.JobStateTypeStopped,
    ].includes(jobState)
    : false;
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
