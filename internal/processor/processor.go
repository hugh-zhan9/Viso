package processor

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Processor struct {
	ScanRoot string // 扫描的根目录
	TrashDir string // 回收站相对于根目录的路径
}

func NewProcessor(scanRoot string, trashName string) (*Processor, error) {
	absRoot, err := filepath.Abs(scanRoot)
	if err != nil {
		return nil, err
	}
	
	trashPath := filepath.Join(absRoot, trashName)
	if err := os.MkdirAll(trashPath, 0755); err != nil {
		return nil, err
	}
	
	return &Processor{
		ScanRoot: absRoot,
		TrashDir: trashPath,
	}, nil
}

// MoveToTrash 将文件安全移动到回收站，并保留其原始路径结构
func (p *Processor) MoveToTrash(srcPath string) (string, error) {
	absSrc, err := filepath.Abs(srcPath)
	if err != nil {
		return "", err
	}

	// 1. 计算相对路径，用于在回收站重建目录结构
	relPath, err := filepath.Rel(p.ScanRoot, absSrc)
	if err != nil {
		return "", fmt.Errorf("failed to calculate relative path: %w", err)
	}

	// 2. 构造目标路径 (在回收站内镜像目录)
	targetPath := filepath.Join(p.TrashDir, relPath)
	targetDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", err
	}

	// 3. 处理同名冲突：在文件名后加时间戳
	if _, err := os.Stat(targetPath); err == nil {
		ext := filepath.Ext(targetPath)
		base := strings.TrimSuffix(targetPath, ext)
		targetPath = fmt.Sprintf("%s_%s%s", base, time.Now().Format("150405"), ext)
	}

	// 4. 执行移动
	// 尝试直接重命名 (同分区秒完成)
	err = os.Rename(absSrc, targetPath)
	if err != nil {
		// 如果跨文件系统移动，os.Rename 会失败，执行 Copy + Remove
		if err := p.copyAndDelete(absSrc, targetPath); err != nil {
			return "", fmt.Errorf("cross-disk move failed: %w", err)
		}
	}

	return targetPath, nil
}

// copyAndDelete 处理跨磁盘分区的移动
func (p *Processor) copyAndDelete(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	if _, err := io.Copy(destination, source); err != nil {
		return err
	}

	source.Close() // 必须先关闭才能删除
	return os.Remove(src)
}
