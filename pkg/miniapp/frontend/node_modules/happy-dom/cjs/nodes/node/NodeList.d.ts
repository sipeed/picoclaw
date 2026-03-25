import * as PropertySymbol from '../../PropertySymbol.cjs';
import Node from './Node.cjs';
/**
 * NodeList.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/NodeList
 */
declare class NodeList<T extends Node> {
    [index: number]: T;
    [PropertySymbol.items]: T[];
    /**
     * Constructor.
     *
     * @param illegalConstructorSymbol Illegal constructor symbol.
     * @param items Items.
     */
    constructor(illegalConstructorSymbol: symbol, items: T[]);
    /**
     * Returns length.
     *
     * @returns Length.
     */
    get length(): number;
    /**
     * Returns `Symbol.toStringTag`.
     *
     * @returns `Symbol.toStringTag`.
     */
    get [Symbol.toStringTag](): string;
    /**
     * Returns `[object NodeList]`.
     *
     * @returns `[object NodeList]`.
     */
    toLocaleString(): string;
    /**
     * Returns `[object NodeList]`.
     *
     * @returns `[object NodeList]`.
     */
    toString(): string;
    /**
     * Returns item by index.
     *
     * @param index Index.
     */
    item(index: number): T;
    /**
     * Returns an iterator, allowing you to go through all values of the key/value pairs contained in this object.
     *
     * @returns Iterator.
     */
    [Symbol.iterator](): IterableIterator<T>;
    /**
     * Returns an iterator, allowing you to go through all values of the key/value pairs contained in this object.
     *
     * @returns Iterator.
     */
    values(): IterableIterator<T>;
    /**
     * Returns an iterator, allowing you to go through all key/value pairs contained in this object.
     *
     * @returns Iterator.
     */
    entries(): IterableIterator<[number, T]>;
    /**
     * Executes a provided callback function once for each DOMTokenList element.
     *
     * @param callback Function.
     * @param thisArg thisArg.
     */
    forEach(callback: (currentValue: any, currentIndex: any, listObj: any) => void, thisArg?: this): void;
    /**
     * Returns an iterator, allowing you to go through all keys of the key/value pairs contained in this object.
     *
     * @returns Iterator.
     */
    keys(): IterableIterator<number>;
}
export default NodeList;
//# sourceMappingURL=NodeList.d.ts.map