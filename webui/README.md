# Bacalhau WebUI

## Dependencies
You will need to install all the dependencies in the `.tool-versions` file.
```bash
asdf install
```

Then install corepack which you'll need for the latest yarn version.
```bash
npm install -g corepack
corepack --yes prepare yarn@latest --activate
```

## Install all yarn dependencies
```bash
yarn install
```


## Spinning up the Dashboard for Development:

For spinning up & testing the dashboard with the API connection to the bacalhau network you can run:

```bash
cd webui

yarn run build

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
