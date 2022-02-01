package helpers

import (
	"fmt"
	"time"

	"github.com/mergestat/timediff"
	"go.riyazali.net/sqlite"
)

// YamlToJson implements yaml_to_json sql function.
// The function signature of the equivalent sql function is:
//     yaml_to_json(string) string
type TimeDiff struct{}

func (y *TimeDiff) Args() int           { return 2 }
func (y *TimeDiff) Deterministic() bool { return true }

func (y *TimeDiff) Apply(context *sqlite.Context, value ...sqlite.Value) {
	var time1, time2 time.Time
	var err error

	time1, err = time.Parse(time.RFC3339, value[0].Text())
	if err != nil {
		context.ResultText(err.Error())
	}
	time2, err = time.Parse(time.RFC3339, value[1].Text())
	if err != nil {
		context.ResultText(err.Error())
	}
	context.ResultText(fmt.Sprintf("%s (%s)", timediff.TimeDiff(time1, timediff.WithStartTime(time2)), time1.Format("2006-01-02")))

}
