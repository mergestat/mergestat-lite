[![Go Reference](https://pkg.go.dev/badge/github.com/mergestat/mergestat.svg)](https://pkg.go.dev/github.com/mergestat/mergestat)
[![BuildStatus](https://github.com/mergestat/mergestat/workflows/tests/badge.svg)](https://github.com/mergestat/mergestat/actions?workflow=tests)
[![Go Report Card](https://goreportcard.com/badge/github.com/mergestat/mergestat)](https://goreportcard.com/report/github.com/mergestat/mergestat)
[![TODOs](https://badgen.net/https/api.tickgit.com/badgen/github.com/mergestat/mergestat/main)](https://www.tickgit.com/browse?repo=github.com/mergestat/mergestat&branch=main)
[![codecov](https://codecov.io/gh/mergestat/mergestat/branch/main/graph/badge.svg)](https://codecov.io/gh/mergestat/mergestat)


# mergestat <a href="https://try.askgit.com/"><img align="right" src="docs/logo.png" alt="MergeStat Logo" height="100"></a>

`mergestat` is a command-line tool for running SQL queries on git repositories and related data sources.
It's meant for ad-hoc querying of source-code on disk through a common interface (SQL), as an alternative to patching together various shell commands.
It can execute queries that look like:
```sql
-- how many commits have been authored by user@email.com?
SELECT count(*) FROM commits WHERE author_email = 'user@email.com'
```

You can try queries on public git repositories without installing anything at [https://mergestat.com/](https://mergestat.com/), in our `Public` workspace.

More in-depth examples and documentation can be found on our dedicated [documentation site](https://docs.mergestat.com/).
