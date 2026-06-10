package storage

// ModelDescriptor 是模型在持久化层的"元数据"包装：
// 附带表名 / i18n 标签 key / 是否允许"手动清空"。
//
// 这样数据存储设置页可以列出每个表的占用情况、并对"非敏感、可重建"的表
// 提供一键清空。新增表时遵循同样规范，i18n 键名由开发者填，UI 自动展示。
type ModelDescriptor struct {
	// Model 是 GORM 模型实例的指针（&Preference{}），用于 AutoMigrate。
	Model any
	// TableName 实际 SQLite 表名，便于前端 i18n 和统计展示。
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

// AllModels 是 GORM AutoMigrate 与设置页统计的注册中心。
// 加新表时在 storage 包内定义结构体，并把对应 ModelDescriptor 追加到这里。
// 初始值在 init() 中组装，completion 版本通过 appendCompletionModels() 追加额外模型。
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

func init() {
	appendCompletionModels()
}

// Preference 按键存储行为偏好。
// Value 存 JSON 字符串，让前端可以直接放任意可序列化值。
//
// Key 作主键；UpdatedAt 为毫秒时间戳，由 Store 手动维护
// （关闭 GORM 的 autoUpdateTime，避免被改写成秒级）。
type Preference struct {
	Key       string `gorm:"primaryKey;size:191" json:"key"`
	Value     string `gorm:"type:text" json:"value"`
	UpdatedAt int64  `gorm:"autoUpdateTime:false" json:"updatedAt"`
}

func (Preference) TableName() string { return "preferences" }

// AppSettings 保存"必有且唯一"的应用级配置。和 Preference 的区别：
//   - 这里的字段是已知有限集
//   - Preference 是开放键值，业务方可以随时塞新键
//
// 始终只有 ID=1 一行；UpdatedAt 毫秒由 Store 手动维护。
type AppSettings struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	ThemeChoice  string `gorm:"size:64" json:"themeChoice"`
	CustomTheme  string `gorm:"type:text" json:"customTheme"`
	LocaleChoice string `gorm:"size:32" json:"localeChoice"`
	UpdatedAt    int64  `gorm:"autoUpdateTime:false" json:"updatedAt"`
}

func (AppSettings) TableName() string { return "app_settings" }

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
