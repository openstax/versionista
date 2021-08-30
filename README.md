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
token: <git hub personal api token>
projects:
  <project name>:
    - repo-organization/repo-name

branches:
  repo-organization/repo-name: feature-branch

aliases:
  <project name>:
    repo-organization/repo-name: MyCustomName

```

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
