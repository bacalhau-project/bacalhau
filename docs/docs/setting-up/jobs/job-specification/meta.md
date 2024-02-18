---
sidebar_label: Meta
---

# Meta Specification

In both the `Job` and `Task` specifications within Bacalhau, the `Meta` block is a versatile element used to attach arbitrary metadata. This metadata isn't utilized for filtering or categorizing jobs; there's a separate [`Labels`](./label) block specifically designated for that purpose. Instead, the `Meta` block is instrumental for embedding additional information for operators or external systems, enhancing clarity and context.

## `Meta` Parameters in Job and Task Specs

The `Meta` block is comprised of key-value pairs, with both keys and values being strings. These pairs aren't constrained by a predefined structure, offering flexibility for users to annotate jobs and tasks with diverse metadata.

### User-Defined Metadata

Users can incorporate any arbitrary key-value pairs to convey descriptive information or context about the job or task.

#### Example:

```json
"Meta": {
    "project": "frontend",
    "version": "1.2.5",
    "owner": "team-alpha",
    "environment": "development"
}
```

- **project**: Identifies the associated project.
- **version**: Specifies the version of the application or service.
- **owner**: Names the responsible team or individual.
- **environment**: Indicates the stage in the development lifecycle.

## Auto-Generated Metadata by Bacalhau

Beyond user-defined metadata, Bacalhau automatically injects specific metadata keys for identification and security purposes.

### Bacalhau Auto-Generated Keys:

- **bacalhau.org/requester.id**: A unique identifier for the orchestrator that handled the job.
- **bacalhau.org/requester.publicKey**: The public key of the requester, aiding in security and validation.
- **bacalhau.org/client.id**: The ID for the client submitting the job, enhancing traceability.

#### Example:

```json
"Meta": {
    "bacalhau.org/requester.id": "QmfZwnVWYjHSchAVxJqXn18Bvd1cpG2ATRYceBBvUGZf2f",
    "bacalhau.org/requester.publicKey": "CAASpgIwggEiMA0GCSqG...BcyEhfEZKnAgMBAAE=",
    "bacalhau.org/client.id": "dfadea67ab6d8c65761c3d879119e11f157923036f945d969d19a51066dc663a"
}
```

### Implications and Utility

- **Identification**: The metadata aids in uniquely identifying jobs and tasks, connecting them to their originators and executors.

- **Context Enhancement**: Metadata can supplement jobs and tasks with additional data, offering insights and context that aren't captured by standard parameters.

- **Security Enhancement**: Auto-generated keys like the requester's public key contribute to the secure handling and execution of jobs and tasks.

While the `Meta` block is distinct from the [`Labels`](./label) block used for filtering, its contribution to providing context, security, and traceability is integral in managing and understanding the diverse jobs and tasks within the Bacalhau ecosystem effectively.
