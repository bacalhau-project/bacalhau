import { generateMockJob } from "../../tests/mocks/jobMock"
import { Job } from "../helpers/jobInterfaces"
import { JobsTable } from "../pages/JobsDashboard/JobsTable/JobsTable"

import TenJobsData from "./storyData/jobsTable/10-jobs.json"

import HundredJobsData from "./storyData/jobsTable/100-jobs.json"

export default {
  component: JobsTable,
  title: "JobsTable",
  tags: ["autodocs"],
}

export const fullDataGenerator = (numJobs: number = 10): Job[] => {
  // If numJobs is not a positive integer, return an empty array
  if (!Number.isInteger(numJobs) || numJobs < 1) {
    console.error("numJobs must be a positive integer")
    return []
  }

  const jobs: Job[] = []
  for (let i = 0; i < numJobs; i += 1) {
    jobs.push(generateMockJob())
  }

  // Read the jobs from the mock data in storyData/10-jobs.json
  return jobs
}

export const Default = {
  args: { data: [] },
}
export const TenJobs = {
  args: { data: TenJobsData },
}
export const HundredJobs = {
  args: { data: HundredJobsData },
}
