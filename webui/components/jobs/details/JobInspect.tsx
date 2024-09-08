import React from 'react';
import { Card, CardContent } from '@/components/ui/card';
import { models_Job } from '@/lib/api/generated';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { vscDarkPlus } from 'react-syntax-highlighter/dist/esm/styles/prism';

const JobInspect = ({ job }: { job: models_Job }) => (
  <Card className="bg-gray-900 text-white">
    <CardContent className="pt-6">
      <SyntaxHighlighter
        language="json"
        style={vscDarkPlus}
        customStyle={{
          backgroundColor: 'transparent',
          padding: '1rem',
          margin: 0,
          borderRadius: '0.5rem',
        }}
      >
        {JSON.stringify(job, null, 2)}
      </SyntaxHighlighter>
    </CardContent>
  </Card>
);

export default JobInspect;