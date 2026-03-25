import HTMLImageElement from './HTMLImageElement.cjs';
import * as PropertySymbol from '../../PropertySymbol.cjs';
/**
 * Image as constructor.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLImageElement/Image.
 */
export default class Image extends HTMLImageElement {
    [PropertySymbol.tagName]: string;
    [PropertySymbol.localName]: string;
    [PropertySymbol.namespaceURI]: string;
    /**
     * Constructor.
     *
     * @param [width] Width.
     * @param [height] Height.
     */
    constructor(width?: number, height?: number);
}
//# sourceMappingURL=Image.d.ts.map