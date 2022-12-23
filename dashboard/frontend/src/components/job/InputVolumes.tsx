import React, { FC } from 'react'
import { SxProps } from '@mui/system'
import Stack from '@mui/material/Stack'
import Box from '@mui/material/Box'
import {
  getShortId,
} from '../../utils/job'
import {
  StorageSpec,
} from '../../types'
import FilPlus from './FilPlus'

const InputVolumes: FC<{
  storageSpecs: StorageSpec[],
  includeDatacap?: boolean,
  sx?: SxProps,
}> = ({
  storageSpecs,
  includeDatacap = false,
  sx = {},
}) => {
  return (
    <Stack direction="row">
      {
        includeDatacap && (
          <Box
            component="div"
            sx={{
              mr: 1,
            }}
          >
            <FilPlus />
          </Box>
        )
      }
      <Box
        component="div"
        sx={{
          width: '100%',
          mr: 1,
          ...sx
        }}
      >
        {
          storageSpecs.map((storageSpec) => {
            let useUrl = ''
            if(storageSpec.URL) {
              const parts = storageSpec.URL.split(':')
              parts.pop()
              useUrl = parts.join(':')
            }
            else if(storageSpec.CID) {
              useUrl = `http://ipfs.io/ipfs/${storageSpec.CID}` 
            }
            return (
              <li key={storageSpec.CID || storageSpec.URL}>
                <a
                  href={ useUrl }
                  target="_blank"
                  rel="noreferrer"
                  style={{
                    fontSize: '0.8em',
                    color: '#333',
                  }}
                >
                  { getShortId(storageSpec.CID || storageSpec.URL || '', 16) }:{ storageSpec.path }
                </a>
              </li>
            )
          })
        }
      </Box>
    </Stack>
  )
}

export default InputVolumes
