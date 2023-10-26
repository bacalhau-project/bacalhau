// src/components/Table.tsx
import React from "react";
import ProgramSummary from "./ProgramSummary";
import styles from "../../styles/Table.module.scss";

interface TableProps {
  headers: string[];
  data: any[][];
}

const Table: React.FC<TableProps> = ({ headers, data }) => {
  return (
    <div className={styles.tableContainer}>
      <table>
        <thead>
          <tr>
            {headers.map((header, index) => (
              <th key={index}>{header}</th>
            ))}
          </tr>
        </thead>
        <tbody>
          {data.map((row, rowIndex) => (
            <tr key={rowIndex}>
              {row.map((cell, cellIndex) => (
                <td key={cellIndex}>
                  {/* Use the ProgramSummary component in the 5th column (cellIndex 4) */}
                  {cellIndex === 3 ? <ProgramSummary data={cell} /> : cell}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
};

export default Table;
