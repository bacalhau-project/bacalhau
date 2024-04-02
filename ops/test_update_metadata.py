import dotenv
import tempfile
import os
from pathlib import Path
from update_metadata import update_metadata_function

dotenv.load_dotenv()

with tempfile.NamedTemporaryFile(suffix=".json") as t:
    print(os.environ["TEST_RUNS_METADATA_FILENAME"])
    os.environ["GOOGLE_APPLICATION_CREDENTIALS"] = t.name
    os.system(
        'echo "${GOOGLE_APPLICATION_CREDENTIALS_CONTENT_B64}" | base64 --decode > "${GOOGLE_APPLICATION_CREDENTIALS}"'
    )
    a = Path(t.name).read_text()
    update_metadata_function(
        os.environ.get("METADATA_BUCKET"), os.environ.get("TEST_RUNS_METADATA_FILENAME")
    )
