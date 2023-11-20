import axios from "axios";
import { Job, JobListRequest, JobsResponse } from "../../helpers/jobInterfaces";
import { NodeListRequest, NodesResponse } from "../../helpers/nodeInterfaces";

// Base configuration for Bacalhau API
const apiHost = process.env.REACT_APP_BACALHAU_API_HOST || "0.0.0.0";
const apiPort = process.env.REACT_APP_BACALHAU_API_PORT || "1234";

const apiConfig = {
  baseURL: `http://${apiHost}:${apiPort}/api/v1`,
  headers: {
    "Content-Type": "application/json",
  },
};

const apiClient = axios.create(apiConfig);

class BacalhauAPI {
  async listJobs(labels?: string[], nextToken?: string): Promise<JobsResponse> {
    const params: JobListRequest = {
      labels: labels ? `env in (${labels.join(",")})` : undefined,
      next_token: nextToken,
    };

    const response = await apiClient.get("/orchestrator/jobs", { params });
    return response.data;
  }

  async submitJob(jobData: Job): Promise<JobsResponse> {
    const response = await apiClient.put("/orchestrator/jobs", jobData);
    return response.data;
  }

  async stopJob(jobId: string): Promise<void> {
    await apiClient.delete(`/orchestrator/jobs/${jobId}`);
  }

  async listNodes(labels?: string[]): Promise<NodesResponse> {
    const params: NodeListRequest = {
      labels: labels ? `env in (${labels.join(",")})` : undefined,
    };

    const response = await apiClient.get("/orchestrator/nodes", { params });
    return response.data;
  }
}

export const bacalhauAPI = new BacalhauAPI();
