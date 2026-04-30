package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync/atomic"
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

	// 使用原子变量存储最新计数和阶段，供动画 goroutine 读取
	var latestTotal, latestVideo int64
	var latestPhase atomic.Value
	latestPhase.Store("")
	sc.OnFileProcessed = func(total int, videoCount int) {
		atomic.StoreInt64(&latestTotal, int64(total))
		atomic.StoreInt64(&latestVideo, int64(videoCount))
	}
	sc.OnPhaseChange = func(phase string) {
		latestPhase.Store(phase)
	}

	// 启动动画 goroutine，每 500ms 刷新一次省略号
	done := make(chan struct{})
	go func() {
		dots := []string{".", "..", "..."}
		i := 0
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				t := atomic.LoadInt64(&latestTotal)
				v := atomic.LoadInt64(&latestVideo)
				phase := latestPhase.Load().(string)
				if phase != "" {
					fmt.Fprintf(os.Stderr, "\r正在扫描中，第 %d 个文件，%d 个视频文件，%s%s   ", t, v, phase, dots[i%3])
				} else {
					fmt.Fprintf(os.Stderr, "\r正在扫描中，第 %d 个文件，%d 个视频文件%s   ", t, v, dots[i%3])
				}
				i++
			}
		}
	}()

	videos, err := sc.Scan(context.Background(), opts.root, opts.samples)
	close(done)
	fmt.Fprintln(os.Stderr)
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

	// 按类型分组
	groups := make(map[string][]rules.Result)
	groupOrder := []string{"文件内容重复", "疑似压缩副本", "疑似从原片截取的片段", "时长过短", "分辨率过低"}

	for _, res := range report {
		category := categorizeResult(res)
		groups[category] = append(groups[category], res)
	}

	for _, cat := range groupOrder {
		items := groups[cat]
		if len(items) == 0 {
			continue
		}
		sort.Slice(items, func(i, j int) bool {
			return extractPath(items[i]) < extractPath(items[j])
		})
		fmt.Fprintf(stdout, "\n【%s】(%d 项)\n", cat, len(items))
		for _, res := range items {
			path := extractPath(res)
			fmt.Fprintf(stdout, "  - %s\n    %s\n", clickablePath(path), makeReasonClickable(res.Reason))
		}
	}

	return nil
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "用法:")
	fmt.Fprintln(w, "  viso scan [目录] [-s 采样点] [-d 最小时长] [-W 最小宽] [-H 最小高]")
}

// categorizeResult 根据 Reason 内容判定分组类型
func categorizeResult(res rules.Result) string {
	switch {
	case strings.Contains(res.Reason, "文件内容重复"):
		return "文件内容重复"
	case strings.Contains(res.Reason, "压缩副本"):
		return "疑似压缩副本"
	case strings.Contains(res.Reason, "截取的片段"):
		return "疑似从原片截取的片段"
	case strings.Contains(res.Reason, "时长过短"):
		return "时长过短"
	case strings.Contains(res.Reason, "分辨率过低"):
		return "分辨率过低"
	default:
		return "其他"
	}
}

// extractPath 从结果中提取文件路径
func extractPath(res rules.Result) string {
	if res.OriginalPath != "" {
		return res.OriginalPath
	}
	// 从 Reason 中提取路径（格式: "... (原件: /path)" 或直接是被检测文件）
	if idx := strings.LastIndex(res.Reason, "(原件: "); idx >= 0 {
		end := strings.LastIndex(res.Reason, ")")
		if end > idx+6 {
			return res.Reason[idx+6 : end]
		}
	}
	return res.Reason
}

// clickablePath 用 OSC 8 转义序列将路径包装为终端可点击链接
func clickablePath(path string) string {
	return fmt.Sprintf("\033]8;;file://%s\033\\%s\033]8;;\033\\", path, path)
}

// makeReasonClickable 将 Reason 中的 (原件: /path) 部分的路径也变为可点击链接
func makeReasonClickable(reason string) string {
	const prefix = "(原件: "
	idx := strings.LastIndex(reason, prefix)
	if idx < 0 {
		return reason
	}
	end := strings.LastIndex(reason, ")")
	if end <= idx+len(prefix) {
		return reason
	}
	originalPath := reason[idx+len(prefix) : end]
	return reason[:idx+len(prefix)] + clickablePath(originalPath) + reason[end:]
}
