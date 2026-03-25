import SVGElement from '../svg-element/SVGElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedString from '../../svg/SVGAnimatedString.js';
import SVGAnimatedLength from '../../svg/SVGAnimatedLength.js';
import SVGAnimatedPreserveAspectRatio from '../../svg/SVGAnimatedPreserveAspectRatio.js';
/**
 * SVGFEImageElement.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGFEImageElement
 */
export default class SVGFEImageElement extends SVGElement {
    [PropertySymbol.crossOrigin]: SVGAnimatedString | null;
    [PropertySymbol.height]: SVGAnimatedLength | null;
    [PropertySymbol.href]: SVGAnimatedString | null;
    [PropertySymbol.preserveAspectRatio]: SVGAnimatedPreserveAspectRatio | null;
    [PropertySymbol.result]: SVGAnimatedString | null;
    [PropertySymbol.width]: SVGAnimatedLength | null;
    [PropertySymbol.x]: SVGAnimatedLength | null;
    [PropertySymbol.y]: SVGAnimatedLength | null;
    /**
     * Returns "crossorigin" attribute.
     *
     * @returns Cross origin.
     */
    get crossOrigin(): string;
    /**
     * Sets "crossorigin" attribute.
     *
     * @param value Cross origin.
     */
    set crossOrigin(value: string);
    /**
     * Returns height.
     *
     * @returns Height.
     */
    get height(): SVGAnimatedLength;
    /**
     * Returns href.
     *
     * @returns Href.
     */
    get href(): SVGAnimatedString;
    /**
     * Returns preserveAspectRatio.
     *
     * @returns PreserveAspectRatio.
     */
    get preserveAspectRatio(): SVGAnimatedPreserveAspectRatio;
    /**
     * Returns result.
     *
     * @returns Result.
     */
    get result(): SVGAnimatedString;
    /**
     * Returns width.
     *
     * @returns Width.
     */
    get width(): SVGAnimatedLength;
    /**
     * Returns x.
     *
     * @returns X.
     */
    get x(): SVGAnimatedLength;
    /**
     * Returns y.
     *
     * @returns Y.
     */
    get y(): SVGAnimatedLength;
}
//# sourceMappingURL=SVGFEImageElement.d.ts.map