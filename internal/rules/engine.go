package rules

import (
	"github.com/vcmaster/viso/internal/video"
)

// Engine 负责执行注册的规则
type Engine struct {
	rules []Rule
}

func NewEngine(rules []Rule) *Engine {
	return &Engine{rules: rules}
}

// Run 评估所有视频并返回命中的结果映射 (Path -> Result)
func (e *Engine) Run(videos []*video.VideoMetadata) map[string]Result {
	ctx := NewContext(videos)
	report := make(map[string]Result)

	for _, v := range videos {
		var bestMatch Result
		for _, rule := range e.rules {
			res, err := rule.Evaluate(v, ctx)
			if err != nil || !res.Matched {
				continue
			}

			// 如果命中多个规则，保留优先级最高的
			if !bestMatch.Matched || res.Priority > bestMatch.Priority {
				bestMatch = res
				bestMatch.RuleName = rule.Name()
			}
		}

		if bestMatch.Matched {
			report[v.Path] = bestMatch
		}
	}

	return report
}
