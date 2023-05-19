import { Button, Divider, Grid, Typography } from "@mui/material";
import { Stack } from "@mui/system";
import React, { FC } from "react";
import { JobModerationSummary, ModerationType } from "../../types";
import { BoldSectionTitle } from "./GeneralText";
import CheckCircleIcon from '@mui/icons-material/CheckCircle'
import CancelIcon from '@mui/icons-material/Cancel'
import { IUserContext } from "../../contexts/user";

type ModerationPanelProps = {
  moderationType: ModerationType
  moderations: JobModerationSummary[]
  user: IUserContext
  icon: any
  onClick: () => void
  children?: React.ReactNode
}

const ModerationPanel: FC<ModerationPanelProps> = ({
  moderationType,
  moderations,
  user,
  icon,
  onClick,
  children,
}) => {
  var moderationRows = moderations.map(moderation => {
    return <Stack direction="row" alignItems="center">
      {
        moderation.moderation.status == true ? (
          <CheckCircleIcon sx={{ fontSize: '2em', color: 'green' }} />
        ) : (
          <CancelIcon sx={{ fontSize: '2em', color: 'red' }} />
        )
      }
      <Typography variant="caption" sx={{ color: '#666', ml: 2, }}>
        Moderated by <strong>{moderation.user.username}</strong> on {new Date(moderation.moderation.created).toLocaleDateString() + ' ' + new Date(moderation.moderation.created).toLocaleTimeString()}
        <br />
        {moderation.moderation.notes || null}
      </Typography>
    </Stack>
  })

  return (
    <Grid container spacing={1} >
      <Grid item xs={6}>
        <BoldSectionTitle>
          {moderationType.substring(0, 1).toUpperCase()}{moderationType.substring(1)} Moderation
        </BoldSectionTitle>
      </Grid>
      <Grid item xs={6} sx={{
        display: 'flex',
        justifyContent: 'flex-end',
      }}>
        {icon}
      </Grid>
      <Grid item xs={12}>
        {children}
      </Grid>
      <Grid item xs={12}>
        <Divider sx={{
          mt: 1,
          mb: 1,
        }} />
      </Grid>
      <Grid item xs={12}>
        {
          moderations.length > 0 ? (
            moderationRows
          ) : (
            <Typography variant="caption" sx={{ color: '#666' }}>
              This job has not been moderated yet
            </Typography>
          )
        }
      </Grid>
      {
        user.user && (
          <>
            <Grid item xs={12}>
              <Divider sx={{
                mt: 1,
                mb: 1,
              }} />
            </Grid>
            <Grid item xs={12}>
              <Button
                variant="outlined"
                color="primary"
                disabled={moderations.length > 0}
                onClick={onClick}
              >
                Moderate Job
              </Button>
            </Grid>
          </>
        )
      }
    </Grid >
  )
}

export default ModerationPanel
