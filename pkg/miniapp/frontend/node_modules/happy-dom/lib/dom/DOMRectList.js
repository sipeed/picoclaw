import * as PropertySymbol from '../PropertySymbol.js';
/**
 * DOM Rect List.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/DOMRectList
 */
export default class DOMRectList extends Array {
    /**
     * Constructor.
     *
     * @param illegalConstructorSymbol Illegal constructor symbol.
     */
    constructor(illegalConstructorSymbol) {
        super();
        // "illegalConstructorSymbol" can be "1" when calling the "splice()" method
        if (illegalConstructorSymbol !== 1 &&
            illegalConstructorSymbol !== PropertySymbol.illegalConstructor) {
            throw new TypeError('Illegal constructor');
        }
    }
    /**
     * Returns item by index.
     *
     * @param index Index.
     */
    item(index) {
        return this[index] ?? null;
    }
}
//# sourceMappingURL=DOMRectList.js.map