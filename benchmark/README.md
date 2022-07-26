# benchmark hack

```bash
make devstack
```
or to test without actually running docker jobs

```
make devstack-noop
```

```bash
make devstack-noop
```

```bash
go install
```

copy paste export commands, e.g.
```
export BACALHAU_API_PORT_0=35601
```

test one job

```
cd benchmark
bash submit.sh
```

```
bash explode.sh
```
