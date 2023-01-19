import React, { FC } from 'react'
import Window, { WindowProps } from './Window'
import TerminalText, { TerminalTextConfig } from './TerminalText'

type TerminalWindowProps = {
  data: any,
  title?: string,
  onClose: {
    (): void,
  }
} & WindowProps & TerminalTextConfig

const TerminalWindow: FC<TerminalWindowProps> = ({
  data,
  title = 'Data',
  color,
  backgroundColor,
  onClose,
  ...windowProps
}) => {
  return (
    <Window
      withCancel
      compact
      title={ title }
      onCancel={ onClose }
      cancelTitle="Close"
      {...windowProps}
    >
      <TerminalText
        data={ data }
        color={ color }
        backgroundColor={ backgroundColor }
      />
    </Window>
  )
}

export default TerminalWindow
