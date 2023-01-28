import React, { FC, useState, useRef, useCallback, useEffect } from 'react'
import { SxProps } from '@mui/system'
import Box from '@mui/material/Box'

const HeightDiv: FC<{
  percent?: number,
  sx?: SxProps,
}> = ({
  percent = 100,
  sx = {},
  children,
}) => {

  const mounted = useRef(true)
  const [ width, setWidth ] = useState(0)
  const ref = useRef<HTMLDivElement>()

  const calculateWidth = () => {
    if(!mounted.current || !ref.current) return
    setWidth(ref.current.offsetWidth * percent)
  }

  useEffect(() => {
    const handleResize = () => calculateWidth()
    handleResize()
    window.addEventListener('resize', handleResize)
    return () => window.removeEventListener('resize', handleResize)
  }, [])

  useEffect(() => {
    mounted.current = true
    return () => {
      mounted.current = false
    }
  }, [])

  return (
    <Box
      component="div"
      sx={{
        width: '100%',
        height: `${width}px`,
        ...sx
      }}
      ref={ ref }
    >
      { children }
    </Box>
  )
}

export default HeightDiv
