import ClassMethodBinder from '../../utilities/ClassMethodBinder.js';
import * as PropertySymbol from '../../PropertySymbol.js';
/**
 * HTMLCollection.
 *
 * We are extending Array here to improve performance.
 * However, we should not expose Array methods to the outside.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLCollection
 */
export default class HTMLCollection {
    [PropertySymbol.query];
    /**
     * Constructor.
     *
     * @param illegalConstructorSymbol Illegal constructor symbol.
     * @param query Query function.
     */
    constructor(illegalConstructorSymbol, query) {
        if (illegalConstructorSymbol !== PropertySymbol.illegalConstructor) {
            throw new TypeError('Illegal constructor');
        }
        this[PropertySymbol.query] = query;
        const methodBinder = new ClassMethodBinder(this, this.constructor !== HTMLCollection ? [this.constructor, HTMLCollection] : [HTMLCollection]);
        return new Proxy(this, {
            get: (target, property) => {
                if (property === 'length') {
                    return query().length;
                }
                if (property in target || typeof property === 'symbol') {
                    methodBinder.bind(property);
                    return target[property];
                }
                const index = Number(property);
                if (!isNaN(index)) {
                    return query()[index];
                }
                return target.namedItem(property) || undefined;
            },
            set(target, property, newValue) {
                methodBinder.bind(property);
                if (typeof property === 'symbol') {
                    target[property] = newValue;
                    return true;
                }
                const index = Number(property);
                if (isNaN(index)) {
                    target[property] = newValue;
                }
                return true;
            },
            deleteProperty(target, property) {
                if (typeof property === 'symbol') {
                    delete target[property];
                    return true;
                }
                const index = Number(property);
                if (isNaN(index)) {
                    delete target[property];
                }
                return true;
            },
            ownKeys() {
                const keys = [];
                const items = query();
                for (let i = 0; i < items.length; i++) {
                    const item = items[i];
                    const name = item.getAttribute('id') || item.getAttribute('name');
                    keys.push(String(i));
                    if (name) {
                        keys.push(name);
                    }
                }
                return keys;
            },
            has(target, property) {
                if (property in target) {
                    return true;
                }
                const items = query();
                const index = Number(property);
                if (!isNaN(index) && index >= 0 && index < items.length) {
                    return true;
                }
                property = String(property);
                for (let i = 0; i < items.length; i++) {
                    const item = items[i];
                    const name = item.getAttribute('id') || item.getAttribute('name');
                    if (name && name === property) {
                        return true;
                    }
                }
                return false;
            },
            defineProperty(target, property, descriptor) {
                methodBinder.preventBinding(property);
                if (property in target) {
                    Object.defineProperty(target, property, descriptor);
                    return true;
                }
                return false;
            },
            getOwnPropertyDescriptor(target, property) {
                if (property in target || typeof property === 'symbol') {
                    return;
                }
                const items = query();
                const index = Number(property);
                if (!isNaN(index) && index >= 0 && index < items.length) {
                    return {
                        value: items[index],
                        writable: false,
                        enumerable: true,
                        configurable: true
                    };
                }
                for (let i = 0; i < items.length; i++) {
                    const item = items[i];
                    const name = item.getAttribute('id') || item.getAttribute('name');
                    if (name && name === property) {
                        return {
                            value: item,
                            writable: false,
                            enumerable: true,
                            configurable: true
                        };
                    }
                }
            }
        });
    }
    /**
     * Returns length.
     *
     * @returns Length.
     */
    get length() {
        return this[PropertySymbol.query]().length;
    }
    /**
     * Returns `Symbol.toStringTag`.
     *
     * @returns `Symbol.toStringTag`.
     */
    get [Symbol.toStringTag]() {
        return this.constructor.name;
    }
    /**
     * Returns `[object HTMLCollection]`.
     *
     * @returns `[object HTMLCollection]`.
     */
    toLocaleString() {
        return `[object ${this.constructor.name}]`;
    }
    /**
     * Returns `[object HTMLCollection]`.
     *
     * @returns `[object HTMLCollection]`.
     */
    toString() {
        return `[object ${this.constructor.name}]`;
    }
    /**
     * Returns item by index.
     *
     * @param index Index.
     */
    item(index) {
        const items = this[PropertySymbol.query]();
        return index >= 0 && items[index] ? items[index] : null;
    }
    /**
     * Returns an iterator, allowing you to go through all values of the key/value pairs contained in this object.
     *
     * @returns Iterator.
     */
    [Symbol.iterator]() {
        const items = this[PropertySymbol.query]();
        return items[Symbol.iterator]();
    }
    /**
     * Returns named item.
     *
     * @param name Name.
     * @returns Node.
     */
    namedItem(name) {
        const items = this[PropertySymbol.query]();
        name = String(name);
        for (const item of items) {
            if (item.getAttribute('id') === name || item.getAttribute('name') === name) {
                return item;
            }
        }
        return null;
    }
}
//# sourceMappingURL=HTMLCollection.js.map