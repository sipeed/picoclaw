import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedLength from '../../svg/SVGAnimatedLength.js';
import SVGAnimatedPreserveAspectRatio from '../../svg/SVGAnimatedPreserveAspectRatio.js';
import SVGAnimatedString from '../../svg/SVGAnimatedString.js';
import SVGGraphicsElement from '../svg-graphics-element/SVGGraphicsElement.js';
/**
 * SVG Image Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGImageElement
 */
export default class SVGImageElement extends SVGGraphicsElement {
    [PropertySymbol.href]: SVGAnimatedString | null;
    [PropertySymbol.preserveAspectRatio]: SVGAnimatedPreserveAspectRatio | null;
    [PropertySymbol.x]: SVGAnimatedLength | null;
    [PropertySymbol.y]: SVGAnimatedLength | null;
    [PropertySymbol.width]: SVGAnimatedLength | null;
    [PropertySymbol.height]: SVGAnimatedLength | null;
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
     * Returns href.
     *
     * @returns Href.
     */
    get href(): SVGAnimatedString;
    /**
     * Returns decoding.
     *
     * @returns Decoding.
     */
    get decoding(): string;
    /**
     * Sets decoding.
     *
     * @param value Decoding.
     */
    set decoding(value: string);
    /**
     * Returns preserve aspect ratio.
     *
     * @returns Preserve aspect ratio.
     */
    get preserveAspectRatio(): SVGAnimatedPreserveAspectRatio;
    /**
     * Returns height.
     *
     * @returns Height.
     */
    get height(): SVGAnimatedLength;
    /**
     * Returns width.
     *
     * @returns Width.
     */
    get width(): SVGAnimatedLength;
    /**
     * Returns x position.
     *
     * @returns X position.
     */
    get x(): SVGAnimatedLength;
    /**
     * Returns y position.
     *
     * @returns Y position.
     */
    get y(): SVGAnimatedLength;
    /**
     * Decodes the image.
     *
     * @returns Promise.
     */
    decode(): Promise<void>;
}
//# sourceMappingURL=SVGImageElement.d.ts.map