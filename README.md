[![GoDev](https://img.shields.io/static/v1?label=godev&message=reference&color=00add8)](https://pkg.go.dev/github.com/askgitdev/askgit)
[![BuildStatus](https://github.com/askgitdev/askgit/workflows/tests/badge.svg)](https://github.com/askgitdev/askgit/actions?workflow=tests)
[![Go Report Card](https://goreportcard.com/badge/github.com/askgitdev/askgit)](https://goreportcard.com/report/github.com/askgitdev/askgit)
[![TODOs](https://badgen.net/https/api.tickgit.com/badgen/github.com/askgitdev/askgit/main)](https://www.tickgit.com/browse?repo=github.com/askgitdev/askgit&branch=main)
[![codecov](https://codecov.io/gh/askgitdev/askgit/branch/main/graph/badge.svg)](https://codecov.io/gh/askgitdev/askgit)

# askgit

`askgit` is a command-line tool for running SQL queries on git repositories.
It's meant for ad-hoc querying of git repositories on disk through a common interface (SQL), as an alternative to patching together various shell commands.
It can execute queries that look like:
```sql
-- how many commits have been authored by user@email.com?
SELECT count(*) FROM commits WHERE author_email = 'user@email.com'
```

You can try queries on public git repositories without installing anything at [https://try.askgit.com/](https://try.askgit.com/)

More in-depth examples and documentation can be found below.

## Installation

### Homebrew

```
brew tap askgitdev/askgit
brew install askgit
```

### Go

[`libgit2`](https://libgit2.org/) is a build dependency (used via [`git2go`](https://github.com/libgit2/git2go)) and must be available on your system for linking.

The following (long 😬) `go install` commands can be used to install a binary via the go toolchain.

On Mac:
```
CGO_CFLAGS=-DUSE_LIBSQLITE3 CGO_LDFLAGS=-Wl,-undefined,dynamic_lookup go install -tags="sqlite_vtable,vtable,sqlite_json1,static,system_libgit2" github.com/askgitdev/askgit
```

On Linux:
```
CGO_CFLAGS=-DUSE_LIBSQLITE3 CGO_LDFLAGS=-Wl,--unresolved-symbols=ignore-in-object-files go install -tags="sqlite_vtable,vtable,sqlite_json1,static,system_libgit2" github.com/askgitdev/askgit
```

See the [`Makefile`](https://github.com/askgitdev/askgit/blob/main/Makefile) for more context.
Checking out this repository and running `make` in the root will produce two files in the `.build` directory:

  1. `askgit` - the CLI binary (which can then be moved into your `$PATH` for use)
  2. `libaskgit.so` - a shared object file [SQLite extension](https://www.sqlite.org/loadext.html) that can be used by SQLite directly

### Using Docker

Build an image locally using docker

```
docker build -t askgit:latest .
```

Or use an official image from [docker hub](https://hub.docker.com/repository/docker/augmentable/askgit)

```
docker pull augmentable/askgit:latest
```

#### Running commands

`askgit` operates on a git repository. This repository needs to be attached as a volume. This example uses the (bash) built-in command `pwd` for the current working directory

> [**pwd**] Print the absolute pathname of the current working directory.

```
docker run --rm -v `pwd`:/repo:ro augmentable/askgit "SELECT * FROM commits"
```

#### Running commands from STDIN

For piping commands via STDIN, the docker command needs to be told to run non-interactively, as well as attaching the repository at `/repo`.

```
cat query.sql | docker run --rm -i -v `pwd`:/repo:ro augmentable/askgit
```

## Usage

```
askgit -h
```

Will output the most up to date usage instructions for your version of the CLI.
Typically the first argument is a SQL query string:

```
askgit "SELECT * FROM commits"
```

Your current working directory will be used as the path to the git repository to query by default.
Use the `--repo` flag to specify an alternate path, or even a remote repository reference (http(s) or ssh).
`askgit` will clone the remote repository to a temporary directory before executing a query.

You can also pass a query in via `stdin`:

```
cat query.sql | askgit
```

By default, output will be an ASCII table.
Use `--format json` or `--format csv` for alternatives.
See `-h` for all the options.

### Tables and Functions

#### Local Git Repository

The following tables access a git repository in the current directory by default.
If the `--repo` flag is specified, they will use the path provided there instead.
A parameter (usually the first) can also be provided to any of the tables below to override the default repo path.
For instance, `SELECT * FROM commits('https://github.com/askgitdev/askgit')` will clone this repo to a temporary directory on disk and return its commits.

##### `commits`

Similar to `git log`, the `commits` table includes all commits in the history of the currently checked out commit.

| Column          | Type     |
|-----------------|----------|
| hash            | TEXT     |
| message         | TEXT     |
| author_name     | TEXT     |
| author_email    | TEXT     |
| author_when     | DATETIME |
| committer_name  | TEXT     |
| committer_email | TEXT     |
| committer_when  | DATETIME |
| parents         | INT      |

Params:
  1. `repository` - path to a local (on disk) or remote (http(s)) repository
  2. `ref` - return commits starting at this ref (i.e. branch name or SHA), defaults to `HEAD`

```sql
-- return all commits starting at HEAD
SELECT * FROM commits

-- specify an alternative repo on disk
SELECT * FROM commits('/some/path/to/repo')

-- clone a remote repo and use it
SELECT * FROM commits('https://github.com/askgitdev/askgit')

-- use the default repo, but provide an alternate branch
SELECT * FROM commits('', 'some-ref')
```

##### `refs`

| Column    | Type |
|-----------|------|
| name      | TEXT |
| type      | TEXT |
| remote    | TEXT |
| full_name | TEXT |
| hash      | TEXT |
| target    | TEXT |

Params:
  1. `repository` - path to a local (on disk) or remote (http(s)) repository

##### `stats`

| Column    | Type |
|-----------|------|
| file_path | TEXT |
| additions | INT  |
| deletions | INT  |

Params:
  1. `repository` - path to a local (on disk) or remote (http(s)) repository
  2. `ref` - commit hash to use for retrieving stats, defaults to `HEAD`
  3. `to_ref` - commit hash to calculate stats relative to

```sql
-- return stats of HEAD
SELECT * FROM stats

-- return stats of a specific commit
SELECT * FROM stats('', 'COMMIT_HASH')

-- return stats for every commit in the current history
SELECT commits.hash, stats.* FROM commits, stats('', commits.hash)
```

##### `files`

| Column     | Type |
|------------|------|
| path       | TEXT |
| executable | BOOL |
| contents   | TEXT |

Params:
  1. `repository` - path to a local (on disk) or remote (http(s)) repository
  2. `ref` - commit hash to use for retrieving files, defaults to `HEAD`

##### `blame`

Similar to `git blame`, the `blame` table includes blame information for all files in the current HEAD.

| Column      | Type     |
|-------------|----------|
| line_no     | INT      |
| commit_hash | TEXT     |

Params:
  1. `repository` - path to a local (on disk) or remote (http(s)) repository
  2. `ref` - commit hash to use for retrieving files, defaults to `HEAD`
  3. `file_path` - path of file to blame

#### Utilities

##### JSON

The [SQLite JSON1 extension](https://www.sqlite.org/json1.html) is included for working with JSON data.

##### `toml_json`

Scalar function that converts `toml` to `json`.

```SQL
SELECT toml_to_json('[some-toml]')

-- +-----------------------------+
-- | TOML_TO_JSON('[SOME-TOML]') |
-- +-----------------------------+
-- | {"some-toml":{}}            |
-- +-----------------------------+
```

##### `xml_to_json`

Scalar function that converts `xml` to `json`.

```SQL
SELECT xml_to_json('<some-xml>hello</some-xml>')

-- +-------------------------------------------+
-- | XML_TO_JSON('<SOME-XML>HELLO</SOME-XML>') |
-- +-------------------------------------------+
-- | {"some-xml":"hello"}                      |
-- +-------------------------------------------+
```

##### `yaml_to_json` and `yml_to_json`

Scalar function that converts `yaml` to `json`.

```SQL
SELECT yaml_to_json('hello: world')

-- +------------------------------+
-- | YAML_TO_JSON('HELLO: WORLD') |
-- +------------------------------+
-- | {"hello":"world"}            |
-- +------------------------------+
```

##### `str_split`

Helper for splitting strings on some separator.

```sql
SELECT str_split('hello,world', ',', 0)

-- +----------------------------------+
-- | STR_SPLIT('HELLO,WORLD', ',', 0) |
-- +----------------------------------+
-- | hello                            |
-- +----------------------------------+
```

```sql
SELECT str_split('hello,world', ',', 1)

-- +----------------------------------+
-- | STR_SPLIT('HELLO,WORLD', ',', 1) |
-- +----------------------------------+
-- | world                            |
-- +----------------------------------+
```


### Example Queries

This will return all commits in the history of the currently checked out branch/commit of the repo.
```sql
SELECT * FROM commits
```

Return the (de-duplicated) email addresses of commit authors:
```sql
SELECT DISTINCT author_email FROM commits
```

Return the commit counts of every author (by email):
```sql
SELECT author_email, count(*) FROM commits GROUP BY author_email ORDER BY count(*) DESC
```

Same as above, but excluding merge commits:
```sql
SELECT author_email, count(*) FROM commits WHERE parents < 2 GROUP BY author_email ORDER BY count(*) DESC
```

Outputs the set of files in the current tree:
```sql
SELECT * FROM files
```


Returns author emails with lines added/removed, ordered by total number of commits in the history (excluding merges):
```sql
SELECT count(DISTINCT commits.hash) AS commits, SUM(additions) AS additions, SUM(deletions) AS deletions, author_email
FROM commits LEFT JOIN stats('', commits.hash)
WHERE commits.parents < 2
GROUP BY author_email ORDER BY commits
```


Returns commit counts by author, broken out by day of the week:

```sql
SELECT
    count(*) AS commits,
    count(CASE WHEN strftime('%w',author_when)='0' THEN 1 END) AS sunday,
    count(CASE WHEN strftime('%w',author_when)='1' THEN 1 END) AS monday,
    count(CASE WHEN strftime('%w',author_when)='2' THEN 1 END) AS tuesday,
    count(CASE WHEN strftime('%w',author_when)='3' THEN 1 END) AS wednesday,
    count(CASE WHEN strftime('%w',author_when)='4' THEN 1 END) AS thursday,
    count(CASE WHEN strftime('%w',author_when)='5' THEN 1 END) AS friday,
    count(CASE WHEN strftime('%w',author_when)='6' THEN 1 END) AS saturday,
    author_email
FROM commits GROUP BY author_email ORDER BY commits
```

#### Exporting

You can use the `askgit export` sub command to save the output of queries into a sqlite database file.
The command expects a path to a db file (which will be created if it doesn't already exist) and a variable number of "export pairs," specified by the `-e` flag.
Each pair represents the name of a table to create and a query to generate its contents.

```
askgit export my-export-file -e commits -e "SELECT * FROM commits" -e files -e "SELECT * FROM files"
```

This can be useful if you're looking to use another tool to examine the data emitted by `askgit`.
Since the exported file is a plain SQLite database, queries should be much faster (as the original git repository is no longer traversed) and you should be able to use any tool that supports querying SQLite database files.
