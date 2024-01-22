# Bacalhau WebUI

This project was bootstrapped with [Create React App](https://github.com/facebook/create-react-app). To help develop the WebUI we have built to interact with the Bacalhau network follow the directions below:

## Getting Started

First, install dependencies:

```bash
npm install
```

Next, run the development server. The page will reload if you make edits:

```bash
npm start
```

Before committing new code, you will need to lint it by running:

```bash
npm run lint
```

Open [http://localhost:3000](http://localhost:3000) with your browser to see the result.

You can start editing the page by modifying `pages/index.tsx`. The page auto-updates as you edit the file.

```bash
npm run build
```

Builds the app for production to the `build` folder.\
It correctly bundles React in production mode and optimizes the build for the best performance.

The build is minified and the filenames include the hashes.\
Your app is ready to be deployed!

See the section about [deployment](https://facebook.github.io/create-react-app/docs/deployment) for more information.

```bash
npm run eject
```

**Note: this is a one-way operation. Once you `eject`, you can’t go back!**

If you aren’t satisfied with the build tool and configuration choices, you can `eject` at any time. This command will remove the single build dependency from your project.

Instead, it will copy all the configuration files and the transitive dependencies (webpack, Babel, ESLint, etc) right into your project so you have full control over them. All of the commands except `eject` will still work, but they will point to the copied scripts so you can tweak them. At this point you’re on your own.

You don’t have to ever use `eject`. The curated feature set is suitable for small and middle deployments, and you shouldn’t feel obligated to use this feature. However we understand that this tool wouldn’t be useful if you couldn’t customize it when you are ready for it.

## Spinning up the Dashboard for Development:

For spinning up & testing the dashboard with the API connection to the bacalhau network you can run:

```bash
cd webui

npm run build

cd..

make build

./bin/$(go env GOOS)/$(go env GOARCH)/bacalhau serve --node-type=requester,compute --peer=none --web-ui
```

The above will spin up your own bacalhau cluster. This will use the default port `1234`. Visit `http://127.0.0.1/` to see WebUI.

## Interaction with Bacalhau

In [`bacalhau.ts`](https://github.com/bacalhau-project/bacalhau/blob/e61b1ebb669043b8b4113437b3035064c0d28f46/dashboard/src/pages/api/bacalhau.ts) you will find Bacalhau API configuration. 

[`webui.go`](https://github.com/bacalhau-project/bacalhau/blob/b6c52302c0bc20a82c3b3eb8b674c7919aab5747/webui/webui.go) serves as a web server to deliver the webui (React code), handling both the serving of static assets embedded in the binary and dynamic routing for client-side navigation.

## Learn More

You can learn more in the [Create React App documentation](https://facebook.github.io/create-react-app/docs/getting-started).

To learn React, check out the [React documentation](https://reactjs.org/).
