package npm

import (
	"context"
	"fmt"

	"go.riyazali.net/sqlite"
)

type GetPackage struct{ *Client }

func (f *GetPackage) Args() int           { return -1 }
func (f *GetPackage) Deterministic() bool { return false }
func (f *GetPackage) Apply(ctx *sqlite.Context, values ...sqlite.Value) {
	switch {
	case len(values) == 0:
		ctx.ResultError(fmt.Errorf("expected a package name"))
		return
	case len(values) == 1:
		if res, err := f.GetPackage(context.Background(), values[0].Text()); err != nil {
			ctx.ResultError(err)
			return
		} else {
			ctx.ResultText(string(res))
		}
	default:
		if res, err := f.GetPackageVersion(context.Background(), values[0].Text(), values[1].Text()); err != nil {
			ctx.ResultError(err)
			return
		} else {
			ctx.ResultText(string(res))
		}
	}
}
