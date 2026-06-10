/// <reference types="vite/client" />

interface ImportMetaEnv {
  /** 构建版本：'lite' | 'completion' */
  readonly VITE_FLAVOR: string
  /** 是否生产构建 */
  readonly PRODUCTION?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
