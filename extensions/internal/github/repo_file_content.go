package github

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/shurcooL/githubv4"
	"go.riyazali.net/sqlite"
)

type repoFileContent struct {
	opts *Options
}

func (f *repoFileContent) Args() int           { return -1 }
func (f *repoFileContent) Deterministic() bool { return false }
func (f *repoFileContent) Apply(ctx *sqlite.Context, values ...sqlite.Value) {
	err := f.opts.RateLimiter.Wait(context.Background())
	if err != nil {
		ctx.ResultError(err)
		return
	}

	var fileContentsQuery struct {
		Repository struct {
			Object struct {
				Blob struct {
					Text string
				} `graphql:"... on Blob"`
			} `graphql:"object(expression: $expression)"`
		} `graphql:"repository(owner: $owner, name: $name)"`
	}

	var owner, name, expression string
	switch len(values) {
	case 0:
		ctx.ResultError(errors.New("need to supply a repo"))
		return
	case 1:
		ctx.ResultError(errors.New("need to supply a file path"))
		return
	case 2:
		splitStr := strings.Split(values[0].Text(), "/")
		if len(splitStr) != 2 {
			ctx.ResultError(errors.New("invalid repo name, must be of format owner/name"))
			return
		}
		owner = splitStr[0]
		name = splitStr[1]
		expression = values[1].Text()
	default:
		owner = values[0].Text()
		name = values[1].Text()
		expression = values[2].Text()
	}

	if !strings.Contains(expression, ":") {
		expression = fmt.Sprintf("HEAD:%s", expression)
	}

	variables := map[string]interface{}{
		"owner":      githubv4.String(owner),
		"name":       githubv4.String(name),
		"expression": githubv4.String(expression),
	}

	err = f.opts.RateLimiter.Wait(context.Background())
	if err != nil {
		ctx.ResultError(err)
		return
	}

	err = f.opts.Client().Query(context.Background(), &fileContentsQuery, variables)
	if err != nil {
		ctx.ResultError(err)
		return
	}
	ctx.ResultText(fileContentsQuery.Repository.Object.Blob.Text)
}

func NewRepoFileContentFunc(opts *Options) sqlite.Function {
	return &repoFileContent{opts}
}
