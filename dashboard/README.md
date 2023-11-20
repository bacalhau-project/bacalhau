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

**N.b. (still in development):**
For testing the API connection look at [bacalhau.ts](https://github.com/bacalhau-project/bacalhau/blob/e61b1ebb669043b8b4113437b3035064c0d28f46/dashboard/src/pages/api/bacalhau.ts). Here you will find the HOST and PORT cofiguration. To test the connection with `go run . devstack` you will need to change the port configuration.

## Project Set Up

This WebUI Dashboard is built with [Next.js](https://nextjs.org/) and is bootstapped with [`create-next-app`](https://github.com/vercel/next.js/tree/canary/packages/create-next-app).

Learn More:
- [Next.js Documentation](https://nextjs.org/docs) - learn about Next.js features and API.
- [Learn Next.js](https://nextjs.org/learn) - an interactive Next.js tutorial.

You can check out [the Next.js GitHub repository](https://github.com/vercel/next.js/) - your feedback and contributions are welcome!
