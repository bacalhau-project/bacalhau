// Select a random job label from the list of available labels in jobLabels - should be between 0 and 3 labels
export function selectRandomLabels(jobLabels: { [key: string]: string[] }): {
  [key: string]: string[]
} {
  const labels: { [key: string]: string[] } = {}
  const numLabels = Math.floor(Math.random() * 4)
  for (let i = 0; i < numLabels; i += 1) {
    const label = selectRandomElements(Object.keys(jobLabels)) as string
    labels[label] = selectRandomElements(jobLabels[label]) as string[]
  }
  return labels
}

export function selectRandomElements(
  allTypes: string[],
  num = 1
): string | string[] {
  if (num === 1) {
    return allTypes[Math.floor(Math.random() * allTypes.length)]
  }
  // Create a variable selections that is strongly typed as an array of strings
  const selections: string[] = []
  for (let i = 0; i < num; i += 1) {
    const randomIdx = Math.floor(Math.random() * allTypes.length)
    selections.push(allTypes[randomIdx])
  }
  return selections
}
