import {
  Job,
  JobShardState,
} from '../types'

export const getShortId = (id: string, length = 8) => {
  return id.slice(0, length)
}

export const getJobShardState = (job: Job): JobShardState | undefined => {
  const nodeStates = job.Status.JobState.Nodes || {}
  const nonEmptyStates = Object.keys(nodeStates)
    .filter(nodeID => {
      const state = nodeStates[nodeID]
      if(!state) return false
      const shardState = (state.Shards || {})['0']
      if(!shardState) return false
      return shardState.State === 'Cancelled' ? false : true
    })
    .map(nodeID => nodeStates[nodeID].Shards['0'])
  return nonEmptyStates[0]
}

export const getShardStateTitle = (shardState: JobShardState | undefined): string => {
  return shardState ?
    shardState.State :
    'Unknown'
}

export const getJobStateTitle = (job: Job) => getShardStateTitle(getJobShardState(job))