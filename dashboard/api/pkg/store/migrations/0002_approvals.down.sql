ALTER TABLE job_moderation
    ADD COLUMN status varchar(255),
    ADD COLUMN job_id varchar(255);

UPDATE job_moderation SET
    job_id = (
        SELECT job_id
        FROM job_moderation_request
        WHERE job_moderation_request.id = job_moderation.request_id
    ),
    status = (CASE
        WHEN approved = true THEN 'yes'
        WHEN approved = false THEN 'no'
    END);

ALTER TABLE job_moderation
    DROP COLUMN approved,
    DROP COLUMN request_id,
    ALTER COLUMN job_id SET NOT NULL,
    ADD FOREIGN KEY (job_id) REFERENCES job(id);

DROP TABLE job_moderation_request;

DROP TYPE moderation_type;
