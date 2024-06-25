#!/bin/bash

set -e

# Function to read .env file
read_env() {
    if [ -f .env ]; then
        while IFS='=' read -r key value
        do
            if [[ ! $key =~ ^# && -n $key ]]; then
                export "$key"="$value"
            fi
        done < .env
    else
        echo ".env file not found"
        exit 1
    fi
}

# Read .env file
read_env

# Check for required variables
required_vars=("BUCKET_NAME" "AWS_ACCESS_KEY_ID" "AWS_SECRET_ACCESS_KEY")
missing_vars=()

for var in "${required_vars[@]}"; do
    if [ -z "${!var}" ]; then
        missing_vars+=("$var")
    fi
done

if [ ${#missing_vars[@]} -ne 0 ]; then
    echo "Missing required environment variables:"
    for var in "${missing_vars[@]}"; do
        echo "- $var"
    done
    echo "Please check your .env file."
    exit 1
fi

# Rest of your script goes here
FILE_NAME="test_file.txt"

# Create a test file
echo "This is a test file for Bacalhau infra" > $FILE_NAME

# Upload the file to S3
aws s3 cp $FILE_NAME s3://$BUCKET_NAME/$FILE_NAME

# Verify the upload
echo "Verifying upload..."
aws s3 ls s3://$BUCKET_NAME/$FILE_NAME

if [ $? -eq 0 ]; then
    echo "File uploaded successfully."
else
    echo "File upload failed."
    exit 1
fi

# Delete the local file
rm $FILE_NAME
echo "Local file deleted."

# Delete the file from S3
aws s3 rm s3://$BUCKET_NAME/$FILE_NAME

# Verify the deletion
echo "Verifying deletion..."
aws s3 ls s3://$BUCKET_NAME/$FILE_NAME

if [ $? -ne 0 ]; then
    echo "File deleted successfully from S3."
else
    echo "File deletion from S3 failed."
    exit 1
fi

echo "Test completed successfully."
