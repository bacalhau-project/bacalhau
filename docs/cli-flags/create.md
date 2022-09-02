---
sidebar_label: 'Bacalhau Create'
sidebar_position: 2
---

# Bacalhau Create

Submit a job to the network in a declarative way by writing a jobspec instead of writing a command. JSON and YAML formats are accepted.

## Usage


```bash
  bacalhau create FILENAME
```

## Examples

An Example jobspec in YAML format

```yaml
apiVersion: v1alpha1
engine: Docker
verifier: Ipfs
job_spec_docker:
  image: gromacs/gromacs
  entrypoint:
    - /bin/bash
    - -c
    - echo 15 | gmx pdb2gmx -f input/1AKI.pdb -o output/1AKI_processed.gro -water spc
  env: []
job_spec_language:
  language: ''
  language_version: ''
  deterministic: false
  context:
    engine: ''
    name: ''
    cid: ''
    path: ''
  command: ''
  program_path: ''
  requirements_path: ''
resources:
  cpu: ''
  gpu: ''
  memory: ''
  disk: ''
inputs:
  - engine: ipfs
    name: ''
    cid: QmeeEB1YMrG6K8z43VdsdoYmQV46gAPQCHotZs9pwusCm9
    path: /input
  - engine_name: urldownload
    name: ''
    url: https://foo.bar.io/foo_data.txt
    path: /app/foo_data_1.txt
outputs:
  - engine: ipfs
    name: output
    cid: ''
    path: /output
annotations: null
```

An Example jobspoec in JSON format

```json
{
  "apiVersion": "v1alpha1",
  "engine": "Docker",
  "verifier": "Ipfs",
  "job_spec_docker": {
      "image": "gromacs/gromacs",
      "entrypoint": [
          "/bin/bash",
          "-c",
          "echo 15 | gmx pdb2gmx -f input/1AKI.pdb -o output/1AKI_processed.gro -water spc"
      ],
      "env": []
  },
  "job_spec_language": {
      "language": "",
      "language_version": "",
      "deterministic": false,
      "context": {
          "engine": "",
          "name": "",
          "cid": "",
          "path": ""
      },
      "command": "",
      "program_path": "",
      "requirements_path": ""
  },
  "resources": {
      "cpu": "",
      "gpu":"",
      "memory": "",
      "disk": ""
  },
  "inputs": [
      {
          "engine": "ipfs",
          "name": "",
          "cid": "QmeeEB1YMrG6K8z43VdsdoYmQV46gAPQCHotZs9pwusCm9",
          "path": "/input"
      }
  ],
  "outputs": [
      {
          "engine": "ipfs",
          "name": "output",
          "cid": "",
          "path": "/output"
      }
  ],
  
  "annotations": null
}
```
