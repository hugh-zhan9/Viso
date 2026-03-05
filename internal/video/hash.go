package video

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"os"
)

const ChunkSize = 64 * 1024 // 64KB

// GetPartialHash 计算视频的采样哈希 (首/中/尾)
func GetPartialHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return "", err
	}

	size := info.Size()
	hash := md5.New()

	// 1. 读取头部
	if _, err := io.CopyN(hash, f, ChunkSize); err != nil && err != io.EOF {
		return "", err
	}

	// 2. 读取中部
	if size > ChunkSize*3 {
		if _, err := f.Seek(size/2, io.SeekStart); err == nil {
			io.CopyN(hash, f, ChunkSize)
		}
	}

	// 3. 读取尾部
	if size > ChunkSize {
		if _, err := f.Seek(size-ChunkSize, io.SeekStart); err == nil {
			io.Copy(hash, f)
		}
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}
