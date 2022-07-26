# benchmark hack

```
make devstack
```
or to test without actually running docker jobs

```
make devstack-noop
```

```
go install
```

copy paste export commands, e.g.
```
export API_PORT_0=35601
```

test one job

```
cd benchmark_hack
./submit.sh
```

```
./explode.sh
```
