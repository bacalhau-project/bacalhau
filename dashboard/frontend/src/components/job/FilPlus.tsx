import React, { FC, useMemo } from 'react'
import { SxProps } from '@mui/system'
import Box from '@mui/material/Box'

const FilPlus: FC<{
  sx?: SxProps,
  imgSize?: number,
  fontSize?: string,
}> = ({
  sx = {},
  imgSize = 24,
  fontSize = '1em',
}) => {
  return (
    <Box
      component="div"
      sx={{
        display: 'flex',
        flexDirection: 'row',
        alignItems: 'center',
        justifyContent: 'center',
        mr: 1,
        mb: 0.5,
        ...sx
      }}
    > 
      <img
        style={{
          width: `${imgSize}px`,
          height: `${imgSize}px`,
        }}
        src="/img/filecoin-logo.png" alt="Filecoin Plus"
      />
      <span style={{fontSize, fontWeight: 'bold', marginTop: '1px', marginLeft: '3px', paddingBottom: '0px'}}>
      +
      </span>
    </Box>
  )
}

export default FilPlus
