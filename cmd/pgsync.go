package cmd

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/askgitdev/askgit/extensions"
	"github.com/askgitdev/askgit/extensions/options"
	"github.com/askgitdev/askgit/pkg/locator"
	"github.com/askgitdev/askgit/pkg/pgsync"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"go.riyazali.net/sqlite"

	_ "github.com/askgitdev/askgit/pkg/sqlite"
	_ "github.com/mattn/go-sqlite3"
)

var pgsyncCmd = &cobra.Command{
	Use:  "pgsync [tableName] [query]",
	Long: `Use this command to sync the results of an askgit query into a Postgres table`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		l := zerolog.New(os.Stderr).
			Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.Stamp}).
			Level(zerolog.DebugLevel).
			With().
			Timestamp().Logger()

		// TODO(patrickdevivo) maybe there should be a "RegisterDefault" method that handles this boilerplate.
		// Basically, register all the default functionality.
		sqlite.Register(
			extensions.RegisterFn(
				options.WithExtraFunctions(),
				options.WithRepoLocator(locator.CachedLocator(locator.MultiLocator())),
				options.WithGitHub(),
				options.WithContextValue("githubToken", os.Getenv("GITHUB_TOKEN")),
				options.WithContextValue("githubPerPage", os.Getenv("GITHUB_PER_PAGE")),
				options.WithContextValue("githubRateLimit", os.Getenv("GITHUB_RATE_LIMIT")),
				options.WithLogger(&l),
			),
		)

		var schemaName string
		tableName := args[0]
		query := args[1]

		if strings.Contains(tableName, ".") {
			s := strings.SplitN(tableName, ".", 2)
			schemaName = s[0]
			tableName = s[1]
		}

		var postgres *sql.DB
		var askgit *sql.DB
		var err error

		if postgres, err = sql.Open("postgres", os.Getenv("POSTGRES_CONNECTION")); err != nil {
			l.Error().Msgf("could not open postgres connection: %v", err)
			return
		}
		defer func() {
			if err := postgres.Close(); err != nil {
				l.Error().Msgf("could not close postgres connection: %v", err)
			}
		}()

		if askgit, err = sql.Open("sqlite3", ":memory:"); err != nil {
			l.Error().Msgf("could not initialize askgit: %v", err)
			return
		}
		defer func() {
			if err := askgit.Close(); err != nil {
				l.Error().Msgf("could not close askgit: %v", err)
			}
		}()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// TODO(patrickdevivo) hacky way of adding a SQL statement to run ahead of sync...
		// added mainly so we can run `SET statement_timeout 0` for connections to bit.io
		if preamble := os.Getenv("PGSYNC_PREAMBLE"); preamble != "" {
			if _, err := postgres.ExecContext(ctx, preamble); err != nil {
				l.Error().Msgf("could not execute preamble: %v", err)
			}
		}

		options := &pgsync.SyncOptions{
			Postgres:   postgres,
			AskGit:     askgit,
			SchemaName: schemaName,
			TableName:  tableName,
			Query:      query,
			Logger:     &logger,
		}

		err = pgsync.Sync(ctx, options)
		if err != nil {
			if !errors.Is(err, sql.ErrTxDone) {
				logger.Error().AnErr("could not sync", err).Msg("error")
			}
		}
	},
}
