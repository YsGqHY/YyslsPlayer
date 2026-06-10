// StorageService：数据文件路径 / 占用空间 / 集合级统计与清空。
//
// 切换路径是后端的"原子操作"——前端只需调用，完成后正常恢复。
// 失败时后端已回滚到旧路径，可继续使用。
import { Service as Binding } from '@bindings/YyslsPlayer/internal/services/storagesvc';

export interface StorageStats {
  path: string;
  isCustom: boolean;
  defaultPath: string;
  sizeBytes: number;
}

// 单个数据集合的展示元数据 + 占用。
// labelKey 是 i18n 键的"后缀"——前端拼成 settings.database.tables.<key>.label。
// estimated=true 表示后端返回估算占用，UI 应注明"估算"。
export interface TableInfo {
  name: string;
  labelKey: string;
  clearable: boolean;
  rowCount: number;
  sizeBytes: number;
  estimated: boolean;
}

export interface TableStats {
  totalBytes: number;
  tables: TableInfo[];
}

type RawStorageStats = Partial<{
  path: string;
  isCustom: boolean;
  defaultPath: string;
  sizeBytes: number | string;
}>;

type RawTableInfo = Partial<{
  name: string;
  labelKey: string;
  clearable: boolean;
  rowCount: number | string;
  sizeBytes: number | string;
  estimated: boolean;
}>;

type RawTableStats = Partial<{
  totalBytes: number | string;
  tables: RawTableInfo[];
}>;

export const StorageService = {
  async getStats(): Promise<StorageStats> {
    const s = await Binding.GetStats() as RawStorageStats;
    return {
      path: s.path ?? '',
      isCustom: Boolean(s.isCustom),
      defaultPath: s.defaultPath ?? '',
      sizeBytes: Number(s.sizeBytes ?? 0),
    };
  },

  async getTableStats(): Promise<TableStats> {
    const s = await Binding.GetTableStats() as RawTableStats;
    return {
      totalBytes: Number(s.totalBytes ?? 0),
      tables: (s.tables ?? []).map<TableInfo>((t) => ({
        name: t.name ?? '',
        labelKey: t.labelKey ?? '',
        clearable: Boolean(t.clearable),
        rowCount: Number(t.rowCount ?? 0),
        sizeBytes: Number(t.sizeBytes ?? 0),
        estimated: Boolean(t.estimated),
      })),
    };
  },

  // newPath 必须是 JSON 数据文件的完整路径（含文件名）。
  async setCustomPath(newPath: string): Promise<void> {
    await Binding.SetCustomPath(newPath);
  },

  async resetPath(): Promise<void> {
    await Binding.ResetPath();
  },

  async clearTable(name: string): Promise<void> {
    await Binding.ClearTable(name);
  },
} as const;
