/**
 * 功能开关：通过 VITE_FLAVOR 环境变量在编译时决定可用功能。
 *
 * 注意：所有 import.meta.env 值在 Vite 构建时静态替换，
 * Lite 版本的构建产物中 completion 相关条件分支会被 tree-shake 移除。
 *
 * 策略：
 *   - 明确设置 VITE_FLAVOR='lite' → 按 lite 构建，tree-shake 移除扒谱功能
 *   - 未设置或设置为其他值 → 按 completion 构建（默认启用扒谱 UI，
 *     后端 GetCapability() 实际控制功能可用性）
 */
export const IS_LITE_FLAVOR = import.meta.env.VITE_FLAVOR === 'lite'
export const ENABLE_COMPLETION_FEATURES = !IS_LITE_FLAVOR
export const ENABLE_MACROS = ENABLE_COMPLETION_FEATURES

export const FEATURES = {
  /**
   * 是否为完全版（含音频转 MIDI 扒谱功能 UI）。
   *
   * 默认启用（VITE_FLAVOR 未设置时）；仅当显式设为 'lite' 时禁用。
   * 即使 FEATURES.transcription === true，后端仍可通过 GetCapability()
   * 返回 transcriptionEnabled: false 来告知前端功能实际不可用。
   */
  get transcription(): boolean {
    return ENABLE_COMPLETION_FEATURES
  },

  /** 是否启用按键宏功能。宏属于 completion/admin 版本能力，lite 构建隐藏入口。 */
  get macros(): boolean {
    return ENABLE_MACROS
  },
} as const

/**
 * 当前构建版本标识。
 */
export type Flavor = 'lite' | 'completion'

export const currentFlavor = (): Flavor => {
  return import.meta.env.VITE_FLAVOR === 'lite' ? 'lite' : 'completion'
}
