create table useraccount (
  id SERIAL PRIMARY KEY,
  created timestamp default current_timestamp,
  username varchar(255),
  hashed_password varchar(255)
);

create table job_moderation (
  id SERIAL PRIMARY KEY,
  job_id varchar(255),
  useraccount_id bigint,
  created timestamp default current_timestamp,
  status varchar(255),
  notes text default '',
  FOREIGN KEY(job_id) REFERENCES job(id),
  FOREIGN KEY(useraccount_id) REFERENCES useraccount(id)
);

create table cid_moderation (
  id SERIAL PRIMARY KEY,
  job_id varchar(255),
  useraccount_id bigint,
  created timestamp default current_timestamp,
  status varchar(255),
  cid varchar(255),
  FOREIGN KEY(job_id) REFERENCES job(id),
  FOREIGN KEY(useraccount_id) REFERENCES useraccount(id)
);
