// import axios from "axios";
// import { resolve } from "../utils/resolver";

// export const SERVER_URL = process.env.REACT_APP_SERVER_URL;

// export interface Resolved {
//     statusCode: number;
//     data: any;
// }

// export const getJobs = async (query?: string, forExplorer?: boolean): Promise<Resolved | void> => {
//     try {
//         const token = forExplorer ? "" : localStorage.getItem("token");

//         const resolved: Resolved = await resolve(
//             axios.get(`${SERVER_URL}/jobs?query=${query ?? ""}`, {
//                 headers: {
//                     "Content-Type": `application/json`,
//                     Authorization: `Bearer ${token}`,
//                 },
//             })
//         );
//         return resolved;
//     } catch (error: any) {
//         console.log(error.message);
//     }
// };

// export const getJobEvents = async (job_id: string): Promise<Resolved | void> => {
//     try {
//         const resolved: Resolved = await resolve(
//             axios.get(`${SERVER_URL}/jobs/events/${job_id}`, {
//                 headers: {
//                     "Content-Type": `application/json`,
//                 },
//             })
//         );
//         return resolved;
//     } catch (error: any) {
//         console.log(error.message);
//     }
// };

// export const getJob = async (
//     id: string,
//     type: string
// ): Promise<Resolved | void> => {
//     try {
//         const token = localStorage.getItem("token");

//         const resolved: Resolved = await resolve(
//             axios.get(`${SERVER_URL}/jobs/state/${id}`, {
//                 headers: {
//                     "Content-Type": `application/json`,
//                     Authorization: `Bearer ${token}`,
//                 },
//             })
//         );
//         return resolved;
//     } catch (error: any) {
//         console.log(error.message);
//     }
// };
