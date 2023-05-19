SELECT job.id::text, input_element ->> 'CID' AS input_output, true AS is_input
FROM job, json_array_elements(jobdata::json -> 'Spec' -> 'inputs') AS input_element
WHERE job.apiversion = 'V1beta1' and input_element ->> 'CID' = $1
UNION ALL
SELECT node_states.JobID, node_states.output_cid AS input_output, false AS is_input
FROM (
         SELECT id AS JobID, key AS NodeID, value -> 'Shards' -> '0' -> 'PublishedResults' ->> 'CID' as output_cid
         FROM job,
             LATERAL jsonb_each(job.statedata::jsonb -> 'Nodes')
         WHERE job.apiversion = 'V1beta1' and statedata != ''
     ) as node_states
WHERE node_states.output_cid = $1;
