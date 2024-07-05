#!/bin/bash

SECRET_NAME="build-agent-env-file"

# Function to store the .env file as a secret
store_secret() {
    if [ -z "$1" ]; then
        echo "Please provide the path to the .env file"
        exit 1
    fi

    if [ ! -f "$1" ]; then
        echo "File not found: $1"
        exit 1
    fi

    gcloud secrets create $SECRET_NAME --data-file="$1" 2>/dev/null || \
    gcloud secrets versions add $SECRET_NAME --data-file="$1"

    echo "Secret '$SECRET_NAME' has been created or updated"
}

# Function to retrieve the secret and source the environment variables
get_and_source_secret() {
    SECRET_CONTENT=$(gcloud secrets versions access latest --secret=$SECRET_NAME)
    if [ -z "$SECRET_CONTENT" ]; then
        echo "Failed to retrieve secret or secret is empty" >&2
        exit 1
    fi

    # echo "Secret content:" >&2
    # echo "---" >&2
    # echo "$SECRET_CONTENT" | sed 's/^/  /' >&2
    # echo "---" >&2

    # Output the secret content with export statements
    echo "$SECRET_CONTENT" | sed 's/^/export /'

    echo "Environment variables from '$SECRET_NAME' have been exported" >&2
}

# Function to list all secrets with their last updated date, versions, and last updated version
list_secrets() {
    first_column_width=30
    second_column_width=15
    third_column_width=15
    fourth_column_width=20

    # Separator bar "-" should be 4 characters longer than all columns
    separator_bar=$(printf '%*s' "$(($first_column_width + $second_column_width + ${third_column_width} + ${fourth_column_width} + 4))" | tr ' ' '-')

    printf "\nListing all secrets with their last updated date, versions, and last updated version:\n\n"

    printf "%-${first_column_width}s %-${second_column_width}s %-${third_column_width}s %-${fourth_column_width}s\n" "NAME" "CREATED" "VERSIONS" "LAST UPDATED"
    echo "$separator_bar"

    secrets_list=$(gcloud secrets list --format="value(name)")

    if [ -z "$secrets_list" ]; then
        echo "No secrets found."
    else
        for secret_name in $secrets_list; do
            secret_info=$(gcloud secrets describe "$secret_name" --format="value(createTime.date('%Y-%m-%d %H:%M:%S'),replication.automatic.status)")
            created_date=$(echo "$secret_info" | cut -d' ' -f1)
            replication_status=$(echo "$secret_info" | cut -d' ' -f2)

            versions_count=$(gcloud secrets versions list "$secret_name" --format="value(name)" | wc -l | tr -d ' ')
            last_updated=$(gcloud secrets versions list "$secret_name" --format="value(createTime.date('%Y-%m-%d %H:%M'))" --sort-by=~createTime --limit=1)

            # Truncate the secret name if it exceeds 30 characters
            if [ ${#secret_name} -gt $first_column_width ]; then
                secret_name="${secret_name:0:$first_column_width}..."
            fi

            printf "%-${first_column_width}s %-${second_column_width}s %-${third_column_width}s %-${fourth_column_width}s\n" "$secret_name" "$created_date" "$versions_count" "$last_updated"
        done
    fi
}

# Main logic
if [ "$1" = "store" ]; then
    store_secret "$2"
elif [ "$1" = "get" ]; then
    get_and_source_secret
elif [ "$1" = "list" ]; then
    list_secrets
else
    cat <<EOF
Usage: $0 [store <path_to_env_file> | get | list]
store: Store the .env file as a secret
get:   Retrieve the secret and source the environment variables
list:  List all secrets with their last updated date

Use 'source <(./manage_env_secret.sh get)' in your shell to source the environment variables
EOF
    exit 1
fi
