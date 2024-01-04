export function capitalizeFirstLetter(input: string) {
  return input.charAt(0).toUpperCase() + input.slice(1);
}

export function fromTimestamp(timestamp: number): Date {
  // Convert nanoseconds to milliseconds
  const milliseconds = timestamp / 1000000;

  // Create a new Date object
  return new Date(milliseconds);
}

export function getShortenedJobID(jobID: string) {
  const parts = jobID.split("-");
  if (parts[0].length > 1) {
    return parts[0];
  } else {
    return parts[0] + "-" + parts[1];
  }
}

export function createLabelArray(label: { [key: string]: string }): string[] {
  const labelArray: string[] = [];
  for (const [key, value] of Object.entries(label)) {
    if (value === "") {
      labelArray.push(key);
    } else {
      labelArray.push(`${key}: ${value}`);
    }
  }
  return labelArray;
}
