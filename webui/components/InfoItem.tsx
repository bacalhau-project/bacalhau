import React from 'react'

interface InfoItemProps {
  label: string
  children: React.ReactNode
}

const InfoItem: React.FC<InfoItemProps> = ({ label, children }) => (
  <div className="grid grid-cols-[120px,1fr] items-center py-1">
    <span className="text-sm font-semibold text-gray-500">{label}:</span>
    <span className="text-sm text-gray-900">{children}</span>
  </div>
)

export default InfoItem
