from datetime import datetime

from airflow import DAG
from bacalhau_airflow.operators import BacalhauSubmitJobOperator

with DAG("bacalhau-sample-dag", start_date=datetime(2021, 1, 1)) as dag:
    op1 = BacalhauSubmitJobOperator(
        task_id="run-1",
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

    op2 = BacalhauSubmitJobOperator(
        task_id="run-2",
        api_version="V1beta1",
        job_spec=dict(
            engine="Docker",
            verifier="Noop",
            publisher="IPFS",
            docker=dict(
                image="ubuntu",
                entrypoint=["echo", "World"],
            ),
            deal=dict(concurrency=1, confidence=0, min_bids=0),
            inputs=[
                dict(
                    cid="QmWG3ZCXTbdMUh6GWq2Pb1n7MMNxPQFa9NMswdZXuVKFUX",
                    path="/another-dataset",
                    storagesource="ipfs",
                )
            ],
        ),
        input_volumes=[
            "{{ task_instance.xcom_pull(task_ids='run-1', key='cids') }}:/datasets",
        ],
    )

    op1 >> op2
