# Bacalhau Docs

This repository manages the documentation for the <a href="https://www.bacalhau.org/">Bacalhau Project</a>. This repo also contains the build scripts and tools to create and contribute to the Bacalhau docs website. <a href="https://docs.bacalhau.org/">Explore the docs →</a></p>

## Table of contents
- [Develop docs locally](#develop-docs-locally)
    - [Prerequisites](#prerequisites)
    - [Installation](#installation)
    - [Render website](#render-website)
- [Documentatation structure](#documentatation-structure)
- [Contributing](#contributing)
  - [Issues and Pull requests](#issues-and-pull-requests)
- [Be Part of the community ](#be-part-of-the-community )
- [Resources](#resources)


## Develop docs locally
Follow these simple example steps to get a local version of the site up and running.

### Prerequisites
- Git ([Installation instructions](https://github.com/git-guides/install-git)), for version control.
- Node.js and `npm` ([Installation instructions](https://treehouse.github.io/installation-guides/mac/node-mac.html)), to run the static site generator [Docusaurus](https://docusaurus.io/docs) used to build this website.

### Installation

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

## Documentatation structure
`docs/` : This is where all the .md files live that control the content of this site. Most contributions happen in this directory.

**Note**: All [code examples](https://docs.bacalhau.org/examples/) live in a dedicated repository https://github.com/bacalhau-project/examples and they are automagically rendered into the [./docs/examples/](https://github.com/bacalhau-project/docs.bacalhau.org/tree/main/docs/examples) folder by github actions.

## Contributing
We would **love ❤️ your help** to improve existing items or make new ones even better!

### Issues and Pull requests
If you find any inconsistencies in the docs, please raise an issue [here →](https://github.com/bacalhau-project/docs.bacalhau.org/issues). Feel free to also submit a pull request with any changes you'd like to see to this repo. Every contribution is more than welcome! 🎈

## Be Part of the community
If you have any questions you can join our [Slack Community](https://join.slack.com/t/bacalhauproject/shared_invite/zt-1sihp4vxf-TjkbXz6JRQpg2AhetPzYYQ) go to **#bacalhau** channel - its open to anyone!


## Resources
- [Bacalhau Website](https://www.bacalhau.org/)
- [Bacalhau Core Code Repository](https://github.com/bacalhau-project/bacalhau)
- [Bacalhau Twitter](https://twitter.com/BacalhauProject)
- [Bacalhau YouTube channel](https://www.youtube.com/channel/UC45IQagLzNR3wdNCUn4vi0A)
- [Bacalhau Newsletter and blog](https://bacalhau.substack.com/)
