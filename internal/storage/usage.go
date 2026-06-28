package storage

// TableUsage 是单类数据的占用统计。
// SQLite 没有便捷的"每表精确字节数"，SizeBytes 用总文件大小按行数占比估算。
type TableUsage struct {
	TableName string `json:"tableName"`
	RowCount  int64  `json:"rowCount"`
	SizeBytes int64  `json:"sizeBytes"`
	Estimated bool   `json:"estimated"`
}

// Usage 汇总各表行数，并按行数占比把数据库文件大小估算到每张表。
func (s *Store) Usage() []TableUsage {
	usages := make([]TableUsage, 0, len(AllModels))
	var totalRows int64
	for _, model := range AllModels {
		var n int64
		s.db().Model(model.Model).Count(&n)
		usages = append(usages, TableUsage{TableName: model.TableName, RowCount: n, Estimated: true})
		totalRows += n
	}

	// 用数据库文件大小（含已分配页）按行数占比分摊到每张表。
	fileSize, _ := FileSize(s.path)
	if totalRows > 0 && fileSize > 0 {
		for i := range usages {
			usages[i].SizeBytes = fileSize * usages[i].RowCount / totalRows
		}
	}
	return usages
}
