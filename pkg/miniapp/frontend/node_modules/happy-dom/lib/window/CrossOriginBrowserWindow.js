import EventTarget from '../event/EventTarget.js';
import DOMException from '../exception/DOMException.js';
import DOMExceptionNameEnum from '../exception/DOMExceptionNameEnum.js';
import * as PropertySymbol from '../PropertySymbol.js';
/**
 * Browser window with limited access due to CORS restrictions in iframes.
 */
export default class CrossOriginBrowserWindow extends EventTarget {
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
                throw new DOMException(`Blocked a frame with origin "${this.parent.location.origin}" from accessing a cross-origin frame.`, DOMExceptionNameEnum.securityError);
            },
            set: () => {
                throw new DOMException(`Blocked a frame with origin "${this.parent.location.origin}" from accessing a cross-origin frame.`, DOMExceptionNameEnum.securityError);
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
//# sourceMappingURL=CrossOriginBrowserWindow.js.map