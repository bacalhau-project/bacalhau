import React, { FC, useMemo } from 'react'
import { SxProps } from '@mui/system'
import Box from '@mui/material/Box'
import Typography from '@mui/material/Typography'

const ShardState: FC<{
  state: string,
  sx?: SxProps,
}> = ({
  state,
  children,
  sx = {},
}) => {
  const shardState = useMemo(() => {
    let color = '#666'
    if(state == 'Error') {
      color = '#990000'
    } else if(state == 'Completed') {
      color = '#009900'
    }
    return (
      <Typography variant="caption" style={{color}} sx={{
        pr: 2,
      }}>
        { state }
      </Typography>
    )
  }, [
    state,
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
      {
        shardState
      }
      {
        children
      }
    </Box>
  )
}

export default ShardState
