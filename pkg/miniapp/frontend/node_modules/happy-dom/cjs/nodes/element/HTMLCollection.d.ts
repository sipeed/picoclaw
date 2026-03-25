import * as PropertySymbol from '../../PropertySymbol.cjs';
import Element from './Element.cjs';
/**
 * HTMLCollection.
 *
 * We are extending Array here to improve performance.
 * However, we should not expose Array methods to the outside.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLCollection
 */
export default class HTMLCollection<T extends Element, NamedItem = T> {
    [index: number]: T;
    protected [PropertySymbol.query]: () => T[];
    /**
     * Constructor.
     *
     * @param illegalConstructorSymbol Illegal constructor symbol.
     * @param query Query function.
     */
    constructor(illegalConstructorSymbol: symbol, query: () => T[]);
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
     * Returns `[object HTMLCollection]`.
     *
     * @returns `[object HTMLCollection]`.
     */
    toLocaleString(): string;
    /**
     * Returns `[object HTMLCollection]`.
     *
     * @returns `[object HTMLCollection]`.
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
     * Returns named item.
     *
     * @param name Name.
     * @returns Node.
     */
    namedItem(name: string): NamedItem | null;
}
//# sourceMappingURL=HTMLCollection.d.ts.map