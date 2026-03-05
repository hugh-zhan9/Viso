package rules

import (
	"fmt"
	"strings"
	"github.com/vcmaster/viso/internal/video"
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

	// 2. 多点视觉特征碰撞 (视觉子集判定)
	for _, other := range ctx.AllFiles {
		if other.Path == v.Path || strings.Contains(other.Path, ".vcmaster-trash") {
			continue
		}

		if len(v.Fingerprints) > 0 && len(other.Fingerprints) > 0 {
			matchCount := 0
			// 对比 v 的每一个采样点，看是否在 other 的采样点中出现
			for _, fpA := range v.Fingerprints {
				for _, fpB := range other.Fingerprints {
					if hammingDistance(fpA, fpB) < 32 {
						matchCount++
						break // 只要 fpA 找到了匹配的 fpB，跳到下一个 fpA
					}
				}
			}

			// 如果 5 个点里有 2 个及以上命中了视觉匹配
			if matchCount >= 2 {
				durationDiff := v.Duration - other.Duration
				if durationDiff < 0 { durationDiff = -durationDiff }

				// 情况 A: 时长非常接近 (同片异质)
				if durationDiff < 3000*1000*1000 { // 3s 误差
					if isSuperior(other, v) {
						return Result{
							Matched:      true,
							Reason:       fmt.Sprintf("疑似压缩副本 (原件: %s)", other.Path),
							RuleName:     "duplicate",
							Priority:     90,
							OriginalPath: other.Path,
						}, nil
					}
				}

				// 情况 B: 视觉匹配但时长不同 (截取片段判定)
				// 只有当 other (原件) 明显更长时，才建议清理短的那个 v
				if other.Duration > v.Duration + 5*1000*1000*1000 {
					return Result{
						Matched:      true,
						Reason:       fmt.Sprintf("疑似从原片截取的片段 (原件: %s)", other.Path),
						RuleName:     "duplicate",
						Priority:     85,
						OriginalPath: other.Path,
					}, nil
				}
			}
		}
	}

	return Result{Matched: false}, nil
}

// hammingDistance 计算视觉差异度
func hammingDistance(a, b []byte) int {
	dist := 0
	for i := 0; i < len(a); i++ {
		diff := int(a[i]) - int(b[i])
		if diff < 0 { diff = -diff }
		if diff > 15 { dist++ }
	}
	return dist
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
