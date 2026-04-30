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
		Width      int    `json:"width"`
		Height     int    `json:"height"`
		Duration   string `json:"duration"`
		BitRate    string `json:"bit_rate"`
		CodecName  string `json:"codec_name"`
		CodecType  string `json:"codec_type"`
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
			cmd := exec.Command("ffmpeg", "-ss", fmt.Sprintf("%.2f", offset), "-i", path, "-vframes", "1", "-vf", "scale=32:32,format=gray", "-f", "rawvideo", "-")
			if fp, err := cmd.Output(); err == nil && len(fp) == 1024 {
				meta.Fingerprints = append(meta.Fingerprints, fp)
				meta.PHashes = append(meta.PHashes, toPHash(fp))
			}
		}
	}

	// 提取音频频谱感知哈希
	if hasAudioStream(data.Streams) && meta.Duration > 0 {
		meta.AudioPHashes = probeAudioPHashes(path, meta.Duration.Seconds())
	}

	return meta, nil
}

// probeAudioPHashes 每 10 秒提取一段音频频谱图并转为感知哈希
func probeAudioPHashes(path string, durationSec float64) [][]byte {
	if durationSec <= 0 {
		return nil
	}

	audioInterval := 10.0 // 每 10 秒一段
	var pHashes [][]byte

	for offset := 0.0; offset < durationSec; offset += audioInterval {
		// 提取 32x32 灰度频谱图 (showspectrumpic 从音频生成频谱)
		cmd := exec.Command("ffmpeg", "-ss", fmt.Sprintf("%.2f", offset), "-i", path,
			"-t", fmt.Sprintf("%.2f", audioInterval),
			"-lavfi", "showspectrumpic=s=32x32:legend=0,format=gray",
			"-frames:v", "1", "-f", "rawvideo", "-")
		fp, err := cmd.Output()
		if err != nil || len(fp) != 1024 {
			continue
		}
		pHashes = append(pHashes, toPHash(fp))
	}
	return pHashes
}

// hasAudioStream 检查视频是否包含音频流
func hasAudioStream(streams []struct {
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	Duration   string `json:"duration"`
	BitRate    string `json:"bit_rate"`
	CodecName  string `json:"codec_name"`
	CodecType  string `json:"codec_type"`
}) bool {
	for _, s := range streams {
		if s.CodecType == "audio" {
			return true
		}
	}
	return false
}

// toPHash 将原始灰度像素转换为二值化感知哈希 (bit array)
// 计算像素均值，>= 均值为 1，否则为 0，压缩为 []byte
func toPHash(pixels []byte) []byte {
	if len(pixels) == 0 {
		return nil
	}
	// 计算均值
	var sum int
	for _, p := range pixels {
		sum += int(p)
	}
	avg := byte(sum / len(pixels))

	// 二值化并压缩为 bit array
	byteLen := (len(pixels) + 7) / 8
	hash := make([]byte, byteLen)
	for i, p := range pixels {
		if p >= avg {
			hash[i/8] |= 1 << (7 - uint(i%8))
		}
	}
	return hash
}
