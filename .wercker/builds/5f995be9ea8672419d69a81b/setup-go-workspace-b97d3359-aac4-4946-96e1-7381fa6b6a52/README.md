# setup-go-workspace

Move your code from the working directory into the `$GOPATH`. Introspection is
done via environment variables, to determine the git provider host, owner, and
repository name. You are able to overwrite this path using a option.

# Options

- `package-dir` (optional) The path inside `$GOPATH/src` to use. If left empty,
the step will use the environment variables to figure out where to put the code.
- `gopath` (optional, default: `$GOPATH`) This should contain a path to the
workspace where the code should be installed. Use this when `$GOPATH` contains
multiple directories.

# Example

Setup a Go workspace using environment variables to figure out the path:

```yaml
build:
    steps:
        - wercker/setup-go-workspace
```

Setup a Go workspace using a fixed path:

```yaml
build:
    steps:
        - wercker/setup-go-workspace:
            package-dir: github.com/wercker/setup-go-workspace
```

# Known issues

## Doesn't work in Single-player mode

Currently the wercker cli does not inject the git details as environment
variables. You can either set the `package-dir` option to a fixed value. Or you
can pass the git details to the wercker cli using the following flags:
`--git-domain`, `--git-owner`, and `--git-repository` (see
`wercker build --help`).

## Changes made during build are not present during a deploy

Currently this step will copy all content from `$WERCKER_SOURCE_DIR` to a
directory inside the `$GOPATH`. It will also change `$WERCKER_SOURCE_DIR` to
point to this new directory. This will ensure that any steps run after this
step will be run in the `$GOPATH`. The store step however will still look at
the original `$WERCKER_SOURCE_DIR` path. So any changes will not be persisted.

The recommended workaround is to put your artifacts in the `$WERCKER_OUTPUT_DIR`
directory. This will ensure that only the files you pick are available during a
deploy.

# License

The MIT License (MIT)

# Changelog

## 1.2.1

- Fix bug where the original folder was deleted.

## 1.2.0

- Remove rsync dependency

## 1.0.4

- Initial release
