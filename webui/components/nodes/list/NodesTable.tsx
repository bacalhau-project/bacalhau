import Link from 'next/link'
import { models_NodeState } from '@/lib/api/generated'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import TruncatedTextWithTooltip from '@/components/TruncatedTextWithTooltip'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { ConnectionStatus, MembershipStatus } from '@/components/nodes/NodeStatus'
import Labels  from '@/components/Labels'
import { NodeResources } from '@/components/nodes/NodeResources'
import { getNodeType } from '@/lib/api/utils'

interface NodesTableProps {
  nodes: models_NodeState[]
  pageSize: number
  setPageSize: (size: number) => void
  pageIndex: number
  onPreviousPage: () => void
  onNextPage: () => void
  hasNextPage: boolean
}

export function NodesTable({
                             nodes = [],
                             pageSize,
                             setPageSize,
                             pageIndex,
                             onPreviousPage,
                             onNextPage,
                             hasNextPage,
                           }: NodesTableProps) {
  return (
    <div>
      <Table>
        <TableHeader className="bg-muted/50">
          <TableRow>
            <TableHead className="p-3 w-28">Node ID</TableHead>
            <TableHead className="w-28">Node Type</TableHead>
            <TableHead className="w-28">Membership</TableHead>
            <TableHead className="w-28">Connection</TableHead>
            <TableHead className="w-60">Resources</TableHead>
            <TableHead className="w-64">Labels</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {nodes.map((node) => (
            <TableRow key={node.Info?.NodeID}>
              <TableCell className="p-3">
                <Link href={`/nodes?id=${node.Info?.NodeID}`}>
                  <TruncatedTextWithTooltip text={node.Info?.NodeID ?? ''} maxLength={10} />
                </Link>
              </TableCell>
              <TableCell>{getNodeType(node)}</TableCell>
              <TableCell><MembershipStatus node={node} /></TableCell>
              <TableCell><ConnectionStatus node={node} /></TableCell>
              <TableCell className="p-2"><NodeResources node={node} /></TableCell>
              <TableCell>
                <Labels labels={node.Info?.Labels} />
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
      <div className="flex items-center justify-between space-x-2 py-4">
        <div className="flex items-center space-x-2">
          <p className="text-sm font-medium">Nodes per page</p>
          <Select
            value={`${pageSize}`}
            onValueChange={(value) => setPageSize(Number(value))}
          >
            <SelectTrigger className="h-8 w-[70px]">
              <SelectValue placeholder={pageSize} />
            </SelectTrigger>
            <SelectContent side="top">
              {[10, 20, 30, 40, 50].map((size) => (
                <SelectItem key={size} value={`${size}`}>
                  {size}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="flex items-center space-x-2">
          <Button
            variant="outline"
            size="sm"
            onClick={onPreviousPage}
            disabled={pageIndex === 0}
          >
            Previous
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={onNextPage}
            disabled={!hasNextPage}
          >
            Next
          </Button>
        </div>
      </div>
    </div>
  )
}