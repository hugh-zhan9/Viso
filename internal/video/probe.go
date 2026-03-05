package video

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"time"
)

type ffprobeOutput struct {
	Streams []struct {
		Width     int    `json:"width"`
		Height    int    `json:"height"`
		Duration  string `json:"duration"`
		BitRate   string `json:"bit_rate"`
		CodecName string `json:"codec_name"`
	} `json:"streams"`
	Format struct {
		Duration string `json:"duration"`
		BitRate  string `json:"bit_rate"`
	} `json:"format"`
}

// ProbeVideo 使用 ffprobe 提取视频元数据，并根据采样点数提取视觉特征
func ProbeVideo(path string, sampleCount int) (*VideoMetadata, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-show_format", "-show_streams", "-of", "json", path)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe error: %w", err)
	}

	var data ffprobeOutput
	if err := json.Unmarshal(out, &data); err != nil {
		return nil, err
	}

	meta := &VideoMetadata{Path: path}
	
	// ... (省略流解析逻辑)
	for _, s := range data.Streams {
		if s.Width > 0 {
			meta.Width = s.Width
			meta.Height = s.Height
			meta.Format = s.CodecName
			break
		}
	}

	durStr := data.Format.Duration
	if durStr == "" && len(data.Streams) > 0 {
		durStr = data.Streams[0].Duration
	}
	if durStr != "" {
		if d, err := strconv.ParseFloat(durStr, 64); err == nil {
			meta.Duration = time.Duration(d * float64(time.Second))
		}
	}

	brStr := data.Format.BitRate
	if brStr == "" && len(data.Streams) > 0 {
		brStr = data.Streams[0].BitRate
	}
	if brStr != "" {
		if br, err := strconv.ParseInt(brStr, 10, 64); err == nil {
			meta.Bitrate = br
		}
	}

	if meta.Duration > 0 && sampleCount > 0 {
		dur := meta.Duration.Seconds()
		// 动态生成采样点：(i-0.5)/N
		for i := 1; i <= sampleCount; i++ {
			offset := dur * (float64(i) - 0.5) / float64(sampleCount)
			cmd := exec.Command("ffmpeg", "-ss", fmt.Sprintf("%.2f", offset), "-i", path, "-vframes", "1", "-vf", "scale=16:16,format=gray", "-f", "rawvideo", "-")
			if fp, err := cmd.Output(); err == nil && len(fp) == 256 {
				meta.Fingerprints = append(meta.Fingerprints, fp)
			}
		}
	}

	return meta, nil
}
