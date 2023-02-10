from airflow.models import BaseOperator
from airflow.models.baseoperator import BaseOperatorLink
from airflow.models.taskinstance import TaskInstanceKey
from airflow.compat.functools import cached_property
from airflow.utils.context import Context
import time
from bacalhau.hooks import BacalhauHook
class BacalhauLink(BaseOperatorLink):
    name = "Bacalhau"

    def get_link(self, operator: BaseOperator, *, ti_key: TaskInstanceKey):
        return "https://docs.bacalhau.org/"

class BacalhauSubmitJobOperator(BaseOperator):
    """Submit a job to the Bacalhau service."""
    ui_color = "#36cbfa"
    ui_fgcolor = "#0554f9"
    custom_operator_name = "BacalhauSubmitJob"

    template_fields = (
        # 'inputs',
        # 'input_volumes',
        # 'job_spec',
        'input_volumes', #Deep nested fields can also be substituted, as long as all intermediate fields are marked as template fields
    )

    def __init__(self,
                 api_version: str,
                 job_spec: dict,
                #  inputs: dict = None,
                 input_volumes: dict = None,
                 **kwargs) -> None:
        super().__init__(**kwargs)
        self.api_version = api_version
        # inject inputs and input_volumes into job_spec

        self.job_spec = job_spec
        # self.inputs = inputs
        self.input_volumes = input_volumes

        

    def execute(self, context: Context) -> dict:
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
                            unravelled_input_volumes.append({
                                "cid": cid,
                                "path": mount_point,
                                "storagesource": "ipfs" # TODO make this configurable (filecoin, etc)
                            })
                    else:
                        unravelled_input_volumes.append({
                            "cid": cids_str,
                            "path": mount_point,
                            "storagesource": "ipfs" # TODO make this configurable (filecoin, etc)
                        })

        if len(unravelled_input_volumes) > 0:
            if "inputs" not in self.job_spec:
                self.job_spec["inputs"] = []
            self.job_spec["inputs"] = self.job_spec["inputs"] + unravelled_input_volumes
        
        print("self.job_spec")
        print(self.job_spec)



        job_id = self.hook.submit_job(
            api_version=self.api_version, job_spec=self.job_spec)
        context["ti"].xcom_push(key="bacalhau_job_id", value=job_id)
        print("job_id")
        print(job_id)

        # use hook to wait for job to complete
        # TODO move this logic to a hook
        while True:
            events = self.hook.get_events(job_id)
            
            terminate = False
            for event in events['events']:
                if "event_name" in event:
                    # TODO fix case when event hangs/errors out/never completes
                    if event["event_name"] == "ComputeError" or event["event_name"] == "ResultsPublished":
                        # print(event)
                        terminate = True
                        break
                    # else:
                    #     print(event)
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
