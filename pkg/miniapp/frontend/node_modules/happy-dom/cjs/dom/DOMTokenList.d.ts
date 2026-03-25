import Element from '../nodes/element/Element.cjs';
import * as PropertySymbol from '../PropertySymbol.cjs';
/**
 * DOM Token List.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/DOMTokenList.
 */
export default class DOMTokenList {
    [index: number]: string;
    private [PropertySymbol.ownerElement];
    private [PropertySymbol.attributeName];
    private [PropertySymbol.cache];
    /**
     * Constructor.
     *
     * @param illegalConstructorSymbol Illegal constructor symbol.
     * @param ownerElement Owner element.
     * @param attributeName Attribute name.
     */
    constructor(illegalConstructorSymbol: symbol, ownerElement: Element, attributeName: string);
    /**
     * Returns length.
     *
     * @returns Length.
     */
    get length(): number;
    /**
     * Set value.
     *
     * @param value Value.
     */
    set value(value: string);
    /**
     * Get value.
     */
    get value(): string;
    /**
     * Returns an iterator, allowing you to go through all values of the key/value pairs contained in this object.
     */
    [Symbol.iterator](): IterableIterator<string>;
    /**
     * Get ClassName.
     *
     * @param index Index.
     * */
    item(index: number | string): string;
    /**
     * Replace Token.
     *
     * @param token Token.
     * @param newToken NewToken.
     */
    replace(token: string, newToken: string): boolean;
    /**
     * Supports.
     *
     * @param _token Token.
     */
    supports(_token: string): boolean;
    /**
     * Returns an iterator, allowing you to go through all values of the key/value pairs contained in this object.
     */
    values(): IterableIterator<string>;
    /**
     * Returns an iterator, allowing you to go through all key/value pairs contained in this object.
     */
    entries(): IterableIterator<[number, string]>;
    /**
     * Executes a provided callback function once for each DOMTokenList element.
     *
     * @param callback
     * @param thisArg
     */
    forEach(callback: (currentValue: any, currentIndex: any, listObj: any) => void, thisArg?: this): void;
    /**
     * Returns an iterator, allowing you to go through all keys of the key/value pairs contained in this object.
     *
     */
    keys(): IterableIterator<number>;
    /**
     * Adds tokens.
     *
     * @param tokens Tokens.
     */
    add(...tokens: string[]): void;
    /**
     * Removes tokens.
     *
     * @param tokens Tokens.
     */
    remove(...tokens: string[]): void;
    /**
     * Check if the list contains a class.
     *
     * @param className Class name.
     * @returns TRUE if it contains.
     */
    contains(className: string): boolean;
    /**
     * Toggle a class name.
     *
     * @param token A string representing the class name you want to toggle.
     * @param [force] If included, turns the toggle into a one way-only operation. If set to `false`, then class name will only be removed, but not added. If set to `true`, then class name will only be added, but not removed.
     * @returns A boolean value, `true` or `false`, indicating whether class name is in the list after the call or not.
     */
    toggle(token: string, force?: boolean): boolean;
    /**
     * Returns token list from attribute value.
     *
     * @see https://infra.spec.whatwg.org/#split-on-ascii-whitespace
     */
    [PropertySymbol.getTokenList](): string[];
    /**
     * Returns DOMTokenList value.
     */
    toString(): string;
}
//# sourceMappingURL=DOMTokenList.d.ts.map