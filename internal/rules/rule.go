package rules

import (
	"github.com/vcmaster/viso/internal/video"
)

// Result 定义规则评估的结果
type Result struct {
	Matched      bool   // 是否命中规则
	RuleName     string // 命中的规则名称
	Reason       string // 命中的理由
	Priority     int    // 优先级
	OriginalPath string // 如果是重复规则，存储原件路径以便对比
}

// Rule 定义了规则的契约
type Rule interface {
	Name() string
	Description() string
	// Evaluate 评估一个视频是否命中该规则
	Evaluate(v *video.VideoMetadata, context *Context) (Result, error)
}

// Context 为规则评估提供全局上下文 (例如所有已扫描文件的映射)
type Context struct {
	AllFiles []*video.VideoMetadata
	// 用于重复检测的缓存
	sizeMap map[int64][]*video.VideoMetadata
}

func NewContext(all []*video.VideoMetadata) *Context {
	ctx := &Context{
		AllFiles: all,
		sizeMap:  make(map[int64][]*video.VideoMetadata),
	}
	// 预先按大小分组
	for _, v := range all {
		ctx.sizeMap[v.Size] = append(ctx.sizeMap[v.Size], v)
	}
	return ctx
}

// GetCandidatesBySize 获取与给定文件大小相同的所有其他文件
func (c *Context) GetCandidatesBySize(v *video.VideoMetadata) []*video.VideoMetadata {
	candidates := c.sizeMap[v.Size]
	var others []*video.VideoMetadata
	for _, cand := range candidates {
		if cand.Path != v.Path {
			others = append(others, cand)
		}
	}
	return others
}
