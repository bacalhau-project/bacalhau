// Select a random job label from the list of available labels in jobLabels - should be between 0 and 3 labels
export function selectRandomLabels(jobLabels: { [key: string]: string[] }) {
  const labels: { [key: string]: string } = {};
  const numLabels = Math.floor(Math.random() * 4);
  for (let i = 0; i < numLabels; i++) {
    const label = selectRandomElements(Object.keys(jobLabels));
    labels[label] = selectRandomElements(jobLabels[label]);
  }
  return labels;
}

export function selectRandomElements(all_types, num = 1) {
  if (num === 1) {
    return all_types[Math.floor(Math.random() * all_types.length)];
  } else {
    // Create a variable selections that is strongly typed as an array of strings
    const selections: string[] = [];
    for (let i = 0; i < num; i++) {
      const randomIdx = Math.floor(Math.random() * all_types.length);
      selections.push(all_types[randomIdx]);
    }
    return selections;
  }
}
