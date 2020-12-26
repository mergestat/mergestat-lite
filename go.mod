module github.com/augmentable-dev/askgit

go 1.13

require (
	github.com/DATA-DOG/go-sqlmock v1.5.0
	github.com/gitsight/go-vcsurl v1.0.0
	github.com/go-openapi/strfmt v0.19.11 // indirect
	github.com/google/go-github v17.0.0+incompatible
	github.com/google/go-querystring v1.0.0 // indirect
	github.com/jedib0t/go-pretty v4.3.0+incompatible
	github.com/jroimartin/gocui v0.4.0
	github.com/libgit2/git2go/v31 v31.3.4
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/mattn/go-sqlite3 v1.14.4
	github.com/nsf/termbox-go v0.0.0-20201107200903-9b52a5faed9e // indirect
	github.com/spf13/cobra v1.1.1
	golang.org/x/crypto v0.0.0-20200820211705-5c72a883971a
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e
	golang.org/x/time v0.0.0-20190308202827-9d24e82272b4
)

replace github.com/mattn/go-sqlite3 v1.14.4 => github.com/patrickdevivo/go-sqlite3 v1.14.6-0.20201219185255-a526406471dd
