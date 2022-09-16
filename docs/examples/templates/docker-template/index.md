---
sidebar_label: "Docker Template"
sidebar_position: 2
---
# Title of Example

> This notebook shows how to use Docker in a notebook. Other simpler templates are available in the [templates](..) directory.

This example requires Docker. If you don't have Docker installed, you can install it from [here](https://docs.docker.com/install/).


```python
!command -v docker >/dev/null 2>&1 || { echo >&2 "I require docker but it's not installed.  Aborting."; exit 1; }
```

## Example Container Run

This example runs a container.


```bash
%%bash
docker run --rm hello-world
```

    
    Hello from Docker!
    This message shows that your installation appears to be working correctly.
    
    To generate this message, Docker took the following steps:
     1. The Docker client contacted the Docker daemon.
     2. The Docker daemon pulled the "hello-world" image from the Docker Hub.
        (arm64v8)
     3. The Docker daemon created a new container from that image which runs the
        executable that produces the output you are currently reading.
     4. The Docker daemon streamed that output to the Docker client, which sent it
        to your terminal.
    
    To try something more ambitious, you can run an Ubuntu container with:
     $ docker run -it ubuntu bash
    
    Share images, automate workflows, and more with a free Docker ID:
     https://hub.docker.com/
    
    For more examples and ideas, visit:
     https://docs.docker.com/get-started/
    


## Example Container Build and Run

And this example shows you how to build a container and run it.


```python
%%writefile Dockerfile
FROM ubuntu:latest
RUN echo "built"
```

    Overwriting Dockerfile



```bash
%%bash
docker build -t myimage .
docker run --rm myimage echo "works!"
```

    works!


    #1 [internal] load build definition from Dockerfile
    #1 sha256:74af154ae2ac123245b8df9e2b42787668440fd24078ceb686663bc3731b2578
    #1 transferring dockerfile: 78B done
    #1 DONE 0.0s
    
    #2 [internal] load .dockerignore
    #2 sha256:da1201b85feb52129655bd8a173ed303b1f452fec29db8561615e8945fa430a1
    #2 transferring context: 2B done
    #2 DONE 0.0s
    
    #3 [internal] load metadata for docker.io/library/ubuntu:latest
    #3 sha256:abd44fdc5704a1f31ce24272a9c459712f2a16d9c7c7ce9bcc7a9b6c11e01aa9
    #3 DONE 0.6s
    
    #4 [1/2] FROM docker.io/library/ubuntu:latest@sha256:20fa2d7bb4de7723f542be5923b06c4d704370f0390e4ae9e1c833c8785644c1
    #4 sha256:ecb55e1c6e99b222a30124fe4f604e87265732d794a19c4d0256314619091953
    #4 DONE 0.0s
    
    #5 [2/2] RUN echo "built"
    #5 sha256:055bd4b2d00eacbbdc2aed5df1c88d58721f8ea4b1b4e6c01536d6e656d64f49
    #5 CACHED
    
    #6 exporting to image
    #6 sha256:e8c613e07b0b7ff33893b694f7759a10d42e180f2b4dc349fb57dc6b71dcab00
    #6 exporting layers done
    #6 writing image sha256:13f4b7e419b299b9e194b5e50af30861ebaea6e93c2e734b413f6a1cc935d863 done
    #6 naming to docker.io/library/myimage done
    #6 DONE 0.0s
    
    Use 'docker scan' to run Snyk tests against images to find vulnerabilities and learn how to fix them

