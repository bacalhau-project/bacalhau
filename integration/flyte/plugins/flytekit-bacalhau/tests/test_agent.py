import pytest
import os
from flytekitplugins.bacalhau import BacalhauTask, BacalhauAgent
from flytekitplugins.bacalhau.task import BacalhauConfig
from flytekit import workflow

from bacalhau_sdk.api import submit
from bacalhau_sdk.config import get_client_id
from bacalhau_apiclient.models.spec import Spec




def test_bacalhau_agent():
    