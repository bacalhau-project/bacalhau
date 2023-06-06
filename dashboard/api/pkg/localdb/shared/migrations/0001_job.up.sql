create table job (
  id varchar(255) PRIMARY KEY,
  created timestamp,
  clientid varchar(255),
  executor varchar(255),
  apiversion varchar(255),
  jobdata text default '',
  statedata text  default ''
);
CREATE INDEX idx_job_clientid ON job (clientid);
CREATE INDEX idx_job_executor ON job (executor);

create table job_annotation (
  id SERIAL PRIMARY KEY,
  job_id varchar(255),
  annotation varchar(255),
  FOREIGN KEY(job_id) REFERENCES job(id)
);
CREATE INDEX idx_job_annotation ON job_annotation (annotation);

create table job_event (
  id SERIAL PRIMARY KEY,
  job_id varchar(255),
  created timestamp,
  apiversion varchar(255),
  eventdata text,
  FOREIGN KEY(job_id) REFERENCES job(id)
);
CREATE INDEX idx_job_event_job_id ON job_event (job_id);

create table local_event (
  id SERIAL PRIMARY KEY,
  job_id varchar(255),
  created timestamp default current_timestamp,
  apiversion varchar(255),
  eventdata text,
  FOREIGN KEY(job_id) REFERENCES job(id)
);
CREATE INDEX idx_local_event_job_id ON local_event (job_id);
