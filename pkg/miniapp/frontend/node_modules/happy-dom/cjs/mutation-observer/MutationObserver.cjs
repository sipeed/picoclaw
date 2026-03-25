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
const PropertySymbol = __importStar(require("../PropertySymbol.cjs"));
const MutationObserverListener_js_1 = __importDefault(require("./MutationObserverListener.cjs"));
/**
 * The MutationObserver interface provides the ability to watch for changes being made to the DOM tree.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/MutationObserver
 */
class MutationObserver {
    #callback;
    #listeners = [];
    #destroyed = false;
    /**
     * Constructor.
     *
     * @param callback Callback.
     */
    constructor(callback) {
        if (!this[PropertySymbol.window]) {
            throw new TypeError(`Failed to construct '${this.constructor.name}': '${this.constructor.name}' was constructed outside a Window context.`);
        }
        this.#callback = callback;
    }
    /**
     * Starts observing.
     *
     * @param target Target.
     * @param options Options.
     */
    observe(target, options) {
        if (this.#destroyed) {
            return;
        }
        if (!target) {
            throw new this[PropertySymbol.window].TypeError(`Failed to execute 'observe' on 'MutationObserver': The first parameter "target" should be of type "Node".`);
        }
        if (options && (options.attributeFilter || options.attributeOldValue)) {
            if (options.attributes === undefined) {
                options = Object.assign({}, options, {
                    attributes: true,
                    attributeFilter: options.attributeFilter,
                    attributeOldValue: options.attributeOldValue
                });
            }
            if (!options.attributes && options.attributeOldValue) {
                throw new this[PropertySymbol.window].TypeError(`Failed to execute 'observe' on 'MutationObserver': The options object may only set 'attributeOldValue' to true when 'attributes' is true or not present.`);
            }
            if (!options.attributes && options.attributeFilter) {
                throw new this[PropertySymbol.window].TypeError(`Failed to execute 'observe' on 'MutationObserver': The options object may only set 'attributeFilter' when 'attributes' is true or not present.`);
            }
        }
        if (options && options.characterDataOldValue) {
            if (options.characterData === undefined) {
                options = Object.assign({}, options, {
                    characterData: true,
                    characterDataOldValue: options.characterDataOldValue
                });
            }
            if (!options.characterData && options.characterDataOldValue) {
                throw new this[PropertySymbol.window].TypeError(`Failed to execute 'observe' on 'MutationObserver': The options object may only set 'characterDataOldValue' to true when 'characterData' is true or not present.`);
            }
        }
        if (!options || (!options.childList && !options.attributes && !options.characterData)) {
            throw new this[PropertySymbol.window].TypeError(`Failed to execute 'observe' on 'MutationObserver': The options object must set at least one of 'attributes', 'characterData', or 'childList' to true.`);
        }
        // Makes sure that attribute names are lower case.
        // TODO: Is this correct?
        options = Object.assign({}, options, {
            attributeFilter: options.attributeFilter
                ? options.attributeFilter.map((name) => name.toLowerCase())
                : null
        });
        /**
         * @see https://developer.mozilla.org/en-US/docs/Web/API/MutationObserver/observe#reusing_mutationobservers
         */
        for (const listener of this.#listeners) {
            if (listener.target === target) {
                listener.options = options;
                return;
            }
        }
        const listener = new MutationObserverListener_js_1.default({
            window: this[PropertySymbol.window],
            options,
            callback: this.#callback.bind(this),
            observer: this,
            target
        });
        this.#listeners.push(listener);
        // Stores all observers on the window object, so that they can be disconnected when the window is closed.
        if (!this[PropertySymbol.window][PropertySymbol.mutationObservers].includes(this)) {
            this[PropertySymbol.window][PropertySymbol.mutationObservers].push(this);
        }
        // Starts observing target node.
        target[PropertySymbol.observeMutations](listener.mutationListener);
    }
    /**
     * Disconnects.
     */
    disconnect() {
        if (this.#listeners.length === 0) {
            return;
        }
        for (const listener of this.#listeners) {
            listener.target[PropertySymbol.unobserveMutations](listener.mutationListener);
            listener.destroy();
        }
        this.#listeners = [];
        const mutationObservers = this[PropertySymbol.window][PropertySymbol.mutationObservers];
        const index = mutationObservers.indexOf(this);
        if (index !== -1) {
            mutationObservers.splice(index, 1);
        }
    }
    /**
     * Returns a list of all matching DOM changes that have been detected but not yet processed by the observer's callback function, leaving the mutation queue empty.
     *
     * @returns Records.
     */
    takeRecords() {
        let records = [];
        for (const listener of this.#listeners) {
            records = records.concat(listener.takeRecords());
        }
        return records;
    }
    /**
     *
     */
    [PropertySymbol.destroy]() {
        this.#destroyed = true;
        this.disconnect();
    }
}
exports.default = MutationObserver;
//# sourceMappingURL=MutationObserver.cjs.map