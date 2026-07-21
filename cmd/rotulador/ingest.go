/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"fmt"
	"image"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"github.com/lewtec/rotulador/annotation"
	"github.com/spf13/cobra"
)

// ingestCmd represents the ingest command
var ingestCmd = &cobra.Command{
	Use:   "ingest",
	Short: "Ingest a folder of files to a folder of images.",
	Long:  `Ingest a folder of files that were extracted from somewhere and organize in a flat hierarchy of images.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if err := cobra.MinimumNArgs(2)(cmd, args); err != nil {
			return err
		}
		inputs := args[0 : len(args)-1]
		output := args[len(args)-1]
		for i, input := range inputs {
			fileInfo, err := os.Stat(input)
			if err != nil {
				return fmt.Errorf("on %dth argument: %w", i+1, err)
			}
			if !fileInfo.IsDir() {
				return fmt.Errorf("on %dth argument: must be a directory", i+1)
			}
		}
		return os.MkdirAll(output, 0777)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		logger, err := getLogger(cmd)
		if err != nil {
			return err
		}
		// --jobs 0 starts no workers. WalkDir then either drops every decoded
		// image (buffer never drained) or deadlocks once the channel buffer
		// fills. Reject it early instead of hanging or silently no-op'ing.
		if jobs < 1 {
			return fmt.Errorf("--jobs must be at least 1, got %d", jobs)
		}
		inputs := args[0 : len(args)-1]
		output := args[len(args)-1]

		crawledFilepaths := make(chan image.Image, 10) // pipeline

		var wg sync.WaitGroup
		ingestWorker := func(queue chan image.Image) {
			defer wg.Done()
			for image := range queue {
				err := annotation.IngestImage(image, output)
				if err != nil {
					annotation.ReportError(cmd.Context(), err, "msg", "ingesting image failed")
				}
			}
		}
		for i := uint(0); i < jobs; i++ {
			wg.Add(1)
			go ingestWorker(crawledFilepaths)
		}

		// Always close the channel and wait for workers before returning.
		// Returning on WalkDir failure without close left workers blocked on
		// range and hung forever on Wait.
		var walkErr error
		for _, input := range inputs {
			if err := filepath.WalkDir(input, func(path string, info fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if info.IsDir() {
					return nil
				}
				img, err := annotation.DecodeImage(path)
				if err != nil {
					// Mixed input folders commonly contain non-images; skip them.
					logger.Debug("skipping non-image file", "path", path, "err", err)
					return nil
				}
				logger.Info("found image", "path", path)
				crawledFilepaths <- img
				return nil
			}); err != nil {
				walkErr = fmt.Errorf("walking input directory %s: %w", input, err)
				break
			}
		}
		close(crawledFilepaths)
		wg.Wait()
		return walkErr
	},
}

var (
	jobs uint
)

func init() {
	rootCmd.AddCommand(ingestCmd)
	ingestCmd.PersistentFlags().UintVarP(&jobs, "jobs", "j", 1, "Amount of concurrent ingestors (must be >= 1)")
}
