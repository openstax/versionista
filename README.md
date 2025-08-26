# versionista : A simple CLI app to cut releases on GitHub

## [![Build Status](https://travis-ci.org/openstax/versionista.svg?branch=master)](https://travis-ci.org/openstax/versionista)

![release screenshot](screenshots/release.png?raw=true "Release Screenshot")

The basic idea is that there's a config file with a GitHub access key and multiple repositories to check.

When it's ran it:

 * Finds the the latest release on each repo
 * Checks if master differs from the last release
 * If there's additional commits, it offers to bump the version and make a release
 * It searches the commits for pull requests and makes a suggested release changelogm, opens an editor to edit if needed
 * it then makes a GitHub release

## Install

Install [from a release](https://github.com/openstax/versionista/releases)

or build manually by checking out code and running `go build` in the source directory

## Example

add a `~/.versionista.yml` file in your home directory with api token:

```
gh_token: <git hub personal api token>
projects:
  <project name>:
    - repo: repo-organization/repo-name
      alias: MyCustomName
      jira: true
      crossLink: true
    - repo: repo-organization/other-repo
      alias: OtherName
      jira: false
      crossLink: false

jira_boards:
  - board-for-project-one
  - board-for-project-two


branches:
  repo-organization/repo-name: feature-branch

```

### Configuration Options

- **alias**: Custom display name for the repository (optional)
- **jira**: Enable/disable JIRA ticket extraction from PR descriptions (default: true)
- **crossLink**: Enable/disable cross-linking to other repositories in the project within release notes (default: false)

When `crossLink` is enabled for a repository, its release notes will include a "Related Releases" section at the top with links to all repositories in the same project, showing either the newly released version or the latest existing version if no release was made.

## Commands

* **release** all repos for a project: `versionista release <project name>`
* **review**  display latest versions of all repos in project: `versionista review <project name>`

Aternatively you can release or review any repository even if it's not listed by using the `organization/name` format like:
`versionista release organization/name`


## Author

Nathan Stitt

## License

MIT.
=======
