-- The following is SQLite SQL.
-- Table valued function commits, columns = [hash, message, author_name, author_email, author_when, committer_name, committer_email, committer_when, parents]
-- Table valued function refs, columns = [name, type, remotate, full_name, hash, target]
-- Table valued function stats, columns = [file_path, additions, deletions]
-- Table valued function files, columns = [path, executable, contents]
-- Table valued function blame, columns = [line_no, commit_hash]

-- list all commits
SELECT hash, message, author_name, author_email, author_when, committer_name, committer_email, committer_when, parents FROM commits;

-- specify an alternative repo on disk
SELECT hash, message, author_name, author_email, author_when, committer_name, committer_email, committer_when, parents FROM commits('/some/path/to/repo');

-- clone a remote repo and use it
SELECT hash, message, author_name, author_email, author_when, committer_name, committer_email, committer_when, parents FROM commits('https://github.com/mergestat/mergestat-lite');

-- use the default repo, but provide an alternate branch/ref
-- list available refs and branches with `SELECT * FROM refs('https://github.com/mergestat/mergestat-lite')`
SELECT hash, message, author_name, author_email, author_when, committer_name, committer_email, committer_when, parents FROM commits('', 'some-ref');

-- list only commits that were authored in the last 30 days
SELECT hash, message, author_name, author_email, author_when, committer_name, committer_email, committer_when, parents FROM commits WHERE author_when > datetime('now', '-30 days');

-- list the file change stats of just the HEAD commit
SELECT file_path, additions, deletions FROM stats;

-- list the file change stats of a specific commit with hash 'COMMIT_HASH'
SELECT file_path, additions, deletions FROM stats('', 'COMMIT_HASH');

-- list the file change stats for every commit in the commit history from the HEAD. We apply an implicit lateral join to get the stats for every commits.
-- this means that for every commit, we look up the stats for that commit.
SELECT commits.*, stats.* FROM commits, stats('', commits.hash);

-- list the file change stats for every commit in the current history but filter for commits that modified some/file.ext
SELECT commits.*, stats.* FROM commits, stats('', commits.hash) WHERE file_path = 'some/file.ext';

-- list the file change stats for every commit in the current history but filter for commits that modified some/file.ext in the last year
SELECT commits.*, stats.* FROM commits, stats('', commits.hash) WHERE file_path = 'some/file.ext' AND author_when > datetime('now', '-1 year');

--