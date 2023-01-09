import React, { FC} from 'react'
import { SxProps } from '@mui/system'
import Stack from '@mui/material/Stack'
import Box from '@mui/material/Box'
import {
  StorageSpec,
  RunCommandResult,
} from '../../types'
import FilPlus from './FilPlus'

const OutputVolumes: FC<{
  outputVolumes: StorageSpec[],
  publishedResults?: StorageSpec,
  includeDatacap?: boolean,
  sx?: SxProps,
}> = ({
  outputVolumes = [],
  publishedResults,
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
          ...sx
        }}
      >
        {
          publishedResults && (
            <li>
              <span
                style={{
                  fontSize: '0.8em',
                  color: '#333',
                }}
              >
                <a target="_blank" href={ `https://ipfs.io/ipfs/${publishedResults.CID}` }>
                  all
                </a>
              </span>
            </li>
          )
        }
        {
          outputVolumes.map((storageSpec) => {
            return (
              <li key={storageSpec.Name}>
                <span
                  style={{
                    fontSize: '0.8em',
                    color: '#333',
                  }}
                >
                  {
                    publishedResults ? (
                      <a target="_blank" href={ `https://ipfs.io/ipfs/${publishedResults.CID}${storageSpec.path}` }>
                        { storageSpec.Name }:{ storageSpec.path }
                      </a>
                    ) : (
                      <span>
                        { storageSpec.Name }:{ storageSpec.path }
                      </span>
                    )
                  }
                  
                </span>
              </li>
            )
          })
        }
      </Box>
    </Stack>
  )
}

export default OutputVolumes
