import DOMRect from './DOMRect.js';
/**
 * DOM Rect List.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/DOMRectList
 */
export default class DOMRectList extends Array<DOMRect> {
    /**
     * Constructor.
     *
     * @param illegalConstructorSymbol Illegal constructor symbol.
     */
    constructor(illegalConstructorSymbol: symbol);
    /**
     * Returns item by index.
     *
     * @param index Index.
     */
    item(index: number): DOMRect;
}
//# sourceMappingURL=DOMRectList.d.ts.map