import React, { FC } from 'react'
import { styled } from '@mui/system'
import Typography from '@mui/material/Typography'

export const SmallText = styled('span')({
  fontSize: '0.8em',
})

export const TinyText = styled('span')({
  fontSize: '0.65em',
})

export const SmallLink = styled('div')({
  fontSize: '0.8em',
  color: 'blue',
  cursor: 'pointer',
  textDecoration: 'underline',
})

export const RequesterNode = styled('span')({
  fontWeight: 'bold',
  color: '#009900',
})

export const BoldSectionTitle: FC = ({
  children,
}) => {
  return (
    <Typography variant="subtitle1" sx={{
      fontWeight: 'bold',
    }}>
      { children }
    </Typography>
  )
}
