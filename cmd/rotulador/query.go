/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/lewtec/rotulador/annotation"
	"github.com/spf13/cobra"
)

func PrintQuery(ctx context.Context, db *sql.Tx, query string, args ...interface{}) error {
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer func() {
		if err := stmt.Close(); err != nil {
			annotation.ReportError(ctx, err, "msg", "failed to close statement")
		}
	}()

	result, err := stmt.QueryContext(ctx, args...)
	if err != nil {
		return err
	}
	defer func() {
		if err := result.Close(); err != nil {
			annotation.ReportError(ctx, err, "msg", "failed to close rows")
		}
	}()

	columns, err := result.Columns()
	if err != nil {
		return err
	}
	if len(columns) > 1 {
		fmt.Println(strings.Join(columns, "\t"))
	}
	pointers := make([]interface{}, len(columns))
	container := make([]string, len(columns))
	for i := 0; i < len(columns); i++ {
		pointers[i] = &container[i]
	}
	for result.Next() {
		if err := result.Scan(pointers...); err != nil {
			return err
		}
		fmt.Println(strings.Join(container, "\t"))
	}
	if err := result.Err(); err != nil {
		return err
	}
	return nil
}

// queryCmd represents the query command
var queryCmd = &cobra.Command{
	Use:   "query [flags] database [stage_index] [option_value] [image_ref]",
	Short: "Queries the annotation database",
	Long: `Query annotations from the database using the current schema
(images keyed by sha256, annotations joined on image_sha256).

Examples:
  # List all distinct stage indexes (phases)
  rotulador query annotations.db

  # List all distinct option values for stage 0
  rotulador query annotations.db 0

  # List images annotated with value "landscape" for stage 0
  rotulador query annotations.db 0 landscape

  # Filter to a specific image by SHA256 or filename
  rotulador query annotations.db 0 landscape image.jpg`,
	RunE: func(cmd *cobra.Command, args []string) error {
		showIDs, err := cmd.Flags().GetBool("show-ids")
		if err != nil {
			return err
		}
		if len(args) < 1 {
			return cmd.Help()
		}
		db, err := annotation.GetDatabase(args[0])
		if err != nil {
			return err
		}
		defer func() {
			if err := db.Close(); err != nil {
				annotation.ReportError(cmd.Context(), err, "msg", "failed to close database")
			}
		}()

		tx, err := db.BeginTx(cmd.Context(), &sql.TxOptions{
			Isolation: sql.LevelReadUncommitted,
		})
		if err != nil {
			return err
		}
		defer func() {
			if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
				annotation.ReportError(cmd.Context(), err, "msg", "failed to rollback transaction")
			}
		}()

		// No stage index provided - list all stages
		if len(args) < 2 {
			return PrintQuery(cmd.Context(), tx, "SELECT DISTINCT stage_index FROM annotations ORDER BY stage_index")
		}

		// Stage index provided, no option value - list all option values for stage
		if len(args) < 3 {
			return PrintQuery(cmd.Context(), tx, "SELECT DISTINCT option_value FROM annotations WHERE stage_index = ?", args[1])
		}

		// Build query to find images with specific annotations (current schema)
		var query string
		if showIDs {
			query = "SELECT images.sha256 "
		} else {
			query = "SELECT images.filename "
		}
		query += "FROM annotations "
		query += "JOIN images ON annotations.image_sha256 = images.sha256 "
		query += "WHERE annotations.stage_index = ? "
		queryArgs := []interface{}{args[1]}

		query += "AND annotations.option_value = ? "
		queryArgs = append(queryArgs, args[2])

		if len(args) >= 4 {
			query += "AND (images.sha256 = ? OR images.filename = ?) "
			queryArgs = append(queryArgs, args[3], args[3])
		}

		query += "ORDER BY images.filename"

		return PrintQuery(cmd.Context(), tx, query, queryArgs...)
	},
}

func init() {
	rootCmd.AddCommand(queryCmd)

	queryCmd.Flags().BoolP("show-ids", "i", false, "Show image SHA256 hashes instead of filenames")
}
