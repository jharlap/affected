# Affected

Affected queries a git repository for a commit range and determines which Go packages are affected, either because their source was modified or because they depend (directly or transitively) on an affected package. This is useful when working in a monorepo to speed up CI tests/builds and for reducing the number of components redeployed by a CD pipeline. A package is considered modified if any file within that package directory was modified. The downside of this is that changing a text file in a library imported by many packages can cause excessive false positives,but was chosen as a strategy because that text file could be part of bindata or similar, thus there's no way to safely know which files are not really part of the program source.

## Usage

Call `affected` with a git commit range in the form `old..new`, for example any of the following:
```
$ affected 2227ca9..HEAD
$ affected master..2227ca9
```

The output is a list of packages which are affected by the commits, ignoring stdlib, suitable for providing to `go test`.

## Requirements

`affected` requires `git` to be available in the path, and will only work on git-managed repos.
