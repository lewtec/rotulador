/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"fmt"
	"os"
    "log"
    "io/fs"
    "sync"

	"image"
	"path/filepath"

	"github.com/lewtec/rotulador/annotation"
	"github.com/spf13/cobra"
)

// ingestCmd represents the ingest command
var ingestCmd = &cobra.Command{
	Use:   "ingest",
	Short: "Ingest a folder of files to a folder of images.",
	Long: `Ingest a folder of files that were extracted from somewhere and organize in a flat hierarchy of images.`,
    Args: func (cmd *cobra.Command, args []string) error {
        if err := cobra.MinimumNArgs(2)(cmd, args); err != nil {
            return err
        }
        inputs := args[0:len(args) - 1]
        output := args[len(args)-1]
        for i, input := range(inputs) {
            fileInfo, err := os.Stat(input)
            if err != nil {
                return fmt.Errorf("on %dth argument: %w", i + 1, err)
            }
            if !fileInfo.IsDir() {
                return fmt.Errorf("on %dth argument: must be a directory", i + 1)
            }
        }
        return os.MkdirAll(output, 0777)
    },
	Run: func(cmd *cobra.Command, args []string) {
        inputs := args[0:len(args) - 1]
        output := args[len(args)-1]

        crawledFilepaths := make(chan image.Image, 10) // pipeline

        var wg sync.WaitGroup
        ingestWorker := func(queue chan image.Image) {
            defer wg.Done()
            for image := range queue {
                err := annotation.IngestImage(image, output)
                if err != nil {
                    log.Printf("Ingesting image error: %s", err)
                }
            }
        }
        defer wg.Wait()
        for i := uint(0); i < jobs; i++ {
            wg.Add(1)
            go ingestWorker(crawledFilepaths)
        }

        for _, input := range inputs {
            err := filepath.WalkDir(input, func(path string, info fs.DirEntry, err error) error {
                if err != nil {
                    return err
                }
                if info.IsDir() {
                    return nil
                }
                img, err := annotation.DecodeImage(path)
                if err != nil {
                    return nil
                }
                log.Printf("found image '%s'", path)
                crawledFilepaths <- img
                return nil
            })
            if err != nil {
                log.Printf("error walking directory: %v", err)
            }
        }
        close(crawledFilepaths)
	},
}

var (
    jobs uint
)

func init() {
	rootCmd.AddCommand(ingestCmd)
    ingestCmd.PersistentFlags().UintVarP(&jobs, "jobs", "j", 1, "Amount of concurrent ingestors")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// ingestCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// ingestCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
