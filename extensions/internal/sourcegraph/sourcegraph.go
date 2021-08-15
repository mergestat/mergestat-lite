package sourcegraph

import (
	"context"

	"github.com/askgitdev/askgit/extensions/options"
	"github.com/pkg/errors"
	"github.com/shurcooL/graphql"
	"go.riyazali.net/sqlite"
	"golang.org/x/oauth2"
)

var sourcegraphUrl string = "https://sourcegraph.com/.api/graphql"

// Register registers GitHub related functionality as a SQLite extension
func Register(ext *sqlite.ExtensionApi, opt *options.Options) (_ sqlite.ErrorCode, err error) {
	sourcegraphOpts := &Options{
		Client: func() *graphql.Client {
			httpClient := oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(
				&oauth2.Token{AccessToken: GetSourcegraphTokenFromCtx(opt.Context)},
			))
			client := graphql.NewClient(sourcegraphUrl, httpClient)
			return client
		},
	}

	if opt.SourcegraphClientGetter != nil {
		sourcegraphOpts.Client = opt.SourcegraphClientGetter
	}

	var modules = map[string]sqlite.Module{
		"sourcegraph_search": NewSourcegraphSearchModule(sourcegraphOpts),
	}

	// register GitHub tables
	for name, mod := range modules {
		if err = ext.CreateModule(name, mod); err != nil {
			return sqlite.SQLITE_ERROR, errors.Wrapf(err, "failed to register Sourcegraph %q module", name)
		}
	}

	// var fns = map[string]sqlite.Function{
	// }

	// register Sourcegraph funcs
	// for name, fn := range fns {
	// 	if err = ext.CreateFunction(name, fn); err != nil {
	// 		return sqlite.SQLITE_ERROR, errors.Wrapf(err, "failed to register Sourcegraph %q function", name)
	// 	}
	// }
	return sqlite.SQLITE_OK, nil
}
