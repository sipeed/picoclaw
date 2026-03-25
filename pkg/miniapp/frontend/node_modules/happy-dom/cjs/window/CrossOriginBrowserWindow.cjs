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
const EventTarget_js_1 = __importDefault(require("../event/EventTarget.cjs"));
const DOMException_js_1 = __importDefault(require("../exception/DOMException.cjs"));
const DOMExceptionNameEnum_js_1 = __importDefault(require("../exception/DOMExceptionNameEnum.cjs"));
const PropertySymbol = __importStar(require("../PropertySymbol.cjs"));
/**
 * Browser window with limited access due to CORS restrictions in iframes.
 */
class CrossOriginBrowserWindow extends EventTarget_js_1.default {
    window = this;
    location;
    // Internal properties
    [PropertySymbol.self] = this;
    // Private properties
    #targetWindow;
    /**
     * Constructor.
     *
     * @param target Target window.
     * @param [parent] Parent window.
     */
    constructor(target, parent) {
        super();
        this[PropertySymbol.parent] = parent ?? this;
        this[PropertySymbol.top] = parent ?? this;
        this.location = new Proxy({}, {
            get: () => {
                throw new DOMException_js_1.default(`Blocked a frame with origin "${this.parent.location.origin}" from accessing a cross-origin frame.`, DOMExceptionNameEnum_js_1.default.securityError);
            },
            set: () => {
                throw new DOMException_js_1.default(`Blocked a frame with origin "${this.parent.location.origin}" from accessing a cross-origin frame.`, DOMExceptionNameEnum_js_1.default.securityError);
            }
        });
        this.#targetWindow = target;
    }
    /**
     * Returns self.
     *
     * @returns Self.
     */
    get self() {
        return this[PropertySymbol.self];
    }
    /**
     * Returns self.
     *
     * @param self Self.
     */
    set self(self) {
        this[PropertySymbol.self] = self;
    }
    /**
     * Returns top.
     *
     * @returns Top.
     */
    get top() {
        return this[PropertySymbol.top];
    }
    /**
     * Returns parent.
     *
     * @returns Parent.
     */
    get parent() {
        return this[PropertySymbol.parent];
    }
    /**
     * Returns parent.
     *
     * @param parent Parent.
     */
    set parent(parent) {
        this[PropertySymbol.parent] = parent;
    }
    /**
     * Returns the opener.
     *
     * @returns Opener.
     */
    get opener() {
        return this.#targetWindow.opener;
    }
    /**
     * Returns the closed state.
     *
     * @returns Closed state.
     */
    get closed() {
        return this.#targetWindow.closed;
    }
    /**
     * Shifts focus away from the window.
     */
    blur() {
        this.#targetWindow.blur();
    }
    /**
     * Gives focus to the window.
     */
    focus() {
        this.#targetWindow.focus();
    }
    /**
     * Closes the window.
     */
    close() {
        this.#targetWindow.close();
    }
    /**
     * Safely enables cross-origin communication between Window objects; e.g., between a page and a pop-up that it spawned, or between a page and an iframe embedded within it.
     *
     * @param message Message.
     * @param [targetOrigin=*] Target origin.
     * @param transfer Transfer. Not implemented.
     */
    postMessage(message, targetOrigin = '*', transfer) {
        this.#targetWindow.postMessage(message, targetOrigin, transfer);
    }
}
exports.default = CrossOriginBrowserWindow;
//# sourceMappingURL=CrossOriginBrowserWindow.cjs.map