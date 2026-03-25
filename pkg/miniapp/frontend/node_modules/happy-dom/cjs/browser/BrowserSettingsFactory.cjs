"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const DefaultBrowserSettings_js_1 = __importDefault(require("./DefaultBrowserSettings.cjs"));
/**
 * Browser settings utility.
 */
class BrowserSettingsFactory {
    /**
     * Returns browser settings.
     *
     * @param [settings] Browser settings.
     * @param [freezeObject] "true" to freeze the object.
     * @returns Settings.
     */
    static createSettings(settings) {
        return {
            ...DefaultBrowserSettings_js_1.default,
            ...settings,
            navigation: {
                ...DefaultBrowserSettings_js_1.default.navigation,
                ...settings?.navigation
            },
            navigator: {
                ...DefaultBrowserSettings_js_1.default.navigator,
                ...settings?.navigator
            },
            timer: {
                ...DefaultBrowserSettings_js_1.default.timer,
                ...settings?.timer
            },
            fetch: {
                ...DefaultBrowserSettings_js_1.default.fetch,
                ...settings?.fetch
            },
            device: {
                ...DefaultBrowserSettings_js_1.default.device,
                ...settings?.device
            }
        };
    }
}
exports.default = BrowserSettingsFactory;
//# sourceMappingURL=BrowserSettingsFactory.cjs.map