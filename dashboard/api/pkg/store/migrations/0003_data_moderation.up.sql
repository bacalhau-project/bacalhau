ALTER TYPE moderation_type ADD VALUE IF NOT EXISTS 'result';

-- A result moderation request is attached to a job moderation request; one
-- result moderation for each result storage spec in the job. All the "requests"
-- are in the same table (and so share an ID namespace) but some of them have
-- extra data – this table exists to hold extra data for result requests. So
-- these rows are just added at the point of creation of the request row.
CREATE TABLE result_moderation_request_extra (
  id SERIAL PRIMARY KEY,
  request_id bigint,
  storage_spec json,
  execution_id json,
  FOREIGN KEY (request_id) REFERENCES job_moderation_request(id)
);

-- We don't expect anything to be in cid_moderation because existing code
-- doesn't use it, so we just drop it.
DROP TABLE cid_moderation;
