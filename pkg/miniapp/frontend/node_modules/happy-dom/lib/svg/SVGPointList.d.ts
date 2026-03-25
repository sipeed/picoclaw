import * as PropertySymbol from '../PropertySymbol.js';
import BrowserWindow from '../window/BrowserWindow.js';
import SVGPoint from './SVGPoint.js';
/**
 * SVGPointList.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGPointList
 */
export default class SVGPointList {
    [index: number]: SVGPoint;
    [PropertySymbol.window]: BrowserWindow;
    [PropertySymbol.getAttribute]: (() => string | null) | null;
    [PropertySymbol.setAttribute]: ((value: string) => void) | null;
    [PropertySymbol.readOnly]: boolean;
    private [PropertySymbol.cache];
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
    constructor(illegalConstructorSymbol: symbol, window: BrowserWindow, options: {
        readOnly?: boolean;
        getAttribute: () => string | null;
        setAttribute: (value: string) => void;
    });
    /**
     * Returns length.
     *
     * @returns Length.
     */
    get length(): number;
    /**
     * Returns length.
     *
     * @returns Length.
     */
    get numberOfItems(): number;
    /**
     * Returns an iterator, allowing you to go through all values of the key/value pairs contained in this object.
     */
    [Symbol.iterator](): IterableIterator<SVGPoint>;
    /**
     * Clears all items from the list.
     */
    clear(): void;
    /**
     * Replace Token.
     *
     * @param newItem New item.
     * @returns The item being replaced.
     */
    initialize(newItem: SVGPoint): SVGPoint;
    /**
     * Returns item at index.
     *
     * @param index Index.
     * @returns The item at the index.
     **/
    getItem(index: number | string): SVGPoint;
    /**
     * Inserts a new item into the list at the specified position. The first item is number 0. If newItem is already in a list, it is removed from its previous list before it is inserted into this list. The inserted item is the item itself and not a copy. If the item is already in this list, note that the index of the item to insert before is before the removal of the item. If the index is equal to 0, then the new item is inserted at the front of the list. If the index is greater than or equal to numberOfItems, then the new item is appended to the end of the list.
     *
     * @param newItem The item to insert into the list.
     * @param index Index.
     * @returns The item being inserted.
     */
    insertItemBefore(newItem: SVGPoint, index: number): SVGPoint;
    /**
     * Replaces an existing item in the list with a new item. If newItem is already in a list, it is removed from its previous list before it is inserted into this list. The inserted item is the item itself and not a copy. If the item is already in this list, note that the index of the item to replace is before the removal of the item.
     *
     * @param newItem The item to insert into the list.
     * @param index Index.
     * @returns The item being replaced.
     */
    replaceItem(newItem: SVGPoint, index: number): SVGPoint;
    /**
     * Removes an existing item from the list.
     *
     * @param index Index.
     * @returns The removed item.
     */
    removeItem(index: number): SVGPoint;
    /**
     * Appends an item to the end of the list.
     *
     * @param newItem The item to add to the list.
     * @returns The item being appended.
     */
    appendItem(newItem: SVGPoint): SVGPoint;
    /**
     * Returns item list from attribute value.
     *
     * @see https://infra.spec.whatwg.org/#split-on-ascii-whitespace
     */
    [PropertySymbol.getItemList](): SVGPoint[];
}
//# sourceMappingURL=SVGPointList.d.ts.map