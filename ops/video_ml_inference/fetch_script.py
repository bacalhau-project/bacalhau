import os
import argparse
from google.cloud import storage


def fetch_files(bucket_name, node_id, total_nodes, destination_dir):
    print(f"Starting download process for Node ID {node_id}/{total_nodes}...")
    storage_client = storage.Client(project="bacalhau-video-processing")
    bucket = storage_client.bucket(bucket_name)

    blobs = list(bucket.list_blobs())
    files_fetched = 0  # Counter for the number of files fetched
    for i, blob in enumerate(blobs):
        if i % total_nodes == node_id - 1:  # Determine if this node should fetch the file
            destination_path = os.path.join(destination_dir, blob.name)
            os.makedirs(os.path.dirname(destination_path), exist_ok=True)  # Ensure the destination directory exists
            blob.download_to_filename(destination_path)
            files_fetched += 1
            print(f"Fetched {blob.name} to {destination_path}")

    print(f"Completed downloading {files_fetched} files for Node ID {node_id}.")


def main():
    # Set up argument parser
    parser = argparse.ArgumentParser(description='Fetch files from Google Cloud Storage.')
    parser.add_argument('--node_id', type=int, required=True, help='Node ID among many')
    parser.add_argument('--total_nodes', type=int, required=True, help='Total number of nodes')
    parser.add_argument('--bucket_name', type=str, required=True, help='Google Cloud Storage bucket name')
    parser.add_argument('--destination_dir', type=str, required=True, help='Destination directory for fetched files')

    # Parse arguments
    args = parser.parse_args()

    # Fetch files
    fetch_files(args.bucket_name, args.node_id, args.total_nodes, args.destination_dir)


if __name__ == '__main__':
    main()
