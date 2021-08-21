[![Go Reference](https://pkg.go.dev/badge/github.com/askgitdev/askgit.svg)](https://pkg.go.dev/github.com/askgitdev/askgit)
[![BuildStatus](https://github.com/askgitdev/askgit/workflows/tests/badge.svg)](https://github.com/askgitdev/askgit/actions?workflow=tests)
[![Go Report Card](https://goreportcard.com/badge/github.com/askgitdev/askgit)](https://goreportcard.com/report/github.com/askgitdev/askgit)
[![TODOs](https://badgen.net/https/api.tickgit.com/badgen/github.com/askgitdev/askgit/main)](https://www.tickgit.com/browse?repo=github.com/askgitdev/askgit&branch=main)
[![codecov](https://codecov.io/gh/askgitdev/askgit/branch/main/graph/badge.svg)](https://codecov.io/gh/askgitdev/askgit)


# askgit <a href="https://try.askgit.com/"><img align="right" src="docs/logo.png" alt="AskGit Logo" height="100"></a>

`askgit` is a command-line tool for running SQL queries on git repositories.
It's meant for ad-hoc querying of git repositories on disk through a common interface (SQL), as an alternative to patching together various shell commands.
It can execute queries that look like:
```sql
-- how many commits have been authored by user@email.com?
SELECT count(*) FROM commits WHERE author_email = 'user@email.com'
```

You can try queries on public git repositories without installing anything at [https://try.askgit.com/](https://try.askgit.com/)

More in-depth examples and documentation can be found below.
Also checkout [our newsletter](https://askgit.substack.com) to stay up to date with feature releases and interesting queries and use cases.

## Installation

### Homebrew

```
brew tap askgitdev/askgit
brew install askgit
```

### Pre-Built Binaries

The [latest releases](https://github.com/askgitdev/askgit/releases) should have pre-built binaries for Mac and Linux.
You can download and add the `askgit` binary somewhere on your `$PATH` to use.
`libaskgit.so` is also available to be loaded as a SQLite run-time extension.

### Go

[`libgit2`](https://libgit2.org/) is a build dependency (used via [`git2go`](https://github.com/libgit2/git2go)) and must be available on your system for linking.

The following (long 😬) `go install` commands can be used to install a binary via the go toolchain.

On Mac:
```
CGO_CFLAGS=-DUSE_LIBSQLITE3 CGO_LDFLAGS=-Wl,-undefined,dynamic_lookup go install -tags="sqlite_vtable,vtable,sqlite_json1,static,system_libgit2" github.com/askgitdev/askgit@latest
```

On Linux:
```
CGO_CFLAGS=-DUSE_LIBSQLITE3 CGO_LDFLAGS=-Wl,--unresolved-symbols=ignore-in-object-files go install -tags="sqlite_vtable,vtable,sqlite_json1,static,system_libgit2" github.com/askgitdev/askgit@latest
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

## Public API

We maintain a free to use, public API for running queries (executed in an AWS Lambda function).
See [this page](docs/API.md) for more information.

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
  2. `rev` - return commits starting at this revision (i.e. branch name or SHA), defaults to `HEAD`

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
  2. `rev` - commit hash (or branch/tag name) to use for retrieving stats, defaults to `HEAD`
  3. `to_rev` - commit hash to calculate stats relative to

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
  2. `rev` - commit hash (or branch/tag name) to use for retrieving files in, defaults to `HEAD`

##### `blame`

Similar to `git blame`, the `blame` table includes blame information for all files in the current HEAD.

| Column      | Type     |
|-------------|----------|
| line_no     | INT      |
| commit_hash | TEXT     |

Params:
  1. `repository` - path to a local (on disk) or remote (http(s)) repository
  2. `rev` - commit hash (or branch/tag name) to use for retrieving blame information from, defaults to `HEAD`
  3. `file_path` - path of file to blame

#### Utilities

##### JSON

The [SQLite JSON1 extension](https://www.sqlite.org/json1.html) is included for working with JSON data.

##### `toml_to_json`

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

##### `go_mod_to_json`

Scalar function that parses a `go.mod` file and returns a JSON representation of it.

```SQL
SELECT go_mod_to_json('<contents-of-go.mod-file>')
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

#### Enry Functions

Functions from the [`enry` project](https://github.com/go-enry/go-enry) are also available as SQL scalar functions

##### `enry_detect_language`

Supply a file path and some source code to detect the language.

```sql
SELECT enry_detect_language('some/path/to/file.go', '<contents of file>')
```

##### `enry_is_binary`

Given a blob, determine if it's a binary file or not (returns 1 or 0).

```sql
SELECT enry_is_binary('<contents of file>')
```

##### `enry_is_configuration`

Detect whether a file path is to a configuration file (returns 1 or 0).

```sql
SELECT enry_is_configuration('some/path/to/file/config.json')
```

##### `enry_is_documentation`

Detect whether a file path is to a documentation file (returns 1 or 0).

```sql
SELECT enry_is_documentation('some/path/to/file/README.md')
```

##### `enry_is_dot_file`

Detect whether a file path is to a dot file (returns 1 or 0).

```sql
SELECT enry_is_dot_file('some/path/to/file/.gitignore')
```

##### `enry_is_generated`

Detect whether a file path is generated (returns 1 or 0).

```sql
SELECT enry_is_generated('some/path/to/file/generated.go', '<contents of file>')
```

##### `enry_is_image`

Detect whether a file path is to an image (returns 1 or 0).

```sql
SELECT enry_is_image('some/path/to/file/image.png')
```

##### `enry_is_test`

Detect whether a file path is to a test file (returns 1 or 0).

```sql
SELECT enry_is_test('some/path/to/file/image.png')
```

##### `enry_is_vendor`

Detect whether a file path is to a vendored file (returns 1 or 0).

```sql
SELECT enry_is_vendor('vendor/file.go')
```



#### GitHub API

You can use `askgit` to query the [GitHub API (v4)](https://docs.github.com/en/graphql).
Constraints in your SQL query are pushed to the GitHub API as much as possible.
For instance, if your query includes an `ORDER BY` clause and if items can be ordered in the GitHub API response (on the specified column), your query can avoid doing a full table scan and rely on the ordering returned by the API.

##### Authenticating

You must provide an authentication token in order to use the GitHub API tables.
You can create a personal access token [following these instructions](https://docs.github.com/en/github/authenticating-to-github/keeping-your-account-and-data-secure/creating-a-personal-access-token).
`askgit` will look for a `GITHUB_TOKEN` environment variable when executing, to use for authentication.
This is also true if running as a runtime loadable extension.

##### Rate Limiting

All API requests to GitHub are [rate limited](https://docs.github.com/en/graphql/overview/resource-limitations#rate-limit).
The following tables make use of the GitHub GraphQL API (v4), which rate limits additionally based on the "complexity" of GraphQL queries.
Generally speaking, the more fields/relations in your GraphQL query, the higher the "cost" of a single API request, and the faster you may reach a rate limit.
Depending on your SQL query, it's hard to know ahead of time what a good client-side rate limit is.
By default, each of the tables below will fetch **100 items per page** and permit **2 API requests per second**.
You can override both of these parameters by setting the following environment variables:

1. `GITHUB_PER_PAGE` - expects an integer between 1 and 100, sets how many items are fetched per-page in API calls that paginate results.
2. `GITHUB_RATE_LIMIT` - expressed in the form `(number of requests) / (number of seconds)` (i.e. `1/3` means at most 1 request per 3 seconds)

If you encounter a rate limit error that looks like `You have exceeded a secondary rate limit`, consider setting the `GITHUB_PER_PAGE` value to a lower number.
If you have a large number of items to scan in your query, it may take longer, but you should avoid hitting a rate limit error.

##### `github_stargazers`

Table-valued-function that returns a list of users who have starred a repository.

| Column     | Type     |
|------------|----------|
| login      | TEXT     |
| email      | TEXT     |
| name       | TEXT     |
| bio        | TEXT     |
| company    | TEXT     |
| avatar_url | TEXT     |
| created_at | DATETIME |
| updated_at | DATETIME |
| twitter    | TEXT     |
| website    | TEXT     |
| location   | TEXT     |
| starred_at | DATETIME |

Params:
  1. `fullNameOrOwner` - either the full repo name `askgitdev/askgit` or just the owner `askgit` (which would require the second argument)
  2. `name` - optional if the first argument is a "full" name, otherwise required - the name of the repo

```sql
SELECT * FROM github_stargazers('askgitdev', 'askgit');
SELECT * FROM github_stargazers('askgitdev/askgit'); -- both are equivalent
```

##### `github_starred_repos`

Table-valued-function that returns a list of repositories a user has starred.

| Column          | Type     |
|-----------------|----------|
| name            | TEXT     |
| url             | TEXT     |
| description     | TEXT     |
| created_at      | DATETIME |
| pushed_at       | DATETIME |
| updated_at      | DATETIME |
| stargazer_count | INT      |
| name_with_owner | TEXT     |
| starred_at      | DATETIME |

Params:
  1. `login` - the `login` of a GitHub user

```sql
SELECT * FROM github_starred_repos('patrickdevivo')
```

##### `github_stargazer_count`

Scalar function that returns the number of stars a GitHub repository has.

Params:
  1. `fullNameOrOwner` - either the full repo name `askgitdev/askgit` or just the owner `askgit` (which would require the second argument)
  2. `name` - optional if the first argument is a "full" name, otherwise required - the name of the repo

```sql
SELECT github_stargazer_count('askgitdev', 'askgit');
SELECT github_stargazer_count('askgitdev/askgit'); -- both are equivalent
```

##### `github_user_repos` and `github_org_repos`

Table-valued function that returns all the repositories belonging to a user or an organization.

| Column                      | Type     |
|-----------------------------|----------|
| created_at                  | DATETIME |
| database_id                 | INT      |
| default_branch_ref_name     | TEXT     |
| default_branch_ref_prefix   | TEXT     |
| description                 | TEXT     |
| disk_usage                  | INT      |
| fork_count                  | INT      |
| homepage_url                | TEXT     |
| is_archived                 | BOOLEAN  |
| is_disabled                 | BOOLEAN  |
| is_fork                     | BOOLEAN  |
| is_mirror                   | BOOLEAN  |
| is_private                  | BOOLEAN  |
| issue_count                 | INT      |
| latest_release_author       | TEXT     |
| latest_release_created_at   | DATETIME |
| latest_release_name         | TEXT     |
| latest_release_published_at | DATETIME |
| license_key                 | TEXT     |
| license_name                | TEXT     |
| name                        | TEXT     |
| open_graph_image_url        | TEXT     |
| primary_language            | TEXT     |
| pull_request_count          | INT      |
| pushed_at                   | DATETIME |
| release_count               | INT      |
| stargazer_count             | INT      |
| updated_at                  | DATETIME |
| watcher_count               | INT      |

Params:
  1. `login` - the `login` of a GitHub user or organization

```sql
SELECT * FROM github_user_repos('patrickdevivo')
SELECT * FROM github_org_repos('askgitdev')
```

##### `github_repo_issues`

Table-valued-function that returns all the issues of a GitHub repository.

| Column                | Type      |
|-----------------------|-----------|
| owner                 | TEXT      |
| reponame              | TEXT      |
| author_login          | TEXT      |
| body                  | TEXT      |
| closed                | BOOLEAN   |
| closed_at             | DATETIME  |
| comment_count         | INT       |
| created_at            | DATETIME  |
| created_via_email     | BOOLEAN   |
| database_id           | TEXT      |
| editor_login          | TEXT      |
| includes_created_edit | BOOLEAN   |
| label_count           | INT       |
| last_edited_at        | DATETIME  |
| locked                | BOOLEAN   |
| milestone_count       | INT       |
| number                | INT       |
| participant_count     | INT       |
| published_at          | DATETIME  |
| reaction_count        | INT       |
| state                 | TEXT      |
| title                 | TEXT      |
| updated_at            | DATETIME  |
| url                   | TEXT      |

Params:
  1. `fullNameOrOwner` - either the full repo name `askgitdev/askgit` or just the owner `askgit` (which would require the second argument)
  2. `name` - optional if the first argument is a "full" name, otherwise required - the name of the repo

```sql
SELECT * FROM github_repo_issues('askgitdev/askgit');
SELECT * FROM github_repo_issues('askgitdev', 'askgit'); -- both are equivalent
```
##### `github_repo_prs`

Table-valued-function that returns all the pull requests of a GitHub repository.

| Column                   | Type     |
|--------------------------|----------|
| additions                | INT      |
| author_login             | TEXT     |
| author_association       | TEXT     |
| base_ref_oid             | TEXT     |
| base_ref_name            | TEXT     |
| base_repository_name     | TEXT     |
| body                     | TEXT     |
| changed_files            | INT      |
| closed                   | BOOLEAN  |
| closed_at                | DATETIME |
| comment_count            | INT      |
| commit_count             | INT      |
| created_at               | TEXT     |
| created_via_email        | BOOLEAN  |
| database_id              | INT      |
| deletions                | INT      |
| editor_login             | TEXT     |
| head_ref_name            | TEXT     |
| head_ref_oid             | TEXT     |
| head_repository_name     | TEXT     |
| is_draft                 | INT      |
| label_count              | INT      |
| last_edited_at           | DATETIME |
| locked                   | BOOLEAN  |
| maintainer_can_modify    | BOOLEAN  |
| mergeable                | TEXT     |
| merged                   | BOOLEAN  |
| merged_at                | DATETIME |
| merged_by                | TEXT     |
| number                   | INT      |
| participant_count        | INT      |
| published_at             | DATETIME |
| review_decision          | TEXT     |
| state                    | TEXT     |
| title                    | TEXT     |
| updated_at               | DATETIME |
| url                      | TEXT     |

Params:
  1. `fullNameOrOwner` - either the full repo name `askgitdev/askgit` or just the owner `askgit` (which would require the second argument)
  2. `name` - optional if the first argument is a "full" name, otherwise required - the name of the repo

```sql
SELECT * FROM github_repo_prs('askgitdev/askgit');
SELECT * FROM github_repo_prs('askgitdev', 'askgit'); -- both are equivalent
```

##### `github_repo_file_content`

Scalar function that returns the contents of a file in a GitHub repository

Params:
  1. `fullNameOrOwner` - either the full repo name `askgitdev/askgit` or just the owner `askgit` (which would require the second argument)
  2. `name` - optional if the first argument is a "full" name, otherwise required - the name of the repo
  3. `expression` - either a simple file path (`README.md`) or a rev-parse suitable expression that includes a path (`HEAD:README.md` or `<some-sha>:README.md`)

```sql
SELECT github_stargazer_count('askgitdev', 'askgit', 'README.md');
SELECT github_stargazer_count('askgitdev/askgit', 'README.md'); -- both are equivalent
```

#### Sourcegraph API (`experimental`!)

You can use `askgit` to query the [Sourcegraph API](https://sourcegraph.com/api/console).

##### Authenticating

You must provide an authentication token in order to use the Sourcegraph API tables.
You can create a personal access token [following these instructions](https://docs.sourcegraph.com/cli/how-tos/creating_an_access_token).
`askgit` will look for a `SOURCEGRAPH_TOKEN` environment variable when executing, to use for authentication.
This is also true if running as a runtime loadable extension.

##### `sourcegraph_search`

Table-valued-function that returns results from a Sourcegraph search.

| Column               | Type |
|----------------------|------|
| __typename           | TEXT |
| results              | TEXT |

`__typename` will be one of `Repository`, `CommitSearchResult`, or `FileMatch`.
`results` will be the JSON value of a search result (will match what's returned from the API)

Params:
  1. `query` - a sourcegraph search query ([docs](https://docs.sourcegraph.com/))

```sql
SELECT sourcegraph_search('askgit');
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
