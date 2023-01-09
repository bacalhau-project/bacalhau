export const formatFloat = (num: number | undefined): number => {
  if (num === undefined) {
    return 0
  }
  return Math.round(num * 100) / 100
}

export const subtractFloat = (a: number | undefined, b: number | undefined): number => {
  const useA = a || 0
  const useB = b || 0
  return formatFloat(useA - useB)
}