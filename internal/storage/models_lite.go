//go:build !completion

package storage

// appendCompletionModels 是 lite 版本的空实现。
// lite 版本不注册任何 completion 专属持久化模型。
func appendCompletionModels() {}
