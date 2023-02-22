# How to contribute

**We are really happy you are reading this!**
Because we need volunteer developers to help this project to come to fruition.

We have highlighted the different ways you can contribute in our [contributing guide](https://docs.bacalhau.org/community/ways-to-contribute).

Here are some essential resources:

  * [Our Bacalhau DOCs](https://docs.bacalhau.org/) tell you who we are
  * [Ticket tracker](https://github.com/orgs/filecoin-project/projects/65) is our day-to-day project management space
  * Mailing list: Join our [developer list](https://bacalhau.substack.com/)
  * Forum is on [Discussions](https://github.com/filecoin-project/bacalhau/discussions) 
  * Slack: [#bacalhau](https://filecoin.io/slack) channel on Filecoin slack 
  
  [//]: <> (Our roadmap is the 10k foot view of where we're going, and)

## Testing

//TODO

## Submitting changes

Please send a [GitHub Pull Request to bacalhau](https://github.com/filecoin-project/bacalhau/tree/main) with a clear list of what you've done (read more about [pull requests](http://help.github.com/pull-requests/)). When you send a pull request, we will love you forever if you include tests. We can always use more test coverage. Please follow our coding conventions (below) and make sure all of your commits are atomic (one feature per commit).

Write a clear log message for your commits. For small changes, one-line messages are fine, but more significant changes should have descriptive messages like this:

    $ git commit -m "Brief summary of the commit
    > 
    > Paragraph describing what changed and its impact and if there are any breaking changes."

## Coding conventions

Start reading our code, and you'll get the hang of it. We optimize for readability:

  * All contributors need to use `.pre-commit-config.yaml` - it will fault tests if not. 
  * We also check during `make` to see if you have the pre-commit commands [installed](https://pre-commit.com/#usage)

Thanks! :heart: :heart: :heart:

Bacalhau Team