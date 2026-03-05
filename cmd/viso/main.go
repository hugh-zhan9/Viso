package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"sort"
	"time"

	"github.com/vcmaster/viso/internal/rules"
	"github.com/vcmaster/viso/internal/scanner"
)

type scanOptions struct {
	root        string
	samples     int
	minDuration time.Duration
	minWidth    int
	minHeight   int
}

var (
	stdout      io.Writer = os.Stdout
	executeScan           = runScan
)

func main() {
	os.Exit(run(os.Args[1:], stdout, os.Stderr))
}

func run(args []string, out io.Writer, errOut io.Writer) int {
	if len(args) == 0 {
		printUsage(errOut)
		return 1
	}

	switch args[0] {
	case "scan":
		opts, err := parseScanArgs(args[1:])
		if err != nil {
			fmt.Fprintf(errOut, "参数错误: %v\n", err)
			return 1
		}
		if err := executeScan(opts); err != nil {
			fmt.Fprintf(errOut, "扫描失败: %v\n", err)
			return 1
		}
		return 0
	default:
		fmt.Fprintf(errOut, "未知子命令: %s\n", args[0])
		printUsage(errOut)
		return 1
	}
}

func parseScanArgs(args []string) (scanOptions, error) {
	fs := flag.NewFlagSet("scan", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	samples := fs.Int("s", 5, "采样点数量")
	fs.IntVar(samples, "samples", 5, "采样点数量")

	minDuration := fs.Duration("d", 5*time.Second, "最小时长")
	fs.DurationVar(minDuration, "duration", 5*time.Second, "最小时长")

	minWidth := fs.Int("W", 480, "最小宽度")
	fs.IntVar(minWidth, "width", 480, "最小宽度")

	minHeight := fs.Int("H", 320, "最小高度")
	fs.IntVar(minHeight, "height", 320, "最小高度")

	if err := fs.Parse(args); err != nil {
		return scanOptions{}, err
	}

	root := "."
	rest := fs.Args()
	if len(rest) > 0 {
		root = rest[0]
	}

	return scanOptions{
		root:        root,
		samples:     *samples,
		minDuration: *minDuration,
		minWidth:    *minWidth,
		minHeight:   *minHeight,
	}, nil
}

func runScan(opts scanOptions) error {
	sc := scanner.NewScanner(runtime.NumCPU())
	videos, err := sc.Scan(context.Background(), opts.root, opts.samples)
	if err != nil {
		return err
	}

	engine := rules.NewEngine([]rules.Rule{
		&rules.DuplicateRule{},
		&rules.DurationRule{MinDuration: opts.minDuration},
		&rules.ResolutionRule{MinWidth: opts.minWidth, MinHeight: opts.minHeight},
	})

	report := engine.Run(videos)

	fmt.Fprintf(stdout, "扫描完成: 共 %d 个视频，命中 %d 个待清理项\n", len(videos), len(report))
	if len(report) == 0 {
		return nil
	}

	paths := make([]string, 0, len(report))
	for path := range report {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	for _, path := range paths {
		res := report[path]
		fmt.Fprintf(stdout, "- [%s] %s (%s)\n", res.RuleName, path, res.Reason)
	}

	return nil
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "用法:")
	fmt.Fprintln(w, "  viso scan [目录] [-s 采样点] [-d 最小时长] [-W 最小宽] [-H 最小高]")
}
