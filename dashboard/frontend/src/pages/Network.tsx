import React, { FC, useState, useEffect, useCallback, useRef } from 'react'
import prettyBytes from 'pretty-bytes'
import Grid from '@mui/material/Grid'
import Container from '@mui/material/Container'
import useApi from '../hooks/useApi'
import useLoadingErrorHandler from '../hooks/useLoadingErrorHandler'
import Box from '@mui/material/Box'
import Card from '@mui/material/Card'
import CardActions from '@mui/material/CardActions'
import CardContent from '@mui/material/CardContent'
import Button from '@mui/material/Button'
import Typography from '@mui/material/Typography'
import {
  ClusterMapResult,
  NodeEvent,
} from '../types'
import {
  getShortId,
} from '../utils/job'
import {
  formatFloat,
  subtractFloat,
} from '../utils/format'

import ForceGraph from '../components/network/ForceGraph'

const Network: FC = () => {
  const [ mapData, setMapData ] = useState<ClusterMapResult>()
  const [ nodeData, setNodeData ] = useState<Record<string, NodeEvent>>({})
  const [ graphSize, setGraphSize ] = useState(0)
  const graphRef = useRef<HTMLDivElement>(null)
  const api = useApi()
  const loadingErrorHandler = useLoadingErrorHandler()

  const loadMapData = useCallback(async () => {
    const handler = loadingErrorHandler(async () => {
      const mapData = await api.get('/api/v1/nodes/map', {})
      setMapData(mapData)
    })
    await handler()
  }, [])

  const loadNodeData = useCallback(async () => {
    const handler = loadingErrorHandler(async () => {
      const nodeData = await api.get<Record<string, NodeEvent>>('/api/v1/nodes', {})
      if (nodeData) {
        Object.keys(nodeData).forEach(nodeID => {
          const originalData = nodeData[nodeID]
          const runningJobsDebugInfo = originalData.DebugInfo.find(info => info.component === 'running_jobs')
          originalData.RunningJobs = []
          if(runningJobsDebugInfo) {
            originalData.RunningJobs = JSON.parse(runningJobsDebugInfo.info)
          }
        })
        setNodeData(nodeData)
      }
    })
    await handler()
  }, [])

  const resizeGraph = useCallback(async () => {
    if(!graphRef.current) return
    setGraphSize(graphRef.current.clientWidth - 20)
  }, [])

  useEffect(() => {
    loadMapData()
    loadNodeData()
  }, [])

  useEffect(() => {
    resizeGraph()
  }, [])

  return (
    <Container maxWidth={ 'xl' } sx={{ mt: 4, mb: 4 }}>
      <Grid container spacing={3}>
        <Grid item xs={12} md={6}>
          <Box sx={{
            display: 'inline-block',
          }}>
            {
              nodeData && Object.keys(nodeData).map((nodeId, i) => {
                const node = nodeData[nodeId]
                return (
                  <Card sx={{ minWidth: 200, display: 'inline-block', m: 1 }} key={ i }>
                    <CardContent>
                      <Typography variant="h5" component="div">
                        { getShortId(node.NodeID) }
                      </Typography>
                      <ul>
                        <li>
                          <Typography variant="body2">
                            <span style={{minWidth: '110px', display: 'inline-block'}}>CPU Usage:</span>
                            <strong>{ subtractFloat(node.TotalCapacity.CPU, node.AvailableCapacity.CPU) }</strong>
                            &nbsp;/&nbsp;
                            <strong>{ formatFloat(node.AvailableCapacity.CPU) }</strong>
                          </Typography>
                        </li>
                        <li>
                          <Typography variant="body2">
                            <span style={{minWidth: '110px', display: 'inline-block'}}>Memory Usage:</span>
                            <strong>{ prettyBytes(subtractFloat(node.TotalCapacity.Memory, node.AvailableCapacity.Memory)) }</strong>
                            &nbsp;/&nbsp;
                            <strong>{ prettyBytes(node.AvailableCapacity.Memory || 0) }</strong>
                          </Typography>
                        </li>
                        <li>
                          <Typography variant="body2">
                            <span style={{minWidth: '110px', display: 'inline-block'}}>Disk Usage:</span>
                            <strong>{ prettyBytes(subtractFloat(node.TotalCapacity.Disk, node.AvailableCapacity.Disk)) }</strong>
                            &nbsp;/&nbsp;
                            <strong>{ prettyBytes(node.AvailableCapacity.Disk || 0) }</strong>
                          </Typography>
                        </li>
                        <li>
                          <Typography variant="body2">
                            <span style={{minWidth: '110px', display: 'inline-block'}}>GPU Usage:</span>
                            <strong>{ subtractFloat(node.TotalCapacity.GPU, node.AvailableCapacity.GPU) }</strong>
                            &nbsp;/&nbsp;
                            <strong>{ node.AvailableCapacity.GPU || 0 }</strong>
                          </Typography>
                        </li>
                        <li>
                          <Typography variant="body2">
                            <span style={{minWidth: '110px', display: 'inline-block'}}>Running Jobs:</span>
                            <strong>{ node.RunningJobs.length || 0 }</strong>
                          </Typography>
                        </li>
                      </ul>
                    </CardContent>
                  </Card>
                )
              })
            }
          </Box>
        </Grid>
        <Grid item xs={12} md={6} ref={graphRef}>
          {
            mapData && graphSize > 0 && (
              <ForceGraph
                data={ mapData }
                size={ graphSize }
              />
            )
          }
        </Grid>
      </Grid>
    </Container>
  )
}

export default Network