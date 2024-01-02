from collections import OrderedDict

import pytest
from urllib3.exceptions import LocationParseError

from flytekitplugins.bacalhau import BacalhauTask
from flytekit import workflow
from flytekit import kwtypes
from flytekit.configuration import Image, ImageConfig, SerializationSettings
from flytekit.extend import get_serializable


test_spec = dict(
    engine="Docker",
    verifier="Noop",
    PublisherSpec={"type": "IPFS"},
    docker={
        "image": "ubuntu",
        "entrypoint": ["echo", "Flyte is awesome!"],
    },
    language={"job_context": None},
    wasm=None,
    resources=None,
    timeout=1800,
    outputs=[
        {
            "storage_source": "IPFS",
            "name": "outputs",
            "path": "/outputs",
        }
    ],
    deal={"concurrency": 1},
    do_not_track=True,
)


def test_local_exec():
    bacalhau_task = BacalhauTask(
        name="hello_world",
        inputs=kwtypes(
            spec=dict,
            api_version=str,
        ),
    )

    assert len(bacalhau_task.interface.inputs) == 2
    assert len(bacalhau_task.interface.outputs) == 1

    # will not run locally
    with pytest.raises(LocationParseError):
        bacalhau_task(
            api_version="V1beta1",
            spec=test_spec,
        )


def test_serialization():
    task_name = "hello_world"
    bacalhau_task = BacalhauTask(
        name=task_name,
        inputs=kwtypes(
            spec=dict,
            api_version=str,
        ),
    )

    @workflow
    def my_wf():
        return bacalhau_task(api_version="V1beta1", spec=test_spec)

    default_img = Image(name="default", fqn="test", tag="tag")
    serialization_settings = SerializationSettings(
        project="proj",
        domain="dom",
        version="123",
        image_config=ImageConfig(default_image=default_img, images=[default_img]),
        env={},
    )

    task_spec = get_serializable(OrderedDict(), serialization_settings, bacalhau_task)

    # check Task
    assert task_spec.template.id.name == task_name
    assert task_spec.template.type == "bacalhau_task"
    assert len(task_spec.template.interface.inputs) == 2
    assert len(task_spec.template.interface.outputs) == 1

    # check Workflow
    workflow_spec = get_serializable(OrderedDict(), serialization_settings, my_wf)
    assert workflow_spec.template.nodes[0].inputs[0].binding.scalar.primitive.string_value == "V1beta1"
    assert workflow_spec.template.nodes[0].inputs[1].binding.scalar.generic["engine"] == test_spec["engine"]
    assert workflow_spec.template.nodes[0].inputs[1].binding.scalar.generic["verifier"] == test_spec["verifier"]
    assert workflow_spec.template.nodes[0].inputs[1].binding.scalar.generic["wasm"] == test_spec["wasm"]
    assert workflow_spec.template.nodes[0].inputs[1].binding.scalar.generic["engine"] == test_spec["engine"]
    assert workflow_spec.template.nodes[0].inputs[1].binding.scalar.generic["resources"] == test_spec["resources"]
    assert workflow_spec.template.nodes[0].inputs[1].binding.scalar.generic["timeout"] == test_spec["timeout"]
    assert workflow_spec.template.nodes[0].inputs[1].binding.scalar.generic["do_not_track"] == test_spec["do_not_track"]
    