import React, { FC } from 'react'
import { SxProps } from '@mui/system'
import Typography from '@mui/material/Typography'
import Box from '@mui/material/Box'

const NumberHighlight: FC<{
  headline: string,
  subline: string,
  backgroundColor: string,
  textColor: string,
  sx?: SxProps,
}> = ({
  headline,
  subline,
  backgroundColor,
  textColor,
  children,
}) => {
  return (
    <Box
      component="div"
      sx={{
        width: '100%',
        borderRadius: '15px',
        height: '260px',
        backgroundColor: backgroundColor,
        display: 'flex',
        flexDirection: 'column',
        justifyContent: 'center',
        alignItems: 'center',
      }}
    >
      { children }
      <Typography
        variant="h3"
        sx={{
          fontWeight: 'bold',
          color: textColor,
        }}
      >
        { headline }
      </Typography>
      <Typography
        variant="subtitle1"
        sx={{
          fontWeight: 'bold',
          color: textColor,
        }}
      >
        { subline }
      </Typography>
    </Box>
  )
}

export default NumberHighlight
