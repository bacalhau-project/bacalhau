import React, { FC } from 'react'
import Box from '@mui/material/Box'

export type TerminalTextConfig = {
  backgroundColor?: string,
  color?: string,
}

const TerminalText: FC<{
  data: any,
} & TerminalTextConfig> = ({
  data,
  backgroundColor = '#000',
  color = '#fff',
}) => {
  return (
    <Box
      component="div"
      sx={{
        width: '100%',
        padding: 2,
        margin: 0,
        backgroundColor,
        overflow: 'auto',
      }}
    >
      <Box
        component="pre"
        sx={{
          padding: 1,
          margin: 0,
          color,
          font: 'Courier',
          fontSize: '12px',
        }}
      >
        { typeof(data) === 'string' ? data : JSON.stringify(data, null, 4) }
      </Box>
    </Box>
  )
}

export default TerminalText
