package midi

// EventFilesDropped 在用户把文件 / 文件夹拖入窗口的 MIDI 列表区域时由后端推送给前端。
// payload 为 FilesDroppedDTO，前端据此调用 ImportPaths 完成导入。
// 与 frontend/src/shared/events.ts 中的事件名保持同步。
const EventFilesDropped = "midi:filesDropped"

// FilesDroppedDTO 携带被拖入的原始路径（可能包含文件夹与非 MIDI 文件，由后端展开与过滤）。
type FilesDroppedDTO struct {
	Paths []string `json:"paths"`
}
