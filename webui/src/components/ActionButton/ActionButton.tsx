import React from "react"
import { useNavigate } from "react-router-dom"
import styles from "./ActionButton.module.scss"

interface ActionButtonProps {
  text: string
  onClick?: () => void
  to?: string
  id?: string
}

export const ActionButton: React.FC<ActionButtonProps> = ({
  text,
  onClick,
  to,
  id,
}) => {
  const navigate = useNavigate()

  const handleClick = () => {
    if (onClick) {
      onClick()
    } else if (to) {
      const path = id ? `${to}/${id}` : to
      navigate(path)
    }
  }

  return (
    <div className={styles.column}>
      <button
        className={styles.actionButton}
        onClick={handleClick}
        type="button"
      >
        {text}
      </button>
    </div>
  )
}
