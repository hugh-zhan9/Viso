## [2026-03-05 19:14] [Feature]
- **Change**: 新增 viso CLI 入口，支持 scan 子命令与参数解析，并输出规则命中结果；补充入口层测试。
- **Risk Analysis**: 新增入口层逻辑可能引入参数兼容性问题（短长参数混用、路径参数位置）；当前已覆盖基础分支，但未覆盖真实 ffprobe/ffmpeg 环境行为，存在运行环境依赖风险。
- **Risk Level**: S2（中级: 局部功能异常、可绕过但影响效率）
- **Changed Files**:
- `cmd/viso/main.go`
- `cmd/viso/main_test.go`
----------------------------------------
## [2026-03-05 19:17] [Bugfix]
- **Change**: 修复 .gitignore 规则，将 viso 改为 /viso，避免误忽略 cmd/viso 源码目录。
- **Risk Analysis**: 忽略规则变更可能影响本地未跟踪文件显示；风险较低，主要是 git 状态噪音变化，不影响运行时行为。
- **Risk Level**: S3（低级: 轻微行为偏差或日志/可观测性影响）
- **Changed Files**:
- `.gitignore`
----------------------------------------
