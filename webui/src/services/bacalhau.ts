import axios from "axios";
import { JobListRequest, JobsResponse } from "../helpers/jobInterfaces";
import { NodeListRequest, NodesResponse } from "../helpers/nodeInterfaces";

// Base configuration for Bacalhau API
const apiHost = "0.0.0.0";
const apiPort = "1234";

const apiConfig = {
  baseURL: `http://${apiHost}:${apiPort}/api/v1`,
  headers: {
    "Content-Type": "application/json",
  },
};

const apiClient = axios.create(apiConfig);

class BacalhauAPI {
  async listJobs(labels?: string[], nextToken?: string): Promise<JobsResponse> {
    try {
      const params: JobListRequest = {
        labels: labels ? `env in (${labels.join(",")})` : undefined,
        next_token: nextToken,
      };
  
      const response = await apiClient.get("/orchestrator/jobs", { params });
      return response.data;
    } catch (error) {
      console.error("An error occurred while listing jobs:", error);
      throw error;
    }
  }

  async listNodes(labels?: string[]): Promise<NodesResponse> {
    try {
      const params: NodeListRequest = {
        labels: labels ? `env in (${labels.join(",")})` : undefined,
      };
      const response = await apiClient.get("/orchestrator/nodes", { params });
      return response.data;
    } catch (error) {
      console.error("An error occurred while listing nodes:", error);
      throw error;
    }
  }
  
}

export const bacalhauAPI = new BacalhauAPI();
