## Getting Started

Install dependencies:

```bash
yarn install
```

Run the development server:

```bash
yarn dev
```

## API Changes

Generate new swagger schema:
```bash
# Run from project root
make generate-swagger
```

Update WebUI generated API:
```bash
yarn generate-api
```

## Testing

Test the WebUI with active Bacalhau nodes:
```bash
# Run from project root
make build-webui
make build-dev
bacalhau serve --compute --orchestrator -c WebUI.Enabled
```

Open [http://localhost:8438](http://localhost:8438) with your browser to see the result.
