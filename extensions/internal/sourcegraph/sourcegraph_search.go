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
	Cloning  []struct {
		Name graphql.String
	}
	Missing []struct {
		Name graphql.String
	}
	Timedout []struct {
		Name graphql.String
	}
	MatchCount          graphql.Int
	ElapsedMilliseconds graphql.Int
	Alert               searchResultAlert `graphql:"... on SearchResults"`
}

type fileMatch struct {
	Repository struct {
		Name graphql.String `json:"name"`
		Url  graphql.String `json:"url"`
	} `json:"repository"`
	File struct {
		Name    graphql.String `json:"name"`
		Path    graphql.String `json:"path"`
		Url     graphql.String `json:"url"`
		Content graphql.String `json:"content"`
		Commit  struct {
			Oid graphql.String `json:"oid"`
		} `json:"commit"`
	}
	LineMatches []struct {
		Preview          graphql.String  `json:"preview"`
		LineNumber       graphql.Int     `json:"lineNumber"`
		OffsetAndLengths [][]graphql.Int `json:"offsetAndLengths"`
	} `json:"lineMatches"`
}

type preview struct {
	Value      graphql.String `json:"value"`
	Highlights highlight      `json:"highlights"`
}

type highlight struct {
	Line      graphql.String `json:"line"`
	Character graphql.String `json:"character"`
	Length    graphql.Int    `json:"length"`
}

type commitSearchResults struct {
	MessagePreview preview `json:"messagePreview"`
	DiffPreview    preview `json:"diffPreview"`
	Label          struct {
		Html graphql.String `json:"html"`
	} `json:"label"`
	Url     graphql.String `json:"url"`
	Matches []struct {
		Url  graphql.String `json:"url"`
		Body struct {
			Html graphql.String `json:"html"`
			Text graphql.String `json:"text"`
		} `json:"body"`
		Highlights []highlight `json:"highlights"`
	}
	Commit struct {
		Repository struct {
			Name graphql.String `json:"name"`
			Url  graphql.String `json:"url"`
		} `json:"repository"`
		Oid     graphql.String `json:"oid"`
		Url     graphql.String `json:"url"`
		Subject graphql.String `json:"subject"`
		Author  struct {
			Date   graphql.String `json:"date"`
			Person struct {
				DisplayName graphql.String `json:"displayName"`
			} `json:"person"`
		} `json:"author"`
	}
}

type repositoryFields struct {
	Name         graphql.String `json:"name"`
	Url          graphql.String `json:"url"`
	ExternalURLs []struct {
		ServiceKind graphql.String `json:"serviceKind"`
		Url         graphql.String `json:"url"`
	} `graphql:"externalURLs" json:"externalURLs"`
	Label struct {
		Html graphql.String `json:"html"`
	} `json:"label"`
}

type searchResultAlert struct {
	Alert struct {
		Title           graphql.String `json:"alertTitle"`
		Description     graphql.String `json:"alertDescription"`
		ProposedQueries []struct {
			Description graphql.String `json:"proposedQueryDescription"`
			Query       graphql.String `json:"proposedQuery"`
		} `json:"proposedQueries"`
	} `json:"alert"`
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
	*Options
	query   string
	current int
	results *searchResults
}

func (i *iterResults) Column(ctx *sqlite.Context, c int) error {
	var current struct {
		Typename                 graphql.String      "graphql:\"__typename\""
		FileMatchFields          fileMatch           "graphql:\"... on FileMatch\""
		CommitSearchResultFields commitSearchResults "graphql:\"... on CommitSearchResult\""
		RepositoryFields         repositoryFields    "graphql:\"... on Repository\""
	}
	if i.current < len(i.results.Results) {
		current = i.results.Results[i.current]
	}
	col := searchCols[c]
	switch col.Name {
	case "__typename":
		ctx.ResultText(string(current.Typename))
	case "results":
		switch current.Typename {
		case "Repository":
			res, err := json.Marshal(current.RepositoryFields)
			if err != nil {
				return err
			}
			ctx.ResultText(string(res))
		case "CommitSearchResult":
			res, err := json.Marshal(current.CommitSearchResultFields)
			if err != nil {
				return err
			}
			ctx.ResultText(string(res))
		case "FileMatch":
			res, err := json.Marshal(current.FileMatchFields)
			if err != nil {
				return err
			}
			ctx.ResultText(string(res))
		default:
			res, err := json.Marshal(i.results.Alert)
			if err != nil {
				ctx.ResultError(err)
				return err
			}
			ctx.ResultText(string(res))
		}
	case "cloning":
		res, err := json.Marshal(i.results.Cloning)
		if err != nil {
			return err
		}
		ctx.ResultText(string(res))
	case "missing":
		res, err := json.Marshal(i.results.Missing)
		if err != nil {
			return err
		}
		ctx.ResultText(string(res))
	case "timed_out":
		res, err := json.Marshal(i.results.Timedout)
		if err != nil {
			return err
		}
		ctx.ResultText(string(res))
	case "match_count":
		ctx.ResultInt(int(i.results.MatchCount))
	case "elapsed_milliseconds":
		ctx.ResultInt(int(i.results.ElapsedMilliseconds))
	}

	return nil
}

func (i *iterResults) Next() (vtab.Row, error) {
	var err error
	if i.current == -1 {
		i.results, err = fetchSearch(context.Background(), &fetchSourcegraphOptions{i.Client(), i.query})
		if err != nil {
			return nil, err
		}
	}

	i.current += 1
	length := len(i.results.Results)

	if i.results == nil || (i.current >= length && i.current > 0) {
		return nil, io.EOF
	}

	return i, nil
}

var searchCols = []vtab.Column{
	{Name: "query", Type: "TEXT", NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, Required: true, OmitCheck: true}}},
	{Name: "__typename", Type: "TEXT"},
	{Name: "cloning", Type: "TEXT", Hidden: true},
	{Name: "elapsed_milliseconds", Type: "INT", Hidden: true},
	{Name: "match_count", Type: "INT", Hidden: true},
	{Name: "missing", Type: "INT", Hidden: true},
	{Name: "timed_out", Type: "TEXT", Hidden: true},
	{Name: "results", Type: "TEXT"},
}

func NewSourcegraphSearchModule(opts *Options) sqlite.Module {
	return vtab.NewTableFunc("sourcegraph_search", searchCols, func(constraints []*vtab.Constraint, orders []*sqlite.OrderBy) (vtab.Iterator, error) {
		var query string
		for _, constraint := range constraints {
			if constraint.Op == sqlite.INDEX_CONSTRAINT_EQ {
				switch constraint.ColIndex {
				case 0:
					query = constraint.Value.Text()
				}
			}
		}
		opts.Logger.Sugar().Infof("running Sourcegraph search: %s", query)
		return &iterResults{opts, query, -1, nil}, nil
	})
}
