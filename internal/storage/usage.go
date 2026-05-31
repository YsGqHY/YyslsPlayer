package storage

// TableUsage 是单类数据的占用统计。
// JSON 存储没有页级精确统计，SizeBytes 始终为序列化片段估算值。
type TableUsage struct {
	TableName string `json:"tableName"`
	RowCount  int64  `json:"rowCount"`
	SizeBytes int64  `json:"sizeBytes"`
	Estimated bool   `json:"estimated"`
}
