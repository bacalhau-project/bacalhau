// src/components/ProgramSummary.tsx
import React from "react"
import styles from "./ProgramSummary.module.scss"
import { Tasks } from "../../../../helpers/jobInterfaces"
import { ReactComponent as Cogwheel } from "../../../../images/cogwheel.svg"
import { ReactComponent as DockerLogo } from "../../../../images/docker.svg"

interface ProgramSummaryProps {
  data: Tasks
}

const getImageSource = (type: string) => {
  switch (type) {
    case "docker":
      return <DockerLogo className={styles.icon} />
    default:
      return <Cogwheel className={styles.icon} />
  }
}

const truncateInput = (text: string, length: number) => {
  if (text.length === 0) return ""
  if (text[0].length <= length) return text
  return `${text[0].substring(0, length)}[cont]`
}

const ProgramSummary: React.FC<ProgramSummaryProps> = ({ data }) => {
  const {
    Type: engineType,
    Params: { Image: image, Parameters: parameters },
  } = data.Engine

  const imageSource = getImageSource(engineType)
  const truncatedInput = truncateInput((parameters || []).join(" "), 100)

  return (
    <div className={styles.programSummary}>
      <div className={styles.iconContainer}>{imageSource}</div>
      <div className={styles.text}>
        <div className={styles.header}>{image}</div>
        <div className={styles.input}>{truncatedInput}</div>
      </div>
    </div>
  )
}

export default ProgramSummary
