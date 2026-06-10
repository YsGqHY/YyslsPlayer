import { currentFlavor, type Flavor } from '@/shared/featureFlags';

export interface UseHomePageResult {
  footerKey: 'home.footer.windowsOnly';
  /** 当前构建版本：lite（轻量版）或 completion（完整版）。 */
  flavor: Flavor;
}

export const useHomePage = (): UseHomePageResult => ({
  footerKey: 'home.footer.windowsOnly',
  flavor: currentFlavor(),
});
