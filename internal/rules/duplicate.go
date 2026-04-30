package rules

import (
	"fmt"
	"math/bits"
	"strings"
	"github.com/vcmaster/viso/internal/video"
)

const (
	// pHashHammingThreshold 感知哈希汉明距离阈值 (1024 bit 中允许 ~10% 差异)
	pHashHammingThreshold = 100
	// matchRatioThreshold 采样点命中率阈值 (60%)
	matchRatioThreshold = 0.6
	// durationToleranceNs 异质重复的时长容忍度 (3秒)
	durationToleranceNs = 3 * 1000 * 1000 * 1000
	// clipDurationDiffNs 截取片段的最小时长差异 (5秒)
	clipDurationDiffNs = 5 * 1000 * 1000 * 1000
)

type DuplicateRule struct{}

func (r *DuplicateRule) Name() string { return "duplicate" }
func (r *DuplicateRule) Description() string { return "识别内容或文件重复的视频（支持多点视觉碰撞）" }

func (r *DuplicateRule) Evaluate(v *video.VideoMetadata, ctx *Context) (Result, error) {
	if strings.Contains(v.Path, ".vcmaster-trash") {
		return Result{Matched: false}, nil
	}

	// 1. 文件级哈希完全一致检测 (File Duplicates)
	candidates := ctx.GetCandidatesBySize(v)
	for _, cand := range candidates {
		if v.PartialHash != "" && v.PartialHash == cand.PartialHash {
			if isSuperior(cand, v) {
				return Result{
					Matched:      true,
					Reason:       fmt.Sprintf("文件内容重复 (原件: %s)", cand.Path),
					RuleName:     "duplicate",
					Priority:     100,
					OriginalPath: cand.Path,
				}, nil
			}
		}
	}

	// 2. 基于感知哈希的视觉匹配
	for _, other := range ctx.AllFiles {
		if other.Path == v.Path || strings.Contains(other.Path, ".vcmaster-trash") {
			continue
		}

		if len(v.PHashes) == 0 || len(other.PHashes) == 0 {
			continue
		}

		// 计算匹配率：v 的每个采样点是否在 other 中找到匹配
		matchCount := 0
		for _, phA := range v.PHashes {
			for _, phB := range other.PHashes {
				if hammingDistance(phA, phB) < pHashHammingThreshold {
					matchCount++
					break
				}
			}
		}

		matchRatio := float64(matchCount) / float64(len(v.PHashes))
		if matchRatio < matchRatioThreshold {
			continue
		}

		durationDiff := v.Duration - other.Duration
		if durationDiff < 0 {
			durationDiff = -durationDiff
		}

		// 音频匹配信息（用于增强置信度描述）
		audioDesc := audioMatchDescription(v, other)

		// 情况 A: 时长非常接近 (同片异质)
		if durationDiff < durationToleranceNs {
			if isSuperior(other, v) {
				return Result{
					Matched:      true,
					Reason:       fmt.Sprintf("疑似压缩副本%s (原件: %s)", audioDesc, other.Path),
					RuleName:     "duplicate",
					Priority:     90,
					OriginalPath: other.Path,
				}, nil
			}
		}

		// 情况 B: 视觉匹配但时长不同 (截取片段判定)
		if other.Duration > v.Duration+clipDurationDiffNs {
			// 时序对齐检测：检查片段是否从原片中截取
			if isTemporalSubset(v.PHashes, other.PHashes) {
				return Result{
					Matched:      true,
					Reason:       fmt.Sprintf("疑似从原片截取的片段%s (原件: %s)", audioDesc, other.Path),
					RuleName:     "duplicate",
					Priority:     85,
					OriginalPath: other.Path,
				}, nil
			}
		}
	}

	return Result{Matched: false}, nil
}

// hammingDistance 使用 XOR bit-count 计算两个感知哈希的汉明距离
func hammingDistance(a, b []byte) int {
	dist := 0
	minLen := len(a)
	if len(b) < minLen {
		minLen = len(b)
	}
	for i := 0; i < minLen; i++ {
		dist += bits.OnesCount8(a[i] ^ b[i])
	}
	// 长度不同的部分也算差异
	if len(a) > len(b) {
		for i := minLen; i < len(a); i++ {
			dist += bits.OnesCount8(a[i])
		}
	} else if len(b) > len(a) {
		for i := minLen; i < len(b); i++ {
			dist += bits.OnesCount8(b[i])
		}
	}
	return dist
}

// isTemporalSubset 检查短序列是否为长序列的时间子集（滑动窗口对齐）
// 即：短片段的采样点是否在长原片中找到连续的位置匹配
func isTemporalSubset(shortPHashes, longPHashes [][]byte) bool {
	if len(shortPHashes) == 0 || len(longPHashes) == 0 {
		return false
	}

	// 用滑动窗口：尝试 longPHashes 的每个起始位置
	for start := 0; start <= len(longPHashes)-1; start++ {
		matchCount := 0
		for i, sp := range shortPHashes {
			// 检查是否超出原片范围
			if start+i >= len(longPHashes) {
				break
			}
			if hammingDistance(sp, longPHashes[start+i]) < pHashHammingThreshold {
				matchCount++
			}
		}
		matchRatio := float64(matchCount) / float64(len(shortPHashes))
		if matchRatio >= matchRatioThreshold {
			return true
		}
	}
	return false
}

// audioMatchDescription 检查音频匹配状态，返回描述文本
// 使用滑动窗口容忍广告插入
func audioMatchDescription(a, b *video.VideoMetadata) string {
	if len(a.AudioPHashes) == 0 || len(b.AudioPHashes) == 0 {
		return ""
	}

	// 选择较长的作为基准，较短的作为待匹配
	shorter, longer := a.AudioPHashes, b.AudioPHashes
	if len(b.AudioPHashes) < len(a.AudioPHashes) {
		shorter, longer = b.AudioPHashes, a.AudioPHashes
	}

	// 滑动窗口：在较长序列中找较短序列的最佳匹配位置
	bestRatio := 0.0
	for start := 0; start <= len(longer)-len(shorter); start++ {
		matchCount := 0
		for i, sp := range shorter {
			if hammingDistance(sp, longer[start+i]) < pHashHammingThreshold {
				matchCount++
			}
		}
		ratio := float64(matchCount) / float64(len(shorter))
		if ratio > bestRatio {
			bestRatio = ratio
		}
	}

	if bestRatio >= matchRatioThreshold {
		return "，音频匹配"
	}
	return ""
}

func isSuperior(a, b *video.VideoMetadata) bool {
	if a.Width*a.Height != b.Width*b.Height {
		return a.Width*a.Height > b.Width*b.Height
	}
	if a.Bitrate != b.Bitrate {
		return a.Bitrate > b.Bitrate
	}
	if !a.ModifiedAt.Equal(b.ModifiedAt) {
		return a.ModifiedAt.Before(b.ModifiedAt)
	}
	return a.Path < b.Path
}
