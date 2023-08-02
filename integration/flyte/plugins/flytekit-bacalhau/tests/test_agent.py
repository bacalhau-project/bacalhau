import json
from dataclasses import asdict
from datetime import timedelta

import grpc
from google.protobuf import struct_pb2
from unittest import mock
from unittest.mock import MagicMock

from flytekit.extend.backend.base_agent import AgentRegistry
from flytekit.interfaces.cli_identifiers import Identifier
from flytekit.models.core.identifier import ResourceType
from flytekit import workflow
from flytekit.models import literals, task, types
from flytekit.models.task import Sql, TaskTemplate
import flytekit.models.interface as interface_models

from flytekitplugins.bacalhau.agent import Metadata


@mock.patch("flytekitplugins.bacalhau.agent.submit")
@mock.patch("flytekitplugins.bacalhau.agent.results")
@mock.patch("flytekitplugins.bacalhau.agent.get_client_id")
def test_bacalhau_agent(mock_get_client_id, mock_results, mock_submit):
    job_id = "dummy_id"

    class MockResponseJobMetadata:
        def __init__(self):
            self.id = job_id

    class MockResponseJob:
        def __init__(self):
            self.metadata = MockResponseJobMetadata()

    class MockSubmitResponse:
        def __init__(self):
            self.job = MockResponseJob()

    class MockResultsCid:
        def __init__(self):
            self.cid = "dummy_cid"

    class MockResultsData:
        def __init__(self):
            self.data = MockResultsCid()
            
    class MockResultsResponse:
        def __init__(self):
            self.results = [MockResultsData()]

    mock_submit.return_value = MockSubmitResponse()
    mock_results.return_value = MockResultsResponse()
    mock_results.state.return_value = "SUCCEEDED"
    mock_get_client_id.return_value = "dummy_client_id"

    ctx = MagicMock(spec=grpc.ServicerContext)
    agent = AgentRegistry.get_agent(ctx, "bacalhau_task")

    task_id = Identifier(
        resource_type=ResourceType.TASK,
        project="project",
        domain="domain",
        name="name",
        version="version",
    )

    task_config = {
        # "Location": "us-central1",
        # "ProjectID": "dummy_project",
    }
    int_type = types.LiteralType(types.SimpleType.INTEGER)
    interfaces = interface_models.TypedInterface(
        {
            "a": interface_models.Variable(int_type, "description1"),
            "b": interface_models.Variable(int_type, "description2"),
        },
        {},
    )
    s = struct_pb2.Struct()
    s.update({
        "key": "value",
        "deal": {
            "concurrency": 1.0,
        }
    })
    task_inputs = literals.LiteralMap(
        {
            "api_version": literals.Literal(scalar=literals.Scalar(primitive=literals.Primitive(string_value="some-api-version"))),
            "client_id": literals.Literal(scalar=literals.Scalar(primitive=literals.Primitive(string_value="some-client-id"))),
            "spec": literals.Literal(scalar=literals.Scalar(generic=s)),
        },
    )
    
    task_metadata = task.TaskMetadata(
        True,
        task.RuntimeMetadata(
            task.RuntimeMetadata.RuntimeType.FLYTE_SDK, "1.0.0", "python"
        ),
        timedelta(days=1),
        literals.RetryStrategy(3),
        True,
        "0.1.1b0",
        "This is deprecated!",
        True,
        "A",
    )

    dummy_template = TaskTemplate(
        id=task_id,
        custom=task_config,
        metadata=task_metadata,
        interface=interfaces,
        type="bacalhau_task",
    )

    metadata_bytes = json.dumps(asdict(Metadata(job_id=job_id))).encode("utf-8")

    assert (
        agent.create(ctx, "/tmp", dummy_template, task_inputs).resource_meta
        == metadata_bytes
    )

    res = agent.get(ctx, metadata_bytes)

    assert (
        res.resource.outputs.literals["results"].scalar.primitive.string_value == "dummy_cid"
    )
    agent.delete(ctx, metadata_bytes)
    # mock_instance.cancel_job.assert_called()
