import JobDetails  from '@/components/jobs/details/JobDetails'

export default function JobDetailsPage({ params }: { params: { id: string } }) {
  return <JobDetails jobId={params.id} />
}
