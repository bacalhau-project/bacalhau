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
```bash
export BACALHAU_API_PORT=35601
```

test one job

```bash
cd benchmark
bash submit.sh
```

```bash
bash explode.sh
```
