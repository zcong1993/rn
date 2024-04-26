package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/urfave/cli/v2"
)

func main() {
	var (
		files  string
		write  bool
		start  int
		prefix string
		revert bool
	)
	dmp := diffmatchpatch.New()
	revertReg := regexp.MustCompile(`^S\d{2}E\d{2} - `)

	app := &cli.App{
		Name:  "rn",
		Usage: "simple tools for renaming files",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "files",
				Value:       "*",
				Usage:       "files matcher",
				Destination: &files,
			},
			&cli.BoolFlag{
				Name:        "write",
				Usage:       "actully do rename",
				Destination: &write,
			},
			&cli.IntFlag{
				Name:        "start",
				Value:       1,
				Usage:       "start number for renaming",
				Destination: &start,
			},
			&cli.StringFlag{
				Name:        "prefix",
				Value:       "S01E",
				Usage:       "prefix for renaming",
				Destination: &prefix,
			},
			&cli.BoolFlag{
				Name:        "revert",
				Usage:       "revert renaming",
				Destination: &revert,
			},
		},
		Action: func(*cli.Context) error {
			fs, err := filepath.Glob(files)
			if err != nil {
				return err
			}
			sort.Strings(fs)
			var res [][2]string
			for _, f := range fs {
				var r [2]string
				r[0] = f
				if revert {
					r[1] = revertReg.ReplaceAllString(f, "")
				} else {
					r[1] = fmt.Sprintf("%s%s - %s", prefix, handleNumber(start), f)
				}
				res = append(res, r)
				start++
				diffs := dmp.DiffMain(r[0], r[1], false)
				fmt.Println(dmp.DiffPrettyText(diffs))
			}
			if write {
				for _, r := range res {
					if err := os.Rename(r[0], r[1]); err != nil {
						return err
					}
				}
			}
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func handleNumber(i int) string {
	if i < 10 {
		return fmt.Sprintf("0%d", i)
	}
	return fmt.Sprintf("%d", i)
}
