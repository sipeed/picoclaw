import * as PropertySymbol from '../PropertySymbol.js';
import NamespaceURI from '../config/NamespaceURI.js';
import StringUtility from '../utilities/StringUtility.js';
import CustomElementUtility from './CustomElementUtility.js';
/**
 * Custom elements registry.
 */
export default class CustomElementRegistry {
    [PropertySymbol.registry] = new Map();
    [PropertySymbol.classRegistry] = new Map();
    [PropertySymbol.callbacks] = new Map();
    [PropertySymbol.destroyed] = false;
    #window;
    /**
     * Constructor.
     *
     * @param window Window.
     */
    constructor(window) {
        if (!window) {
            throw new TypeError('Illegal constructor');
        }
        this.#window = window;
    }
    /**
     * Defines a custom element class.
     *
     * @param name Tag name of element.
     * @param elementClass Element class.
     * @param [options] Options.
     * @param [options.extends] Extends tag name.
     */
    define(name, elementClass, options) {
        if (this[PropertySymbol.destroyed]) {
            return;
        }
        if (!CustomElementUtility.isValidCustomElementName(name)) {
            throw new this.#window.DOMException(`Failed to execute 'define' on 'CustomElementRegistry': "${name}" is not a valid custom element name`);
        }
        if (this[PropertySymbol.registry].has(name)) {
            throw new this.#window.DOMException(`Failed to execute 'define' on 'CustomElementRegistry': the name "${name}" has already been used with this registry`);
        }
        if (this[PropertySymbol.classRegistry].has(elementClass)) {
            throw new this.#window.DOMException("Failed to execute 'define' on 'CustomElementRegistry': this constructor has already been used with this registry");
        }
        const tagName = StringUtility.asciiUpperCase(name);
        elementClass.prototype[PropertySymbol.window] = this.#window;
        elementClass.prototype[PropertySymbol.ownerDocument] = this.#window.document;
        elementClass.prototype[PropertySymbol.tagName] = tagName;
        elementClass.prototype[PropertySymbol.localName] = name;
        elementClass.prototype[PropertySymbol.namespaceURI] = NamespaceURI.html;
        // ObservedAttributes should only be called once by CustomElementRegistry (see #117)
        const observedAttributes = new Set();
        const elementObservervedAttributes = elementClass.observedAttributes;
        if (Array.isArray(elementObservervedAttributes)) {
            for (const attribute of elementObservervedAttributes) {
                observedAttributes.add(String(attribute).toLowerCase());
            }
        }
        this[PropertySymbol.registry].set(name, {
            elementClass,
            extends: options && options.extends ? options.extends.toLowerCase() : null,
            observedAttributes,
            livecycleCallbacks: {
                connectedCallback: elementClass.prototype.connectedCallback,
                disconnectedCallback: elementClass.prototype.disconnectedCallback,
                attributeChangedCallback: elementClass.prototype.attributeChangedCallback
            }
        });
        this[PropertySymbol.classRegistry].set(elementClass, name);
        const callbacks = this[PropertySymbol.callbacks].get(name);
        if (callbacks) {
            this[PropertySymbol.callbacks].delete(name);
            for (const callback of callbacks) {
                callback();
            }
        }
    }
    /**
     * Returns a defined element class.
     *
     * @param name Tag name of element.
     * @returns HTMLElement Class defined or undefined.
     */
    get(name) {
        return this[PropertySymbol.registry].get(name)?.elementClass;
    }
    /**
     * Upgrades a custom element directly, even before it is connected to its shadow root.
     *
     * Not implemented yet.
     *
     * @param _root Root node.
     */
    upgrade(_root) {
        // Do nothing
    }
    /**
     * When defined.
     *
     * @param name Tag name of element.
     */
    whenDefined(name) {
        if (this[PropertySymbol.destroyed]) {
            return Promise.reject(new this.#window.DOMException(`Failed to execute 'whenDefined' on 'CustomElementRegistry': The custom element registry has been destroyed.`));
        }
        if (!CustomElementUtility.isValidCustomElementName(name)) {
            return Promise.reject(new this.#window.DOMException(`Failed to execute 'whenDefined' on 'CustomElementRegistry': Invalid custom element name: "${name}"`));
        }
        if (this.get(name)) {
            return Promise.resolve();
        }
        return new Promise((resolve) => {
            const callbacks = this[PropertySymbol.callbacks].get(name);
            if (callbacks) {
                callbacks.push(resolve);
            }
            else {
                this[PropertySymbol.callbacks].set(name, [resolve]);
            }
        });
    }
    /**
     * Reverse lookup searching for name by given element class.
     *
     * @param elementClass Class constructor.
     * @returns Found tag name or `null`.
     */
    getName(elementClass) {
        return this[PropertySymbol.classRegistry].get(elementClass) || null;
    }
    /**
     * Destroys the registry.
     */
    [PropertySymbol.destroy]() {
        this[PropertySymbol.destroyed] = true;
        for (const definition of this[PropertySymbol.registry].values()) {
            definition.elementClass.prototype[PropertySymbol.window] = null;
            definition.elementClass.prototype[PropertySymbol.ownerDocument] = null;
            definition.elementClass.prototype[PropertySymbol.tagName] = null;
            definition.elementClass.prototype[PropertySymbol.localName] = null;
            definition.elementClass.prototype[PropertySymbol.namespaceURI] = null;
        }
        this[PropertySymbol.registry] = new Map();
        this[PropertySymbol.classRegistry] = new Map();
        this[PropertySymbol.callbacks] = new Map();
    }
}
//# sourceMappingURL=CustomElementRegistry.js.map