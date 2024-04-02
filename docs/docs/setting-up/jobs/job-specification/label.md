---

sidebar_label: Label
---

# Labels Specification

The `Labels` block within a `Job` specification plays a crucial role in Bacalhau, serving as a mechanism for filtering jobs. By attaching specific labels to jobs, users can quickly and effectively filter and manage jobs via both the Command Line Interface (CLI) and Application Programming Interface (API) based on various criteria.

## `Labels` Parameters

Labels are essentially key-value pairs attached to jobs, allowing for detailed categorizations and filtrations. Each label consists of a `Key` and a `Value`. These labels can be filtered using operators to pinpoint specific jobs fitting certain criteria.

### Filtering Operators

Jobs can be filtered using the following operators:

- `in`: Checks if the key's value matches any within a specified list of values.
- `notin`: Validates that the key's value isn’t within a provided list of values.
- `exists`: Checks for the presence of a specified key, regardless of its value.
- `!`: Validates the absence of a specified key. (i.e., DoesNotExist)
- `gt`: Checks if the key's value is greater than a specified value.
- `lt`: Checks if the key's value is less than a specified value.
- `= & ==`: Used for exact match comparisons between the key’s value and a specified value.
- `!=`: Validates that the key’s value doesn't match a specified value.

### Example Usage

Filter jobs with a label whose key is "environment" and value is "development":

```shell
bacalhau job list --labels 'environment=development'
```

Filter jobs with a label whose key is "version" and value is greater than "2.0":

```shell
bacalhau job list --labels 'version gt 2.0'
```

Filter jobs with a label "project" existing:

```shell
bacalhau job list --labels 'project'
```

Filter jobs without a "project" label:

```shell
bacalhau job list --labels '!project'
```

### Practical Applications

- **Job Management**: Enables efficient management of jobs by categorizing them based on distinct attributes or criteria.
- **Automation**: Facilitates the automation of job deployment and management processes by allowing scripts and tools to target specific categories of jobs.
- **Monitoring & Analytics**: Enhances monitoring and analytics by grouping jobs into meaningful categories, allowing for detailed insights and analysis.

## Conclusion

The `Labels` block is instrumental in the enhanced management, filtering, and operation of jobs within Bacalhau. By understanding and utilizing the available operators and label parameters effectively, users can optimize their workflow, automate processes, and achieve detailed insights into their jobs.
