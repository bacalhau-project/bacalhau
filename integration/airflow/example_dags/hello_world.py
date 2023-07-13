"""
The first task of this DAG prints out "Hello World" to stdout,
then automatically pipe its output into the subsequent task as an input file.
The second task will simply print out the content of its input file.
"""
from datetime import datetime

from airflow import DAG
from bacalhau_airflow.operators import BacalhauSubmitJobOperator

with DAG("bacalhau-helloworld-dag", start_date=datetime(2023, 3, 1)) as dag:
    task_1 = BacalhauSubmitJobOperator(
        task_id="task_1",
        api_version="V1beta1",
        job_spec=dict(
            engine="Docker",
            publisher="IPFS",
            docker=dict(
                image="ubuntu",
                entrypoint=["echo", "Hello World"],
            ),
            deal=dict(concurrency=1),
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
            publisher="IPFS",
            docker=dict(
                image="ubuntu",
                entrypoint=["cat", "/task_1_output/stdout"],
            ),
            deal=dict(concurrency=1),
        ),
    )

    task_1 >> task_2
