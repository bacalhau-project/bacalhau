-- Dropping values from an enum is not supported.
DROP TABLE result_moderation_request_extra;

-- Unfortunately we do have to recreate this useless table so that the
-- migrations can come up and down cleanly.
CREATE TABLE cid_moderation (
  id SERIAL PRIMARY KEY,
  job_id varchar(255),
  useraccount_id bigint,
  created timestamp default current_timestamp,
  status varchar(255),
  cid varchar(255),
  FOREIGN KEY(job_id) REFERENCES job(id),
  FOREIGN KEY(useraccount_id) REFERENCES useraccount(id)
);
