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

function getAPIConfig(property: "host" | "port" | "base", defaultValue: string): string {
  const declared = document.querySelector(`link[rel=api-${property}]`)?.getAttribute("href");
  const useDefault = declared === undefined || declared?.match(/\{{2}/) || declared === "" || declared === null;
  return useDefault ? defaultValue : (declared || "");
}

const host = getAPIConfig("host", document.location.hostname)
const port = getAPIConfig("port", "1234")
const base = getAPIConfig("base", "api/v1")
export const bacalhauAPI = new BacalhauAPI(`${document.location.protocol}//${host}:${port}/${base}`);
