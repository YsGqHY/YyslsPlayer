let currentProjectId: number | null = null;

export const EditorSelectionService = {
  setProjectId(projectId: number): void {
    currentProjectId = projectId > 0 ? projectId : null;
  },

  getProjectId(): number | null {
    return currentProjectId;
  },

  clear(): void {
    currentProjectId = null;
  },
} as const;
