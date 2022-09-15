# Bacalhau Docs

This repository contains the user documentation of the Bacalhau Project.
These are accessible at https://docs.bacalhau.org/.

All code examples live in a dedicated repository https://github.com/bacalhau-project/examples and they are automagically rendered into the [./docs/examples/](https://github.com/bacalhau-project/docs.bacalhau.org/tree/main/docs/examples) folder.

* [Bacalhau Website](https://www.bacalhau.org/)
* [Bacalhau Core Code Repository](https://github.com/filecoin-project/bacalhau)
* [Twitter](https://twitter.com/BacalhauProject)
* [YouTube Channel](https://www.youtube.com/channel/UC45IQagLzNR3wdNCUn4vi0A)

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
