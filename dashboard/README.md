# Bacalhau WebUI

To help develop the WebUI we have built to interact with the Bacalhau network follow the directions below:

## Getting Started

First, install dependencies:

```bash
npm install
```

Next, run the development server:

```bash
npm run dev
```

Before commiting new code, you will need to lint it by running:

```bash
npm run lint
```

Open [http://localhost:3000](http://localhost:3000) with your browser to see the result.

You can start editing the page by modifying `pages/index.tsx`. The page auto-updates as you edit the file.

[API routes](https://nextjs.org/docs/api-routes/introduction) can be accessed on [http://localhost:3000/api/hello](http://localhost:3000/api/hello). This endpoint can be edited in `pages/api/hello.ts`.

The `pages/api` directory is mapped to `/api/*`. Files in this directory are treated as [API routes](https://nextjs.org/docs/api-routes/introduction) instead of React pages.

### Spnning up the Dashboard (still in development):
For spinning up & testing the dashboard with the API connection to the bacalhau network you can run:

```bash
bacalhau serve --node-type requester,compute
``` 

to spin up your own bacalhau cluster. This will use the default port `1234`.

In [`bacalhau.ts`](https://github.com/bacalhau-project/bacalhau/blob/e61b1ebb669043b8b4113437b3035064c0d28f46/dashboard/src/pages/api/bacalhau.ts) you will find the HOST and PORT cofiguration. Test the connection with: 

```bash
go run . devstack
``` 
You will need to change the port configuration in [`bacalhau.ts`](https://github.com/bacalhau-project/bacalhau/blob/e61b1ebb669043b8b4113437b3035064c0d28f46/dashboard/src/pages/api/bacalhau.ts) to the one generated when you run the command.

## Project Set Up

This WebUI Dashboard is built with [Next.js](https://nextjs.org/) and is bootstapped with [`create-next-app`](https://github.com/vercel/next.js/tree/canary/packages/create-next-app).

Learn More:
- [Next.js Documentation](https://nextjs.org/docs) - learn about Next.js features and API.
- [Learn Next.js](https://nextjs.org/learn) - an interactive Next.js tutorial.

You can check out [the Next.js GitHub repository](https://github.com/vercel/next.js/) - your feedback and contributions are welcome!
