[![Go Reference](https://pkg.go.dev/badge/github.com/askgitdev/askgit.svg)](https://pkg.go.dev/github.com/askgitdev/askgit)
[![BuildStatus](https://github.com/askgitdev/askgit/workflows/tests/badge.svg)](https://github.com/askgitdev/askgit/actions?workflow=tests)
[![Go Report Card](https://goreportcard.com/badge/github.com/askgitdev/askgit)](https://goreportcard.com/report/github.com/askgitdev/askgit)
[![TODOs](https://badgen.net/https/api.tickgit.com/badgen/github.com/askgitdev/askgit/main)](https://www.tickgit.com/browse?repo=github.com/askgitdev/askgit&branch=main)
[![codecov](https://codecov.io/gh/askgitdev/askgit/branch/main/graph/badge.svg)](https://codecov.io/gh/askgitdev/askgit)


# mergestat <a href="https://try.askgit.com/"><img align="right" src="docs/logo.png" alt="AskGit Logo" height="100"></a>

`mergestat` is a command-line tool for running SQL queries on git repositories and related data sources.
It's meant for ad-hoc querying of source-code on disk through a common interface (SQL), as an alternative to patching together various shell commands.
It can execute queries that look like:
```sql
-- how many commits have been authored by user@email.com?
SELECT count(*) FROM commits WHERE author_email = 'user@email.com'
```

You can try queries on public git repositories without installing anything at [https://mergestat.com/](https://mergestat.com/), in our `Public` workspace.

More in-depth examples and documentation can be found on our dedicated [documentation site](https://docs.mergestat.com/).
