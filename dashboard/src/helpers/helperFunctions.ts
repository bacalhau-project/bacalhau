export function capitalizeFirstLetter(input: string) {
  return input.charAt(0).toUpperCase() + input.slice(1);
}

export function formatTimestamp(timestamp: number) {
  // Convert nanoseconds to milliseconds
  const milliseconds = timestamp / 1000000;

  // Create a new Date object
  const date = new Date(milliseconds);

  // Use toLocaleString() or any other method to format the date
  return date.toLocaleString(); // or any other formatting you desire
}
