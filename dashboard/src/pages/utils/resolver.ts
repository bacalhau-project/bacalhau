import axios, { AxiosResponse, AxiosError } from 'axios';

interface Resolved {
    data: null | any;
    error: null | AxiosError;
    statusCode: number;
}

export async function resolve(promise: Promise<AxiosResponse<any>>): Promise<Resolved> {
    const resolved: Resolved = {
        data: null,
        error: null,
        statusCode: 0,
    }

    try {
        const res: AxiosResponse = await promise;
        resolved.data = res.data;
        resolved.statusCode = res.status;
    } catch (e) {
        const axiosError = e as AxiosError;
        if (axiosError.response) {
            resolved.data = axiosError.response;
            resolved.statusCode = axiosError.response.status;
        }
        resolved.error = axiosError;
    }

    return resolved;
}
