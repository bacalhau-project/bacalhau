import axios from "axios";
import { JobsResponse } from "../../helpers/jobInterfaces";
import { NodesResponse } from "../../helpers/nodeInterfaces";

// Base configuration for Bacalhau API
const apiConfig = {
  baseURL: "http://0.0.0.0:52662/api/v1", // TODO: replace this with flexible port
  headers: {
    "Content-Type": "application/json",
  },
};

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
