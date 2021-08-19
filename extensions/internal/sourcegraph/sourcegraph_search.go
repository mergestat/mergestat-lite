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
	Results []struct {
		Typename graphql.String `graphql:"__typename"`

		FileMatchFields file_match `graphql:"... on FileMatch"`

		CommitSearchResultFields commit_search_result `graphql:"... on CommitSearchResult"`

		RepositoryFields repository_fields `graphql:"... on Repository"`
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
	SearchResultsAlertFields search_results_alert_fields `graphql:"... on SearchResults"`
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
		Preview          graphql.String
		LineNumber       graphql.Int
		OffsetAndLengths [][]graphql.Int
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
		Highlights []highlight
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
	ExternalURLs []struct {
		ServiceKind graphql.String
		Url         graphql.String
	} `graphql:"externalURLs"`
	Label struct {
		Html graphql.String
	}
}
type search_results_alert_fields struct {
	Alert struct {
		Title           graphql.String
		Description     graphql.String
		ProposedQueries []struct {
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
		Search struct {
			Results search_results
		} `graphql:"search(query: $query, version: V2)"`
	}

	variables := map[string]interface{}{
		"query": graphql.String(input.Query),
	}
	println(input.Query)
	println(fmt.Sprint(variables["query"]))

	err := input.Client.Query(ctx, &sourcegraphQuery, variables)

	if err != nil {
		return nil, err
	}

	return &sourcegraphQuery.Search.Results, nil
}

type iterResults struct {
	query   string
	client  *graphql.Client
	current int
	results *search_results
}

func (i *iterResults) Column(ctx *sqlite.Context, c int) error {

	col := searchCols[c]
	switch i.results.Results[i.current].Typename {
	case "FileMatch":
		switch col.Name {
		case "file_commit":
			ctx.ResultText(string(i.results.Results[i.current].FileMatchFields.File.Commit.Oid))
		case "file_content":
			ctx.ResultText(string(i.results.Results[i.current].FileMatchFields.File.Content))
		case "file_name":
			ctx.ResultText(string(i.results.Results[i.current].FileMatchFields.File.Name))
		case "file_path":
			ctx.ResultText(string(i.results.Results[i.current].FileMatchFields.File.Path))
		case "file_url":
			ctx.ResultText(string(i.results.Results[i.current].FileMatchFields.File.Url))
		case "linematches_preview":
			var p string
			for _, x := range i.results.Results[i.current].FileMatchFields.LineMatches {
				p += "\n" + string(x.Preview)
			}
			ctx.ResultText(p)
		case "linematches_line_no":
			var p string
			for _, x := range i.results.Results[i.current].FileMatchFields.LineMatches {
				p += "\n" + string(x.LineNumber)
			}
			ctx.ResultText(p)
		case "linematches_offset_and_length":
			var p string
			for _, x := range i.results.Results[i.current].FileMatchFields.LineMatches {
				p += "\n" + string(fmt.Sprint(x.OffsetAndLengths))
			}
			ctx.ResultText(p)
		}

	case "CommitSearchResult":
		switch col.Name {
		case "CSR_message_preview_value":
			ctx.ResultText(string(i.results.Results[i.current].CommitSearchResultFields.MessagePreview.Value))
		case "CSR_message_preview_highlight":
			ctx.ResultText(string(i.results.Results[i.current].CommitSearchResultFields.MessagePreview.Highlights.Line))
		case "CSR_diff_preview_value":
			ctx.ResultText(string(i.results.Results[i.current].CommitSearchResultFields.DiffPreview.Value))
		case "CSR_diff_preview_highlight":
			ctx.ResultText(string(i.results.Results[i.current].CommitSearchResultFields.DiffPreview.Highlights.Line))
		case "CSR_label":
			ctx.ResultText(string(i.results.Results[i.current].CommitSearchResultFields.Label.Html))
		case "CSR_url":
			ctx.ResultText(string(i.results.Results[i.current].CommitSearchResultFields.Url))
		case "CSR_matches_url":
			ctx.ResultText(string(i.results.Results[i.current].CommitSearchResultFields.Matches.Url))
		case "CSR_matches_body_html":
			ctx.ResultText(string(i.results.Results[i.current].CommitSearchResultFields.Matches.Body.Html))
		case "CSR_matches_body_text":
			ctx.ResultText(string(i.results.Results[i.current].CommitSearchResultFields.Matches.Body.Text))
		case "CSR_matches_highlights_line":
			var line string
			for _, x := range i.results.Results[i.current].CommitSearchResultFields.Matches.Highlights {
				line += string(x.Line) + "\n"
			}
			ctx.ResultText(line)
		case "CSR_matches_highlights_character":
			var character string
			for _, x := range i.results.Results[i.current].CommitSearchResultFields.Matches.Highlights {
				character += string(x.Character) + "\n"
			}
			ctx.ResultText(character)
		case "CSR_matches_highlights_length":
			var length string
			for _, x := range i.results.Results[i.current].CommitSearchResultFields.Matches.Highlights {
				length += string(x.Length) + "\n"
			}
			ctx.ResultText(length)
		case "CSR_commit_repository_name":
			ctx.ResultText(string(i.results.Results[i.current].CommitSearchResultFields.Commit.Repository.name))
		case "CSR_commit_hash":
			ctx.ResultText(string(i.results.Results[i.current].CommitSearchResultFields.Commit.Oid))
		case "CSR_commit_url":
			ctx.ResultText(string(i.results.Results[i.current].CommitSearchResultFields.Commit.Url))
		case "CSR_commit_subject":
			ctx.ResultText(string(i.results.Results[i.current].CommitSearchResultFields.Commit.Subject))
		case "CSR_commit_author_date":
			ctx.ResultText(string(i.results.Results[i.current].CommitSearchResultFields.Commit.Author.Date))
		case "CSR_commit_author_displayname":
			ctx.ResultText(string(i.results.Results[i.current].CommitSearchResultFields.Commit.Author.Person.DisplayName))
		}
	case "Repository":
		switch col.Name {
		case "repository_name":
			ctx.ResultText(string(i.results.Results[i.current].RepositoryFields.Name))
		case "repository_url":
			ctx.ResultText(string(i.results.Results[i.current].RepositoryFields.Url))
		case "repository_externalurl_servicetype":
			var serviceType string
			for _, x := range i.results.Results[i.current].RepositoryFields.ExternalURLs {
				serviceType += string(x.ServiceKind) + "\n"
			}
			ctx.ResultText(serviceType)
		case "repository_externalurl_url":
			var url string
			for _, x := range i.results.Results[i.current].RepositoryFields.ExternalURLs {
				url += string(x.Url) + "\n"
			}
			ctx.ResultText(url)
		case "repository_label_html":
			ctx.ResultText(string(i.results.Results[i.current].RepositoryFields.Label.Html))
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
	case "search_results_alert_fields_title":
		ctx.ResultText(string(i.results.SearchResultsAlertFields.Alert.Title))
	case "search_results_alert_fields_description":
		ctx.ResultText(string(i.results.SearchResultsAlertFields.Alert.Description))
	case "search_results_alert_fields_proposedQueries_descriptions":
		var descriptions string
		for _, x := range i.results.SearchResultsAlertFields.Alert.ProposedQueries {
			descriptions += string(x.Description)
		}
		ctx.ResultText(descriptions)
	case "search_results_alert_fields_proposedQueries_queries":
		var queries string
		for _, x := range i.results.SearchResultsAlertFields.Alert.ProposedQueries {
			queries += string(x.Query)
		}
		ctx.ResultText(queries)
	}

	return nil
}
func graphqlStrArrToString(strArr []graphql.String) string {
	var ret string
	for _, x := range strArr {
		ret += string(x) + "\n"
	}
	return ret
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
	//length := len(i.results.Results)

	if i.results == nil || i.current >= 10 /*length*/ {
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
	{Name: "search_results_alert_title", Type: sqlite.SQLITE_TEXT},
	{Name: "search_results_alert_description", Type: sqlite.SQLITE_TEXT},
	{Name: "search_results_alert_proposedQueries_descriptions", Type: sqlite.SQLITE_TEXT},
	{Name: "search_results_alert_proposedQueries_queries", Type: sqlite.SQLITE_TEXT},
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
					// TODO: allow auth token to be passed in as second parameter
					// case 1:
					// 	auth_token = constraint.Value.Text()
				}
			}
		}

		return &iterResults{query, opts.Client(), -1, nil}, nil
	})
}
