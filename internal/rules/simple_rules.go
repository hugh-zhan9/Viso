package rules

import (
	"fmt"
	"time"
	"github.com/vcmaster/viso/internal/video"
)

type DurationRule struct {
	MinDuration time.Duration
}

func (r *DurationRule) Name() string { return "duration" }
func (r *DurationRule) Description() string { return fmt.Sprintf("时长短于 %v 的视频", r.MinDuration) }

func (r *DurationRule) Evaluate(v *video.VideoMetadata, ctx *Context) (Result, error) {
	if v.Duration < r.MinDuration {
		return Result{
			Matched:  true,
			Reason:   fmt.Sprintf("时长过短 (%v)", v.Duration),
			Priority: 50,
		}, nil
	}
	return Result{Matched: false}, nil
}

type ResolutionRule struct {
	MinWidth  int
	MinHeight int
}

func (r *ResolutionRule) Name() string { return "resolution" }
func (r *ResolutionRule) Description() string { return fmt.Sprintf("分辨率低于 %dx%d 的视频", r.MinWidth, r.MinHeight) }

func (r *ResolutionRule) Evaluate(v *video.VideoMetadata, ctx *Context) (Result, error) {
	if v.Width < r.MinWidth || v.Height < r.MinHeight {
		return Result{
			Matched:  true,
			Reason:   fmt.Sprintf("分辨率过低 (%dx%d)", v.Width, v.Height),
			Priority: 40,
		}, nil
	}
	return Result{Matched: false}, nil
}
