---
sidebar_label: 'CLI Flags' sidebar_position: 1
---

# CLI Commands
  devstack    Start a cluster of bacalhau nodes for testing and development
  help        Help about any command
  list        List jobs on the network
  run         Run a job on the network
  serve       Start the bacalhau compute node

## Top level flags
      --api-host string   The host for the client and server to communicate on (via REST). (default "bootstrap.production.bacalhau.org")
      --api-port int      The port for the client and server to communicate on (via REST). (default 1234)
  -h, --help              help for bacalhau

# List
  -h, --help               help for list
      --hide-header        do not print the column headers.
      --id-filter string   filter by Job List to IDs matching substring.
      --no-style           remove all styling from table output.
  -n, --number int         print the first NUM jobs instead of the first 10. (default 10)
      --output string      The output format for the list of jobs (json or text) (default "text")
      --reverse            reverse order of table - for time sorting, this will be newest first.
      --sort-by Column     sort by field, defaults to creation time, with newest first [Allowed "id", "created_at"].
      --wide               Print full values in the table results