import React, { FC, useMemo } from 'react'
import { SxProps } from '@mui/system'
import Box from '@mui/material/Box'
import {
  Job,
} from '../../types'

import ShardState from './ShardState'

import {
  getJobShardState,
  getShardStateTitle,
} from '../../utils/job'

const JobState: FC<{
  job: Job,
  sx?: SxProps,
}> = ({
  job,
  sx = {},
}) => {
  const shardState = useMemo(() => {
    return getShardStateTitle(getJobShardState(job))
  }, [
    job,
  ])

  return (
    <Box
      component="div"
      sx={{
        display: 'flex',
        flexDirection: 'row',
        alignItems: 'center',
        justifyContent: 'flex-start',
        ...sx
      }}
    >
      <div
        style={{
          minWidth: '70px',
        }}
      >
        <ShardState
          state={ shardState }
        />
      </div>
    </Box>
  )
}

export default JobState
