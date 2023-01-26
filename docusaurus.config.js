// @ts-check
// Note: type annotations allow type checking and IDEs autocompletion

<script defer data-domain="docs.bacalhau.org" src="https://plausible.io/js/script.js"></script>


const lightCodeTheme = require("prism-react-renderer/themes/github");
const darkCodeTheme = require("prism-react-renderer/themes/dracula");

const simplePlantUML = require("@akebifiky/remark-simple-plantuml");

/** @type {import('@docusaurus/types').Config} */
const config = {
  title: "Bacalhau Docs",
  tagline: "Docs for Bacalhau",
  url: "https://docs.bacalhau.org/",
  baseUrl: "/",
  onBrokenLinks: "throw",
  onBrokenMarkdownLinks: "warn",
  favicon: "img/favicon.ico",
  organizationName: "bacalhau-project", // Usually your GitHub org/user name.
  projectName: "docs.bacalhau.org", // Usually your repo name.

  presets: [
    [
      "@docusaurus/preset-classic",
      {
        docs: {
          routeBasePath: "/",
          sidebarPath: require.resolve("./sidebars.js"),
          editUrl: `https://github.com/bacalhau-project/docs.bacalhau.org/blob/main/`,
          remarkPlugins: [simplePlantUML],
          showLastUpdateAuthor: false,
          showLastUpdateTime: true,
        },
        gtag: {
          trackingID: 'G-D6NP6P151C',
        },
      },
    ],
  ],

  themeConfig:
    /** @type {import('@docusaurus/preset-classic').ThemeConfig} */
    ({
      navbar: {
        title: "Bacalhau Docs",
        logo: {
          alt: "Bacalhau Docs Logo",
          src: "img/logo.png",
        },
        items: [
          {
            type: "doc",
            docId: "intro",
            position: "left",
            label: "Documentation",
          },
          // { to: "/blog", label: "Blog", position: "left" },
          {
            href: "https://github.com/filecoin-project/bacalhau",
            label: "GitHub",
            position: "right",
          },
        ],
      },
      blog: false,
      algolia :{
        appId: 'VLYBZRSZSU',
        apiKey: '833f6b97e76869f94a4a564c4c67f7d9',
        indexName: 'dev_bac'
      },
      footer: {
        style: "dark",
        links: [
          {
            title: "Docs",
            items: [
              {
                label: "Getting Started",
                to: "/getting-started/installation",
              },
              {
                label: "About Bacalhau",
                to: "/",
              },
            ],
          },
          {
            title: "Community",
            items: [
              {
                label: "Stack Overflow",
                href: "https://stackoverflow.com/questions/tagged/bacalhau",
              },
              {
                label: "Twitter",
                href: "https://twitter.com/BacalhauProject",
              },
            ],
          },
          {
            title: "More",
            items: [
              // {
              //   label: "Blog",
              //   to: "https://docs.bacalhau.org/blog",
              // },
              {
                label: "GitHub",
                href: "https://github.com/filecoin-project/bacalhau",
              },
            ],
          },
        ],
        copyright: `Copyright © ${new Date().getFullYear()} Bacalhau, Inc. Built with Docusaurus.`,
      },
      prism: {
        theme: darkCodeTheme,
        lightTheme: lightCodeTheme,
        darkTheme: darkCodeTheme,
      },
    }),
};

module.exports = config;
