package scr

type Config struct {
	// 输出根目录，例如: origami
	OutputRoot string
	// 自定义 GetName 拼接前缀；为空则使用源包名
	NamePrefix string
	// 最大递归生成层次（<=0 表示不限制）
	MaxDepth int

	// 黑名单配置
	Blacklist BlacklistConfig

	// 依赖包映射配置
	PackageMappings map[string]string

	// 文件固定替换，准备生成的文件时检查，如果匹配则替换而不是新生成
	FixedReplace map[string]string
}

// BlacklistConfig 黑名单配置
type BlacklistConfig struct {
	// 包路径黑名单-只能生成 data.AnyValue
	Packages []string
}
