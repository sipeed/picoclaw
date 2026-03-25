import * as PropertySymbol from '../PropertySymbol.js';
/**
 * Storage.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/Storage
 */
export default class Storage {
    [PropertySymbol.data]: {
        [key: string]: string;
    };
    /**
     * Constructor.
     */
    constructor();
    /**
     * Returns length.
     *
     * @returns Length.
     */
    get length(): number;
    /**
     * Returns name of the nth key.
     *
     * @param index Index.
     * @returns Name.
     */
    key(index: number): string | null;
    /**
     * Sets item.
     *
     * @param name Name.
     * @param item Item.
     */
    setItem(name: string, item: string): void;
    /**
     * Returns item.
     *
     * @param name Name.
     * @returns Item.
     */
    getItem(name: string): string | null;
    /**
     * Removes item.
     *
     * @param name Name.
     */
    removeItem(name: string): void;
    /**
     * Clears storage.
     */
    clear(): void;
}
//# sourceMappingURL=Storage.d.ts.map