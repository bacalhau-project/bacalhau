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
    if (job.Spec.Engine == "Docker") {
      return (
        <img
          style={{
            width: `${imgSize}px`,
            marginRight: '10px',
          }}
          src="/img/docker-logo.png" alt="Docker"
        />
      )
    } else if(job.Spec.Engine == "Wasm") {
      return (
        <img
          style={{
            width: `${imgSize}px`,
            height: `${imgSize}px`,
          }}
          src="/img/wasm-logo.png" alt="WASM"
        />
      )
    }
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
      let programCID = ''
      let entryPoint = job.Spec.Wasm.EntryPoint
      let useUrl = ''
      if (job.Spec.Contexts.length > 0) {
        programCID = job.Spec.Contexts[0].CID || ''
        useUrl = `http://ipfs.io/ipfs/${programCID}` 
      }
      return (
        <Box component="div" sx={{
          pl: 1,
        }}>
          <Typography variant="caption" style={{fontWeight: 'bold'}}>
              <a
                href={ useUrl }
                target="_blank"
                rel="noreferrer"
                style={{
                  fontSize: '0.8em',
                  color: '#333',
                }}
              >
                { getShortId(programCID, 16) }:{ entryPoint }
              </a>
          </Typography>
        </Box>
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
