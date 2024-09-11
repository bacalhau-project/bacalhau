import React from 'react'
import { models_NodeState } from '@/lib/api/generated'
import { Cpu, HardDrive, CircuitBoard } from 'lucide-react'

interface NodeResourcesProps {
  node: models_NodeState
  variant?: 'compact' | 'default' | 'large'
}

const formatResource = (
  available: number | undefined,
  total: number | undefined,
  unit: string
): string => {
  if (available === undefined || total === undefined) {
    return 'N/A'
  }
  return `${available.toFixed(1)}/${total.toFixed(1)} ${unit}`
}

const formatMemoryOrDisk = (bytes: number | undefined): number => {
  if (bytes === undefined) {
    return 0
  }
  return Number((bytes / (1024 * 1024 * 1024)).toFixed(1))
}

const ResourceBar: React.FC<{
  available: number
  total: number
  icon: React.ReactNode
  label: string
  unit: string
  variant: 'compact' | 'default' | 'large'
}> = ({ available, total, icon, label, unit, variant }) => {
  const percentage = (available / total) * 100
  return (
    <div
      className={`flex items-center space-x-2 ${variant === 'large' ? 'mb-5' : variant === 'compact' ? 'mb-0.5' : 'mb-1'}`}
    >
      {variant !== 'compact' && (
        <div
          className={`${variant === 'large' ? 'w-24' : 'w-20 text-sm '} flex items-center`}
        >
          <span className="mr-2">{icon}</span>
          {label}
        </div>
      )}
      <div
        className={`flex-grow bg-gray-100 rounded-md ${variant === 'large' ? 'h-4' : variant === 'compact' ? 'h-1' : 'h-2'}`}
      >
        <div
          className={`bg-blue-500 rounded-md ${variant === 'large' ? 'h-4' : variant === 'compact' ? 'h-1' : 'h-2'}`}
          style={{ width: `${percentage}%` }}
        ></div>
      </div>
      <div
        className={`${variant === 'large' ? 'w-32' : variant === 'compact' ? 'w-16 text-sm ' : 'w-24 text-sm '} text-right`}
      >
        {formatResource(available, total, unit)}
      </div>
    </div>
  )
}

export const NodeResources: React.FC<NodeResourcesProps> = ({
  node,
  variant = 'default',
}) => {
  const availableCapacity = node.Info?.ComputeNodeInfo?.AvailableCapacity
  const maxCapacity = node.Info?.ComputeNodeInfo?.MaxCapacity

  if (!availableCapacity || !maxCapacity) {
    return (
      <span className="text-sm text-gray-500">
        No resource information available
      </span>
    )
  }

  return (
    <div className="w-full">
      <ResourceBar
        available={availableCapacity.CPU || 0}
        total={maxCapacity.CPU || 1}
        icon={<Cpu size={variant === 'large' ? 20 : 14} />}
        label="CPU"
        unit="cores"
        variant={variant}
      />
      <ResourceBar
        available={formatMemoryOrDisk(availableCapacity.Memory)}
        total={formatMemoryOrDisk(maxCapacity.Memory)}
        icon={<CircuitBoard size={variant === 'large' ? 20 : 14} />}
        label="Memory"
        unit="GB"
        variant={variant}
      />
      <ResourceBar
        available={formatMemoryOrDisk(availableCapacity.Disk)}
        total={formatMemoryOrDisk(maxCapacity.Disk)}
        icon={<HardDrive size={variant === 'large' ? 20 : 14} />}
        label="Disk"
        unit="GB"
        variant={variant}
      />
      {availableCapacity.GPU !== undefined && maxCapacity.GPU !== undefined && (
        <ResourceBar
          available={availableCapacity.GPU}
          total={maxCapacity.GPU}
          icon={<Cpu size={variant === 'large' ? 20 : 14} />}
          label="GPU"
          unit="units"
          variant={variant}
        />
      )}
    </div>
  )
}
