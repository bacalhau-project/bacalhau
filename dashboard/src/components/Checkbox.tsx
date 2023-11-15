import React from "react";

interface CheckboxProps {
  checked: boolean;
  onChange: () => void;
  label?: string;
}

const Checkbox: React.FC<CheckboxProps> = ({ checked, onChange, label }) => {
  return (
    <label>
      {label && <span>{label}</span>}
      <input type="checkbox" checked={checked} onChange={onChange} />
    </label>
  );
};

export default Checkbox;
