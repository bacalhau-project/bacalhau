## running locally

Have a few ternminal windows.

### 1.

```bash
go run . serve --port 8080
```

### 2.

```bash
go run . serve --jsonrpc-port=1235 --peer /ip4/127.0.0.1/tcp/8080/p2p/<peer id printed by 1>
```