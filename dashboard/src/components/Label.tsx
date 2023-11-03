import React from "react";

interface LabelProps {
  text: string;
  backgroundColor: string;
  textColor: string;
}

const Label: React.FC<LabelProps> = ({ text, backgroundColor, textColor }) => {
  return (
    <button
      style={{
        backgroundColor,
        color: textColor,
        padding: "5px 15px",
        borderRadius: "20px",
        border: "none",
        fontSize: "14px",
      }}
    >
      {text}
    </button>
  );
};

export default Label;
