import React, { FC, useState, useEffect, useCallback, useContext, useMemo } from 'react'
import { A } from 'hookrouter'
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
import RefreshIcon from '@mui/icons-material/Refresh'

import useApi from '../hooks/useApi'
import {
  JobInfo, JobRelation,
  ModerateRequest,
  ModerationType,
} from '../types'
import {
  getShortId,
} from '../utils/job'
import InputVolumes from '../components/job/InputVolumes'
import OutputVolumes from '../components/job/OutputVolumes'
import JobState from '../components/job/JobState'
import ShardState from '../components/job/ShardState'
import JobProgram from '../components/job/JobProgram'
import FilPlus from '../components/job/FilPlus'

import {
  SmallText,
  SmallLink,
  TinyText,
  BoldSectionTitle,
  RequesterNode,
} from '../components/widgets/GeneralText'
import Accordion from "@mui/material/Accordion"
import AccordionDetails from "@mui/material/AccordionDetails"
import AccordionSummary from "@mui/material/AccordionSummary"
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import Link from "@mui/material/Link";
import ModerationPanel from '../components/widgets/ModerationSummary'
import ModerationWindow from '../components/widgets/ModerationWindow'
import Table from "@mui/material/Table";
import TableBody from "@mui/material/TableBody";
import TableCell from "@mui/material/TableCell"
import TableContainer from "@mui/material/TableContainer"
import TableRow from "@mui/material/TableRow";
import TerminalWindow from '../components/widgets/TerminalWindow'
import useLoadingErrorHandler from '../hooks/useLoadingErrorHandler'
import useSnackbar from '../hooks/useSnackbar'
import { UserContext } from '../contexts/user'
import {styled} from "@mui/system";


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
  const [ datacapWindowOpen, setDatacapWindowOpen ] = useState(false)
  const [ execModWindowOpen, setExecModWindowOpen ] = useState(false)

  const [ jobInfo, setJobInfo ] = useState<JobInfo>()
  const [ jobOutputRelation, setJobOutputRelation] = useState<JobRelation[]>([]);
  const [jobInputRelation, setJobInputRelation] = useState<JobRelation[]>([]);

  const [ jsonWindow, setJsonWindow ] = useState<JSONWindowConfig>()
  const api = useApi()
  const loadingErrorHandler = useLoadingErrorHandler()

  const nodeStateIDs = useMemo(() => {
    if(!jobInfo) return []
    const cancelledNodeIDs: string[] = []
    const nonCancelledNodeIDs: string[] = []
    Object.keys(jobInfo.state.Nodes || []).map(nodeID => {
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
  }, [
      id,
  ])

  const loadInputRelationInfo = useCallback(async () => {
    const handler = loadingErrorHandler(async () => {
      const info = await api.get(`/api/v1/job/${id}/inputs`);
      setJobInputRelation(info);
    });
    await handler();
  }, [
     id,
  ]);

  const loadOutputRelationInfo = useCallback(async () => {
    const handler = loadingErrorHandler(async () => {
      const info = await api.get(`/api/v1/job/${id}/outputs`);
      setJobOutputRelation(info);
    });
    await handler();
  }, [
     id,
  ]);

  const groupByCID = (jobRelation) => {
    const groups = {};

    if (jobRelation) {
      jobRelation.forEach((relation) => {
        const cid = relation.cid;
        if (!groups[cid]) {
          groups[cid] = [];
        }
        groups[cid].push(relation);
      });
    }

    return groups;
  };

  const submitDatacapModeration = useCallback(async (approved: boolean, reason: string) => {
    if(!user.user) return
    const data: ModerateRequest = { approved: approved, reason: reason }
    const result = await api.post(`/api/v1/job/${id}/datacap`, data)
    if(!result) {
      snackbar.error('failed to assign datacap')
      return
    }
    await loadInfo()
    const not = approved ? " " : "not "
    snackbar.success(`Datacap has ${not}been assigned to this job.`)
  }, [
    id,
    user,
    loadInfo,
  ])

  const submitExecModeration = useCallback(async (approved: boolean, reason: string) => {
    if(!user.user) return
      const data: ModerateRequest = { approved: approved, reason: reason }
      const result = await api.post(`/api/v1/job/${id}/exec`, data)
      if(!result) {
        snackbar.error('Failed to moderate execution')
        return
      }
      await loadInfo()
      const not = approved ? " " : "not "
      snackbar.success(`Job will ${not}be scheduled.`)
    }, [
      id,
      user,
      loadInfo,
  ])

  const closeDatacapModeration = useCallback(async () => { setDatacapWindowOpen(false) }, [])
  const closeExecModeration = useCallback(async () => { setExecModWindowOpen(false) }, [])

  useEffect(() => {
    loadInfo();
    loadInputRelationInfo();
    loadOutputRelationInfo();
  }, [id]);

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
          <Paper sx={{ p: 2, mt: 2, }}>
            <ModerationPanel
              moderationType={ModerationType.Datacap}
              moderations={jobInfo.moderations}
              user={user}
              icon={<FilPlus/>}
              onClick={async () => { setDatacapWindowOpen(true) }}
            />
          </Paper>
          <Paper sx={{ p: 2, mt: 2, }}>
            <ModerationPanel
              moderationType={ModerationType.Execution}
              moderations={jobInfo.moderations}
              user={user}
              icon={<div/>}
              onClick={async () => { setExecModWindowOpen(true) }}
            />
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
        datacapWindowOpen && (
          <ModerationWindow
            open
            title="Award Datacap To This Job?"
            prompt="The compute node that publishes the results will be awarded Datacap if they make a deal for those results."
            onCancel={closeDatacapModeration}
            onSubmit={closeDatacapModeration}
            onModerate={submitDatacapModeration}
          />
        )
      }
      {
        execModWindowOpen && (
          <ModerationWindow
            open
            title="Allow this job to be executed?"
            prompt="The job will be scheduled on an appropriate compute node."
            onCancel={closeExecModeration}
            onSubmit={closeExecModeration}
            onModerate={submitExecModeration}
          />
        )
      }
      <Grid container spacing={3}>
        <Grid item xs={12}>
          <Grid container spacing={3}>
            <Grid item xs={12} lg={4}>
                <Typography variant="h6">Job(s) Producing Input</Typography>
                {Object.keys(groupByCID(jobInputRelation)).length > 0 ? (
                    Object.entries(groupByCID(jobInputRelation)).map(([cid, relations]) => (
                        <Accordion key={cid}>
                          <AccordionSummary expandIcon={<ExpandMoreIcon />}>
                            <Typography variant="subtitle1">CID: {cid}</Typography>
                          </AccordionSummary>
                          <AccordionDetails>
                            <TableContainer>
                              <Table sx={{ minWidth: 50 }} size="small">
                                <TableBody>
                                  {relations.map((relation, index) => (
                                      <TableRow key={index}>
                                        <TableCell>
                                          <SmallText>
                                            <Link href={`/jobs/${relation.job_id}`} onClick={loadInfo}>
                                              {relation.job_id}
                                            </Link>
                                          </SmallText>
                                        </TableCell>
                                      </TableRow>
                                  ))}
                                </TableBody>
                              </Table>
                            </TableContainer>
                          </AccordionDetails>
                        </Accordion>
                    ))
                ) : (
                    <Typography variant="body1">No job relations found.</Typography>
                )}
            </Grid>
            <Grid item xs={12} lg={4}>
                <Typography variant="h6">Job(s) Operating on Output</Typography>
                {Object.keys(groupByCID(jobOutputRelation)).length > 0 ? (
                    Object.entries(groupByCID(jobOutputRelation)).map(([cid, relations]) => (
                        <Accordion key={cid}>
                          <AccordionSummary expandIcon={<ExpandMoreIcon />}>
                            <Typography variant="subtitle1">{cid}</Typography>
                          </AccordionSummary>
                          <AccordionDetails>
                            <TableContainer>
                              <Table sx={{ minWidth: 50 }} size="small">
                                <TableBody>
                                  {relations.map((relation, index) => (
                                      <TableRow key={index}>
                                        <TableCell>
                                          <SmallText>
                                            <Link href={`/jobs/${relation.job_id}`} onClick={loadInfo}>
                                              {relation.job_id}
                                            </Link>
                                          </SmallText>
                                        </TableCell>
                                      </TableRow>
                                  ))}
                                </TableBody>
                              </Table>
                            </TableContainer>
                          </AccordionDetails>
                        </Accordion>
                    ))
                ) : (
                    <Typography variant="body1">No job relations found.</Typography>
                )}
            </Grid>
          </Grid>
        </Grid>
      </Grid>
    </Container>
  )
}

export default JobPage
