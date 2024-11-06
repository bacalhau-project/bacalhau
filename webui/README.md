## Getting Started

Install dependencies:

```bash
npm install
```

Run the development server:

```bash
npm run dev
```

Generate API:
```bash
npm run generate-api
```

Test the WebUI with active Bacalhau nodes:
```bash
# Run from project root
make build-webui
make build-dev
bacalhau serve --node-type=requester,compute --web-ui
```

Open [http://localhost:8438](http://localhost:8438) with your browser to see the result.
