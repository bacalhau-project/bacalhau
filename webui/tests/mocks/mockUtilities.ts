export function selectRandomKeyAndValue(allKVs: Record<string, string[]>): {
  key: string
  value: string
} {
  const keys = Object.keys(allKVs)
  const key: string = keys[Math.floor(Math.random() * keys.length)]
  const value: string = selectRandomElements(allKVs[key], 1) as string
  return { key, value }
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
