package storage

// ModelDescriptor 是模型在持久化层的"元数据"包装：
// 附带数据集合名 / i18n 标签 key / 是否允许"手动清空"。
//
// 这样数据存储设置页可以列出每个数据集合的占用情况、并对"非敏感、可重建"的集合
// 提供一键清空。新增集合时遵循同样规范，i18n 键名由开发者填，UI 自动展示。
type ModelDescriptor struct {
	// Model 是数据模型实例的指针（&Preference{}）。
	Model any
	// TableName JSON 数据集合名，便于前端 i18n 和统计展示。
	TableName string
	// LabelKey 设置页展示的 i18n 路径，文案在
	// settings.database.tables.<key>.{label,description}。
	LabelKey string
	// Clearable=true 表示允许在设置页手动清空。
	// 涉及"用户输入" / "敏感凭据" / "无法重建的关键状态"的集合设为 false。
	// 当前核心集合的判断：
	//   - preferences   = true：UI 行为开关，重置即默认
	//   - app_settings  = false：主题选择 / 自定义主题 / 语言，"全清"会重置用户视觉设置，应让用户走具体的"重置"动作
	Clearable bool
}

// AllModels 是 JSON 数据集合与设置页统计的注册中心。
// 加新集合时在 storage 包内定义结构体，并把对应 ModelDescriptor 追加到这里。
var AllModels = []ModelDescriptor{
	{
		Model:     &Preference{},
		TableName: "preferences",
		LabelKey:  "preferences",
		Clearable: true,
	},
	{
		Model:     &AppSettings{},
		TableName: "app_settings",
		LabelKey:  "appSettings",
		Clearable: false,
	},
	{
		Model:     &MidiProject{},
		TableName: MidiProjectsTable,
		LabelKey:  "midiProjects",
		Clearable: false,
	},
	{
		Model:     &MidiEvent{},
		TableName: MidiEventsTable,
		LabelKey:  "midiEvents",
		Clearable: false,
	},
	{
		Model:     &MidiProfile{},
		TableName: MidiProfilesTable,
		LabelKey:  "midiProfiles",
		Clearable: false,
	},
	{
		Model:     &Keymap36{},
		TableName: Keymap36Table,
		LabelKey:  "keymap36",
		Clearable: false,
	},
	{
		Model:     &PlayHistory{},
		TableName: PlayHistoryTable,
		LabelKey:  "playHistory",
		Clearable: true,
	},
	{
		Model:     &HotkeyBinding{},
		TableName: HotkeyBindingsTable,
		LabelKey:  "hotkeyBindings",
		Clearable: false,
	},
}

// Preference 按键存储行为偏好。
// Value 存 JSON 字符串，让前端可以直接放任意可序列化值。
type Preference struct {
	Key       string `json:"key"`
	Value     string `json:"value"`
	UpdatedAt int64  `json:"updatedAt"`
}

// AppSettings 保存"必有且唯一"的应用级配置。和 Preference 的区别：
//   - 这里的字段是已知有限集
//   - Preference 是开放键值，业务方可以随时塞新键
type AppSettings struct {
	ID           uint   `json:"id"`
	ThemeChoice  string `json:"themeChoice"`
	CustomTheme  string `json:"customTheme"`
	LocaleChoice string `json:"localeChoice"`
	UpdatedAt    int64  `json:"updatedAt"`
}

// FindDescriptor 按集合名查找 ModelDescriptor；找不到返回 nil。
// storagesvc.ClearTable 用它做白名单校验，避免任意集合名被清空。
func FindDescriptor(tableName string) *ModelDescriptor {
	for i := range AllModels {
		if AllModels[i].TableName == tableName {
			return &AllModels[i]
		}
	}
	return nil
}
