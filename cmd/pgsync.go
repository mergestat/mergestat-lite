package cmd

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"strings"

	"github.com/mergestat/mergestat/pkg/pgsync"
	"github.com/spf13/cobra"

	_ "github.com/mattn/go-sqlite3"
	_ "github.com/mergestat/mergestat/pkg/sqlite"
)

var pgsyncCmd = &cobra.Command{
	Use:  "pgsync [tableName] [query]",
	Long: `Use this command to sync the results of a mergestat query into a Postgres table`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		registerExt()

		var schemaName string
		tableName := args[0]
		query := args[1]

		if strings.Contains(tableName, ".") {
			s := strings.SplitN(tableName, ".", 2)
			schemaName = s[0]
			tableName = s[1]
		}

		var postgres *sql.DB
		var mergestat *sql.DB
		var err error

		if postgres, err = sql.Open("postgres", os.Getenv("POSTGRES_CONNECTION")); err != nil {
			logger.Error().Msgf("could not open postgres connection: %v", err)
			return
		}
		defer func() {
			if err := postgres.Close(); err != nil {
				logger.Error().Msgf("could not close postgres connection: %v", err)
			}
		}()

		if mergestat, err = sql.Open("sqlite3", ":memory:"); err != nil {
			logger.Error().Msgf("could not initialize mergestat: %v", err)
			return
		}
		defer func() {
			if err := mergestat.Close(); err != nil {
				logger.Error().Msgf("could not close mergestat: %v", err)
			}
		}()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// TODO(patrickdevivo) hacky way of adding a SQL statement to run ahead of sync...
		// added mainly so we can run `SET statement_timeout 0` for connections to bit.io
		if preamble := os.Getenv("PGSYNC_PREAMBLE"); preamble != "" {
			if _, err := postgres.ExecContext(ctx, preamble); err != nil {
				logger.Error().Msgf("could not execute preamble: %v", err)
			}
		}

		options := &pgsync.SyncOptions{
			Postgres:   postgres,
			MergeStat:  mergestat,
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
