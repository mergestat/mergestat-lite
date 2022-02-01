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
type TimeDiffNow struct{}

func (y *TimeDiffNow) Args() int           { return 1 }
func (y *TimeDiffNow) Deterministic() bool { return true }

func (y *TimeDiffNow) Apply(context *sqlite.Context, value ...sqlite.Value) {
	var time1 time.Time
	var err error

	time1, err = time.Parse(time.RFC3339, value[0].Text())
	if err != nil {
		context.ResultText(err.Error())
	}
	context.ResultText(fmt.Sprintf("%s (%s)", timediff.TimeDiff(time1), time1.Format("2006-01-02")))

}
