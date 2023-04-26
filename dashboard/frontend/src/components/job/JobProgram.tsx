import React, { FC, useMemo } from 'react'
import { SxProps } from '@mui/system'
import Box from '@mui/material/Box'
import Typography from '@mui/material/Typography'
import {
  Job,
} from '../../types'
import {
  getShortId,
} from '../../utils/job'

const JobProgram: FC<{
  job: Job,
  sx?: SxProps,
  imgSize?: number,
  fontSize?: string,
}> = ({
  job,
  sx = {},
  imgSize = 36,
  fontSize = '1em',
}) => {
  const engineLogo = useMemo(() => {
    const imageSrc = `/img/${job.Spec.Engine.toLowerCase()}-logo.png`
    return (
        <img
          style={{
            width: `${imgSize}px`,
            marginRight: '10px',
          }}
          src={imageSrc}
          alt={job.Spec.Engine}
        />
      )
  }, [
    job,
  ])

  const programDetails = useMemo(() => {
    if (job.Spec.Engine == "Docker") {
      const image = job.Spec.Docker?.Image || ''
      const entrypoint = job.Spec.Docker?.Entrypoint || []
      const details = `${image} ${(entrypoint || []).join(' ')}`
      return (
        <div>
          <div>
            <Typography variant="caption" style={{fontWeight: 'bold'}}>
              { image }
            </Typography>
          </div>
          <div>
            <Typography variant="caption" style={{color: '#666'}}>
              { (entrypoint || []).join(' ') }
            </Typography>
          </div>
        </div>
      )
    } else if (job.Spec.Engine == "Wasm") {
      let programCID = job.Spec.Wasm.EntryModule?.CID || ''
      let entryPoint = job.Spec.Wasm.EntryPoint
      let params = job.Spec.Wasm.Parameters
      let useUrl = `http://ipfs.io/ipfs/${programCID}`
      return (
        <div>
          <a href={ useUrl } target="_blank" rel="noreferrer" style={{color: 'black'}}>
            <Typography variant="caption" style={{fontWeight: 'bold', color: 'black'}}>
                { getShortId(programCID, 16) }:{ entryPoint }
            </Typography>
          </a>
          <div>
            <Typography variant="caption" style={{color: '#666'}}>
              { (params || []).join(' ') }
            </Typography>
          </div>
        </div>
      )
    } else {
      return 'unknown'
    }
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
      <div>
        { engineLogo }
      </div>
      <div>
        { programDetails }
      </div>


    </Box>
  )
}

export default JobProgram
