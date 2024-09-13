import React, { useState, useMemo } from 'react'
import { Card, CardContent } from '@/components/ui/card'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { Button } from '@/components/ui/button'
import { Filter, X } from 'lucide-react'
import { apimodels_ListJobHistoryResponse } from '@/lib/api/generated'
import { shortID, formatTime } from '@/lib/api/utils'

const colors = [
  'text-blue-800',
  'text-purple-800',
  'text-orange-700',
  'text-pink-700',
  'text-cyan-700',
  'text-yellow-700',
]

const getColorForExecutionID = (
  executionID: string | undefined,
  colorMap: Record<string, string>,
  colorIndex: number
) => {
  if (!executionID) {
    return ''
  }
  if (colorMap[executionID]) {
    return colorMap[executionID]
  } else {
    const newColor = colors[colorIndex % colors.length]
    colorMap[executionID] = newColor
    return newColor
  }
}

const JobHistory = ({
  history,
}: {
  history?: apimodels_ListJobHistoryResponse
}) => {
  const [colorMap, setColorMap] = useState<Record<string, string>>({})
  const [searchTerm, setSearchTerm] = useState('')
  const [showJobOnly, setShowJobOnly] = useState(false)
  const [filterExecutionID, setFilterExecutionID] = useState<string | null>(
    null
  )

  const filteredHistory = useMemo(() => {
    return history?.Items?.filter((item) => {
      const matchesSearch =
        item.ExecutionID?.toLowerCase().includes(searchTerm.toLowerCase()) ||
        item.Event?.Topic?.toLowerCase().includes(searchTerm.toLowerCase()) ||
        item.Event?.Message?.toLowerCase().includes(searchTerm.toLowerCase())
      const isJobEvent = !item.ExecutionID
      const matchesJobOnly = !showJobOnly || isJobEvent
      const matchesExecutionID =
        !filterExecutionID || item.ExecutionID === filterExecutionID
      return matchesSearch && matchesJobOnly && matchesExecutionID
    })
  }, [history, searchTerm, showJobOnly, filterExecutionID])

  const toggleFilter = (executionID: string | null) => {
    setFilterExecutionID((prevID) =>
      prevID === executionID ? null : executionID
    )
  }

  let colorIndex = 0

  return (
    <Card>
      <CardContent className="pt-6">
        <div className="flex items-center space-x-4 mb-6">
          <Input
            placeholder="Search by Execution ID, Topic, or Message"
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="flex-grow"
          />
          <div className="flex items-center space-x-2 flex-shrink-0">
            <Switch
              checked={showJobOnly}
              onCheckedChange={setShowJobOnly}
              id="job-only-switch"
            />
            <label htmlFor="job-only-switch">Job Events Only</label>
          </div>
        </div>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-1/6">Time</TableHead>
              <TableHead className="w-1/6">ExecutionID</TableHead>
              <TableHead className="w-1/6">Topic</TableHead>
              <TableHead className="w-2/6">Message</TableHead>
              <TableHead className="w-1/12">Filter</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {filteredHistory?.map((item, index) => {
              const isExecutionEvent = !!item.ExecutionID
              let rowClass = ''

              if (isExecutionEvent) {
                rowClass = getColorForExecutionID(
                  item.ExecutionID,
                  colorMap,
                  colorIndex
                )
                colorIndex++
              } else {
                rowClass = 'font-medium'
              }

              return (
                <TableRow key={index} className={rowClass}>
                  <TableCell>{formatTime(item.Time, true)}</TableCell>
                  <TableCell>
                    {isExecutionEvent ? shortID(item.ExecutionID) : '-'}
                  </TableCell>
                  <TableCell>{item.Event?.Topic}</TableCell>
                  <TableCell>{item.Event?.Message}</TableCell>
                  <TableCell className="p-0 w-10">
                    {isExecutionEvent ? (
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => toggleFilter(item.ExecutionID!)}
                      >
                        {filterExecutionID === item.ExecutionID ? (
                          <X size={16} />
                        ) : (
                          <Filter size={16} />
                        )}
                      </Button>
                    ) : (
                      <div className="h-10 w-10" /> // Placeholder for job events
                    )}
                  </TableCell>
                </TableRow>
              )
            })}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  )
}

export default JobHistory
