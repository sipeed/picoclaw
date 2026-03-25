"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const BrowserNavigationCrossOriginPolicyEnum_js_1 = __importDefault(require("../enums/BrowserNavigationCrossOriginPolicyEnum.cjs"));
const DetachedBrowserFrame_js_1 = __importDefault(require("../detached-browser/DetachedBrowserFrame.cjs"));
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
/**
 * Browser frame validator.
 */
class BrowserFrameValidator {
    /**
     * Returns true if the frame navigation complies with the cross origin policy.
     *
     * @param frame Frame.
     * @param toURL URL.
     * @returns True if the frame navigation complies with the cross origin policy.
     */
    static validateCrossOriginPolicy(frame, toURL) {
        const settings = frame.page.context.browser.settings;
        let fromURL = frame.page.mainFrame.window.location;
        if (frame[PropertySymbol.openerFrame]) {
            fromURL = frame[PropertySymbol.openerFrame].window.location;
        }
        else if (frame.parentFrame) {
            fromURL = frame.parentFrame.window.location;
        }
        if (settings.navigation.crossOriginPolicy === BrowserNavigationCrossOriginPolicyEnum_js_1.default.sameOrigin &&
            fromURL.protocol !== 'about:' &&
            toURL.protocol !== 'about:' &&
            toURL.protocol !== 'javascript:' &&
            fromURL.origin !== toURL.origin) {
            return false;
        }
        if (settings.navigation.crossOriginPolicy ===
            BrowserNavigationCrossOriginPolicyEnum_js_1.default.strictOrigin &&
            fromURL.protocol === 'https:' &&
            toURL.protocol === 'http:') {
            return false;
        }
        return true;
    }
    /**
     * Returns true if navigation is allowed for the frame.
     *
     * @param frame Frame.
     * @returns True if navigation is allowed for the frame.
     */
    static validateFrameNavigation(frame) {
        const settings = frame.page.context.browser.settings;
        // When using the Window instance directly and not via the Browser API we should not navigate the browser frame.
        if (frame instanceof DetachedBrowserFrame_js_1.default &&
            frame.page.context === frame.page.context.browser.defaultContext &&
            frame.page.context.pages[0] === frame.page &&
            frame.page.mainFrame === frame) {
            return false;
        }
        if (settings.navigation.disableMainFrameNavigation && frame.page.mainFrame === frame) {
            return false;
        }
        if (settings.navigation.disableChildFrameNavigation && frame.page.mainFrame !== frame) {
            return false;
        }
        if (settings.navigation.disableChildPageNavigation && !!frame[PropertySymbol.openerFrame]) {
            return false;
        }
        return true;
    }
}
exports.default = BrowserFrameValidator;
//# sourceMappingURL=BrowserFrameValidator.cjs.map