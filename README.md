[![Go Reference](https://pkg.go.dev/badge/github.com/mergestat/mergestat.svg)](https://pkg.go.dev/github.com/mergestat/mergestat)
[![BuildStatus](https://github.com/mergestat/mergestat/workflows/tests/badge.svg)](https://github.com/mergestat/mergestat/actions?workflow=tests)
[![Go Report Card](https://goreportcard.com/badge/github.com/mergestat/mergestat)](https://goreportcard.com/report/github.com/mergestat/mergestat)
[![TODOs](https://badgen.net/https/api.tickgit.com/badgen/github.com/mergestat/mergestat/main)](https://www.tickgit.com/browse?repo=github.com/mergestat/mergestat&branch=main)
[![codecov](https://codecov.io/gh/mergestat/mergestat/branch/main/graph/badge.svg)](https://codecov.io/gh/mergestat/mergestat)
[![Twitter Follow](https://img.shields.io/twitter/follow/mergestat)](https://twitter.com/mergestat)


# mergestat <a href="https://app.mergestat.com/"><img align="right" src="https://github.com/mergestat/mergestat/raw/main/docs/logo.png" alt="MergeStat Logo" height="100"></a>

`mergestat` is a command-line tool for running SQL queries on git repositories and related data sources.
It's meant for ad-hoc querying of source-code on disk through a common interface (SQL), as an alternative to patching together various shell commands.
It can execute queries that look like:
```sql
-- how many commits have been authored by user@email.com?
SELECT count(*) FROM commits WHERE author_email = 'user@email.com'
```

You can try queries on public git repositories without installing anything at [app.mergestat.com](https://app.mergestat.com/), in our `Public` workspace.

More in-depth examples and documentation can be found on our dedicated [**documentation site**](https://docs.mergestat.com/).

Join our community on [Slack](https://join.slack.com/t/mergestatcommunity/shared_invite/zt-xvvtvcz9-w3JJVIdhLgEWrVrKKNXOYg) if you have questions, or just to say hi 🎉.

## Installation

See the [**full instructions**](https://docs.mergestat.com/getting-started-cli/installation) in our documentation.

### Homebrew

```bash
brew tap mergestat/mergestat
brew install mergestat
```

### Docker
```bash
docker run -v "${PWD}:/repo" mergestat/mergestat "select count(*) from commits"
```

### Examples

SQL queries can be executed in the CLI on local or remote git repositories.
Remote repos are cloned to a temporary directory at runtime.

![CLI SQL Screenshot](./docs/cli-query-example.png)

The `--format` flag can be used to output `json`, `ndjson`, `csv` and more (see `mergestat -h`).
This can be useful for piping/using with other tools.

Higher level commands such as `mergestat summarize commits` generate reports without requiring a SQL input.
Learn more [here](https://docs.mergestat.com/getting-started-cli/summarize-commits) about the available flags such as `--start` to change the date range and `--json` to output as JSON.

![CLI Summarize Commits Screenshot](./docs/cli-summarize-example.png)

[**Learn more in our docs**](https://docs.mergestat.com/)

[**Try live queries**](https://app.mergestat.com/)

