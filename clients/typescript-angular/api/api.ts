export * from './health.service';
import { HealthService } from './health.service';
export * from './job.service';
import { JobService } from './job.service';
export * from './misc.service';
import { MiscService } from './misc.service';
export const APIS = [HealthService, JobService, MiscService];
