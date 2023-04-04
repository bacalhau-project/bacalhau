import { FC, useCallback, useState } from "react";
import { FormControl, FormControlLabel, FormLabel, Grid, Radio, RadioGroup, TextField, Typography } from "@mui/material";
import Window, { WindowProps } from "./Window";

type ModerationWindowProps = {
  title: string,
  prompt: string,
  onModerate: {
    (approved: boolean, reason: string): void,
  },
} & WindowProps

const positive: string = "Yes"
const negative: string = "No"

const ModerationWindow: FC<ModerationWindowProps> = ({
  title,
  prompt,
  onModerate,
  ...windowProps
}) => {
  const [ moderationResult, setModerationResult ] = useState(false)
  const [ moderationNotes, setModerationNotes ] = useState('')

  const closeModeration = useCallback(async () => {
    setModerationResult(false)
    setModerationNotes('')
    windowProps.onCancel?.()
  }, [])

  const submitModeration = useCallback(async () => {
    onModerate(moderationResult, moderationNotes)
    windowProps.onSubmit?.()
  }, [moderationResult, moderationNotes])

  return (
  <Window
    size="md"
    title="Moderate Job"
    submitTitle="Confirm"
    withCancel
    {...windowProps}
    onCancel={closeModeration}
    onSubmit={submitModeration}
  >
    <Grid container spacing={0}>
      <Grid item xs={12}>
        <FormControl>
          <FormLabel>{title}</FormLabel>
          <RadioGroup
            row
            value={moderationResult ? positive : negative}
            onChange={(e) => setModerationResult(e.target.value == positive)}
          >
            <FormControlLabel value={positive} control={<Radio />} label={positive} />
            <FormControlLabel value={negative} control={<Radio />} label={negative} />
          </RadioGroup>
        </FormControl>
      </Grid>
      <Grid item xs={12}>
        <Typography gutterBottom variant="caption">
          {prompt}
        </Typography>
      </Grid>
      <Grid item xs={12} sx={{
        mt: 4,
      }}>
        <TextField
          label="Notes"
          fullWidth
          multiline
          rows={4}
          value={moderationNotes}
          onChange={(e) => setModerationNotes(e.target.value)}
        />
      </Grid>
    </Grid>
  </Window>)
}

export default ModerationWindow
