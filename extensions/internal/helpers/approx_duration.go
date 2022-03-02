package helpers

import (
	"fmt"
	"math"
	"time"

	"github.com/mergestat/timediff"
	"github.com/mergestat/timediff/locale"
	"go.riyazali.net/sqlite"
)

// ApproxDuration pretty prints a duration given in days showing
// x years (y months)
// x years
// x months (y days)
// x months
// x days
type ApproxDuration struct{}

func (y *ApproxDuration) Args() int           { return 1 }
func (y *ApproxDuration) Deterministic() bool { return true }

func (y *ApproxDuration) Apply(context *sqlite.Context, value ...sqlite.Value) {
	d := time.Duration(value[0].Float() * 24 * float64(time.Hour.Nanoseconds()))

	f := timediff.WithCustomFormatters(locale.Formatters{
		time.Second:           func(_ time.Duration) string { return "<none>" },
		44 * time.Second:      func(_ time.Duration) string { return "a few seconds" },
		89 * time.Second:      func(_ time.Duration) string { return "1 minute" },
		44 * time.Minute:      func(d time.Duration) string { return fmt.Sprintf("%.0f minutes", math.Ceil(d.Minutes())) },
		89 * time.Minute:      func(_ time.Duration) string { return "1 hour" },
		21 * time.Hour:        func(d time.Duration) string { return fmt.Sprintf("%.0f hours", math.Ceil(d.Hours())) },
		35 * time.Hour:        func(_ time.Duration) string { return "1 day" },
		25 * (24 * time.Hour): func(d time.Duration) string { return fmt.Sprintf("%.0f days", math.Ceil(d.Hours()/24.0)) },
		45 * (24 * time.Hour): func(_ time.Duration) string { return "1 month" },
		10 * (24 * time.Hour) * 30: func(d time.Duration) string {
			return fmt.Sprintf("%.0f months", math.Ceil(d.Hours()/(24.0*30)))
		},
		17 * (24 * time.Hour) * 30: func(d time.Duration) string {
			return fmt.Sprintf("1 year (%.0f months)", math.Round(d.Hours()/(24.0*30)))
		},
		1<<63 - 1: func(d time.Duration) string {
			return fmt.Sprintf("%.0f years (%.0f months)", math.Ceil(d.Hours()/(24.0*30*12)), math.Round(d.Hours()/(24.0*30)))
		},
	})

	context.ResultText(timediff.TimeDiff(time.Now().Add(-d), f))
}
