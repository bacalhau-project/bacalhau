import React from 'react'
import { Box, Cpu } from 'lucide-react'
import { models_Job } from '@/lib/api/generated'

interface JobEngineDisplayProps {
  job: models_Job
}

const DockerIcon: React.FC<{ className?: string }> = ({ className }) => (
  <svg
    xmlns="http://www.w3.org/2000/svg"
    aria-label="Docker"
    role="img"
    viewBox="0 0 512 512"
    width="24px"
    height="24px"
    fill="none"
  >
    <g id="SVGRepo_bgCarrier" stroke-width="0"></g>
    <g
      id="SVGRepo_tracerCarrier"
      stroke-linecap="round"
      stroke-linejoin="round"
    ></g>
    <g id="SVGRepo_iconCarrier">
      <rect width="512" height="512" rx="15%" fill="#ffffff"></rect>
      <path
        stroke="#066da5"
        stroke-width="38"
        d="M296 226h42m-92 0h42m-91 0h42m-91 0h41m-91 0h42m8-46h41m8 0h42m7 0h42m-42-46h42"
      ></path>
      <path
        fill="#066da5"
        d="m472 228s-18-17-55-11c-4-29-35-46-35-46s-29 35-8 74c-6 3-16 7-31 7H68c-5 19-5 145 133 145 99 0 173-46 208-130 52 4 63-39 63-39"
      ></path>
    </g>
  </svg>
)

const JobEngineDisplay: React.FC<JobEngineDisplayProps> = ({ job }) => {
  const getEngineIcon = (engineType: string) => {
    switch (engineType.toLowerCase()) {
      case 'docker':
        return <DockerIcon className="w-5 h-5 text-blue-500" />
      case 'wasm':
        return <Cpu className="w-5 h-5 text-green-500" />
      default:
        return <Box className="w-5 h-5 text-gray-500" />
    }
  }
  const engineType = job.Tasks?.[0].Engine?.Type || 'unknown'

  return (
    <div className="flex items-center space-x-2">
      {getEngineIcon(engineType)}
      <span className="text-sm font-medium">{engineType}</span>
    </div>
  )
}

export default JobEngineDisplay
