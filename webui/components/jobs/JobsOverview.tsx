'use client'

import React, {useEffect, useState} from 'react';
import { JobsTable } from './JobsTable';
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import {models_Job, OrchestratorService} from "@/lib/api/generated";
import { useApi } from '@/app/providers/ApiProvider';


export function JobsOverview() {
  const [jobs, setJobs] = useState<models_Job[]>([]);
  const [search, setSearch] = useState('');
  const { isInitialized } = useApi();

  useEffect(() => {
    async function fetchJobs() {
      if (!isInitialized) return;

      try {
        const response = await OrchestratorService.orchestratorListJobs();
        setJobs(response.Items ?? []);
      } catch (error) {
        console.error('Error fetching jobs:', error);
        setJobs([]);
      }
    }

    fetchJobs();
  }, [isInitialized]);

  const filteredJobs = jobs.filter(job =>
    (job.ID?.toLowerCase().includes(search.toLowerCase()) ?? false) ||
    (job.Name?.toLowerCase().includes(search.toLowerCase()) ?? false)
  )

  return (
    <div className="container mx-auto py-8">
      <h1 className="text-3xl font-bold mb-8">Jobs overview</h1>
      <div className="flex justify-between items-center mb-6">
        <Input
          className="max-w-sm"
          type="search"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="Search jobs..."
        />
        <Button>Submit Job</Button>
      </div>
      <JobsTable jobs={filteredJobs} />
    </div>
  );
}
