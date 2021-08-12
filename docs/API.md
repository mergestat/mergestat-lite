## AskGit Public API

The [demo site](https://try.askgit.com/) makes use of a publicly available AskGit HTTP API which can be called directly.
Queries are executed in an AWS lambda function and have the following limitations:

1. At most 500 rows will be returned
2. Queries time out after 30 seconds
3. Query results must be under 10 MB

You can use this API by sending a `POST` request to `https://try.askgit.com/api/query`.
Make sure you set a `Content-Type: application/json` header.
The request body shoud have the following fields:

1. `repo` - the repository to query
2. `query` - the SQL query to execute

The JSON response will have the following fields:

1. `runningTime` - query runtime in ms
2. `rows` - array of objects representing query results
3. `columnNames` - array of column names in the result set
4. `columnTypes` - array of column types in the result set


```
curl -X POST -H "content-type: application/json" https://try.askgit.com/api/query -d '{"query": "select * from commits limit 1", "repo": "https://github.com/askgitdev/askgit"}'
```

Will yield something like:

```json
{
  "runningTime": 381563788,
  "rows": [
    {
      "author_email": "patrick.devivo@gmail.com",
      "author_name": "Patrick DeVivo",
      "author_when": "2021-06-27T15:14:22-04:00",
      "committer_email": "noreply@github.com",
      "committer_name": "GitHub",
      "committer_when": "2021-06-27T15:14:22-04:00",
      "hash": "4c0ba314f9551e6d316feb973b607a20b624f46a",
      "message": "Merge pull request #128 from askgitdev/change-import-paths\n\nChange import paths to reflect new org owner (`askgitdev`)",
      "parents": 2
    }
  ],
  "columnNames": [
    "hash",
    "message",
    "author_name",
    "author_email",
    "author_when",
    "committer_name",
    "committer_email",
    "committer_when",
    "parents"
  ],
  "columnTypes": [
    "TEXT",
    "TEXT",
    "TEXT",
    "TEXT",
    "DATETIME",
    "TEXT",
    "TEXT",
    "DATETIME",
    "INT"
  ]
}
```