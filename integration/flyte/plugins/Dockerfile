FROM python:3.8-slim-buster

LABEL Enrico Rotundo <enrico@winder.ai>
LABEL org.opencontainers.image.source=https://github.com/flyteorg/flytekit

WORKDIR /root
ENV PYTHONPATH /root

# flytekit will autoload the agent if package is installed.
RUN pip install flytekitplugins-bacalhau
CMD pyflyte serve --port 8000
