import React from "react";
import styles from "./Table.module.scss";

interface TableProps {
  data: {headers: string[], rows: { [key: string]: React.ReactNode }[]};
  style?: React.CSSProperties;
}

const Table: React.FC<TableProps> = ({ data, style }) => {
  return (
    <div className={styles.tableContainer} style={style}>
      <table>
        <thead>
          <tr>
            {data.headers.map((header) => (
              <th key={header}>{header}</th>
            ))}
          </tr>
        </thead>
        <tbody>
          {data.rows.map((row, rowIndex) => (
            <tr key={rowIndex}>
            {data.headers.map((header) => (
              <td key={`${header}-${rowIndex}`}>{row[header]}</td>
            ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
};

export default Table;
