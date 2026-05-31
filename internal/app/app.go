// Package app 封装 Wails 应用的启动逻辑。
// main.go 只保留 embed.FS 与 Run 调用，业务装配集中在这里。
package app

import (
	"embed"
	"fmt"

	"YyslsPlayer/internal/events"
	"YyslsPlayer/internal/services/appsettings"
	"YyslsPlayer/internal/services/hotkey"
	"YyslsPlayer/internal/services/keysim"
	"YyslsPlayer/internal/services/midi"
	"YyslsPlayer/internal/services/player"
	"YyslsPlayer/internal/services/preferences"
	"YyslsPlayer/internal/services/storagesvc"
	"YyslsPlayer/internal/storage"
	"YyslsPlayer/internal/utils/logx"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Run 创建 Wails 应用、注册服务、打开主窗口并启动事件循环。
// assets 由调用方（main.go）通过 //go:embed 注入。
func Run(assets embed.FS) error {
	// —— 工具层先就绪：日志 → 后续模块都能直接用 ——
	if err := logx.Init(defaultLogConfig()); err != nil {
		// 日志初始化失败不应阻断启动；降级到默认 stderr logger（logx.For 内部已兜底）
		fmt.Println("warn: logx init failed:", err)
	}

	events.Register()

	// —— 持久化层启动顺序 ——
	// 1) 读 storage.json：用户自定义路径 / 默认路径
	// 2) 尝试打开当前生效的数据文件；若自定义路径失效，则自动回退到默认 JSON 存储
	// 3) 包成 Holder，业务 service 全部走 holder.Current() 拿活跃存储
	cfgMgr, err := storage.LoadConfig()
	if err != nil {
		return fmt.Errorf("load storage config: %w", err)
	}
	db, dbPath, err := openStorageWithRecovery(cfgMgr)
	if err != nil {
		return err
	}
	holder := storage.NewHolder(db)
	logx.For("app").Info("storage opened", "path", dbPath)

	playerSvc := player.New(keysim.New(nil))

	// 全局热键服务：OS 级快捷键控制演奏（切到游戏也生效）。
	// playback 类动作直接作用于 playerSvc，紧急松键最关键。
	hotkeySvc := hotkey.New(holder, playerSvc)

	app := application.New(application.Options{
		Name:        "YyslsPlayer",
		Description: "燕云流音：面向《燕云十六声》36 键模式的 MIDI 导入、预览与按键模拟演奏工具",
		Services: []application.Service{
			application.NewService(preferences.New(holder)),
			application.NewService(appsettings.New(holder)),
			application.NewService(storagesvc.New(holder, cfgMgr)),
			application.NewService(midi.New(holder)),
			application.NewService(playerSvc),
			application.NewService(hotkeySvc),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
	})
	playerSvc.AttachEmitter(player.EventEmitterFunc(func(name string, payload any) {
		app.Event.Emit(name, payload)
	}))
	hotkeySvc.AttachEmitter(hotkey.EventEmitterFunc(func(name string, payload any) {
		app.Event.Emit(name, payload)
	}))

	// 注册全局热键（读持久化绑定 + OS 注册 + 启动消息循环）。
	// 注册失败逐项标记，不阻断应用启动。
	if err := hotkeySvc.Start(); err != nil {
		logx.For("app").Warn("hotkey start failed", "error", err)
	}
	defer hotkeySvc.Stop()

	app.Window.NewWithOptions(windowOptions())

	defer logx.Close()
	return app.Run()
}
