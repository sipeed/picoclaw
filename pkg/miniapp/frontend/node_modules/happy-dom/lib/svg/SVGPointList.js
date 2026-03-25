import ClassMethodBinder from '../utilities/ClassMethodBinder.js';
import DOMExceptionNameEnum from '../exception/DOMExceptionNameEnum.js';
import * as PropertySymbol from '../PropertySymbol.js';
import SVGPoint from './SVGPoint.js';
const ATTRIBUTE_SEPARATOR_REGEXP = /[\t\f\n\r, ]+/;
/**
 * SVGPointList.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGPointList
 */
export default class SVGPointList {
    [PropertySymbol.window];
    [PropertySymbol.getAttribute] = null;
    [PropertySymbol.setAttribute] = null;
    [PropertySymbol.readOnly] = false;
    [PropertySymbol.cache] = {
        items: [],
        attributeValue: ''
    };
    /**
     * Constructor.
     *
     * @param illegalConstructorSymbol Illegal constructor symbol.
     * @param window Window.
     * @param options Options.
     * @param options.getAttribute Get attribute.
     * @param options.setAttribute Set attribute.
     * @param [options.readOnly] Read only.
     */
    constructor(illegalConstructorSymbol, window, options) {
        if (illegalConstructorSymbol !== PropertySymbol.illegalConstructor) {
            throw new TypeError('Illegal constructor');
        }
        this[PropertySymbol.window] = window;
        this[PropertySymbol.readOnly] = !!options.readOnly;
        this[PropertySymbol.getAttribute] = options.getAttribute || null;
        this[PropertySymbol.setAttribute] = options.setAttribute || null;
        const methodBinder = new ClassMethodBinder(this, [SVGPointList]);
        return new Proxy(this, {
            get: (target, property) => {
                if (property === 'length' || property === 'numberOfItems') {
                    return target[PropertySymbol.getItemList]().length;
                }
                if (property in target || typeof property === 'symbol') {
                    methodBinder.bind(property);
                    return target[property];
                }
                const index = Number(property);
                if (!isNaN(index)) {
                    return target[PropertySymbol.getItemList]()[index];
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
            ownKeys(target) {
                return Object.keys(target[PropertySymbol.getItemList]());
            },
            has(target, property) {
                if (property in target) {
                    return true;
                }
                if (typeof property === 'symbol') {
                    return false;
                }
                const index = Number(property);
                return !isNaN(index) && index >= 0 && index < target[PropertySymbol.getItemList]().length;
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
                const items = target[PropertySymbol.getItemList]();
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
     * Returns length.
     *
     * @returns Length.
     */
    get length() {
        return this[PropertySymbol.getItemList]().length;
    }
    /**
     * Returns length.
     *
     * @returns Length.
     */
    get numberOfItems() {
        return this[PropertySymbol.getItemList]().length;
    }
    /**
     * Returns an iterator, allowing you to go through all values of the key/value pairs contained in this object.
     */
    [Symbol.iterator]() {
        return this[PropertySymbol.getItemList]().values();
    }
    /**
     * Clears all items from the list.
     */
    clear() {
        if (this[PropertySymbol.readOnly]) {
            throw new this[PropertySymbol.window].TypeError(`Failed to execute 'clear' on 'SVGPointList': The object is read-only.`);
        }
        for (const item of this[PropertySymbol.cache].items) {
            item[PropertySymbol.getAttribute] = null;
            item[PropertySymbol.setAttribute] = null;
        }
        this[PropertySymbol.cache].items = [];
        this[PropertySymbol.cache].attributeValue = '';
        this[PropertySymbol.setAttribute]('');
    }
    /**
     * Replace Token.
     *
     * @param newItem New item.
     * @returns The item being replaced.
     */
    initialize(newItem) {
        if (this[PropertySymbol.readOnly]) {
            throw new this[PropertySymbol.window].TypeError(`Failed to execute 'initialize' on 'SVGPointList': The object is read-only.`);
        }
        if (arguments.length < 1) {
            throw new this[PropertySymbol.window].TypeError(`Failed to execute 'initialize' on 'SVGPointList': 1 arguments required, but only ${arguments.length} present.`);
        }
        if (!(newItem instanceof SVGPoint)) {
            throw new this[PropertySymbol.window].TypeError(`Failed to execute 'appendItem' on 'SVGPointList': parameter 1 is not of type 'SVGPoint'.`);
        }
        for (const item of this[PropertySymbol.cache].items) {
            item[PropertySymbol.getAttribute] = null;
            item[PropertySymbol.setAttribute] = null;
        }
        newItem[PropertySymbol.getAttribute] = () => newItem[PropertySymbol.attributeValue];
        newItem[PropertySymbol.setAttribute] = () => {
            this[PropertySymbol.cache].attributeValue = this[PropertySymbol.getItemList]()
                .map((item) => item[PropertySymbol.attributeValue] || '0 0')
                .join(' ');
            this[PropertySymbol.setAttribute](this[PropertySymbol.cache].attributeValue);
        };
        this[PropertySymbol.cache].items = [newItem];
        this[PropertySymbol.cache].attributeValue = newItem[PropertySymbol.attributeValue];
        this[PropertySymbol.setAttribute](newItem[PropertySymbol.attributeValue]);
        return newItem;
    }
    /**
     * Returns item at index.
     *
     * @param index Index.
     * @returns The item at the index.
     **/
    getItem(index) {
        const items = this[PropertySymbol.getItemList]();
        if (typeof index === 'number') {
            return items[index] ? items[index] : null;
        }
        index = Number(index);
        index = isNaN(index) ? 0 : index;
        return items[index] ? items[index] : null;
    }
    /**
     * Inserts a new item into the list at the specified position. The first item is number 0. If newItem is already in a list, it is removed from its previous list before it is inserted into this list. The inserted item is the item itself and not a copy. If the item is already in this list, note that the index of the item to insert before is before the removal of the item. If the index is equal to 0, then the new item is inserted at the front of the list. If the index is greater than or equal to numberOfItems, then the new item is appended to the end of the list.
     *
     * @param newItem The item to insert into the list.
     * @param index Index.
     * @returns The item being inserted.
     */
    insertItemBefore(newItem, index) {
        if (this[PropertySymbol.readOnly]) {
            throw new this[PropertySymbol.window].TypeError(`Failed to execute 'insertItemBefore' on 'SVGPointList': The object is read-only.`);
        }
        if (arguments.length < 2) {
            throw new this[PropertySymbol.window].TypeError(`Failed to execute 'insertItemBefore' on 'SVGPointList': 2 arguments required, but only ${arguments.length} present.`);
        }
        if (!(newItem instanceof SVGPoint)) {
            throw new this[PropertySymbol.window].TypeError(`Failed to execute 'insertItemBefore' on 'SVGPointList': parameter 1 is not of type 'SVGPoint'.`);
        }
        const items = this[PropertySymbol.getItemList]();
        const existingIndex = items.indexOf(newItem);
        if (existingIndex !== -1) {
            items.splice(existingIndex, 1);
        }
        if (index < 0) {
            index = 0;
        }
        else if (index > items.length) {
            index = items.length;
        }
        items.splice(index, 0, newItem);
        newItem[PropertySymbol.getAttribute] = () => newItem[PropertySymbol.attributeValue];
        newItem[PropertySymbol.setAttribute] = () => {
            this[PropertySymbol.cache].attributeValue = this[PropertySymbol.getItemList]()
                .map((item) => item[PropertySymbol.attributeValue] || '0 0')
                .join(' ');
            this[PropertySymbol.setAttribute](this[PropertySymbol.cache].attributeValue);
        };
        this[PropertySymbol.cache].attributeValue = items
            .map((item) => item[PropertySymbol.attributeValue] || '0 0')
            .join(' ');
        this[PropertySymbol.setAttribute](this[PropertySymbol.cache].attributeValue);
        return newItem;
    }
    /**
     * Replaces an existing item in the list with a new item. If newItem is already in a list, it is removed from its previous list before it is inserted into this list. The inserted item is the item itself and not a copy. If the item is already in this list, note that the index of the item to replace is before the removal of the item.
     *
     * @param newItem The item to insert into the list.
     * @param index Index.
     * @returns The item being replaced.
     */
    replaceItem(newItem, index) {
        if (this[PropertySymbol.readOnly]) {
            throw new this[PropertySymbol.window].TypeError(`Failed to execute 'replaceItem' on 'SVGPointList': The object is read-only.`);
        }
        if (arguments.length < 2) {
            throw new this[PropertySymbol.window].TypeError(`Failed to execute 'replaceItem' on 'SVGPointList': 2 arguments required, but only ${arguments.length} present.`);
        }
        if (!(newItem instanceof SVGPoint)) {
            throw new this[PropertySymbol.window].TypeError(`Failed to execute 'replaceItem' on 'SVGPointList': parameter 1 is not of type 'SVGPoint'.`);
        }
        const items = this[PropertySymbol.getItemList]();
        const existingIndex = items.indexOf(newItem);
        if (existingIndex === index) {
            return newItem;
        }
        if (existingIndex !== -1) {
            items.splice(existingIndex, 1);
        }
        if (index < 0) {
            index = 0;
        }
        else if (index >= items.length) {
            index = items.length - 1;
        }
        if (items[index]) {
            items[index][PropertySymbol.getAttribute] = null;
            items[index][PropertySymbol.setAttribute] = null;
        }
        const replacedItem = items[index];
        items[index] = newItem;
        newItem[PropertySymbol.getAttribute] = () => newItem[PropertySymbol.attributeValue];
        newItem[PropertySymbol.setAttribute] = () => {
            this[PropertySymbol.cache].attributeValue = this[PropertySymbol.getItemList]()
                .map((item) => item[PropertySymbol.attributeValue] || '0 0')
                .join(' ');
            this[PropertySymbol.setAttribute](this[PropertySymbol.cache].attributeValue);
        };
        this[PropertySymbol.cache].attributeValue = items
            .map((item) => item[PropertySymbol.attributeValue] || '0 0')
            .join(' ');
        this[PropertySymbol.setAttribute](this[PropertySymbol.cache].attributeValue);
        return replacedItem;
    }
    /**
     * Removes an existing item from the list.
     *
     * @param index Index.
     * @returns The removed item.
     */
    removeItem(index) {
        if (this[PropertySymbol.readOnly]) {
            throw new this[PropertySymbol.window].TypeError(`Failed to execute 'removeItem' on 'SVGPointList': The object is read-only.`);
        }
        if (arguments.length < 1) {
            throw new this[PropertySymbol.window].TypeError(`Failed to execute 'removeItem' on 'SVGPointList': 1 argument required, but only ${arguments.length} present.`);
        }
        const items = this[PropertySymbol.getItemList]();
        index = Number(index);
        if (isNaN(index)) {
            index = 0;
        }
        if (index >= items.length) {
            throw new this[PropertySymbol.window].DOMException(`Failed to execute 'removeItem' on 'SVGPointList':  The index provided (${index}) is greater than the maximum bound.`, DOMExceptionNameEnum.indexSizeError);
        }
        if (index < 0) {
            throw new this[PropertySymbol.window].DOMException(`Failed to execute 'removeItem' on 'SVGPointList':  The index provided (${index}) is negative.`, DOMExceptionNameEnum.indexSizeError);
        }
        const removedItem = items[index];
        if (removedItem) {
            removedItem[PropertySymbol.getAttribute] = null;
            removedItem[PropertySymbol.setAttribute] = null;
        }
        items.splice(index, 1);
        this[PropertySymbol.setAttribute](items.map((item) => item[PropertySymbol.attributeValue] || '0 0').join(' '));
        return removedItem;
    }
    /**
     * Appends an item to the end of the list.
     *
     * @param newItem The item to add to the list.
     * @returns The item being appended.
     */
    appendItem(newItem) {
        if (this[PropertySymbol.readOnly]) {
            throw new this[PropertySymbol.window].TypeError(`Failed to execute 'appendItem' on 'SVGPointList': The object is read-only.`);
        }
        if (arguments.length < 1) {
            throw new this[PropertySymbol.window].TypeError(`Failed to execute 'appendItem' on 'SVGPointList': 1 argument required, but only ${arguments.length} present.`);
        }
        if (!(newItem instanceof SVGPoint)) {
            throw new this[PropertySymbol.window].TypeError(`Failed to execute 'appendItem' on 'SVGPointList': parameter 1 is not of type 'SVGPoint'.`);
        }
        const items = this[PropertySymbol.getItemList]();
        const existingIndex = items.indexOf(newItem);
        if (existingIndex !== -1) {
            items.splice(existingIndex, 1);
        }
        items.push(newItem);
        newItem[PropertySymbol.getAttribute] = () => newItem[PropertySymbol.attributeValue];
        newItem[PropertySymbol.setAttribute] = () => {
            this[PropertySymbol.cache].attributeValue = this[PropertySymbol.getItemList]()
                .map((item) => item[PropertySymbol.attributeValue] || '0 0')
                .join(' ');
            this[PropertySymbol.setAttribute](this[PropertySymbol.cache].attributeValue);
        };
        this[PropertySymbol.cache].attributeValue = items
            .map((item) => item[PropertySymbol.attributeValue] || '0 0')
            .join(' ');
        this[PropertySymbol.setAttribute](this[PropertySymbol.cache].attributeValue);
        return newItem;
    }
    /**
     * Returns item list from attribute value.
     *
     * @see https://infra.spec.whatwg.org/#split-on-ascii-whitespace
     */
    [PropertySymbol.getItemList]() {
        const attributeValue = this[PropertySymbol.getAttribute]() ?? '';
        const cache = this[PropertySymbol.cache];
        if (cache.attributeValue === attributeValue) {
            return cache.items;
        }
        if (cache.items.length) {
            for (const item of cache.items) {
                item[PropertySymbol.getAttribute] = null;
                item[PropertySymbol.setAttribute] = null;
            }
        }
        // It is possible to make this statement shorter by using Array.from() and Set, but this is faster when comparing using a bench test.
        const items = [];
        const trimmed = attributeValue.trim();
        if (trimmed) {
            const parts = trimmed.split(ATTRIBUTE_SEPARATOR_REGEXP);
            for (let i = 0, max = parts.length; i < max; i += 2) {
                const x = parseFloat(parts[i]);
                const y = parts[i + 1] !== undefined ? ' ' + parseFloat(parts[i + 1]) : '';
                const item = new SVGPoint(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                    readOnly: this[PropertySymbol.readOnly],
                    getAttribute: () => item[PropertySymbol.attributeValue],
                    setAttribute: () => {
                        this[PropertySymbol.cache].attributeValue = this[PropertySymbol.getItemList]()
                            .map((item) => item[PropertySymbol.attributeValue] || '0 0')
                            .join(' ');
                        this[PropertySymbol.setAttribute](this[PropertySymbol.cache].attributeValue);
                    }
                });
                item[PropertySymbol.attributeValue] = `${x}${y}`;
                items.push(item);
            }
        }
        cache.attributeValue = attributeValue;
        cache.items = items;
        return items;
    }
}
//# sourceMappingURL=SVGPointList.js.map