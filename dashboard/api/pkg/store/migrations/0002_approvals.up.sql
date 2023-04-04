CREATE TYPE moderation_type AS ENUM('datacap', 'execution');

CREATE TABLE job_moderation_request (
  id SERIAL PRIMARY KEY,
  job_id varchar(255),
  created timestamp default current_timestamp,
  request_type moderation_type NOT NULL,
  callback varchar(255)
  -- Note that we specifically do NOT add a foreign key here.
  -- This is so that moderation requests can arrive for jobs we haven't seen yet.
  -- FOREIGN KEY(job_id) REFERENCES job(id)
);

INSERT INTO job_moderation_request (job_id, created, request_type) (
    SELECT job_id, created, 'datacap'::moderation_type
    FROM job_moderation
);

ALTER TABLE job_moderation
    ADD COLUMN request_id bigint,
    ADD COLUMN approved boolean NOT NULL DEFAULT false;

UPDATE job_moderation SET
    request_id = (
        SELECT id
        FROM job_moderation_request
        WHERE job_moderation_request.job_id = job_moderation.job_id
    ),
    approved = (CASE
        WHEN LOWER(status) = 'yes' THEN true
        WHEN LOWER(status) = 'no' THEN false
    END);

ALTER TABLE job_moderation
    DROP COLUMN status,
    DROP COLUMN job_id,
    ALTER COLUMN request_id SET NOT NULL;
    -- ADD FOREIGN KEY (request_id) REFERENCES job_moderation_request(id);
