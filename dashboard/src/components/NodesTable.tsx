// src/components/NodesTable.tsx
import React from "react";
import styles from "../../styles/NodesTable.module.scss";
import Label from "./Label";
import ActionButton from "./ActionButton";
import { Node, ParsedNodeData } from "../helpers/nodeInterfaces";

interface TableProps {
  data: Node[];
}

const labelColorMap: { [key: string]: string } = {
  healthy: "green",
  warning: "orange",
  critical: "red",
  offline: "blue",
  unknown: "grey"
};

function parseData(nodes: Node[]): ParsedNodeData[] {
  return nodes.map((node) => {
    return {
      id: node.PeerInfo.ID,
      name: node.Labels.name ? node.Labels.name : node.PeerInfo.ID,
      type: node.NodeType,
      labels: node.Labels.env, // TODO
      action: "Action"
    };
  });
}

const JobsTable: React.FC<TableProps> = ({ data }) => {
  const parsedData = parseData(data);
  return (
    <div className={styles.tableContainer}>
      <table>
        <thead>
          <tr>
            <th>Node ID</th>
            <th>Name</th>
            <th>Type</th>
            <th>Labels</th>
            <th>Action</th>
          </tr>
        </thead>
        <tbody>
          {parsedData.map((nodeData, index) => (
            <tr key={index}>
              <td className={styles.id}>{nodeData.id}</td>
              <td className={styles.name}>{nodeData.name}</td>
              <td className={styles.type}>{nodeData.type}</td>
              <td className={styles.label}>
                <Label
                  text={nodeData.labels}
                  color="green"
                />
              </td>
              <td className={styles.action}>
                <ActionButton text="View" />
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
};

export default JobsTable;
