
import React from 'react';
import { Badge } from '@/components/ui/badge';

interface JobLabelsProps {
  labels: Record<string, string>;
}

const JobLabels: React.FC<JobLabelsProps> = ({ labels }) => {
  const getColorForLabel = (key: string) => {
    const colors = [
      'bg-blue-100 text-blue-800',
      'bg-green-100 text-green-800',
      'bg-yellow-100 text-yellow-800',
      'bg-purple-100 text-purple-800',
      'bg-pink-100 text-pink-800',
      'bg-indigo-100 text-indigo-800',
    ];

    // Use a hash function to consistently map keys to colors
    const hash = key.split('').reduce((acc, char) => char.charCodeAt(0) + ((acc << 5) - acc), 0);
    return colors[Math.abs(hash) % colors.length];
  };

  return (
    <div className="flex flex-wrap gap-2">
      {Object.entries(labels).map(([key, value]) => (
        <Badge key={key} className={`text-xs ${getColorForLabel(key)}`}>
          {key}: {value}
        </Badge>
      ))}
    </div>
  );
};

export { JobLabels };
