import { Buffer } from "buffer"
import axios, { AxiosError, AxiosInstance } from "axios"
import {
  JobExecutionsResponse,
  JobListRequest,
  JobResponse,
  JobsResponse,
} from "../helpers/jobInterfaces"
import { NodeListRequest, NodesResponse } from "../helpers/nodeInterfaces"
import { Authentication, AuthnRequest, AuthnResponse, ListAuthnMethodsResponse, Requirement } from "../helpers/authInterfaces"

export class BacalhauAPI {
  apiClient: AxiosInstance

  constructor(baseURL: string) {
    this.apiClient = axios.create({
      baseURL,
      headers: {
        "Content-Type": "application/json",
      },
    })
  }

  async listJobs(labels?: string[], nextToken?: string): Promise<JobsResponse> {
    try {
      const params: JobListRequest = {
        order_by: "created_at",
        reverse: true,
        limit: 10,
        labels: labels ? `env in (${labels.join(",")})` : undefined,
        next_token: nextToken,
      }
      const response = await this.apiClient.get("/orchestrator/jobs", {
        params,
      })
      return response.data as JobsResponse
    } catch (error) {
      console.error("An error occurred while listing jobs:", error)
      throw error
    }
  }

  async listNodes(labels?: string[]): Promise<NodesResponse> {
    try {
      const params: NodeListRequest = {
        labels: labels ? `env in (${labels.join(",")})` : undefined,
      }
      const response = await this.apiClient.get("/orchestrator/nodes", {
        params,
      })
      return response.data as NodesResponse
    } catch (error) {
      console.error("An error occurred while listing nodes:", error)
      throw error
    }
  }

  async describeJob(jobId: string): Promise<JobResponse> {
    try {
      const response = await this.apiClient.get(`/orchestrator/jobs/${jobId}`)
      return response.data as JobResponse
    } catch (error) {
      console.error(
        `An error occurred while fetching details for job ID: ${jobId}`,
        error
      )
      throw error
    }
  }

  async jobExecution(jobId: string): Promise<JobExecutionsResponse> {
    try {
      const response = await this.apiClient.get(
        `/orchestrator/jobs/${jobId}/executions`
      )
      return response.data as JobExecutionsResponse
    } catch (error) {
      console.error(
        `An error occurred while fetching details for job executions: ${jobId}`,
        error
      )
      throw error
    }
  }

  async authMethods(): Promise<{ [key: string]: Requirement }> {
    try {
      const response = await this.apiClient.get("/auth")
      return (response.data as ListAuthnMethodsResponse).Methods
    } catch (error) {
      console.log(error)
      throw error
    }
  }

  async authenticate(req: AuthnRequest): Promise<Authentication> {
    try {
      const methodData = Buffer.from(JSON.stringify(req.MethodData)).toString("base64")
      const request = { Name: req.Name, MethodData: methodData }
      const response = await this.apiClient.post(`/auth/${req.Name}`, request)
      const data = response.data as AuthnResponse
      if (data.Authentication.success) {
        this.apiClient.defaults.headers.common.Authorization = `Bearer ${data.Authentication.token}`
      }

      return data.Authentication
    } catch (error) {
      if (error instanceof AxiosError && error.response?.status == 401) {
        const data = error.response.data as AuthnResponse
        return data.Authentication
      }
      throw error
    }
  }
}

function getAPIConfig(
  property: "host" | "port" | "base",
  defaultValue: string
): string {
  const declared = document
    .querySelector(`link[rel=api-${property}]`)
    ?.getAttribute("href")
  const useDefault =
    declared === undefined ||
    declared?.match(/\{{2}/) ||
    declared === "" ||
    declared === null
  return useDefault ? defaultValue : declared || ""
}

const host = getAPIConfig("host", document.location.hostname)
const port = getAPIConfig("port", "1234")
const base = getAPIConfig("base", "api/v1")
export const bacalhauAPI = new BacalhauAPI(
  `${document.location.protocol}//${host}:${port}/${base}`
)
