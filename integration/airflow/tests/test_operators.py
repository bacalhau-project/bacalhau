"""Tests for `bacalhau_airflow.operators` package."""

import unittest
from datetime import datetime
import pendulum

from bacalhau_airflow.operators import BacalhauSubmitJobOperator
from bacalhau_airflow.hooks import BacalhauHook

from airflow.models.dag import DAG
from airflow.utils.types import DagRunType
from airflow.utils.state import DagRunState

DEFAULT_DATE = pendulum.datetime(2022, 3, 4, tz="America/Toronto")

clock = str(int(datetime.now().timestamp()))
TEST_TASK_ID = "my_custom_operator_task" + clock
TEST_DAG_ID = "my_custom_operator_dag" + clock


class TestBacalhauAirflowOperator(unittest.TestCase):
    def setUp(self):
        """Set up test fixtures, if any."""
        super().setUp()
        self.dag = DAG(
            dag_id=TEST_DAG_ID,
            default_args={"owner": "airflow", "start_date": DEFAULT_DATE},
        )
        self.task = BacalhauSubmitJobOperator(
            task_id=TEST_TASK_ID,
            dag=self.dag,
            api_version="V1beta1",
            job_spec=dict(
                engine="Docker",
                verifier="Noop",
                publisher="Estuary",
                docker=dict(
                    image="ubuntu",
                    entrypoint=["echo", "TestBacalhauSubmitJobOperator"],
                ),
                deal=dict(concurrency=1, confidence=0, min_bids=0),
            ),
        )

    def tearDown(self):
        """Tear down test fixtures, if any."""

    def test_hook(self):
        """Test hook property."""

        operator = BacalhauSubmitJobOperator(
            task_id="test", api_version="V1beta1", job_spec={}
        )
        hook = operator.hook
        self.assertIsNotNone(hook)
        self.assertIsInstance(hook, BacalhauHook)

    def test_get_hook(self):
        """Test get_hook method."""
        operator = BacalhauSubmitJobOperator(
            task_id="test", api_version="V1beta1", job_spec={}
        )
        hook = operator.get_hook()
        self.assertIsNotNone(hook)
        self.assertIsInstance(hook, BacalhauHook)

    def test_execute(self):

        dagrun = self.dag.create_dagrun(
            state=DagRunState.RUNNING,
            execution_date=DEFAULT_DATE,
            # data_interval=DEFAULT_DATE,
            start_date=DEFAULT_DATE,
            run_type=DagRunType.MANUAL,
        )
        ti = dagrun.get_task_instance(task_id=TEST_TASK_ID)
        ti.task = self.dag.get_task(task_id=TEST_TASK_ID)
        result = ti.task.execute(ti.get_template_context())

        self.assertIsNotNone(result)
        self.assertIsInstance(result, str)
