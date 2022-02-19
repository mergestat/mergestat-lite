package helpers

import (
	"time"

	"github.com/mergestat/timediff"
	"go.riyazali.net/sqlite"
)

// TimeDiff implements a timediff pretty print function
// using github.com/mergestat/timediff
type TimeDiff struct{}

func (y *TimeDiff) Args() int           { return -1 }
func (y *TimeDiff) Deterministic() bool { return false }

func (y *TimeDiff) Apply(context *sqlite.Context, value ...sqlite.Value) {
	var time1, time2 time.Time
	var err error
	switch len(value) {
	case 0:
		context.ResultText("NULL")
	case 1:
		time1, err = time.Parse(time.RFC3339, value[0].Text())
		if err != nil {
			context.ResultError(err)
		}
		context.ResultText(timediff.TimeDiff(time1))
	case 2:
		time1, err = time.Parse(time.RFC3339, value[0].Text())
		if err != nil {
			context.ResultError(err)
		}
		time2, err = time.Parse(time.RFC3339, value[1].Text())
		if err != nil {
			context.ResultError(err)
		}
		context.ResultText(timediff.TimeDiff(time1, timediff.WithStartTime(time2)))
	case 3:
		time1, err = time.Parse(value[2].Text(), value[0].Text())
		if err != nil {
			context.ResultError(err)
		}
		time2, err = time.Parse(value[2].Text(), value[1].Text())
		if err != nil {
			context.ResultError(err)
		}
		context.ResultText(timediff.TimeDiff(time1, timediff.WithStartTime(time2)))
	default:
		context.ResultText("valid timediff format : timediff(time ,time2? ,inputFormat?)")
	}
}
