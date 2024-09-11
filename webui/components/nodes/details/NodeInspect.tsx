import React from 'react'
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { models_NodeState } from '@/lib/api/generated'
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter'
import { vscDarkPlus } from 'react-syntax-highlighter/dist/esm/styles/prism'

interface NodeInspectProps {
  node: models_NodeState
}

const NodeInspect: React.FC<NodeInspectProps> = ({ node }) => {
  return (
    <Card className="bg-gray-900 text-white">
      <CardContent>
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
          {JSON.stringify(node, null, 2)}
        </SyntaxHighlighter>
      </CardContent>
    </Card>
  )
}

export default NodeInspect