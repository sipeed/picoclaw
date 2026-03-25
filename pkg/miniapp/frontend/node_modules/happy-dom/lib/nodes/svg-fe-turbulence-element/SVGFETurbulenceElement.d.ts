import SVGElement from '../svg-element/SVGElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedLength from '../../svg/SVGAnimatedLength.js';
import SVGAnimatedString from '../../svg/SVGAnimatedString.js';
import SVGAnimatedNumber from '../../svg/SVGAnimatedNumber.js';
import SVGAnimatedInteger from '../../svg/SVGAnimatedInteger.js';
import SVGAnimatedEnumeration from '../../svg/SVGAnimatedEnumeration.js';
/**
 * SVGFETurbulenceElement.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGFETurbulenceElement
 */
export default class SVGFETurbulenceElement extends SVGElement {
    static readonly SVG_TURBULENCE_TYPE_UNKNOWN = 0;
    static readonly SVG_TURBULENCE_TYPE_FRACTALNOISE = 1;
    static readonly SVG_TURBULENCE_TYPE_TURBULENCE = 2;
    static readonly SVG_STITCHTYPE_UNKNOWN = 0;
    static readonly SVG_STITCHTYPE_STITCH = 1;
    static readonly SVG_STITCHTYPE_NOSTITCH = 2;
    [PropertySymbol.baseFrequencyX]: SVGAnimatedNumber | null;
    [PropertySymbol.baseFrequencyY]: SVGAnimatedNumber | null;
    [PropertySymbol.height]: SVGAnimatedLength | null;
    [PropertySymbol.numOctaves]: SVGAnimatedInteger | null;
    [PropertySymbol.result]: SVGAnimatedString | null;
    [PropertySymbol.seed]: SVGAnimatedNumber | null;
    [PropertySymbol.stitchTiles]: SVGAnimatedEnumeration | null;
    [PropertySymbol.type]: SVGAnimatedEnumeration | null;
    [PropertySymbol.width]: SVGAnimatedLength | null;
    [PropertySymbol.x]: SVGAnimatedLength | null;
    [PropertySymbol.y]: SVGAnimatedLength | null;
    /**
     * Returns baseFrequencyX.
     *
     * @returns Base frequency x.
     */
    get baseFrequencyX(): SVGAnimatedNumber;
    /**
     * Returns baseFrequencyY.
     *
     * @returns Base frequency y.
     */
    get baseFrequencyY(): SVGAnimatedNumber;
    /**
     * Returns height.
     *
     * @returns Height.
     */
    get height(): SVGAnimatedLength;
    /**
     * Returns numOctaves.
     *
     * @returns Num octaves.
     */
    get numOctaves(): SVGAnimatedInteger;
    /**
     * Returns result.
     *
     * @returns Result.
     */
    get result(): SVGAnimatedString;
    /**
     * Returns seed.
     *
     * @returns Seed.
     */
    get seed(): SVGAnimatedNumber;
    /**
     * Returns stitchTiles.
     *
     * @returns Stitch tiles.
     */
    get stitchTiles(): SVGAnimatedEnumeration;
    /**
     * Returns type.
     *
     * @returns Type.
     */
    get type(): SVGAnimatedEnumeration;
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
//# sourceMappingURL=SVGFETurbulenceElement.d.ts.map