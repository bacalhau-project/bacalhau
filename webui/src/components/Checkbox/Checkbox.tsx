import React from "react"

interface CheckboxProps {
  checked: boolean | undefined
  onChange: () => void
  label?: string
}

const Checkbox: React.FC<CheckboxProps> = ({ checked, onChange, label }) => {
  return (
    <div>
      <label>
        <span>{label && <span>{label}</span>}</span>
        <input type="checkbox" checked={checked} onChange={onChange} />
      </label>
    </div>
  )
}

export default Checkbox
