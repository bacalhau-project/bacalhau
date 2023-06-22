-- Create the job_info table
CREATE TABLE job_info
(
    id         VARCHAR(255) PRIMARY KEY,
    created    TIMESTAMPTZ DEFAULT NOW(),
    updated    TIMESTAMPTZ DEFAULT NOW(),
    info       JSONB,
    apiversion VARCHAR(255)
);
CREATE INDEX idx_apiversion ON job_info (apiversion);

-- Create job summary view
CREATE VIEW job_summary_view AS
SELECT
    id,
    created,
    apiversion,
    CASE WHEN apiversion = 'V1beta2' THEN info->'State'->>'State'
        END AS state,
    CASE
        WHEN apiversion LIKE 'V1beta%' THEN info->'Job'->'Metadata'->>'ClientID'
        END AS clientid,
    CASE
        WHEN apiversion LIKE 'V1beta%' THEN info->'Job'->'Spec'->>'Engine'
        END AS executor
FROM
    job_info
ORDER BY
    created DESC;

-- Create a trigger function to update the 'updated' column
CREATE
    OR REPLACE FUNCTION update_updated_column()
    RETURNS TRIGGER AS $$
BEGIN
    NEW.updated := NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create a trigger to execute the update_updated_column function on UPDATE
CREATE TRIGGER update_updated_trigger
    BEFORE UPDATE ON job_info
    FOR EACH ROW
EXECUTE FUNCTION update_updated_column();
