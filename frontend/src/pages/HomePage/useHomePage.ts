export interface UseHomePageResult {
  footerKey: 'home.footer.windowsOnly';
}

// HomePage ViewModel：仅暴露结构化状态，文案由 View 层通过 i18n 解析。
export const useHomePage = (): UseHomePageResult => ({
  footerKey: 'home.footer.windowsOnly',
});
