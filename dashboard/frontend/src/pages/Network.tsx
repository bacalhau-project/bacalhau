import React, { FC, useState, useEffect, useCallback, useRef } from 'react'
import prettyBytes from 'pretty-bytes'
import Grid from '@mui/material/Grid'
import Container from '@mui/material/Container'
import useApi from '../hooks/useApi'
import useLoadingErrorHandler from '../hooks/useLoadingErrorHandler'
import Box from '@mui/material/Box'
import Card from '@mui/material/Card'
import CardContent from '@mui/material/CardContent'
import Typography from '@mui/material/Typography'
import {
  NodeInfo,
} from '../types'
import {
  getShortId,
} from '../utils/job'
import {
  formatFloat,
  subtractFloat,
} from '../utils/format'

const Network: FC = () => {
  const [ nodeData, setNodeData ] = useState<Record<string, NodeInfo>>({})
  const api = useApi()
  const loadingErrorHandler = useLoadingErrorHandler()

  const loadNodeData = useCallback(async () => {
    const handler = loadingErrorHandler(async () => {
      const nodeData = await api.get<Record<string, NodeInfo>>('/api/v1/nodes', {})
      if (nodeData) {
        setNodeData(nodeData)
      }
    })
    await handler()
  }, [])


  useEffect(() => {
    loadNodeData()
  }, [])

  return (
    <Container maxWidth={ 'xl' } sx={{ mt: 4, mb: 4 }}>
      <Grid container spacing={3}>
        <Grid item xs={12}>
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
                        { getShortId(node.PeerInfo.ID) }
                      </Typography>
                      <ul>
                        <li>
                          <Typography variant="body2">
                            <span style={{minWidth: '110px', display: 'inline-block'}}>CPU Usage:</span>
                            <strong>{ subtractFloat(node.ComputeNodeInfo.MaxCapacity.CPU, node.ComputeNodeInfo.AvailableCapacity.CPU) }</strong>
                            &nbsp;/&nbsp;
                            <strong>{ formatFloat(node.ComputeNodeInfo.MaxCapacity.CPU) }</strong>
                          </Typography>
                        </li>
                        <li>
                          <Typography variant="body2">
                            <span style={{minWidth: '110px', display: 'inline-block'}}>Memory Usage:</span>
                            <strong>{ prettyBytes(subtractFloat(node.ComputeNodeInfo.MaxCapacity.Memory, node.ComputeNodeInfo.AvailableCapacity.Memory)) }</strong>
                            &nbsp;/&nbsp;
                            <strong>{ prettyBytes(node.ComputeNodeInfo.MaxCapacity.Memory || 0) }</strong>
                          </Typography>
                        </li>
                        <li>
                          <Typography variant="body2">
                            <span style={{minWidth: '110px', display: 'inline-block'}}>Disk Usage:</span>
                            <strong>{ prettyBytes(subtractFloat(node.ComputeNodeInfo.MaxCapacity.Disk, node.ComputeNodeInfo.AvailableCapacity.Disk)) }</strong>
                            &nbsp;/&nbsp;
                            <strong>{ prettyBytes(node.ComputeNodeInfo.MaxCapacity.Disk || 0) }</strong>
                          </Typography>
                        </li>
                        <li>
                          <Typography variant="body2">
                            <span style={{minWidth: '110px', display: 'inline-block'}}>GPU Usage:</span>
                            <strong>{ subtractFloat(node.ComputeNodeInfo.MaxCapacity.GPU, node.ComputeNodeInfo.AvailableCapacity.GPU) }</strong>
                            &nbsp;/&nbsp;
                            <strong>{ node.ComputeNodeInfo.MaxCapacity.GPU || 0 }</strong>
                          </Typography>
                        </li>
                        <li>
                          <Typography variant="body2">
                            <span style={{minWidth: '110px', display: 'inline-block'}}>Running Jobs:</span>
                            <strong>{ node.ComputeNodeInfo.RunningExecutions || 0 }</strong>
                          </Typography>
                        </li>
                        <li>
                          <Typography variant="body2">
                            <span style={{minWidth: '110px', display: 'inline-block'}}>Enqueued Jobs:</span>
                            <strong>{ node.ComputeNodeInfo.EnqueuedExecutions || 0 }</strong>
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
      </Grid>
    </Container>
  )
}

export default Network