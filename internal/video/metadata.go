package video

import (
	"fmt"
	"time"
)

// VideoMetadata 存储视频文件的所有关键特征
type VideoMetadata struct {
	Path        string        `json:"path"`         // 绝对路径
	Size        int64         `json:"size"`         // 字节数
	ModifiedAt  time.Time     `json:"modified_at"`  // 修改时间
	Extension   string        `json:"extension"`    // 后缀名
	
	// 以下信息由 ffprobe 提取
	Duration    time.Duration `json:"duration"`     // 时长
	Width       int           `json:"width"`        // 宽度
	Height      int           `json:"height"`       // 高度
	Bitrate     int64         `json:"bitrate"`      // 码率 (bps)
	Format      string        `json:"format"`       // 编码格式
	
	// 用于去重的高级特征
	PartialHash  string        `json:"partial_hash"`  // 采样哈希
	Fingerprints [][]byte      `json:"fingerprints"`  // 视觉特征指纹集合 (多点采样)
	FullHash     string        `json:"full_hash"`
	PHash       string        `json:"p_hash"`       // 视觉指纹 (Perceptual Hash)
}

// Resolution 返回分辨率的显示字符串
func (v *VideoMetadata) Resolution() string {
	return fmt.Sprintf("%dx%d", v.Width, v.Height)
}
