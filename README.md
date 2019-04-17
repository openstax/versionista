# versionista : A simple CLI app to cut releases on github

## [![Build Status](https://travis-ci.org/openstax/versionista.svg?branch=master)](https://travis-ci.org/openstax/versionista)

![release screenshot](screenshots/release.png?raw=true "Release Screenshot")

Has a config file with a github access key and repos to check.
 * When it’s ran it checks the latest release on each one,
 * Checks if master differes from the release
 * If there's additional commits, it offers to bump the version and make a release
 * searches the commits for pull requests and makes a suggested release changelog, opens an editor to edit if needed
 * Makes a release

## Install

Install [from a relase](https://github.com/openstax/versionista/releases)

or build and install via go get:

```
go get github.com/openstax/versionista
```

## Example

add a `~/.versionista.yml` file in your home directory with api token:

```
token: <github personal api token>
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
