import React, { FC, useState, useEffect, useMemo, useCallback } from 'react'
import bluebird from 'bluebird'
import { A, navigate, useQueryParams } from 'hookrouter'
import Grid from '@mui/material/Grid'
import Container from '@mui/material/Container'
import TextField from '@mui/material/TextField'
import Button from '@mui/material/Button'
import Box from '@mui/material/Box'
import IconButton from '@mui/material/IconButton'
import Tooltip from '@mui/material/Tooltip'
import FormControl from '@mui/material/FormControl'
import FormLabel from '@mui/material/FormLabel'
import FormGroup from '@mui/material/FormGroup'
import FormControlLabel from '@mui/material/FormControlLabel'
import Checkbox from '@mui/material/Checkbox'
import {
  DataGrid,
  GridColDef,
  GridSortModel,
  GridSortDirection,
} from '@mui/x-data-grid'

import {
  getShortId,
  getJobStateTitle,
} from '../utils/job'
import {
  Job,
  AnnotationSummary,
} from '../types'

import RefreshIcon from '@mui/icons-material/Refresh'
import InfoIcon from '@mui/icons-material/InfoOutlined';
import InputVolumes from '../components/job/InputVolumes'
import OutputVolumes from '../components/job/OutputVolumes'
import JobState from '../components/job/JobState'
import JobProgram from '../components/job/JobProgram'
import useLoadingErrorHandler from '../hooks/useLoadingErrorHandler'
import useApi from '../hooks/useApi'

const DEFAULT_PAGE_SIZE = 25
const PAGE_SIZES = [10, 25, 50, 100]

const columns: GridColDef[] = [
  {
    field: 'actions',
    headerName: 'Actions',
    width: 50,
    sortable: false,
    filterable: false,
    renderCell: (params: any) => {
      return (
        <Box
          sx={{
            display: 'flex',   
            justifyContent: 'flex-start',
            alignItems: 'center',
            width: '100%',
          }}
          component="div"
        >
          <IconButton
            component="label"
            onClick={ () => navigate(`/jobs/${params.row.job.Metadata.ID}`) }
          >
            <InfoIcon color="primary" />
          </IconButton>
        </Box>
      )
    },
  },
  {
    field: 'id',
    headerName: 'ID',
    width: 100,
    sortable: false,
    filterable: false,
    renderCell: (params: any) => {
      return (
        <span style={{
          fontSize: '0.8em'
        }}>
          <A href={`/jobs/${params.row.job.Metadata.ID}`}>{ getShortId(params.row.job.Metadata.ID) }</A>
        </span>
      )
    },
  },
  {
    field: 'date',
    headerName: 'Date',
    width: 120,
    sortable: true,
    filterable: false,
    renderCell: (params: any) => {
      return (
        <span style={{
          fontSize: '0.8em'
        }}>{ params.row.date }</span>
      )
    },
  },
  {
    field: 'inputs',
    headerName: 'Inputs',
    width: 260,
    sortable: false,
    filterable: false,
    renderCell: (params: any) => {
      return (
        <InputVolumes
          storageSpecs={ params.row.inputs }
          includeDatacap={ false }
        />
      )
    },
  },
  {
    field: 'program',
    headerName: 'Program',
    flex: 1,
    minWidth: 200,
    sortable: false,
    filterable: false,
    renderCell: (params: any) => {
      return (
        <JobProgram
          job={ params.row.job }
        />
      )
    },
  },
  {
    field: 'outputs',
    headerName: 'Outputs',
    width: 200,
    sortable: false,
    filterable: false,
    renderCell: (params: any) => {
      return (
        <A href={`/jobs/${params.row.job.ID}`} style={{color: '#333'}}>
          <OutputVolumes
            outputVolumes={ params.row.outputs }
            includeDatacap={ false }
          />
        </A>
      )
    },
  },
  {
    field: 'state',
    headerName: 'State',
    width: 140,
    sortable: false,
    filterable: false,
    renderCell: (params: any) => {
      return (
        <JobState
          job={ params.row.job }
        />
      )
    },
  },
]

const Jobs: FC = () => {
  const [ findJobID, setFindJobID ] = useState('')
  const [ jobs, setJobs ] = useState<Job[]>([])
  const [ jobsCount, setJobsCount ] = useState(0)
  const [ annotations, setAnnotations ] = useState<AnnotationSummary[]>([])
  const [ queryParams, setQueryParams ] = useQueryParams()
  const api = useApi()
  const loadingErrorHandler = useLoadingErrorHandler()

  // annoyingly hookrouter queryParams mutates it's object
  // so we need this to know if the query params have changed
  const qs = JSON.stringify(queryParams)

  const rows = useMemo(() => {
    return jobs.map(job => {
      const {
        inputs = [],
        outputs = [],
      } = job.Spec
      return {
        job,
        id: getShortId(job.Metadata.ID),
        date: new Date(job.Metadata.CreatedAt).toLocaleDateString() + ' ' + new Date(job.Metadata.CreatedAt).toLocaleTimeString(),
        inputs,
        outputs,
        shardState: getJobStateTitle(job),
      }
    })
  }, [
    jobs,
  ])

  const sortModel = useMemo<GridSortModel | undefined>(() => {
    if(!queryParams.sort_field || !queryParams.sort_order) return [{
      field: 'date',
      sort: 'desc' as GridSortDirection,
    }]
    return [{
      field: queryParams.sort_field,
      sort: queryParams.sort_order as GridSortDirection,
    }]
  }, [
    qs,
  ])

  const page_size = useMemo(() => {
    if(queryParams.page_size) {
      let t = parseInt(queryParams.page_size)
      return isNaN(t) ? DEFAULT_PAGE_SIZE : t
    } else {
      return DEFAULT_PAGE_SIZE
    }
  }, [
    qs,
  ])

  const activeAnnotations = useMemo<string[]>(() => {
    return queryParams.annotations ? queryParams.annotations.split(',') : []
  }, [
    qs,
    annotations,
  ])

  const updateAnnotation = useCallback((annotation: string, active: boolean) => {
    let newAnnotations = activeAnnotations.filter(a => a != annotation)
    if(active) {
      newAnnotations = [...newAnnotations, annotation]
    }    
    setQueryParams({
      annotations: newAnnotations.join(','),
    })
  }, [
    activeAnnotations,
  ])
  
  const loadAnnotations = useCallback(async () => {
    const data = await api.get('/api/v1/summary/annotations')
    setAnnotations(data)
  }, [])

  const loadJobs = useCallback(async (params: Record<string, string>) => {
    const handler = loadingErrorHandler(async () => {
      let page = parseInt(params.page)
      let page_size = parseInt(params.page_size)
      if (isNaN(page)) page = 0
      if (isNaN(page_size)) page_size = DEFAULT_PAGE_SIZE
      const activeAnnotations = queryParams.annotations ? queryParams.annotations.split(',') : []
      const query = {
        return_all: true,
        // we only really support sorting by date
        sort_by: 'created_at',
        sort_reverse: params.sort_order == 'asc' ? false : true,
        limit: page_size,
        offset: page * page_size,
        include_tags: activeAnnotations,
      }
      const {
        jobs,
        count,
      } = await bluebird.props({
        jobs: await api.post('/api/v1/jobs', query),
        count: await api.post('/api/v1/jobs/count', query),
      })
      setJobs(jobs)
      setJobsCount(count.count)
    })
    await handler()
  }, [])

  const reloadJobs = useCallback(async () => {
    loadJobs(queryParams)
  }, [
    queryParams,
  ])

  // the grid does this annoying
  const handleSortModelChange = useCallback(() => {
    setQueryParams({
      sort_field: 'date',
      sort_order: queryParams.sort_order == 'asc' ? 'desc' : 'asc',
    })
  }, [
    setQueryParams,
    qs,
  ])

  const handlePageChange = useCallback((page: number) => {
    setQueryParams({
      page,
    })
  }, [
    setQueryParams,
    qs,
  ])

  const handlePageSizeChange = useCallback((page_size: number) => {
    setQueryParams({
      page: 0,
      page_size,
    })
  }, [
    setQueryParams,
    qs,
  ])

  const findJob = useCallback(async () => {
    const handler = loadingErrorHandler(async () => {
      if(!findJobID) throw new Error(`please enter a job id`)
      try {
        const job = await api.get(`/api/v1/job/${findJobID}`) as Job
        navigate(`/jobs/${job.Metadata.ID}`)
      } catch(err: any) {
        throw new Error(`could not load job with id ${findJobID}: ${err.toString()}`)
      }
    })
    await handler()
  }, [
    findJobID,
  ])

  useEffect(() => {
    loadJobs(queryParams)
    loadAnnotations()
  }, [
    qs,
  ])

  return (
    <Container
      maxWidth={ 'xl' }
      sx={{
        mt: 4,
        mb: 4,
        height: '100%',
      }}
    >
      <Box
        component="div"
        sx={{
          display: 'flex',
          flexDirection: 'column',
          height: '100%',
        }}
      >
        <Box
          component="div"
          sx={{
            display: 'flex',
            flexDirection: 'row',
            alignItems: 'center',
            justifyContent: 'space-between',
          }}
        >
          <Box
            component="div"
            sx={{
              flexGrow: 1,
            }}
          >
            <TextField
              fullWidth
              size="small"
              label="Find Job by ID"
              value={ findJobID }
              sx={{
                backgroundColor: 'white',
              }}
              onChange={ (e) => setFindJobID(e.target.value) }
            />
          </Box>
          <Box
            component="div"
            sx={{
              flexGrow: 0,
              whiteSpace: 'nowrap'
            }}
          >
            <Button
              size="small"
              variant="outlined"
              sx={{
                height: '35px',
                ml: 2,
              }}
              onClick={ findJob }
            >
              Find&nbsp;Job
            </Button>

            <Tooltip title="Refresh">
              <IconButton aria-label="delete" color="primary" onClick={ reloadJobs }>
                <RefreshIcon />
              </IconButton>
            </Tooltip>
            
          </Box>
        </Box>
        <Box
          component="div"
          sx={{
            flexGrow: 1,
            mt: 2,
          }}
        >
          <div style={{ height: '100%', width: '100%' }}>
            <DataGrid
              rows={rows}
              rowCount={jobsCount}
              columns={columns}
              pageSize={page_size}
              rowsPerPageOptions={PAGE_SIZES}
              paginationMode="server"
              sortingMode="server"
              sortModel={sortModel}
              onSortModelChange={handleSortModelChange}
              onPageChange={handlePageChange}
              onPageSizeChange={handlePageSizeChange}
            />
          </div>
        </Box>
      </Box>
    </Container>
  )
}

export default Jobs