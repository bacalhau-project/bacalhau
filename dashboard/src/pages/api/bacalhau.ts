import axios from "axios";
import { JobsResponse } from "../../helpers/jobInterfaces";
import { NodesResponse } from "../../helpers/nodeInterfaces";

// Base configuration for Bacalhau API
const apiHost = process.env.REACT_APP_BACALHAU_API_HOST || '0.0.0.0';
const apiPort = process.env.REACT_APP_BACALHAU_API_PORT || '51331';

const apiConfig = {
  baseURL: `http://${apiHost}:${apiPort}/api/v1`,
  headers: {
    "Content-Type": "application/json",
  },
};

// http://0.0.0.0:51331/api/v1/orchestrator/nodes

const apiClient = axios.create(apiConfig);

class BacalhauAPI {
  async listJobs(labels?: string[], nextToken?: string): Promise<JobsResponse> {
    const params: any = {};
    if (labels) {
      params.labels = `env in (${labels.join(",")})`;
    }
    if (nextToken) {
      params.next_token = nextToken;
    }
    const response = await apiClient.get("/orchestrator/jobs", { params });
    return response.data;
  }

  async submitJob(jobData: any): Promise<JobsResponse> {
    const response = await apiClient.put("/orchestrator/jobs", jobData);
    return response.data;
  }

  async stopJob(jobId: string): Promise<void> {
    await apiClient.delete(`/orchestrator/jobs/${jobId}`);
  }

  async listNodes(labels?: string[]): Promise<NodesResponse> {
    const params: any = {};
    if (labels) {
      params.labels = `env in (${labels.join(",")})`;
    }
    const response = await apiClient.get("/orchestrator/nodes", { params });
    return response.data;
  }
}

export const bacalhauAPI = new BacalhauAPI();
