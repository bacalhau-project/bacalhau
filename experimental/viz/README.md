# visualize a bacalhau devstack cluster


Visualize a 300 node devstack stress test cluster that's running on 10.0.0.{1,2,3} on API ports 10000-10099 (100 nodes).

```
go run main.go 10.0.0.1 10000 10099 10.0.0.2 10000 10099 10.0.0.3 10000 10099
```

```
open http://localhost:31337
```

## running against our production nodes

If you want to visualize our production nodes, you can do so by running:

```bash
args=""
for ip in $(gcloud compute instances list | grep bacalhau-vm | awk '{print $5}'); do
  args="$args $ip 1234 1234"
done
go run main.go $args
```