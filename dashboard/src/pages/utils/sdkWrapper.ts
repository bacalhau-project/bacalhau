// utils/sdkWrapper.ts

async function fetchList(): Promise<any> {
    const response = await fetch('/api/list');
    if (!response.ok) {
      throw new Error('Failed to fetch list');
    }
    const data = await response.json();
    return data;
  }
  
  export { fetchList };
  