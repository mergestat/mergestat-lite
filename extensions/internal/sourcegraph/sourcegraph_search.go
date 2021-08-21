package sourcegraph

import (
	"context"
	"encoding/json"
	"io"

	"github.com/augmentable-dev/vtab"
	"github.com/shurcooL/graphql"
	"go.riyazali.net/sqlite"
)

type searchResults struct {
	Results []struct {
		Typename                 graphql.String      `graphql:"__typename"`
		FileMatchFields          fileMatch           `graphql:"... on FileMatch"`
		CommitSearchResultFields commitSearchResults `graphql:"... on CommitSearchResult"`
		RepositoryFields         repositoryFields    `graphql:"... on Repository"`
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
	SearchResultsAlertFields searchResultAlertFields `graphql:"... on SearchResults"`
}

type fileMatch struct {
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

type commitSearchResults struct {
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

type repositoryFields struct {
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

type searchResultAlertFields struct {
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

func fetchSearch(ctx context.Context, input *fetchSourcegraphOptions) (*searchResults, error) {
	var sourcegraphQuery struct {
		Search struct {
			Results searchResults
		} `graphql:"search(query: $query, version: V2)"`
	}

	variables := map[string]interface{}{
		"query": graphql.String(input.Query),
	}

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
	results *searchResults
}

func (i *iterResults) Column(ctx *sqlite.Context, c int) error {
	current := i.results.Results[i.current]
	col := searchCols[c]
	switch col.Name {
	case "results":
		switch current.Typename {
		case "Repository":
			res, err := json.Marshal(current.RepositoryFields)
			if err != nil {
				ctx.ResultError(err)
			}
			ctx.ResultText(string(res))
		case "CommitSearchResult":
			res, err := json.Marshal(current.CommitSearchResultFields)
			if err != nil {
				ctx.ResultError(err)
			}
			ctx.ResultText(string(res))
		case "FileMatch":
			res, err := json.Marshal(current.FileMatchFields)
			if err != nil {
				ctx.ResultError(err)
			}
			ctx.ResultText(string(res))
		}
	case "__typename":
		ctx.ResultText(string(current.Typename))
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

func (i *iterResults) Next() (vtab.Row, error) {
	var err error
	if i.current == -1 {
		i.results, err = fetchSearch(context.Background(), &fetchSourcegraphOptions{i.client, i.query})
		if err != nil {
			return nil, err
		}
	}

	i.current += 1
	length := len(i.results.Results)

	if i.results == nil || i.current >= length {
		return nil, io.EOF
	}

	return i, nil
}

var searchCols = []vtab.Column{
	{Name: "query", Type: sqlite.SQLITE_TEXT, NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, Required: true, OmitCheck: true}}},
	{Name: "sourcegraph_token", Type: sqlite.SQLITE_TEXT, NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ}}},
	{Name: "cloning", Type: sqlite.SQLITE_TEXT},
	{Name: "elapsed_milliseconds", Type: sqlite.SQLITE_INTEGER},
	{Name: "match_count", Type: sqlite.SQLITE_INTEGER},
	{Name: "missing", Type: sqlite.SQLITE_INTEGER},
	{Name: "results", Type: sqlite.SQLITE_TEXT},
	{Name: "search_results_alert_fields", Type: sqlite.SQLITE_TEXT},
	{Name: "search_results_alert_title", Type: sqlite.SQLITE_TEXT},
	{Name: "search_results_alert_description", Type: sqlite.SQLITE_TEXT},
	{Name: "search_results_alert_proposed_queries_descriptions", Type: sqlite.SQLITE_TEXT},
	{Name: "search_results_alert_proposed_queries_queries", Type: sqlite.SQLITE_TEXT},
	{Name: "timed_out", Type: sqlite.SQLITE_TEXT},
	{Name: "__typename", Type: sqlite.SQLITE_TEXT},
}

func NewSourcegraphSearchModule(opts *Options) sqlite.Module {
	return vtab.NewTableFunc("github_repo_issues", searchCols, func(constraints []*vtab.Constraint, orders []*sqlite.OrderBy) (vtab.Iterator, error) {
		var query string
		for _, constraint := range constraints {
			if constraint.Op == sqlite.INDEX_CONSTRAINT_EQ {
				switch constraint.ColIndex {
				case 0:
					query = constraint.Value.Text()
				}
			}
		}

		return &iterResults{query, opts.Client(), -1, nil}, nil
	})
}
