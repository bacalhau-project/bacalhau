# Example of sharding with video filters

```bash
cid="Qmd9CBYpdgCLuCKRtKRRggu24H72ZUrGax5A9EYvrbC72j"
time bacalhau docker run \
  -v $cid:/inputs \
  --cpu 2 \
  --memory 1Gb \
  --wait \
  --wait-timeout-secs 10000 \
  binocarlos/video-resize-example \
  bash /entrypoint.sh /inputs /outputs
time bacalhau docker run \
  -v $cid:/inputs \
  --cpu 2 \
  --memory 1Gb \
  --wait \
  --wait-timeout-secs 10000 \
  --sharding-glob-pattern "/inputs/*.mp4" \
  --sharding-batch-size 1 \
  binocarlos/video-resize-example \
  bash /entrypoint.sh /inputs /outputs
```