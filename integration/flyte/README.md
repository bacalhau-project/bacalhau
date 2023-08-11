# Flyte Bacalhau Plugin

This repo adheres to the [Flyte official guidelines](https://github.com/flyteorg/flytekit/tree/master/plugins#guidelines-) for flytekit plugins and is structured such that the `plugins/flytekit-bacalhau` folder can be merged into Flytekit repository.

## Install 

The only prerequisite is having `flytekit` installed (via pip) ⚡️

```
pip install flytekitplugins-bacalhau
```

## Examples

Here's a Hello World workflow submitting a job to Bacalhau. Run it with: `pyflyte run my-wf-file.py my_workflow`

```python
from flytekit import workflow, kwtypes
from flytekitplugins.bacalhau import BacalhauTask

bacalhau_task = BacalhauTask(
    name="hello_world",
    inputs=kwtypes(
        spec=dict,
        api_version=str,
    ),
)


@workflow
def my_workflow():
    my_bacalhau_task = bacalhau_task(
        api_version="V1beta1",
        spec=dict(
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
        ),
    )
    print(my_bacalhau_task)
```

Curious to see more complex workflows 🧐?
Find more [in the examples folder](./plugins/flytekit-bacalhau/examples/) ✨.


## Kubernetes Deployment

To deploy Flyte on Kubernetes, please follow the [official instructions](https://docs.flyte.org/en/latest/deployment/index.html).
You'll have to install our [custom Bacalhau Agent](https://docs.flyte.org/projects/cookbook/en/latest/auto_examples/extend_flyte/agent_service.html#update-flyteagent) as well; to faciliate you in doing that we do provide extra [deployment instructions for GKE](./DEPLOYMENT.md) 

## Development :computer:

Similarly to the [official development guidelines](https://docs.flyte.org/projects/flytekit/en/latest/contributing.html#contribute-code), we use a virtual environment to develop this Flytekit plugin.

### 1. Setup (Do Once)

```bash
virtualenv ~/.virtualenvs/flytekit-bacalhau
source ~/.virtualenvs/flytekit-bacalhau/bin/activate
make setup
```

> It is important to maintain separate virtualenvs for `flytekit development` and `flytekit` use because installing a Python library in editable mode will link it to your source code. The behavior will change as you work on the code, check out different branches, etc.

This will install Flytekit dependencies and Flytekit in editable mode. This links your virtual Python’s site-packages with your local repo folder, allowing your local changes to take effect when the same Python interpreter runs import flytekit.


### 2. Plugin Development

```bash
source ~/.virtualenvs/flytekit-bacalhau/bin/activate
cd plugins
pip install -e .
```

This should install all the plugins in editable mode as well.

#### Unit tests

```bash
make test
```

### Formatting, Linting, etc.

```bash
source ~/.virtualenvs/flytekit/bin/activate
make fmt
make lint
make spellcheck
```

---

**Questions?** Feel free to contact the author [@enricorotundo](https://github.com/enricorotundo) at [winder.ai](winder.ai)