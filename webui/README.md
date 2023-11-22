# Bacalhau WebUI

This project was bootstrapped with [Create React App](https://github.com/facebook/create-react-app). To help develop the WebUI we have built to interact with the Bacalhau network follow the directions below:

## Getting Started

First, install dependencies:

```bash
npm install
```

Next, run the development server:

```bash
npm start
```

The page will reload if you make edits.\
You will also see any lint errors in the console.

### `npm test`

Launches the test runner in the interactive watch mode.\
See the section about [running tests](https://facebook.github.io/create-react-app/docs/running-tests) for more information.

Before commiting new code, you will need to lint it by running:

```bash
npm lint
```

Open [http://localhost:3000](http://localhost:3000) with your browser to see the result.

You can start editing the page by modifying `pages/index.tsx`. The page auto-updates as you edit the file.

[API routes](https://nextjs.org/docs/api-routes/introduction) can be accessed on [http://localhost:3000/api/hello](http://localhost:3000/api/hello). This endpoint can be edited in `pages/api/hello.ts`.

The `pages/api` directory is mapped to `/api/*`. Files in this directory are treated as [API routes](https://nextjs.org/docs/api-routes/introduction) instead of React pages.

### `npm run build`

Builds the app for production to the `build` folder.\
It correctly bundles React in production mode and optimizes the build for the best performance.

The build is minified and the filenames include the hashes.\
Your app is ready to be deployed!

See the section about [deployment](https://facebook.github.io/create-react-app/docs/deployment) for more information.

### `npm run eject`

**Note: this is a one-way operation. Once you `eject`, you can’t go back!**

If you aren’t satisfied with the build tool and configuration choices, you can `eject` at any time. This command will remove the single build dependency from your project.

Instead, it will copy all the configuration files and the transitive dependencies (webpack, Babel, ESLint, etc) right into your project so you have full control over them. All of the commands except `eject` will still work, but they will point to the copied scripts so you can tweak them. At this point you’re on your own.

You don’t have to ever use `eject`. The curated feature set is suitable for small and middle deployments, and you shouldn’t feel obligated to use this feature. However we understand that this tool wouldn’t be useful if you couldn’t customize it when you are ready for it.

## Spnning up the Dashboard (still in development):
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

## Learn More

You can learn more in the [Create React App documentation](https://facebook.github.io/create-react-app/docs/getting-started).

To learn React, check out the [React documentation](https://reactjs.org/).
