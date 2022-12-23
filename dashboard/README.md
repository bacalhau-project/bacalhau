## dev

You will need 3 terminal panes:

Start devstack:

```bash
export PREDICTABLE_API_PORT=1
make devstack
```

Start the api:

```bash
cd dashboard/api
go run main.go 127.0.0.1 20000 20000 127.0.0.1 20001 20001 127.0.0.1 20001 20001
```

Start the frontend:

```bash
cd dashboard/frontend
yarn install
yarn dev
```

Open the browser: http://127.0.0.1:8080