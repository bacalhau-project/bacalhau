import React from 'react';
import { Badge } from '@/components/ui/badge';

type Label = string | [string, string];

interface LabelsProps {
  labels: Label[] | Record<string, string> | undefined;
  color?: string; // New prop for specifying a uniform color
}

const getColorForLabel = (key: string) => {
  const colors = [
    'bg-blue-100 text-blue-800',
    'bg-green-100 text-green-800',
    'bg-yellow-100 text-yellow-800',
    'bg-purple-100 text-purple-800',
    'bg-pink-100 text-pink-800',
    'bg-indigo-100 text-indigo-800',
  ];

  const hash = key
    .split('')
    .reduce((acc, char) => char.charCodeAt(0) + ((acc << 5) - acc), 0);
  return colors[Math.abs(hash) % colors.length];
};

const Labels: React.FC<LabelsProps> = ({ labels, color }) => {
  if (!labels || (Array.isArray(labels) && labels.length === 0) || (typeof labels === 'object' && Object.keys(labels).length === 0)) {
    return null;
  }

  const renderLabel = (key: string, value?: string) => (
    <Badge
      key={key}
      className={`text-xs ${color || getColorForLabel(key)}`}
    >
      {value ? `${key}: ${value}` : key}
    </Badge>
  );

  const renderLabels = () => {
    if (Array.isArray(labels)) {
      return labels.map((label) => {
        if (typeof label === 'string') {
          return renderLabel(label);
        } else if (Array.isArray(label)) {
          const [key, value] = label;
          return renderLabel(key, value);
        }
        return null;
      });
    } else if (typeof labels === 'object') {
      return Object.entries(labels).map(([key, value]) => renderLabel(key, value));
    }
    return null;
  };

  return <div className="flex flex-wrap gap-2">{renderLabels()}</div>;
};

export default Labels;