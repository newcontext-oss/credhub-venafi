# Contributing

This document describes how to contribute to this project.
Proposals for changes to this document are welcome.

## Table of Contents

[Code of Conduct](#code-of-conduct)

[Asking Questions](#asking-questions)

[Project Technologies](#project-technologies)

## Code of Conduct

Contributors to this project are expected to adhere to the
[Code of Conduct](CODE_OF_CONDUCT.md). Any unacceptable conduct
should be reported to opensource@newcontext.com.

## Questions

Questions about the project may be posed on the
[GitHub issue tracker][github-issue-tracker].

## Project Technologies

Familiarity with the following technologies is important in
understanding the design and behaviour of this project.

- [Go][go]
- [VenaVenafi Trust Protection Platform][venafi]
- [Credhub][credhub]

## Reporting Bugs

Bugs must be reported on the
[GitHub issue tracker](github-issue-tracker). Any information that will assist in the maintainers reproducing the bug should be included.

## Suggesting Changes

Changes should be suggested on the
[GitHub issue tracker](github-issue-tracker). Submitting a pull request with an implementation of the changes is also encouraged but not required.

## Developing

The development workflow for this project follows
[standard GitHub workflow](fork-a-repo).

### Unit Testing

[Golang testing package][gotest] is used as the unit testing framework.

The following command will execute the unit tests.

> Executing unit tests with Go's testing package

```sh
go test -v
```

The json files under [testdata](testdata) contain supporting json files for testing.

<!-- Markdown links and image definitions -->
[credhub]: https://docs.cloudfoundry.org/credhub/
[fork-a-repo]: https://help.github.com/articles/fork-a-repo/
[github-issue-tracker]: https://github.com/newcontext-oss/credhub-venafi/issues
[go]: https://golang.org/
[gotest]: https://golang.org/pkg/testing
[testdata]: https://github.com/newcontext-oss/credhub-venafi/tree/master/testdata
[venafi]: https://venafi.com
