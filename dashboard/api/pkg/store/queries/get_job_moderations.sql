SELECT job_moderation.id,
    job_moderation_request.id,
    job_moderation.useraccount_id,
    job_moderation.created,
    job_moderation.approved,
    job_moderation.notes,
    useraccount.id,
    useraccount.created,
    useraccount.username,
    job_moderation_request.id,
    job_moderation_request.job_id,
    job_moderation_request.request_type,
    job_moderation_request.created,
    job_moderation_request.callback
FROM job_moderation
    INNER JOIN job_moderation_request ON job_moderation.request_id = job_moderation_request.id
    INNER JOIN useraccount ON job_moderation.useraccount_id = useraccount.id
WHERE job_id = $1
