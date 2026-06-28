// Package app 封装 Wails 应用的启动逻辑。
// main.go 只保留 embed.FS 与 Run 调用，业务装配集中在这里。
package app

import (
	"embed"
	"fmt"

	"YyslsPlayer/internal/events"
	"YyslsPlayer/internal/services/appearance"
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
	wailsevents "github.com/wailsapp/wails/v3/pkg/events"
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
	// 2) 尝试打开当前生效的数据文件；若自定义路径失效，则自动回退到默认 SQLite 数据库
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

	ksDriver := keysim.NewDefaultDriver()
	playerKeysimSvc := keysim.New(ksDriver)
	playerSvc := player.New(playerKeysimSvc)

	// 全局热键服务：OS 级快捷键控制演奏（切到游戏也生效）。
	// playback 类动作直接作用于 playerSvc，紧急松键最关键。
	hotkeySvc := hotkey.New(holder, playerSvc)

	services := []application.Service{
		application.NewService(preferences.New(holder)),
		application.NewService(appsettings.New(holder)),
		application.NewService(appearance.New()),
		application.NewService(storagesvc.New(holder, cfgMgr)),
		application.NewService(midi.New(holder)),
		application.NewService(playerSvc),
		application.NewService(hotkeySvc),
	}
	// completion 版本会追加 transcription、macro 等服务；lite 版本保持原样。
	services, transcriptionSvc, macroSvc := registerCompletionServices(services, holder, ksDriver, hotkeySvc, playerSvc)

	app := application.New(application.Options{
		Name:        "YyslsPlayer",
		Description: "燕云流音：面向《燕云十六声》36 键模式的 MIDI 导入、预览与按键模拟演奏工具",
		Services:    services,
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
	if macroSvc != nil {
		macroSvc.AttachEmitter(func(name string, payload any) {
			app.Event.Emit(name, payload)
		})
	}

	// transcription service：completion 版本需要 emitter 和生命周期管理
	if transcriptionSvc != nil {
		transcriptionSvc.SetApp(app)
		transcriptionSvc.AttachEmitter(func(name string, payload any) {
			app.Event.Emit(name, payload)
		})
		transcriptionSvc.Start()
		defer transcriptionSvc.Shutdown()
	}

	// completion 宏服务先把已启用宏同步为 hotkey 外部绑定快照；hotkey manager 启动后统一 apply。
	if macroSvc != nil {
		macroSvc.Start()
	}

	// 注册全局热键（读持久化绑定 + OS 注册 + 启动消息循环）。
	// 注册失败逐项标记，不阻断应用启动。
	if err := hotkeySvc.Start(); err != nil {
		logx.For("app").Warn("hotkey start failed", "error", err)
	}
	defer hotkeySvc.Stop()
	if macroSvc != nil {
		defer macroSvc.Stop()
	}

	window := app.Window.NewWithOptions(windowOptions())

	// 文件拖放：用户把 MIDI 文件 / 文件夹拖入窗口的列表区域时，
	// Wails 通过 WindowFilesDropped 事件把真实绝对路径送到后端（浏览器侧拿不到本地路径）。
	// 这里把路径转发为强类型前端事件，由前端 MidiService 调用 ImportPaths 完成导入。
	window.OnWindowEvent(wailsevents.Common.WindowFilesDropped, func(e *application.WindowEvent) {
		files := e.Context().DroppedFiles()
		if len(files) == 0 {
			return
		}
		app.Event.Emit(midi.EventFilesDropped, midi.FilesDroppedDTO{Paths: files})
	})

	defer logx.Close()
	return app.Run()
}
