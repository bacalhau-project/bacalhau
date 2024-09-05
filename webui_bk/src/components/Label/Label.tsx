import React from "react"
import styles from "./Label.module.scss"

interface LabelProps {
  text: string
  color: string
}

const Label: React.FC<LabelProps> = ({ text, color }) => {
  const labelClass = `${styles.label} ${styles[`label-${color}`] || ""}`
  return (
    <div className={styles.column}>
      <button className={labelClass} type="button">
        {text}
      </button>
    </div>
  )
}

export default Label
