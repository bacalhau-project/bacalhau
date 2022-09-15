---
sidebar_label: 'Video Filter Sharding'
sidebar_position: 2
---
# Example of sharding with video filters


```python
%env cid=Qmd9CBYpdgCLuCKRtKRRggu24H72ZUrGax5A9EYvrbC72j

!bacalhau docker run \
    -v ${cid}:/inputs \
    --cpu 2 \
    --memory 1Gb \
    binocarlos/video-resize-example \
    bash /entrypoint.sh /inputs /outputs
```

    env: cid=Qmd9CBYpdgCLuCKRtKRRggu24H72ZUrGax5A9EYvrbC72j
    Qmd9CBYpdgCLuCKRtKRRggu24H72ZUrGax5A9EYvrbC72j



```python
!bacalhau docker run \
    -v $cid:/inputs \
    --cpu 2 \
    --memory 1Gb \
    --sharding-base-path "/inputs" \
    --sharding-glob-pattern "*.mp4" \
    --sharding-batch-size 1 \
    binocarlos/video-resize-example \
    bash /entrypoint.sh /inputs /outputs
```

    d70adb2f-587d-4cc3-9eb4-1377da9bdc47



```python

```
