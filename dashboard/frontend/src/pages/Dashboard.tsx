import React, { FC, useCallback, useEffect, useState, useMemo } from 'react'
import bluebird from 'bluebird'
import {
  ComposedChart,
  Line,
  Area,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
  PieChart,
  Pie,
  Sector,
  Cell,
  RadialBarChart,
  RadialBar,
} from 'recharts'

import Box from '@mui/material/Box'
import Grid from '@mui/material/Grid'
import Container from '@mui/material/Container'
import Paper from '@mui/material/Paper'

import NumberHighlight from '../components/dashboard/NumberHighlight'
import AutoAwesomeMotionIcon from '@mui/icons-material/AutoAwesomeMotion'
import TimelineIcon from '@mui/icons-material/Timeline'
import PersonIcon from '@mui/icons-material/Person'
import CodeIcon from '@mui/icons-material/Code'

import useLoadingErrorHandler from '../hooks/useLoadingErrorHandler'
import useApi from '../hooks/useApi'

import {
  DashboardSummary,
} from '../types'

const blue = '#0088FE'
const green = '#00C49F'
const yellow = '#FFBB28'
const COLORS = [blue, green, yellow, '#FF8042'];

const Dashboard: FC = () => {
  const api = useApi()
  const loadingErrorHandler = useLoadingErrorHandler()

  const [ data, setData ] = useState<DashboardSummary>()

  const loadData = useCallback(async () => {
    const handler = loadingErrorHandler(async () => {
      const data: DashboardSummary = await bluebird.props({
        annotations: await api.get('/api/v1/summary/annotations'),
        jobMonths: await api.get('/api/v1/summary/jobmonths'),
        jobExecutors: await api.get('/api/v1/summary/jobexecutors'),
        totalJobs: await api.get('/api/v1/summary/totaljobs'),
        totalEvents: await api.get('/api/v1/summary/totaljobevents'),
        totalUsers: await api.get('/api/v1/summary/totalusers'),
        totalExecutors: await api.get('/api/v1/summary/totalexecutors'),
      })
      setData(data)
    })
    await handler()
  }, [])

  const barGraphData = useMemo(() => {
    if(!data?.jobMonths) return []
    const mappedData = data?.jobMonths.map((month) => ({
      name: month.month,
      jobs: month.count,
    }))
    return [{
      name: '',
      jobs: 0,
    }].concat(mappedData)
  }, [
    data,
  ])

  const pieChartData = useMemo(() => {
    if(!data?.jobExecutors) return []
    return data.jobExecutors.map((executor, index) => ({
      name: executor.executor,
      value: executor.count,
    }))
  }, [
    data,
  ])

  useEffect(() => {
    loadData()
  }, [])

  if(!data) return null

  return (
    <Container maxWidth={ 'xl' } sx={{ mt: 4, mb: 4 }}>
      <Grid container spacing={3}>
        <Grid item xs={12} md={3}>
          <NumberHighlight
            headline={ `${data.totalJobs.count}` }
            subline="Total Jobs"
            backgroundColor="#D1E9FC"
            textColor="#061B64"
          >
            <AutoAwesomeMotionIcon
              sx={{
                fontSize: '64px',
                mb: 2,
                color: '#061B64'
              }}
            />
          </NumberHighlight>
        </Grid>
        <Grid item xs={12} md={3}>
          <NumberHighlight
            headline={ `${data.totalEvents.count}` }
            subline="Total Events"
            backgroundColor="#D0F2FF"
            textColor="#264B90"
          >
            <TimelineIcon
              sx={{
                fontSize: '64px',
                mb: 2,
                color: '#264B90'
              }}
            />
          </NumberHighlight>
        </Grid>
        <Grid item xs={12} md={3}>
          <NumberHighlight
            headline={ `${data.totalUsers.count}` }
            subline="Unique Users"
            backgroundColor="#FFF7CD"
            textColor="#7F5509"
          >
            <PersonIcon
              sx={{
                fontSize: '64px',
                mb: 2,
                color: '#7F5509'
              }}
            />
          </NumberHighlight>
        </Grid>

        
        <Grid item xs={12} md={3}>
          <NumberHighlight
            headline={ `${data.totalExecutors.count}` }
            subline="Executors"
            backgroundColor="#FFE7D9"
            textColor="#7A0C2E"
          >
            <CodeIcon
              sx={{
                fontSize: '64px',
                mb: 2,
                color: '#7A0C2E'
              }}
            />
          </NumberHighlight>
        </Grid>

        <Grid item xs={12} md={8}>
          <Box
            component="div"
            sx={{
              height: '400px',
              backgroundColor: '#fff',
              borderRadius: '15px',
              padding: '20px',
            }}
          >
            <ResponsiveContainer width="100%" height="100%">
              <ComposedChart
                data={barGraphData}
                margin={{
                  top: 20,
                  right: 20,
                  bottom: 20,
                  left: 20,
                }}
              >
                <CartesianGrid stroke="#f5f5f5" />
                <XAxis dataKey="name" scale="band" />
                <YAxis />
                <Tooltip />
                <Legend />
                <Bar dataKey="jobs" barSize={20} fill={ blue } />
                <Line type="monotone" dataKey="jobs" stroke={ green } />
              </ComposedChart>
            </ResponsiveContainer>
          </Box>
        </Grid>

        <Grid item xs={12} md={4}>
          <Box
            component="div"
            sx={{
              height: '400px',
              backgroundColor: '#fff',
              borderRadius: '15px',
              padding: '20px',
            }}
          >
            <ResponsiveContainer width="100%" height="100%">
              <PieChart>
                <Pie
                  data={pieChartData}
                  innerRadius={40}
                  outerRadius={80}
                  fill="#8884d8"
                  paddingAngle={5}
                  dataKey="value"
                >
                  {pieChartData.map((entry, index) => (
                    <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                  ))}
                </Pie>
                <Legend
                  verticalAlign="bottom"
                  layout="vertical"
                  formatter={(value, entry: any, index) => {
                    return `${value} ${entry.payload?.value}`
                  }}
                />
              </PieChart>
            </ResponsiveContainer>
          </Box>
        </Grid>
      </Grid>
    </Container>
  )
}

export default Dashboard