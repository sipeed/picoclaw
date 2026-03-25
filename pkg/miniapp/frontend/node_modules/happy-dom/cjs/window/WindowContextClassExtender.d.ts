import BrowserWindow from './BrowserWindow.cjs';
/**
 * Extends classes with a "window" property, so that they internally can access it's Window context.
 *
 * By using WindowBrowserContext, the classes can get access to their Browser context, for accessing settings or navigating the browser.
 */
export default class WindowContextClassExtender {
    /**
     * Extends classes with a "window" property.
     *
     * @param window Window.
     */
    static extendClasses(window: BrowserWindow): void;
}
//# sourceMappingURL=WindowContextClassExtender.d.ts.map