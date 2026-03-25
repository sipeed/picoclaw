import EventTarget from '../../event/EventTarget.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import ClassMethodBinder from '../../utilities/ClassMethodBinder.js';
/**
 * TextTrackList.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/TextTrackList
 */
export default class TextTrackList extends EventTarget {
    // Internal properties
    [PropertySymbol.items] = [];
    // Events
    onaddtrack = null;
    onchange = null;
    onremovetrack = null;
    /**
     * Constructor.
     *
     * @param illegalConstructorSymbol Illegal constructor symbol.
     * @param items Items.
     */
    constructor(illegalConstructorSymbol, items) {
        super();
        if (illegalConstructorSymbol !== PropertySymbol.illegalConstructor) {
            throw new TypeError('Illegal constructor');
        }
        this[PropertySymbol.items] = items;
        const methodBinder = new ClassMethodBinder(this, [TextTrackList, EventTarget]);
        return new Proxy(this, {
            get: (target, property) => {
                if (property === 'length') {
                    return items.length;
                }
                if (property in target || typeof property === 'symbol') {
                    methodBinder.bind(property);
                    return target[property];
                }
                const index = Number(property);
                if (!isNaN(index)) {
                    return items[index];
                }
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
                return Object.keys(items);
            },
            has(target, property) {
                if (property in target) {
                    return true;
                }
                if (typeof property === 'symbol') {
                    return false;
                }
                const index = Number(property);
                return !isNaN(index) && index >= 0 && index < items.length;
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
                const index = Number(property);
                if (!isNaN(index) && items[index]) {
                    return {
                        value: items[index],
                        writable: false,
                        enumerable: true,
                        configurable: true
                    };
                }
            }
        });
    }
    /**
     * Returns the number of TextTrack objects in the TextTrackList.
     *
     * @returns Number of TextTrack objects.
     */
    get length() {
        return this[PropertySymbol.items].length;
    }
    /**
     * Returns `Symbol.toStringTag`.
     *
     * @returns `Symbol.toStringTag`.
     */
    get [Symbol.toStringTag]() {
        return 'TextTrackList';
    }
    /**
     * Returns `[object NodeList]`.
     *
     * @returns `[object NodeList]`.
     */
    toLocaleString() {
        return '[object TextTrackList]';
    }
    /**
     * Returns `[object NodeList]`.
     *
     * @returns `[object NodeList]`.
     */
    toString() {
        return '[object TextTrackList]';
    }
    /**
     * Returns an iterator, allowing you to go through all values of the key/value pairs contained in this object.
     *
     * @returns Iterator.
     */
    [Symbol.iterator]() {
        const items = this[PropertySymbol.items];
        return items[Symbol.iterator]();
    }
    /**
     * Returns the TextTrack found within the TextTrackList whose id matches the specified string. If no match is found, null is returned.
     *
     * @param id Text track cue identifier.
     * @returns TextTrack.
     */
    getTrackById(id) {
        for (const cue of this[PropertySymbol.items]) {
            if (cue.id === id) {
                return cue;
            }
        }
        return null;
    }
}
//# sourceMappingURL=TextTrackList.js.map