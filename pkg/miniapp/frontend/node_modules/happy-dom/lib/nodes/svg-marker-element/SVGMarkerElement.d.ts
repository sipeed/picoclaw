import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedEnumeration from '../../svg/SVGAnimatedEnumeration.js';
import SVGAnimatedLength from '../../svg/SVGAnimatedLength.js';
import SVGAnimatedAngle from '../../svg/SVGAnimatedAngle.js';
import SVGAnimatedRect from '../../svg/SVGAnimatedRect.js';
import SVGAnimatedPreserveAspectRatio from '../../svg/SVGAnimatedPreserveAspectRatio.js';
import SVGAngle from '../../svg/SVGAngle.js';
import SVGElement from '../svg-element/SVGElement.js';
/**
 * SVG Rect Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGMarkerElement
 */
export default class SVGMarkerElement extends SVGElement {
    static readonly SVG_MARKER_ORIENT_UNKNOWN = 0;
    static readonly SVG_MARKER_ORIENT_AUTO = 1;
    static readonly SVG_MARKER_ORIENT_ANGLE = 2;
    static readonly SVG_MARKERUNITS_UNKNOWN = 0;
    static readonly SVG_MARKERUNITS_USERSPACEONUSE = 1;
    static readonly SVG_MARKERUNITS_STROKEWIDTH = 2;
    readonly SVG_MARKER_ORIENT_UNKNOWN = 0;
    readonly SVG_MARKER_ORIENT_AUTO = 1;
    readonly SVG_MARKER_ORIENT_ANGLE = 2;
    [PropertySymbol.markerUnits]: SVGAnimatedEnumeration | null;
    [PropertySymbol.markerWidth]: SVGAnimatedLength | null;
    [PropertySymbol.markerHeight]: SVGAnimatedLength | null;
    [PropertySymbol.orientType]: SVGAnimatedEnumeration | null;
    [PropertySymbol.orientAngle]: SVGAnimatedAngle | null;
    [PropertySymbol.refX]: SVGAnimatedLength | null;
    [PropertySymbol.refY]: SVGAnimatedLength | null;
    [PropertySymbol.viewBox]: SVGAnimatedRect | null;
    [PropertySymbol.preserveAspectRatio]: SVGAnimatedPreserveAspectRatio | null;
    /**
     * Returns marker units.
     *
     * @returns Marker units.
     */
    get markerUnits(): SVGAnimatedEnumeration;
    /**
     * Returns marker width.
     *
     * @returns Marker width.
     */
    get markerWidth(): SVGAnimatedLength;
    /**
     * Returns marker height.
     *
     * @returns Marker height.
     */
    get markerHeight(): SVGAnimatedLength;
    /**
     * Returns orient type.
     *
     * @returns Orient type.
     */
    get orientType(): SVGAnimatedEnumeration;
    /**
     * Returns orient angle.
     *
     * @returns Orient angle.
     */
    get orientAngle(): SVGAnimatedAngle;
    /**
     * Returns ref x.
     *
     * @returns Ref x.
     */
    get refX(): SVGAnimatedLength;
    /**
     * Returns ref y.
     *
     * @returns Ref y.
     */
    get refY(): SVGAnimatedLength;
    /**
     * Returns view box.
     *
     * @returns View box.
     */
    get viewBox(): SVGAnimatedRect;
    /**
     * Returns preserve aspect ratio.
     *
     * @returns Preserve aspect ratio.
     */
    get preserveAspectRatio(): SVGAnimatedPreserveAspectRatio;
    /**
     * Sets the value of the orient attribute to auto.
     */
    setOrientToAuto(): void;
    /**
     * Sets the value of the orient attribute to an angle.
     *
     * @param angle Angle.
     */
    setOrientToAngle(angle: SVGAngle): void;
}
//# sourceMappingURL=SVGMarkerElement.d.ts.map