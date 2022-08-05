# Example of sharding with video filters

```bash
cid=$(ipfs add folder-with-videos-inside)
bacalhau docker run \
  -v $cid:/inputs \
  -o results:/results \
  binocarlos/bacalhau-video-sharding \
  bash /entrypoint.sh /inputs /results
```