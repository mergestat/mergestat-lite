package sourcegraph

import (
	"context"
	"fmt"
	"io"

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
			Repository_fields []repository_fields `graphql:"...RepositoryFields"`
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
			Oid graphql.String
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
	switch i.results.Results.Typename {
	case "FileMatch":
		switch col.Name {
		case "file_commit":
			ctx.ResultText(string(i.results.Results.FileMatch.FileMatchFields[i.current].File.Commit.Oid))
		case "file_content":
			ctx.ResultText(string(i.results.Results.FileMatch.FileMatchFields[i.current].File.Content))
		case "file_name":
			ctx.ResultText(string(i.results.Results.FileMatch.FileMatchFields[i.current].File.Name))
		case "file_path":
			ctx.ResultText(string(i.results.Results.FileMatch.FileMatchFields[i.current].File.Path))
		case "file_url":
			ctx.ResultText(string(i.results.Results.FileMatch.FileMatchFields[i.current].File.Url))
		case "linematches_preview":
			var p string
			for _, x := range i.results.Results.FileMatch.FileMatchFields[i.current].LineMatches {
				p += "\n" + string(x.Preview)
			}
			ctx.ResultText(p)
		case "linematches_line_no":
			var p string
			for _, x := range i.results.Results.FileMatch.FileMatchFields[i.current].LineMatches {
				p += "\n" + string(x.LineNumber)
			}
			ctx.ResultText(p)
		case "linematches_offset_and_length":
			var p string
			for _, x := range i.results.Results.FileMatch.FileMatchFields[i.current].LineMatches {
				p += "\n" + string(x.OffsetAndLength)
			}
			ctx.ResultText(p)
		case "linematches_limit_hit":
			var p string
			for _, x := range i.results.Results.FileMatch.FileMatchFields[i.current].LineMatches {
				p += "\n" + fmt.Sprint(x.LimitHit)
			}
			ctx.ResultText(p)
		}

	case "CommitSearchResult":
		switch col.Name {
		case "CSR_message_preview_value":
			ctx.ResultText(string(i.results.Results.CommitSearchResults.Commit_search_result[i.current].MessagePreview.Value))
		case "CSR_message_preview_highlight":
			ctx.ResultText(string(i.results.Results.CommitSearchResults.Commit_search_result[i.current].MessagePreview.Highlights.Line))
		case "CSR_diff_preview_value":
			ctx.ResultText(string(i.results.Results.CommitSearchResults.Commit_search_result[i.current].DiffPreview.Value))
		case "CSR_diff_preview_highlight":
			ctx.ResultText(string(i.results.Results.CommitSearchResults.Commit_search_result[i.current].DiffPreview.Highlights.Line))
		case "CSR_label":
			ctx.ResultText(string(i.results.Results.CommitSearchResults.Commit_search_result[i.current].Label.Html))
		case "CSR_url":
			ctx.ResultText(string(i.results.Results.CommitSearchResults.Commit_search_result[i.current].Url))
		case "CSR_matches_url":
			ctx.ResultText(string(i.results.Results.CommitSearchResults.Commit_search_result[i.current].Matches.Url))
		case "CSR_matches_body_html":
			ctx.ResultText(string(i.results.Results.CommitSearchResults.Commit_search_result[i.current].Matches.Body.Html))
		case "CSR_matches_body_text":
			ctx.ResultText(string(i.results.Results.CommitSearchResults.Commit_search_result[i.current].Matches.Body.Text))
		case "CSR_matches_highlights_line":
			ctx.ResultText(string(i.results.Results.CommitSearchResults.Commit_search_result[i.current].Matches.Highlights.Line))
		case "CSR_matches_highlights_character":
			ctx.ResultText(string(i.results.Results.CommitSearchResults.Commit_search_result[i.current].Matches.Highlights.Character))
		case "CSR_matches_highlights_length":
			ctx.ResultInt(int(i.results.Results.CommitSearchResults.Commit_search_result[i.current].Matches.Highlights.Length))
		case "CSR_commit_repository_name":
			ctx.ResultText(string(i.results.Results.CommitSearchResults.Commit_search_result[i.current].Commit.Repository.name))
		case "CSR_commit_hash":
			ctx.ResultText(string(i.results.Results.CommitSearchResults.Commit_search_result[i.current].Commit.Oid))
		case "CSR_commit_url":
			ctx.ResultText(string(i.results.Results.CommitSearchResults.Commit_search_result[i.current].Commit.Url))
		case "CSR_commit_subject":
			ctx.ResultText(string(i.results.Results.CommitSearchResults.Commit_search_result[i.current].Commit.Subject))
		case "CSR_commit_author_date":
			ctx.ResultText(string(i.results.Results.CommitSearchResults.Commit_search_result[i.current].Commit.Author.Date))
		case "CSR_commit_author_displayname":
			ctx.ResultText(string(i.results.Results.CommitSearchResults.Commit_search_result[i.current].Commit.Author.Person.DisplayName))
		}
	case "Repository":
		switch col.Name {
		case "repository_name":
			ctx.ResultText(string(i.results.Results.Repository.Repository_fields[i.current].Name))
		case "repository_url":
			ctx.ResultText(string(i.results.Results.Repository.Repository_fields[i.current].Url))
		case "repository_externalurl_servicetype":
			ctx.ResultText(string(i.results.Results.Repository.Repository_fields[i.current].ExternalURLs.ServiceType))
		case "repository_externalurl_url":
			ctx.ResultText(string(i.results.Results.Repository.Repository_fields[i.current].ExternalURLs.Url))
		case "repository_label_html":
			ctx.ResultText(string(i.results.Results.Repository.Repository_fields[i.current].Label.Html))
		}
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
		for _, result := range i.results.SearchResultsAlertFields {
			alerts += string(result.Alert.Title + result.Alert.Description + "\n" + result.Alert.ProposedQueries.Description)
		}
		ctx.ResultText(alerts)
	}
	return nil
}

func (i *iterResults) Next() (vtab.Row, error) {
	var err error
	println(i.current)
	if i.current == -1 {
		i.results, err = fetchSearch(context.Background(), &fetchSourcegraphOptions{i.client, i.query})
		if err != nil {
			return nil, err
		}
	}
	//results, err := fetchPR(context.Background(), &fetchPROptions{i.client, owner, name, i.perPage, cursor, i.prOrder})

	i.current += 1
	var length int
	switch i.results.Results.Typename {
	case "FileMatch":
		length = len(i.results.Results.FileMatch.FileMatchFields)
	case "CommitSearchResults":
		length = len(i.results.Results.CommitSearchResults.Commit_search_result)
	case "RepositoryFields":
		length = len(i.results.Results.Repository.Repository_fields)
	}

	if i.results == nil || i.current >= length {
		return nil, io.EOF
	}

	return i, nil
}

var searchCols = []vtab.Column{
	{Name: "query", Type: sqlite.SQLITE_TEXT, NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, Required: true, OmitCheck: true}}},
	{Name: "sourcegraph_token", Type: sqlite.SQLITE_TEXT, NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ}}},
	{Name: "CSR_commit_author_date", Type: sqlite.SQLITE_TEXT},
	{Name: "CSR_commit_author_displayname", Type: sqlite.SQLITE_TEXT},
	{Name: "CSR_commit_hash", Type: sqlite.SQLITE_TEXT},
	{Name: "CSR_commit_repository_name", Type: sqlite.SQLITE_TEXT},
	{Name: "CSR_commit_subject", Type: sqlite.SQLITE_INTEGER},
	{Name: "CSR_commit_url", Type: sqlite.SQLITE_TEXT},
	{Name: "CSR_diff_preview_highlight", Type: sqlite.SQLITE_INTEGER, OrderBy: vtab.ASC | vtab.DESC},
	{Name: "CSR_diff_preview_value", Type: sqlite.SQLITE_TEXT, OrderBy: vtab.ASC | vtab.DESC},
	{Name: "CSR_label", Type: sqlite.SQLITE_INTEGER},
	{Name: "CSR_matches_body_html", Type: sqlite.SQLITE_TEXT},
	{Name: "CSR_matches_body_text", Type: sqlite.SQLITE_TEXT},
	{Name: "CSR_matches_highlights_character", Type: sqlite.SQLITE_INTEGER},
	{Name: "CSR_matches_highlights_length", Type: sqlite.SQLITE_INTEGER},
	{Name: "CSR_matches_highlights_line", Type: sqlite.SQLITE_TEXT},
	{Name: "CSR_matches_url", Type: sqlite.SQLITE_INTEGER},
	{Name: "CSR_message_preview_highlight", Type: sqlite.SQLITE_INTEGER},
	{Name: "CSR_message_preview_value", Type: sqlite.SQLITE_INTEGER},
	{Name: "CSR_url", Type: sqlite.SQLITE_INTEGER},
	{Name: "cloning", Type: sqlite.SQLITE_TEXT},
	{Name: "elapsed_milliseconds", Type: sqlite.SQLITE_INTEGER},
	{Name: "file_commit", Type: sqlite.SQLITE_TEXT},
	{Name: "file_content", Type: sqlite.SQLITE_TEXT},
	{Name: "file_name", Type: sqlite.SQLITE_TEXT, OrderBy: vtab.ASC | vtab.DESC},
	{Name: "file_path", Type: sqlite.SQLITE_TEXT},
	{Name: "file_url", Type: sqlite.SQLITE_TEXT},
	{Name: "linematches_limit_hit", Type: sqlite.SQLITE_TEXT},
	{Name: "linematches_line_no", Type: sqlite.SQLITE_INTEGER},
	{Name: "linematches_offset_and_length", Type: sqlite.SQLITE_INTEGER},
	{Name: "linematches_preview", Type: sqlite.SQLITE_TEXT},
	{Name: "match_count", Type: sqlite.SQLITE_INTEGER},
	{Name: "missing", Type: sqlite.SQLITE_INTEGER},
	{Name: "repository_externalurl_servicetype", Type: sqlite.SQLITE_INTEGER},
	{Name: "repository_externalurl_url", Type: sqlite.SQLITE_INTEGER},
	{Name: "repository_label_html", Type: sqlite.SQLITE_TEXT},
	{Name: "repository_name", Type: sqlite.SQLITE_INTEGER},
	{Name: "repository_url", Type: sqlite.SQLITE_TEXT},
	{Name: "search_results_alert_fields", Type: sqlite.SQLITE_TEXT},
	{Name: "timed_out", Type: sqlite.SQLITE_TEXT, OrderBy: vtab.ASC | vtab.DESC},
}

func NewSourcegraphSearchModule(opts *Options) sqlite.Module {
	return vtab.NewTableFunc("github_repo_issues", searchCols, func(constraints []*vtab.Constraint, orders []*sqlite.OrderBy) (vtab.Iterator, error) {
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
