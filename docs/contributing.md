
## Contributing Guide

We welcome all contributions to Bacalhau  documentation ❤️

- [Contributing Guide](#contributing-guide)
- [Documentation Structure](#documentation-structure)
- [Issues](#issues)
- [Pull Requests](#pull-requests)
- [Editing Docs Locally](#editing-docs-locally)
- [Creating a new page](#creating-a-new-page)
- [Waiting for a review](#waiting-for-a-review)
- [Merging your fix](#merging-your-fix)
- [Finishing up](#finishing-up)


## Documentation Structure

`docs/` : This is where all the .md files live that control the content of this site. Most contributions happen in this directory.

**Note**: All code examples live in a dedicated repository [Bacalhau-project/examples](https://github.com/bacalhau-project/examples) and they are automagically rendered into the `./docs/examples/` folder by github actions.

## Issues

All issues involving the Bacalhau docs themselves can be found here in the Bacalhau repo under the [**Issues** tab](https://github.com/bacalhau-project/bacalhau/issues). Here you can see all the issues that are currently open.

If you spot anything that conflicts in our docs or have feedbacks, suggestions, hints etc, you can create an issue.

**Note**: For a step by step process on how to create an issue, please refer to this [Github guide](https://docs.github.com/en/issues/tracking-your-work-with-issues/creating-an-issue)

## Pull Requests

If you want to make documentation changes, you can submit a pull request.

For minor changes like typos, sentence fixes, click **Edit this page**, located at the bottom of each document. This will take you to the source file on GitHub, where you can submit a pull request for your changes.

For larger edits or new documents, you can choose to edit [docs locally](#editing-docs-locally).


## Editing Docs Locally

If you want to submit a pull request to update the docs, you will need to [make a fork](https://docs.github.com/en/get-started/quickstart/fork-a-repo) of this repo and clone it.


1. Create a branch and switch to it:

    `git checkout -b <branch-name>`

2. Add or modify the Markdown files in these directories according to our style guide.

3. When you are happy with your changes, commit them with a message summarizing what you did:

    `git commit -am "commit message"`

4. Push your branch up:

    `git push origin <branch-name>`

**Note**: Our documentation is written in markdown format.

## Creating a new page

New pages added should be placed under the docs directory. Each new page should contain **front matter** which provides additional metadata for your doc page. For example
```
---
sidebar_position: 3
sidebar_label: 'Social Media'
description: 'Find Bacalhau on your favorite social media platform'
---
```

## Waiting for a review

All pull requests from the community must be reviewed by at least one project member before they are merged in. Depending on the size of the pull request, this could take anywhere from a few minutes to a few days to review everything. Depending on the complexity of the pull request, there may be further discussion regarding your changes. Keep returning to GitHub and checking your notifications page to make sure you don't miss anything.

## Merging your fix

Once your pull request has been approved, a project member with the correct rights will merge it. You'll be notified as soon as the merge is complete.

## Finishing up

So there you have it!  We're always on the lookout for contributors to help us improve the Bacalhau docs and make the internet better for everyone, so keep up the good work!
