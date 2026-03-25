"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
/**
 * Mutation Observer Listener.
 */
class MutationObserverListener {
    target;
    options;
    mutationListener;
    #window;
    #observer;
    #callback;
    #records = [];
    #timeout = null;
    /**
     * Constructor.
     *
     * @param init Options.
     * @param init.window Window.
     * @param init.options Options.
     * @param init.target Target.
     * @param init.observer Observer.
     * @param init.callback Callback.
     */
    constructor(init) {
        this.options = init.options;
        this.target = init.target;
        this.mutationListener = {
            options: init.options,
            callback: new WeakRef((record) => this.report(record))
        };
        this.#window = init.window;
        this.#observer = init.observer;
        this.#callback = init.callback;
    }
    /**
     * Reports mutations.
     *
     * @param record Record.
     */
    report(record) {
        if (!this.#records) {
            return;
        }
        this.#records.push(record);
        if (this.#timeout) {
            this.#window.clearTimeout(this.#timeout);
        }
        this.#timeout = this.#window.setTimeout(() => {
            const records = this.#records;
            if (records?.length > 0) {
                this.#records = [];
                this.#callback(records, this.#observer);
            }
        });
    }
    /**
     * Destroys the listener.
     */
    takeRecords() {
        if (this.#timeout) {
            this.#window.clearTimeout(this.#timeout);
        }
        const records = this.#records;
        this.#records = [];
        return records;
    }
    /**
     * Destroys the listener.
     */
    destroy() {
        if (this.#timeout) {
            this.#window.clearTimeout(this.#timeout);
        }
        this.options = null;
        this.target = null;
        this.mutationListener = null;
        this.#window = null;
        this.#observer = null;
        this.#callback = null;
        this.#timeout = null;
        this.#records = null;
    }
}
exports.default = MutationObserverListener;
//# sourceMappingURL=MutationObserverListener.cjs.map