# Bacalhau Docs

This repository contains the user documentation of the [Bacalhau project](https://github.com/filecoin-project/bacalhau).
These are accessible at https://docs.bacalhau.org/.

# Contribute

## Found inconsistencies in the docs?

Please feel free to open an issue or submit a pull request to this repo, every contribution is more than welcome! :balloon:

## Develop docs locally

### Prerequistes

* Git ([Installation instructions](https://github.com/git-guides/install-git)), for version control.
* Node.js and `npm` ([Installation instructions](https://treehouse.github.io/installation-guides/mac/node-mac.html)), to run the static site generator [Docusaurus](https://docusaurus.io/docs) used to build this website.

### Install Node.js dependencies

```
git clone https://github.com/bacalhau-project/docs.bacalhau.org.git
cd docs.bacalhau.org/
npm install
```

### Render website

```
npm run start
```

The rendered site will be accessible at http://localhost:3000/
