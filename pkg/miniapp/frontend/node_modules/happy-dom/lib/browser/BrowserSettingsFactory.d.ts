import IBrowserSettings from './types/IBrowserSettings.js';
import IOptionalBrowserSettings from './types/IOptionalBrowserSettings.js';
/**
 * Browser settings utility.
 */
export default class BrowserSettingsFactory {
    /**
     * Returns browser settings.
     *
     * @param [settings] Browser settings.
     * @param [freezeObject] "true" to freeze the object.
     * @returns Settings.
     */
    static createSettings(settings?: IOptionalBrowserSettings): IBrowserSettings;
}
//# sourceMappingURL=BrowserSettingsFactory.d.ts.map