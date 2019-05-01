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

Install [from a relase](https://github.com/openstax/versionista/releases)

or build and install via go get:

```
go get github.com/openstax/versionista
```

## Example

add a `~/.versionista.yml` file in your home directory with api token:

```
token: <git hub personal api token>
projects:
  <project name>:
    - repo-organization/repo-name

```

to release all repos for a project run:
`versionista release <project name>`

Aternatively you can release any repo even if it's not listed by using the `organization/name` format like:
`versionista release organization/name`


## Author

Nathan Stitt

## License

MIT.
=======
