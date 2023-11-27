import axios, { AxiosInstance } from "axios";
import { JobListRequest, JobsResponse } from "../helpers/jobInterfaces";
import { NodeListRequest, NodesResponse } from "../helpers/nodeInterfaces";

class BacalhauAPI {
  apiClient: AxiosInstance

  constructor(baseURL: string) {
    this.apiClient = axios.create({
      baseURL: baseURL,
      headers: {
        "Content-Type": "application/json",
      },
    })
  }

  async listJobs(labels?: string[], nextToken?: string): Promise<JobsResponse> {
    try {
      const params: JobListRequest = {
        limit: 10,
        labels: labels ? `env in (${labels.join(",")})` : undefined,
        next_token: nextToken,
      };

      const response = await this.apiClient.get("/orchestrator/jobs", { params });
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
      const response = await this.apiClient.get("/orchestrator/nodes", { params });
      return response.data;
    } catch (error) {
      console.error("An error occurred while listing nodes:", error);
      throw error;
    }
  }

}

const host = document.querySelector("link[rel=api-host]")?.getAttribute("href") || document.location.host;
const port = document.querySelector("link[rel=api-port]")?.getAttribute("href") || "1234";
const base = document.querySelector("link[rel=api-base]")?.getAttribute("href") || "api/v1";
export const bacalhauAPI = new BacalhauAPI(`${document.location.protocol}//${host}:${port}/${base}`);
