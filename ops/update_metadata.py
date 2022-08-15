import google.cloud
from google.cloud import storage
import os
import sys
import datetime


def update_metadata_function(BUCKET_NAME: str, FILE_NAME: str):
    storage_client = storage.Client()
    bucket = storage_client.bucket(BUCKET_NAME)
    blob = bucket.get_blob(FILE_NAME)

    if not blob:
        print(f"Could not access the file at 'gs://{BUCKET_NAME}/{FILE_NAME}'.", file=sys.stderr)

    metadata = {"x-goog-meta-last-updated": f"{datetime.datetime.utcnow().isoformat()}Z"}
    blob.metadata = metadata
    blob.patch()


if __name__ == "__main__":
    if not os.environ.get("GOOGLE_APPLICATION_CREDENTIALS"):
        print("'GOOGLE_APPLICATION_CREDENTIALS' env variable not set.", file=sys.stderr)
        sys.exit(1)

    print(sys.argv)

    args = sys.argv

    print(os.path.basename(__file__))

    if args[0] == os.path.basename(__file__):
        args = args[1:]

    if not len(args) == 2:
        print("Please provide the 'bucket name' and 'metadata file'.", file=sys.stderr)
        sys.exit(1)

    BUCKET_NAME = sys.argv[1]
    FILE_NAME = sys.argv[2]
    update_metadata_function(BUCKET_NAME=BUCKET_NAME, FILE_NAME=FILE_NAME)
