"""
Airflow operators for Bacalhau.
"""
import time

from airflow.compat.functools import cached_property
from airflow.models import BaseOperator
from airflow.models.baseoperator import BaseOperatorLink
from airflow.models.taskinstance import TaskInstanceKey
from airflow.utils.context import Context
from bacalhau_airflow.hooks import BacalhauHook


class BacalhauLink(BaseOperatorLink):
    """Link to the Bacalhau service."""

    name = "Bacalhau"

    def get_link(self, operator: BaseOperator, *, ti_key: TaskInstanceKey):
        """Get the URL of the Bacalhau public service."""
        return "https://docs.bacalhau.org/"


class BacalhauSubmitJobOperator(BaseOperator):
    """Submit a job to the Bacalhau service."""

    ui_color = "#36cbfa"
    ui_fgcolor = "#0554f9"
    custom_operator_name = "BacalhauSubmitJob"

    template_fields = ("input_volumes",)

    def __init__(
        self,
        api_version: str,
        job_spec: dict,
        #  inputs: dict = None,
        input_volumes: list = [],
        **kwargs
    ) -> None:
        """Constructor of the operator to submit a Bacalhau job.

        Args:
            api_version (str): The API version to use. Example: "V1beta1".
            job_spec (dict): A dictionary with the job specification. See example dags for more details.
            input_volumes (list, optional):
                Use this parameter to pipe an upstream's output into a Bacalhau task.

                This makes use of Airflow's XComs to support communication between tasks.
                Please learn more about XComs here: https://airflow.apache.org/docs/apache-airflow/stable/core-concepts/xcoms.html

                Every task of `BacalhauSubmitJobOperator` stores an XCom key-value named `cids` (type `str`), a CID comma-separated list of the output shards.
                That way, a downstream task can use the `input_volumes` parameter to mount the upstream's output shards into its own input volumes.

                The format of this parameter is a list of strings, where each string is a pair of `cid` and `mount_point` separated by a colon.
                Defaults to [].

                For example, the list `[ "{{ task_instance.xcom_pull(task_ids='run-1', key='cids') }}:/datasets" ]` takes all shards created by task "run-1" and mounts them at "/datasets".
        """
        super().__init__(**kwargs)
        self.api_version = api_version
        # inject inputs and input_volumes into job_spec

        self.job_spec = job_spec
        # self.inputs = inputs
        self.input_volumes = input_volumes

    def execute(self, context: Context) -> str:
        """Execute the operator.

        Args:
            context (Context):

        Returns:
            str: The job ID created.
        """

        # TODO do the same for inputs?

        # TODO manage the case when 1+ cids are passed in input_volumes and must be mounted in children mount points
        # 'failed to create container: Error response from daemon: Duplicate

        unravelled_input_volumes = []
        if self.input_volumes and len(self.input_volumes) > 0:
            for input_volume in self.input_volumes:
                if type(input_volume) == str:
                    cids_str, mount_point = input_volume.split(":")
                    if "," in cids_str:
                        cids = cids_str.split(",")
                        for cid in cids:
                            unravelled_input_volumes.append(
                                {
                                    "cid": cid,
                                    "path": mount_point,
                                    "storagesource": "ipfs",  # TODO make this configurable (filecoin, etc)
                                }
                            )
                    else:
                        unravelled_input_volumes.append(
                            {
                                "cid": cids_str,
                                "path": mount_point,
                                "storagesource": "ipfs",  # TODO make this configurable (filecoin, etc)
                            }
                        )

        if len(unravelled_input_volumes) > 0:
            if "inputs" not in self.job_spec:
                self.job_spec["inputs"] = []
            self.job_spec["inputs"] = self.job_spec["inputs"] + unravelled_input_volumes

        print("self.job_spec")
        print(self.job_spec)

        job_id = self.hook.submit_job(
            api_version=self.api_version, job_spec=self.job_spec
        )
        context["ti"].xcom_push(key="bacalhau_job_id", value=job_id)
        print("job_id")
        print(job_id)

        # use hook to wait for job to complete
        # TODO move this logic to a hook
        while True:
            events = self.hook.get_events(job_id)

            terminate = False
            print(events["events"])
            if events["events"]:
                for event in events["events"]:
                    if "type" in event:
                        if event["type"] == "JobLevel":
                            if "job_state" in event:
                                if event["job_state"] and "new" in event["job_state"]:
                                    # TODO fix case when event hangs/errors out/never completes
                                    if (
                                        event["job_state"]["new"] == "ComputeError"
                                        or event["job_state"]["new"] == "Completed"
                                        or event["job_state"]["new"] == "Cancelled"
                                        or event["job_state"]["new"] == "Error"
                                    ):
                                        terminate = True
                                        break
                if terminate:
                    break
                print("clock it ticking...")
                time.sleep(2)

        # fetch all shards' resulting CIDs
        results = self.hook.get_results(job_id)
        # join CIDs comma separated..
        cids = []
        for result in results:
            cids.append(result["data"]["cid"])
        cids_str = ",".join(cids)
        # print(cids_str)
        context["ti"].xcom_push(key="cids", value=cids_str)

        return job_id

    @cached_property
    def hook(self):
        """Create and return an BacalhauHook (cached)."""
        return BacalhauHook()

    def get_hook(self):
        """Create and return an BacalhauHook (cached)."""
        return self.hook
