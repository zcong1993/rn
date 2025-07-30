package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/urfave/cli/v3"
)

func main() {
	app := &cli.Command{
		Name:  "rn",
		Usage: "simple tools for renaming files and so on",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "files",
				Value: "*",
				Usage: "files matcher",
			},
			&cli.BoolFlag{
				Name:  "write",
				Usage: "actually do rename",
			},
			&cli.IntFlag{
				Name:  "start",
				Value: 1,
				Usage: "start number for renaming",
			},
			&cli.StringFlag{
				Name:  "prefix",
				Value: "S01E",
				Usage: "prefix for renaming",
			},
			&cli.BoolFlag{
				Name:  "revert",
				Usage: "revert renaming",
			},
			&cli.IntFlag{
				Name:  "width",
				Value: 2,
				Usage: "number width",
			},
		},
		Action: rn(),

		Commands: []*cli.Command{
			{
				Name:  "mv",
				Usage: "Batch move files according to config file",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "config",
						Usage: "config file path, default is .mv.json",
					},
					&cli.BoolFlag{
						Name:  "write",
						Usage: "actually perform move operation (default is dry-run only)",
					},
				},
				Action: mv(),
			},
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

func rn() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		files := cmd.String("files")
		write := cmd.Bool("write")
		start := cmd.Int("start")
		prefix := cmd.String("prefix")
		revert := cmd.Bool("revert")
		width := cmd.Int("width")

		fs, err := filepath.Glob(files)
		if err != nil {
			return err
		}

		sort.Strings(fs)
		dmp := diffmatchpatch.New()
		revertReg := regexp.MustCompile(`^S\d{2}E\d{2,4} - `)

		var res [][2]string
		for _, f := range fs {
			if strings.HasPrefix(filepath.Base(f), ".") {
				continue
			}

			var r [2]string
			r[0] = f
			if revert {
				r[1] = revertReg.ReplaceAllString(f, "")
			} else {
				r[1] = fmt.Sprintf("%s%s - %s", prefix, padNumber(width, start), f)
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
	}
}

type MoveRule struct {
	Regex string `json:"regex"`
	Dest  string `json:"dest"`
}

func mv() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		configFile := cmd.String("config")
		if configFile == "" {
			configFile = ".mv.json"
		}
		write := cmd.Bool("write")

		data, err := os.ReadFile(configFile)
		if err != nil {
			return fmt.Errorf("failed to read config file: %w", err)
		}

		var rules []MoveRule
		if err := json.Unmarshal(data, &rules); err != nil {
			return fmt.Errorf("failed to parse config file: %w", err)
		}

		compiledRules := make([]struct {
			re   *regexp.Regexp
			dest string
		}, len(rules))

		for i, rule := range rules {
			re, err := regexp.Compile(rule.Regex)
			if err != nil {
				return fmt.Errorf("failed to compile regex %s: %w", rule.Regex, err)
			}
			compiledRules[i].re = re
			compiledRules[i].dest = rule.Dest
		}

		dryRunColor := color.New(color.FgYellow).SprintFunc()
		moveColor := color.New(color.FgGreen).SprintFunc()
		pathColor := color.New(color.FgCyan).SprintFunc()

		for _, rule := range compiledRules {
			files, err := filepath.Glob("*")
			if err != nil {
				return fmt.Errorf("failed to scan directory: %w", err)
			}

			for _, name := range files {
				if !rule.re.MatchString(name) {
					continue
				}
				destPath := filepath.Join(rule.dest, name)
				if write {
					if err := os.Rename(name, destPath); err != nil {
						return fmt.Errorf("failed to move %s -> %s: %w", name, destPath, err)
					}
					fmt.Printf("%s: %s %s %s\n",
						moveColor("Moved"),
						pathColor(name),
						color.New(color.FgMagenta).Sprint("→"),
						pathColor(destPath),
					)
				} else {
					fmt.Printf("%s: %s %s %s\n",
						dryRunColor("[dry-run] Will move"),
						pathColor(name),
						color.New(color.FgMagenta).Sprint("→"),
						pathColor(destPath),
					)
				}
			}
		}

		return nil
	}
}

func padNumber(width, num int) string {
	return fmt.Sprintf("%0*d", width, num)
}
