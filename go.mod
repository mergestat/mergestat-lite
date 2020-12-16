module github.com/augmentable-dev/askgit

go 1.13

require (
	github.com/DATA-DOG/go-sqlmock v1.5.0
	github.com/gitsight/go-vcsurl v1.0.0
	github.com/google/go-github v17.0.0+incompatible
	github.com/google/go-querystring v1.0.0 // indirect
	github.com/jroimartin/gocui v0.4.0
	github.com/kr/text v0.2.0 // indirect
	github.com/libgit2/git2go/v31 v31.4.7
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/mattn/go-sqlite3 v1.14.4
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/nsf/termbox-go v0.0.0-20201107200903-9b52a5faed9e // indirect
	github.com/olekukonko/tablewriter v0.0.4
	github.com/spf13/cobra v1.1.1
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45
	golang.org/x/sync v0.0.0-20190423024810-112230192c58
	golang.org/x/time v0.0.0-20190308202827-9d24e82272b4
	gopkg.in/check.v1 v1.0.0-20200227125254-8fa46927fb4f // indirect
	gopkg.in/yaml.v2 v2.3.0 // indirect
)

replace github.com/mattn/go-sqlite3 v1.14.4 => github.com/patrickdevivo/go-sqlite3 v1.14.6-0.20201211024840-146d4a910383
