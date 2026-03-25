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
Object.defineProperty(exports, "__esModule", { value: true });
const PropertySymbol = __importStar(require("../PropertySymbol.cjs"));
/**
 * API for accessing the Browser in a Window context without exposing the Browser as accessible properties.
 *
 * The Browser should never be exposed to scripts, as the scripts could then manipulate it, which would lead to security issues.
 */
class WindowBrowserContext {
    static [PropertySymbol.browserFrames] = new Map();
    static [PropertySymbol.windowInternalId] = 0;
    #window;
    /**
     * Browser window.
     *
     * @param window Window.
     */
    constructor(window) {
        this.#window = window;
    }
    /**
     * Returns the browser settings of the window.
     *
     * @returns Browser settings.
     */
    getSettings() {
        return this.getBrowserFrame()?.page?.context?.browser?.settings || null;
    }
    /**
     * Returns the browser.
     *
     * @returns Browser.
     */
    getBrowser() {
        return this.getBrowserFrame()?.page?.context?.browser || null;
    }
    /**
     * Returns the browser page.
     *
     * @returns Browser page.
     */
    getBrowserPage() {
        return this.getBrowserFrame()?.page || null;
    }
    /**
     * Returns the browser context.
     *
     * @returns Browser context.
     */
    getBrowserContext() {
        return this.getBrowserFrame()?.page?.context || null;
    }
    /**
     * Returns the browser frame of the window.
     *
     * @returns Browser frame.
     */
    getBrowserFrame() {
        if (!this.#window) {
            return null;
        }
        return (this.constructor[PropertySymbol.browserFrames].get(this.#window[PropertySymbol.internalId]) || null);
    }
    /**
     * Returns the async task manager of the window.
     *
     * @returns Async task manager.
     */
    getAsyncTaskManager() {
        return this.getBrowserFrame()?.[PropertySymbol.asyncTaskManager] || null;
    }
    /**
     * Assigns the window to a browser frame.
     *
     * @param window Window.
     * @param browserFrame Browser frame.
     */
    static setWindowBrowserFrameRelation(window, browserFrame) {
        const browserFrames = this[PropertySymbol.browserFrames];
        if (window[PropertySymbol.internalId] === -1) {
            window[PropertySymbol.internalId] = this[PropertySymbol.windowInternalId];
            this[PropertySymbol.windowInternalId]++;
        }
        browserFrames.set(window[PropertySymbol.internalId], browserFrame);
    }
    /**
     * Assigns the window to a browser frame.
     *
     * @param window Window.
     * @param browserFrame Browser frame.
     */
    static removeWindowBrowserFrameRelation(window) {
        this[PropertySymbol.browserFrames].delete(window[PropertySymbol.internalId]);
    }
}
exports.default = WindowBrowserContext;
//# sourceMappingURL=WindowBrowserContext.cjs.map