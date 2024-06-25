#!/bin/bash

set -e
# Create a log file
LOG_FILE="aws_operations.log"
touch $LOG_FILE

# Function to run AWS commands and log output
run_aws_command() {
    echo "Running: $1" >> $LOG_FILE
    eval "$1" >> $LOG_FILE 2>&1
    return $?
}

update_env_var() {
    local key=$1
    local value=$2
    if grep -q "^$key=" .env; then
        sed -i '' "s|^$key=.*|$key=$value|" .env
    else
        echo "$key=$value" >> .env
    fi
}

# Function to run AWS commands, log output, and return result
run_aws_command_with_output() {
    echo "Running: $1" >> $LOG_FILE
    result=$(eval "$1" 2>> $LOG_FILE)
    echo "$result" >> $LOG_FILE
    echo "$result"
}

# Generate a secure random password
PASSWORD=$(openssl rand -base64 32)

# AWS account ID - load account id from aws cli
ACCOUNT_ID=$(run_aws_command_with_output "aws sts get-caller-identity --query Account --output text")

# Check if the IAM user already exists
if run_aws_command "aws iam get-user --user-name bacalhau-infra" 2>/dev/null; then
    echo "User bacalhau-infra already exists. Skipping user creation."
else
    # Create the IAM user
    run_aws_command "aws iam create-user --user-name bacalhau-infra"
    echo "User bacalhau-infra created successfully."
fi

# Check existing access keys and delete the oldest if necessary
EXISTING_KEYS=$(run_aws_command_with_output "aws iam list-access-keys --user-name bacalhau-infra --query 'AccessKeyMetadata[*].[AccessKeyId,CreateDate]' --output text")
NUM_KEYS=$(echo "$EXISTING_KEYS" | wc -l)

if [ "$NUM_KEYS" -ge 2 ]; then
    echo "Two access keys already exist. Deleting the oldest one."
    OLDEST_KEY=$(echo "$EXISTING_KEYS" | sort -k2 | head -n1 | awk '{print $1}')
    run_aws_command "aws iam delete-access-key --user-name bacalhau-infra --access-key-id $OLDEST_KEY"
    echo "Deleted access key: $OLDEST_KEY"
fi

# Create a new access key for the user
ACCESS_KEY=$(run_aws_command_with_output "aws iam create-access-key --user-name bacalhau-infra")
echo "New access key created successfully."

# Create the policy document as a variable
POLICY_DOCUMENT=$(cat << 'EOF'
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ec2:*",
        "s3:*",
        "iam:PassRole"
      ],
      "Resource": "*"
    }
  ]
}
EOF
)

# Check if the policy already exists
EXISTING_POLICY=$(run_aws_command "aws iam get-policy --policy-arn arn:aws:iam::$ACCOUNT_ID:policy/BacalhauInfraPolicy" 2>/dev/null)

if [ $? -eq 0 ]; then
    echo "Policy BacalhauInfraPolicy already exists. Updating the policy."
    POLICY_VERSION=$(run_aws_command "aws iam create-policy-version --policy-arn arn:aws:iam::$ACCOUNT_ID:policy/BacalhauInfraPolicy --policy-document '$POLICY_DOCUMENT' --set-as-default")
    POLICY_ARN="arn:aws:iam::$ACCOUNT_ID:policy/BacalhauInfraPolicy"
else
    echo "Creating new policy BacalhauInfraPolicy."
    POLICY=$(run_aws_command_with_output "aws iam create-policy --policy-name BacalhauInfraPolicy --policy-document '$POLICY_DOCUMENT'")
    POLICY_ARN=$(echo $POLICY | jq -r '.Policy.Arn')
fi

# Attach the policy to the user
run_aws_command "aws iam attach-user-policy --user-name bacalhau-infra --policy-arn $POLICY_ARN"



# Check if login profile exists and update or create accordingly
if run_aws_command "aws iam get-login-profile --user-name bacalhau-infra" 2>/dev/null; then
    echo "Login profile for bacalhau-infra already exists. Updating password."
    run_aws_command "aws iam update-login-profile --user-name bacalhau-infra --password '$PASSWORD' --password-reset-required"
else
    echo "Creating new login profile for bacalhau-infra."
    run_aws_command "aws iam create-login-profile --user-name bacalhau-infra --password '$PASSWORD' --password-reset-required"
fi

# Ensure .env file exists
touch .env

# Update or add variables to .env file
update_env_var "AWS_ACCOUNT_ID" "$ACCOUNT_ID"
update_env_var "AWS_USER_NAME" "bacalhau-infra"
update_env_var "AWS_USER_PASSWORD" "$PASSWORD"
update_env_var "AWS_ACCESS_KEY_ID" "$(echo $ACCESS_KEY | jq -r '.AccessKey.AccessKeyId')"
update_env_var "AWS_SECRET_ACCESS_KEY" "$(echo $ACCESS_KEY | jq -r '.AccessKey.SecretAccessKey')"
update_env_var "AWS_POLICY_ARN" "$POLICY_ARN"

echo "Credentials and other important variables have been updated in the .env file."

# Add an email alias to the user
run_aws_command "aws iam tag-user --user-name bacalhau-infra --tags Key=email,Value=bacalhau-infra@expanso.io"

# Verify the user creation and permissions
run_aws_command "aws iam get-user --user-name bacalhau-infra"
run_aws_command "aws iam list-attached-user-policies --user-name bacalhau-infra"

# Output sensitive information
echo "User created/updated successfully. Credentials have been saved to .env file."
echo "Please ensure to keep the .env file secure and do not share it."

echo "Script completed. Check $LOG_FILE for detailed output."
