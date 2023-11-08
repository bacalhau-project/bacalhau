import axios from 'axios';
import { Job } from "../../interfaces";


// Base configuration for Bacalhau API
const apiConfig = {
  baseURL: 'http://0.0.0.0:52509/api/v1',
//   port: '52509',
//   endpointPrefix: '/api/v1',
  headers: {
    'Content-Type': 'application/json',
  },
};

// Creating an instance of axios with the base configuration
const apiClient = axios.create(apiConfig);
console.log("apiClient", apiClient)

interface NodeList {
  // Define the structure for NodeList object
}

class BacalhauAPI {
  // List jobs with optional label filtering and pagination
  async listJobs(labels?: string[], nextToken?: string): Promise<Job[]> {
    try {
      const params: any = {};
      if (labels) {
        params.labels = `env in (${labels.join(',')})`;
      }
      if (nextToken) {
        params.next_token = nextToken;
      }
      const response = await apiClient.get('/orchestrator/jobs', { params });
      console.log("RESPPONNNSEEE", response)
      return response.data;

    } catch (error) {
      // Handle error
      throw error;
    }
  }

  // Submit a new job
  async submitJob(jobData: any): Promise<Job> {
    try {
      const response = await apiClient.put('/orchestrator/jobs', jobData);
      return response.data;
    } catch (error) {
      // Handle error
      throw error;
    }
  }

  // Stop a job
  async stopJob(jobId: string): Promise<void> {
    try {
      await apiClient.delete(`/orchestrator/jobs/${jobId}`);
    } catch (error) {
      // Handle error
      throw error;
    }
  }

  // List nodes with optional label filtering
  async listNodes(labels?: string[]): Promise<NodeList> {
    try {
      const params: any = {};
      if (labels) {
        params.labels = `env in (${labels.join(',')})`;
      }
      const response = await apiClient.get('/orchestrator/nodes', { params });
      return response.data;
    } catch (error) {
      // Handle error
      throw error;
    }
  }
  
  // Fetch agent info
  async getAgentInfo(): Promise<any> {
    try {
      const response = await apiClient.get('/agent/info');
      return response.data;
    } catch (error) {
      // Handle error
      throw error;
    }
  }

  // Check agent health
  async checkAgentHealth(): Promise<any> {
    try {
      const response = await apiClient.get('/agent/health');
      return response.data;
    } catch (error) {
      // Handle error
      throw error;
    }
  }
}

export const bacalhauAPI = new BacalhauAPI();
