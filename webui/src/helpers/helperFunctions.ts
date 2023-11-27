export function capitalizeFirstLetter(input: string) {
  return input.charAt(0).toUpperCase() + input.slice(1);
}

export function fromTimestamp(timestamp: number): Date {
  // Convert nanoseconds to milliseconds
  const milliseconds = timestamp / 1000000;

  // Create a new Date object
  return new Date(milliseconds);
}
