import React from "react"
import { useNavigate } from "react-router-dom"
import { ReactComponent as ViewIcon } from "../../images/view-icon.svg"
import styles from "./ActionButton.module.scss"

interface ActionButtonProps {
  text: string
  onClick?: () => void
  to?: string
  id?: string
}

const ActionButton: React.FC<ActionButtonProps> = ({
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
        <ViewIcon className={styles.viewIcon} />
        {text}
      </button>
    </div>
  )
}

export { ActionButton }
