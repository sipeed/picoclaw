import Element from '../nodes/element/Element.cjs';
/**
 * Dataset factory.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLElement/dataset
 */
export default class DOMStringMap {
    [key: string]: string;
    /**
     * Constructor.
     *
     * @param illegalConstructorSymbol Illegal constructor symbol.
     * @param element Element.
     */
    constructor(illegalConstructorSymbol: symbol, element: Element);
}
//# sourceMappingURL=DOMStringMap.d.ts.map