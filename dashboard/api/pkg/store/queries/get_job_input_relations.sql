WITH job_inputs AS (
    SELECT
        id AS JobID,
        input_element ->> 'StorageSource' as storage_source,
        input_element ->> 'CID' AS input
    FROM
        job,
        json_array_elements(jobdata::json -> 'Spec' -> 'inputs') AS input_element
    WHERE job.apiversion = 'V1beta1' and input_element ->> 'StorageSource' = 'IPFS'
),
     job_outputs AS (
         SELECT
             node_states.JobID,
             node_states.NodeID,
             node_states.storage_source,
             node_states.output_cid
         FROM (
                  SELECT
                      id                                             AS JobID,
                      key                                            AS NodeID,
                      value -> 'Shards' -> '0' ->> 'State'           AS State,
                      value -> 'Shards' -> '0' -> 'PublishedResults' AS PublishedResults,
                      value -> 'Shards' -> '0' -> 'PublishedResults' ->> 'StorageSource' as storage_source,
                      value -> 'Shards' -> '0' -> 'PublishedResults' ->> 'CID' as output_cid
                  FROM job, LATERAL jsonb_each(job.statedata::jsonb -> 'Nodes')
                  WHERE job.apiversion = 'V1beta1' and statedata != ''
              ) as node_states
         WHERE node_states.State = 'Completed' and node_states.storage_source = 'IPFS'
     )

SELECT
    job_outputs.JobID AS job_id,
    job_outputs.output_cid AS cid
FROM
    job_inputs
        JOIN
    job_outputs
    ON
            job_inputs.input = job_outputs.output_cid
WHERE
        job_inputs.JobID = $1
ORDER BY job_id DESC;