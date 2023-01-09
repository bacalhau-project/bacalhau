import React, { FC, useState, useEffect, useCallback, useContext, useMemo } from 'react'
import bluebird from 'bluebird'
import { A, navigate } from 'hookrouter'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Stack from '@mui/material/Stack'
import Grid from '@mui/material/Grid'
import Container from '@mui/material/Container'
import Typography from '@mui/material/Typography'
import Paper from '@mui/material/Paper'
import IconButton from '@mui/material/IconButton'
import Tooltip from '@mui/material/Tooltip'
import Divider from '@mui/material/Divider'
import FormControl from '@mui/material/FormControl'
import FormLabel from '@mui/material/FormLabel'
import RadioGroup from '@mui/material/RadioGroup'
import TextField from '@mui/material/TextField'
import FormControlLabel from '@mui/material/FormControlLabel'
import Radio from '@mui/material/Radio'
import RefreshIcon from '@mui/icons-material/Refresh'

import useApi from '../hooks/useApi'
import {
  JobInfo,
  JobModeration,
} from '../types'
import {
  getShortId,
  getJobStateTitle,
} from '../utils/job'
import InputVolumes from '../components/job/InputVolumes'
import OutputVolumes from '../components/job/OutputVolumes'
import JobState from '../components/job/JobState'
import ShardState from '../components/job/ShardState'
import JobProgram from '../components/job/JobProgram'
import FilPlus from '../components/job/FilPlus'
import CheckCircleIcon from '@mui/icons-material/CheckCircle'
import CancelIcon from '@mui/icons-material/Cancel'

import {
  SmallText,
  SmallLink,
  TinyText,
  BoldSectionTitle,
  RequesterNode,
} from '../components/widgets/GeneralText'
import TerminalWindow from '../components/widgets/TerminalWindow'
import useLoadingErrorHandler from '../hooks/useLoadingErrorHandler'
import { UserContext } from '../contexts/user'
import Window from '../components/widgets/Window'
import useSnackbar from '../hooks/useSnackbar'

type JSONWindowConfig = {
  title: string,
  data: any,
}

const InfoRow: FC<{
  title: string,
  rightAlign?: boolean,
  withDivider?: boolean,
}> = ({
  title,
  rightAlign = false,
  withDivider = false,
  children,
}) => {
  return (
    <>
      <Grid item xs={3}>
        <Typography variant="caption">
          { title }:
        </Typography>
      </Grid>
      <Grid item xs={9} sx={{
        pl: 8,
        display: 'flex',
        alignItems: 'center',
        justifyContent: rightAlign ? 'flex-end' : 'flex-start',
      }}>
        { children }
      </Grid>
      {
        withDivider && (
          <Grid item xs={12}>
            <Divider sx={{
              mt: 1,
              mb: 1,
            }} />
          </Grid>
        )
      }
    </>
  )
}

const JobPage: FC<{
  id: string,
}> = ({
  id,
}) => {
  const user = useContext(UserContext)
  const snackbar = useSnackbar()
  const [ moderationWindowOpen, setModerationWindowOpen ] = useState(false)
  const [ jobInfo, setJobInfo ] = useState<JobInfo>()
  const [ jsonWindow, setJsonWindow ] = useState<JSONWindowConfig>()
  const [ moderationResult, setModerationResult ] = useState('')
  const [ moderationNotes, setModerationNotes ] = useState('')
  const api = useApi()
  const loadingErrorHandler = useLoadingErrorHandler()

  const nodeStateIDs = useMemo(() => {
    if(!jobInfo) return []
    const cancelledNodeIDs: string[] = []
    const nonCancelledNodeIDs: string[] = []
    Object.keys(jobInfo.state.Nodes).map(nodeID => {
      const nodeState = jobInfo.state.Nodes[nodeID]
      let seenCancelledShard = false
      Object.keys(nodeState.Shards).map((shardIndex, i) => {
        const shardState = nodeState.Shards[shardIndex as unknown as number]
        if(shardState.State == 'Cancelled') {
          seenCancelledShard = true
        }
      })
      if(seenCancelledShard) {
        cancelledNodeIDs.push(nodeID)
      }
      else {
        nonCancelledNodeIDs.push(nodeID)
      }
    })
    return nonCancelledNodeIDs.concat(cancelledNodeIDs)
  }, [
    jobInfo,
  ])

  const isRequesterNodeID = useCallback((id: string): boolean => {
    if(!jobInfo) return false
    return jobInfo.job.Status.Requester.RequesterNodeID == id
  }, [
    jobInfo,
  ])

  const loadInfo = useCallback(async () => {
    const handler = loadingErrorHandler(async () => {
      const info = await api.get(`/api/v1/job/${id}/info`)
      setJobInfo(info)
    })
    await handler()
  }, [])

  const submitModeration = useCallback(async () => {
    if(!user.user) return
    const data: Partial<JobModeration> = {
      job_id: id,
      user_account_id: user.user.id,
      status: moderationResult,
      notes: moderationNotes,
    }
    const result = await api.post('/api/v1/admin/moderate', data)
    if(!result) {
      snackbar.error('failed to assign datacap')
      return
    }
    await loadInfo()
    setModerationWindowOpen(false)
    snackbar.success('datacap has been assigned to this job')
  }, [
    id,
    user,
    moderationResult,
    moderationNotes,
    loadInfo,
  ])

  const closeModeration = useCallback(async () => {
    setModerationResult('')
    setModerationNotes('')
    setModerationWindowOpen(false)
  }, [])

  useEffect(() => {
    loadInfo()
  }, [])

  if(!jobInfo) return null

  return (
    <Container maxWidth={ 'xl' } sx={{ mt: 4, mb: 4 }}>
      <Grid container spacing={3}>
        <Grid item md={12} lg={4}>
          <Paper
            sx={{
              p: 2,
            }}
          >
            <Grid container spacing={1}>
              <Grid item xs={6}>
                <BoldSectionTitle>
                  Job Details
                </BoldSectionTitle>
              </Grid>
              <Grid item xs={6} sx={{
                display: 'flex',
                justifyContent: 'flex-end',
              }}>
                <Tooltip title="Refresh">
                  <IconButton aria-label="delete" color="primary" onClick={ loadInfo }>
                    <RefreshIcon />
                  </IconButton>
                </Tooltip>
              </Grid>
              <InfoRow title="ID">
                <SmallText>
                  { jobInfo.job.Metadata.ID }
                </SmallText>
              </InfoRow>
              <InfoRow title="Date">
                <SmallText>
                  { new Date(jobInfo.job.Metadata.CreatedAt).toLocaleDateString() + ' ' + new Date(jobInfo.job.Metadata.CreatedAt).toLocaleTimeString()}
                </SmallText>
              </InfoRow>
              <InfoRow title="Concurrency">
                <SmallText>
                  { jobInfo.job.Spec.Deal.Concurrency }
                </SmallText>
              </InfoRow>
              <InfoRow title="Shards">
                <SmallText>
                { jobInfo.job.Spec.ExecutionPlan.ShardsTotal }
                </SmallText>
              </InfoRow>
              <InfoRow title="State" withDivider>
                <JobState
                  job={ jobInfo.job }
                />
              </InfoRow>
              <InfoRow title="Inputs" withDivider>
                <InputVolumes
                  storageSpecs={ jobInfo.job.Spec.inputs || [] }
                />
              </InfoRow>
              <Grid item xs={12} sx={{
                direction: 'column',
                display: 'flex',
                justifyContent: 'center',
              }}>
                <Box
                  sx={{
                    cursor: 'pointer',
                  }}
                  onClick={() => setJsonWindow({
                    title: 'Program',
                    data: jobInfo.job.Spec,
                  })}
                >
                  <JobProgram
                    job={ jobInfo.job }
                  />
                </Box>
                <br />
                
              </Grid>
              <Grid item xs={12} sx={{
                direction: 'column',
                display: 'flex',
                justifyContent: 'center',
              }}>
                <SmallLink
                  onClick={() => setJsonWindow({
                    title: 'Program',
                    data: jobInfo.job.Spec,
                  })}
                >
                  view info
                </SmallLink>
              </Grid>
              <Grid item xs={12}>
                <Divider sx={{
                  mt: 1,
                  mb: 1,
                }} />
              </Grid>
              <InfoRow title="Outputs" withDivider>
                <OutputVolumes
                  outputVolumes={ jobInfo.job.Spec.outputs || [] }
                />
              </InfoRow>
              <InfoRow title="Annotations" withDivider>
                <Stack direction="row">
                  <Box
                    component="div"
                    sx={{
                      width: '100%',
                      mr: 1,
                    }}
                  >
                    {
                      (jobInfo.job.Spec.Annotations || []).map((annotation, index) => (
                        <li
                          key={ index }
                          style={{
                            fontSize: '0.8em',
                            color: '#333',
                          }}
                        >
                          { annotation }
                        </li>
                      ))
                    }
                  </Box>
                </Stack>
              </InfoRow>
            </Grid>
          </Paper>
          <Paper
            sx={{
              p: 2,
              mt: 2,
            }}
          >
            <Grid container spacing={1}>
              <Grid item xs={6}>
                <BoldSectionTitle>
                  Moderation
                </BoldSectionTitle>
              </Grid>
              <Grid item xs={6} sx={{
                display: 'flex',
                justifyContent: 'flex-end',
              }}>
                <FilPlus />
              </Grid>
              <Grid item xs={12}>
                <Divider sx={{
                  mt: 1,
                  mb: 1,
                }} />
              </Grid>
              <Grid item xs={12}>
                {
                  jobInfo.moderation.moderation ? (
                    <Stack
                      direction="row"
                      alignItems="center"
                    >
                      {
                        jobInfo.moderation.moderation.status == 'yes' ? (
                          <CheckCircleIcon
                            sx={{
                              fontSize: '4em',
                              color: 'green'
                            }}
                          />
                        ) : (
                          <CancelIcon
                            sx={{
                              fontSize: '4em',
                              color: 'red'
                            }}
                          />
                        )
                      }
                      <Typography variant="caption" sx={{
                        color: '#666',
                        ml: 2,
                      }}>
                        Moderated by <strong>{ jobInfo.moderation.user.username }</strong> on { new Date(jobInfo.moderation.moderation.created).toLocaleDateString() + ' ' + new Date(jobInfo.moderation.moderation.created).toLocaleTimeString() }
                        <br />
                        { jobInfo.moderation.moderation.notes || null }
                      </Typography>
                    </Stack>
                  ) : (
                    <Typography variant="caption" sx={{
                      color: '#666'
                    }}>
                      This job has not been moderated yet
                    </Typography>
                  )
                }
              </Grid>
              {
                user.user && (
                  <>
                    <Grid item xs={12}>
                      <Divider sx={{
                        mt: 1,
                        mb: 1,
                      }} />
                    </Grid>
                    <Grid item xs={12}>
                      <Button
                        variant="outlined"
                        color="primary"
                        disabled={ jobInfo.moderation.moderation ? true : false }
                        onClick={ () => {
                          setModerationWindowOpen(true)
                        }}
                      >
                        Moderate Job
                      </Button>
                    </Grid>
                  </>
                )
              }
            </Grid>
          </Paper>
        </Grid>
        <Grid item md={12} lg={4}>
          <Paper
            sx={{
              p: 2,
              mb: 2,
            }}
          >
            <Grid container spacing={1}>
              <Grid item xs={12}>
                <BoldSectionTitle>
                  Nodes
                </BoldSectionTitle>
              </Grid>
              <Grid item xs={3}>
                <Typography variant="caption">
                  Requester Node:
                </Typography>
              </Grid>
              <Grid item xs={9}>
                <SmallText>
                  <RequesterNode>
                    { getShortId(jobInfo.job.Status.Requester.RequesterNodeID) }
                  </RequesterNode>
                </SmallText>
              </Grid>
            </Grid>
          </Paper>
          {
            nodeStateIDs.map(nodeID => {
              const nodeState = jobInfo.state.Nodes[nodeID]
              return (
                <Paper
                  key={ nodeID }
                  sx={{
                    p: 2,
                    mb: 2,
                  }}
                >
                  <Grid container spacing={0.5}>
                    <Grid item xs={12}>
                      <BoldSectionTitle>
                        <A href="/network">
                          { getShortId(nodeID) }
                        </A>
                      </BoldSectionTitle>
                    </Grid>
                    {
                      Object.keys(nodeState.Shards).map((shardIndex, i) => {
                        const shardState = nodeState.Shards[shardIndex as unknown as number]
                        return (
                          <React.Fragment key={ shardIndex }>
                            <InfoRow title="Shard Index">
                              <SmallText>
                                { shardIndex }
                              </SmallText>
                            </InfoRow>
                            <InfoRow title="State">
                              <SmallText>
                                <ShardState state={ shardState.State } />
                              </SmallText>
                            </InfoRow>
                            {
                              shardState.RunOutput && (
                                <>
                                  <InfoRow title="Status">
                                    <TinyText>
                                      exitCode: { shardState.RunOutput?.exitCode } &nbsp;
                                      <span style={{color:'#999'}}>{ shardState.Status }</span>
                                    </TinyText>
                                  </InfoRow>
                                  {
                                    shardState.RunOutput?.stdout && (
                                      <InfoRow title="stdout">
                                        <TinyText>
                                          <span style={{color:'#999'}}>{ shardState.RunOutput?.stdout }</span>
                                        </TinyText>
                                      </InfoRow>
                                    )
                                  }
                                  {
                                    shardState.RunOutput?.stderr && (
                                      <InfoRow title="stderr">
                                        <TinyText>
                                          <span style={{color:'#999'}}>{ shardState.RunOutput?.stderr }</span>
                                        </TinyText>
                                      </InfoRow>
                                    )
                                  }
                                  <InfoRow title="Outputs" withDivider={ i < Object.keys(nodeState.Shards).length - 1 }>
                                    <OutputVolumes
                                      outputVolumes={ jobInfo.job.Spec.outputs || [] }
                                      publishedResults={ shardState.PublishedResults }
                                    />
                                  </InfoRow>
                                </>
                              )
                            }
                          </React.Fragment>
                        )
                      })
                    }
                  </Grid>
                </Paper>
              )
            })
          }
        </Grid>
        <Grid item md={12} lg={4}>
          <Paper
            sx={{
              p: 2,
            }}
          >
            <Grid container spacing={0.5}>
              <Grid item xs={8}>
                <BoldSectionTitle>
                  Events
                </BoldSectionTitle>
              </Grid>
              <Grid item xs={4} sx={{
                display: 'flex',
                justifyContent: 'flex-end',
              }}>
                <SmallLink
                  onClick={() => setJsonWindow({
                    title: 'Events',
                    data: jobInfo.events,
                  })}
                >
                  view all
                </SmallLink>
              </Grid>
              <Grid item xs={4}>
                <SmallText>
                  <strong>Node</strong>
                </SmallText>
              </Grid>
              <Grid item xs={4}>
              <SmallText>
                  <strong>Event</strong>
                </SmallText>
              </Grid>
              <Grid item xs={4}>
                <SmallText>
                  <strong>Date</strong>
                </SmallText>
              </Grid>
              {
                jobInfo.events.map((event, i) => {
                  return (
                    <React.Fragment key={ i }>
                      <Grid item xs={4}>
                        <SmallText>
                          {
                            isRequesterNodeID(event.SourceNodeID) && (event.TargetNodeID || event.EventName == 'Created') ? (
                              <RequesterNode>
                                { getShortId(event.SourceNodeID) }
                              </RequesterNode>
                            ) : getShortId(event.SourceNodeID)
                          }
                        </SmallText>
                      </Grid>
                      <Grid item xs={4}>
                        <SmallLink
                          onClick={() => setJsonWindow({
                            title: 'Event',
                            data: event,
                          })}
                        >
                          { event.EventName }
                        </SmallLink>
                      </Grid>
                      <Grid item xs={4}>
                        <TinyText>
                          { new Date(event.EventTime).toLocaleDateString() + ' ' + new Date(event.EventTime).toLocaleTimeString()}
                        </TinyText>
                      </Grid>
                      
                    </React.Fragment>
                  )
                })
              }
            </Grid>
          </Paper>
        </Grid>
      </Grid>
      {
        jsonWindow && (
          <TerminalWindow
            open
            title={ jsonWindow.title }
            backgroundColor="#fff"
            color="#000"
            data={ jsonWindow.data }
            onClose={ () => setJsonWindow(undefined) }
          />
        )
      }
      {
        moderationWindowOpen && (
          <Window
            open
            size="md"
            title="Moderate Job"
            submitTitle="Confirm"
            withCancel
            onCancel={ closeModeration }
            onSubmit={ submitModeration }
          >
            <Grid container spacing={ 0 }>
              <Grid item xs={ 12 }>
                <FormControl>
                  <FormLabel>Award Datacap To This Job?</FormLabel>
                  <RadioGroup
                    row
                    value={ moderationResult }
                    onChange={ (e) => setModerationResult(e.target.value) }
                  >
                    <FormControlLabel value="yes" control={<Radio />} label="Yes" />
                    <FormControlLabel value="No" control={<Radio />} label="No" />
                  </RadioGroup>
                </FormControl>
              </Grid>
              <Grid item xs={ 12 }>
                <Typography gutterBottom variant="caption">
                  The compute node that publishes the results will be awarded Datacap if they make a deal for those results.
                </Typography>
              </Grid>
              <Grid item xs={ 12 } sx={{
                mt: 4,
              }}>
                <TextField
                  label="Notes"
                  fullWidth
                  multiline
                  rows={4}
                  value={ moderationNotes }
                  onChange={ (e) => setModerationNotes(e.target.value) }
                />
              </Grid>
            </Grid>
          </Window>
        )
      }
    </Container>
  )
}

export default JobPage

