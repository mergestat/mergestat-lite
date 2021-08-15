package sourcegraph

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/augmentable-dev/vtab"
	"github.com/shurcooL/graphql"
	"go.riyazali.net/sqlite"
)

/*
query ($query: String!) {
  search(query: $query, version: V2) {
    results {
      results {
        __typename
        ... on FileMatch {
          ...FileMatchFields
        }
        ... on CommitSearchResult {
          ...CommitSearchResultFields
        }
        ... on Repository {
          ...RepositoryFields
        }
      }
      limitHit
      cloning {
        name
      }
      missing {
        name
      }
      timedout {
        name
      }
      matchCount
      elapsedMilliseconds
      ...SearchResultsAlertFields
    }
  }
}

fragment FileMatchFields on FileMatch {
  repository {
    name
    url
  }
  file {
    name
    path
    url
    content
    commit {
      oid
    }
  }
  lineMatches {
    preview
    lineNumber
    offsetAndLengths
    limitHit
  }
}

fragment CommitSearchResultFields on CommitSearchResult {
  messagePreview {
    value
    highlights {
      line
      character
      length
    }
  }
  diffPreview {
    value
    highlights {
      line
      character
      length
    }
  }
  label {
    html
  }
  url
  matches {
    url
    body {
      html
      text
    }
    highlights {
      character
      line
      length
    }
  }
  commit {
    repository {
      name
    }
    oid
    url
    subject
    author {
      date
      person {
        displayName
      }
    }
  }
}

fragment RepositoryFields on Repository {
  name
  url
  externalURLs {
    serviceType
    url
  }
  label {
    html
  }
}

fragment SearchResultsAlertFields on SearchResults {
  alert {
    title
    description
    proposedQueries {
      description
      query
    }
  }
}
*/
type search_results struct {
	Results struct {
		Typename  graphql.String `graphql:"__typename"`
		FileMatch struct {
			FileMatchFields []file_match `graphql:"...FileMatchFields"`
		} `graphql:"... on FileMatch"`
		CommitSearchResults struct {
			Commit_search_result []commit_search_result `graphql:"...CommitSearchResultFields"`
		} `graphql:"... on CommitSearchResults"`
		Repository struct {
			Repository_fields []repository_fields `graphql:"...RepositoryFields`
		} `graphql:"... on Repository"`
	}
	LimitHit graphql.Boolean
	Cloning  struct {
		Name graphql.String
	}
	Missing struct {
		Name graphql.String
	}
	Timedout struct {
		Name graphql.String
	}
	MatchCount               graphql.Int
	ElapsedMilliseconds      graphql.Int
	SearchResultsAlertFields []search_results_alert_fields `graphql:"...SearchResultsAlertFields"`
}
type file_match struct {
	Repository struct {
		Name graphql.String
		Url  graphql.String
	}
	File struct {
		Name    graphql.String
		Path    graphql.String
		Url     graphql.String
		Content graphql.String
		Commit  struct {
			oid graphql.String
		}
	}
	LineMatches []struct {
		Preview         graphql.String
		LineNumber      graphql.Int
		OffsetAndLength graphql.String
		LimitHit        graphql.Boolean
	}
}
type preview struct {
	Value      graphql.String
	Highlights highlight
}
type highlight struct {
	Line      graphql.String
	Character graphql.String
	Length    graphql.Int
}
type commit_search_result struct {
	MessagePreview preview
	DiffPreview    preview
	Label          struct {
		Html graphql.String
	}
	Url     graphql.String
	Matches struct {
		Url  graphql.String
		Body struct {
			Html graphql.String
			Text graphql.String
		}
		Highlights highlight
	}
	Commit struct {
		Repository struct {
			name graphql.String
		}
		Oid     graphql.String
		Url     graphql.String
		Subject graphql.String
		Author  struct {
			Date   graphql.String
			Person struct {
				DisplayName graphql.String
			}
		}
	}
}
type repository_fields struct {
	Name         graphql.String
	Url          graphql.String
	ExternalURLs struct {
		ServiceType graphql.String
		Url         graphql.String
	}
	Label struct {
		Html graphql.String
	}
}
type search_results_alert_fields struct {
	Alert struct {
		Title           graphql.String
		Description     graphql.String
		ProposedQueries struct {
			Description graphql.String
			Query       graphql.String
		}
	}
}

type fetchSourcegraphOptions struct {
	Client *graphql.Client
	Query  string
}

func fetchSearch(ctx context.Context, input *fetchSourcegraphOptions) (*search_results, error) {
	var sourcegraphQuery struct {
		Query struct {
			Search struct {
				Results search_results
			} `graphql:"search(query: $query, version: V2)"`
		} `graphql:"query($query: String!)"`
	}

	variables := map[string]interface{}{
		"query": graphql.String(input.Query),
	}

	err := input.Client.Query(ctx, &sourcegraphQuery, variables)

	if err != nil {
		return nil, err
	}

	return &sourcegraphQuery.Query.Search.Results, nil
}

type iterResults struct {
	query   string
	client  *graphql.Client
	current int
	results *search_results
}

func (i *iterResults) Column(ctx *sqlite.Context, c int) error {
	col := searchCols[c]
	switch col.Type{
	case "FileMatch":
		switch col.Name{
		case "file_commit":
			ctx.ResultText(i.results.Results.FileMatch.FileMatchFields[i.current].File.Commit.oid)
		case "file_content":
			ctx.ResultText(string(i.results.Results.FileMatch.FileMatchFields[i.current].File.Content))
		case "file_name":
			ctx.ResultText(string(i.results.Results.FileMatch.FileMatchFields[i.current].File.Name))
		case "file_path":
			ctx.ResultText(string(i.results.Results.FileMatch.FileMatchFields[i.current].File.Path))
		case "file_url":
			ctx.ResultText(string(i.results.Results.FileMatch.FileMatchFields[i.current].File.Url))
		case "line_matches_preview":
			var p string
			for _,x := range(i.results.Results.FileMatch.FileMatchFields[i.current].LineMatches){
				p+="\n"+string(x.Preview)
			}
			ctx.ResultText(p)
		}

	case "CommitSearchResult":

	case "Repository":

	}
	
	switch col.Name {
	case "cloning":
		ctx.ResultText(string(i.results.Cloning.Name))
	case "missing":
		ctx.ResultText(string(i.results.Missing.Name))
	case "timed_out":
		ctx.ResultText(string(i.results.Timedout.Name))
	case "match_count":
		ctx.ResultInt(int(i.results.MatchCount))
	case "elapsed_milliseconds":
		ctx.ResultInt(int(i.results.ElapsedMilliseconds))
	case "search_results_alert_fields":
		var alerts string
		for _,result:=range(i.results.SearchResultsAlertFields){
			alerts+=string(result.Alert.Title+result.Alert.Description+"\n"+result.Alert.ProposedQueries.Description)
		}
		ctx.ResultText(alerts)
	
	return nil
}

func (i *iterResults) Next() (vtab.Row, error) {
	i.current += 1
	length := len(i.results.Results.CommitSearchResults.Commit_search_result) + len(i.results.Results.Repository.Repository_fields) + len(i.results.Results.FileMatch.FileMatchFields)
	if i.results == nil || i.current >= length {
		return nil, io.EOF
	}

	return i, nil
}

var searchCols = []vtab.Column{
	{Name: "owner", Type: sqlite.SQLITE_TEXT, NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, Required: true, OmitCheck: true}}},
	{Name: "reponame", Type: sqlite.SQLITE_TEXT, NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ}}},
	{Name: "author_login", Type: sqlite.SQLITE_TEXT},
	{Name: "body", Type: sqlite.SQLITE_TEXT},
	{Name: "closed", Type: sqlite.SQLITE_INTEGER},
	{Name: "closed_at", Type: sqlite.SQLITE_TEXT},
	{Name: "comment_count", Type: sqlite.SQLITE_INTEGER, OrderBy: vtab.ASC | vtab.DESC},
	{Name: "created_at", Type: sqlite.SQLITE_TEXT, OrderBy: vtab.ASC | vtab.DESC},
	{Name: "created_via_email", Type: sqlite.SQLITE_INTEGER},
	{Name: "database_id", Type: sqlite.SQLITE_TEXT},
	{Name: "editor_login", Type: sqlite.SQLITE_TEXT},
	{Name: "includes_created_edit", Type: sqlite.SQLITE_INTEGER},
	{Name: "label_count", Type: sqlite.SQLITE_INTEGER},
	{Name: "last_edited_at", Type: sqlite.SQLITE_TEXT},
	{Name: "locked", Type: sqlite.SQLITE_INTEGER},
	{Name: "milestone_count", Type: sqlite.SQLITE_INTEGER},
	{Name: "number", Type: sqlite.SQLITE_INTEGER},
	{Name: "participant_count", Type: sqlite.SQLITE_INTEGER},
	{Name: "published_at", Type: sqlite.SQLITE_TEXT},
	{Name: "reaction_count", Type: sqlite.SQLITE_INTEGER},
	{Name: "state", Type: sqlite.SQLITE_TEXT},
	{Name: "title", Type: sqlite.SQLITE_TEXT},
	{Name: "updated_at", Type: sqlite.SQLITE_TEXT, OrderBy: vtab.ASC | vtab.DESC},
	{Name: "url", Type: sqlite.SQLITE_TEXT},
}

func NewSourcegraphSearchModule(opts *Options) sqlite.Module {
	return vtab.NewTableFunc("github_repo_issues", issuesCols, func(constraints []*vtab.Constraint, orders []*sqlite.OrderBy) (vtab.Iterator, error) {
		var query string
		for _, constraint := range constraints {
			if constraint.Op == sqlite.INDEX_CONSTRAINT_EQ {
				switch constraint.ColIndex {
				case 0:
					query = constraint.Value.Text()
					// case 1:
					// 	auth_token = constraint.Value.Text()
				}
			}
		}

		return &iterResults{query, opts.Client(), -1, nil}, nil
	})
}
