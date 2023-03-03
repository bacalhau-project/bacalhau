from datetime import datetime

from airflow import DAG
from bacalhau_airflow.operators import BacalhauSubmitJobOperator

with DAG("bacalhau-helloworld-dag", start_date=datetime(2023, 3, 1)) as dag:
    task_1 = BacalhauSubmitJobOperator(
        task_id="task_1",
        api_version="V1beta1",
        job_spec=dict(
            engine="Docker",
            verifier="Noop",
            publisher="IPFS",
            docker=dict(
                image="ubuntu",
                entrypoint=["echo", "Hello"],
            ),
            deal=dict(concurrency=1, confidence=0, min_bids=0),
        ),
    )

    task_2 = BacalhauSubmitJobOperator(
        task_id="task_2",
        api_version="V1beta1",
        input_volumes=[
            "{{ task_instance.xcom_pull(task_ids='task_1', key='cids') }}:/task_1_output",
        ],
        job_spec=dict(
            engine="Docker",
            verifier="Noop",
            publisher="IPFS",
            docker=dict(
                image="ubuntu",
                entrypoint=["echo", "$(cat /task_1_output/stdout)", "World"],
            ),
            deal=dict(concurrency=1, confidence=0, min_bids=0),
        ),
    )

    task_1 >> task_2
