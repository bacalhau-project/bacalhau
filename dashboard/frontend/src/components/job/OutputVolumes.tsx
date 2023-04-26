import { FC} from 'react'
import { SxProps } from '@mui/system'
import Stack from '@mui/material/Stack'
import Box from '@mui/material/Box'
import { StorageSpec } from '../../types'
import FilPlus from './FilPlus'
import StorageSpecRow from './StorageSpec'

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
    <Stack direction="column">
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
      {
        publishedResults && (<StorageSpecRow spec={publishedResults} name="All results"/>)
      }
      {
        outputVolumes.map((storageSpec) => <StorageSpecRow spec={storageSpec}/>)
      }
    </Stack>
  )
}

export default OutputVolumes
