

from datetime import datetime
from airflow import DAG
from bacalhau.operators import BacalhauSubmitJobOperator


with DAG('run-me', start_date=datetime(2021, 1, 1)) as dag:
    op1 = BacalhauSubmitJobOperator(
        task_id='run-me',
        api_version='V1beta1',
        job_spec=dict(
            engine="Docker",
            verifier="Noop",
            publisher="Estuary",
            docker=dict(
                image="ubuntu",
                entrypoint=["echo", "Hello World!"],
            ),
            deal=dict(concurrency=1, confidence=0, min_bids=0),
        )
    )

    op1
