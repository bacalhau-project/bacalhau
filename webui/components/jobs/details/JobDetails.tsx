'use client'
import React, { useState, useEffect } from 'react';
import { OrchestratorService, apimodels_GetJobResponse } from '@/lib/api/generated';
import { useApi } from '@/app/providers/ApiProvider';
import { JobInformation } from './JobInformation';
import JobActions from './JobActions';
import JobTabs from './JobTabs';

const JobDetailsPage = ({ jobId }: { jobId: string }) => {
  const [jobData, setJobData] = useState<apimodels_GetJobResponse | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const { isInitialized } = useApi();

  const fetchJobData = async () => {
    if (!isInitialized) return;

    setIsLoading(true);
    setError(null);

    try {
      const response = await OrchestratorService.orchestratorGetJob(
        jobId,
        'history,executions',
        undefined // limit
      );
      setJobData(response);
    } catch (error) {
      console.error('Error fetching job data:', error);
      setError('Failed to fetch job data. Please try again.');
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchJobData();
  }, [isInitialized, jobId]);

  const handleJobUpdated = () => {
    fetchJobData();
  };

  if (isLoading) {
    return <div>Loading...</div>;
  }

  if (error) {
    return <div className="text-red-500">{error}</div>;
  }

  if (!jobData || !jobData.Job) {
    return <div>Job not found.</div>;
  }

  const { Job, History, Executions } = jobData;

  return (
    <div className="container mx-auto p-4">
      <div className="flex justify-between items-center mb-4">
        <h1 className="text-2xl font-bold">{Job.Name}</h1>
        <JobActions job={Job} onJobUpdated={handleJobUpdated} />
      </div>
      <JobInformation job={Job} />
      <JobTabs job={Job} history={History} executions={Executions} />
    </div>
  );
};

export default JobDetailsPage;