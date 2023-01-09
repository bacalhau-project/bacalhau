import React, { useCallback } from 'react'
import MuiSnackbar from '@mui/material/Snackbar'
import Alert from '@mui/material/Alert'
import {
  SnackbarContext,
} from '../../contexts/snackbar'

const Snackbar: React.FC = () => {
  const snackbarContext = React.useContext(SnackbarContext)

  const handleClose = useCallback(() => {
    snackbarContext.setSnackbar('')
  }, [])

  if(!snackbarContext.snackbar) return null

  return (
    <MuiSnackbar
      open
      autoHideDuration={ 5000 }
      anchorOrigin={{ vertical: 'top', horizontal: 'center' }}
      onClose={ handleClose }
    >
      <Alert
        severity={ snackbarContext.snackbar.severity }
        elevation={ 6 }
        variant="filled"
        onClose={ handleClose }
      >
        { snackbarContext.snackbar.message }
      </Alert>
    </MuiSnackbar>
  )
}

export default Snackbar
