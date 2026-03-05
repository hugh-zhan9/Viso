package scanner

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vcmaster/viso/internal/video"
	"golang.org/x/sync/errgroup"
)

var videoExtensions = map[string]bool{
	".mp4": true, ".mkv": true, ".avi": true, ".mov": true,
	".flv": true, ".ts":  true, ".wmv": true, ".m4v": true,
}

// Scanner 并发扫描视频元数据
type Scanner struct {
	Concurrency int
}

func NewScanner(concurrency int) *Scanner {
	if concurrency <= 0 {
		concurrency = 4
	}
	return &Scanner{Concurrency: concurrency}
}

var skipExtensions = map[string]bool{
	".crdownload": true, ".download": true, ".part": true,
	".aria2": true, ".tmp": true, ".xltd": true,
}

func (s *Scanner) Scan(ctx context.Context, root string, sampleCount int) ([]*video.VideoMetadata, error) {
	g, ctx := errgroup.WithContext(ctx)
	paths := make(chan string, 100)

	// ... (遍历逻辑不变)
	g.Go(func() error {
		defer close(paths)
		return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			
			// 绝对屏蔽：只要路径中包含回收站目录名，直接跳过
			if strings.Contains(path, ".viso-trash") {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			if info.IsDir() {
				return nil
			}
			
			// 排除下载中的临时文件
			ext := strings.ToLower(filepath.Ext(path))
			if skipExtensions[ext] {
				return nil
			}

			// 排除 5 分钟内修改过的活跃文件 (可能正在下载/写入)
			if time.Since(info.ModTime()) < 5*time.Minute {
				return nil
			}

			if videoExtensions[ext] {
				select {
				case paths <- path:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			return nil
		})
	})

	// 2. 并发提取元数据
	results := make(chan *video.VideoMetadata, 100)
	for i := 0; i < s.Concurrency; i++ {
		g.Go(func() error {
			for path := range paths {
				// 传递 sampleCount
				meta, err := video.ProbeVideo(path, sampleCount)
				if err != nil {
					meta = &video.VideoMetadata{Path: path}
				}
				
				// 获取文件基础属性
				info, _ := os.Stat(path)
				if info != nil {
					meta.Size = info.Size()
					meta.ModifiedAt = info.ModTime()
					meta.Extension = filepath.Ext(path)
				}

				// 计算采样哈希
				hash, _ := video.GetPartialHash(path)
				meta.PartialHash = hash

				select {
				case results <- meta:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			return nil
		})
	}

	// 3. 关闭结果通道
	go func() {
		g.Wait()
		close(results)
	}()

	var all []*video.VideoMetadata
	for res := range results {
		all = append(all, res)
	}

	return all, g.Wait()
}
