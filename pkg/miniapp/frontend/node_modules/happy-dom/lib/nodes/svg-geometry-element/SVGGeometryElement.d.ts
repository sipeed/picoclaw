import SVGGraphicsElement from '../svg-graphics-element/SVGGraphicsElement.js';
import SVGAnimatedNumber from '../../svg/SVGAnimatedNumber.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import SVGPoint from '../../svg/SVGPoint.js';
/**
 * SVG Geometry Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGGeometryElement
 */
export default class SVGGeometryElement extends SVGGraphicsElement {
    [PropertySymbol.pathLength]: SVGAnimatedNumber | null;
    /**
     * Returns path length.
     *
     * @returns Path length.
     */
    get pathLength(): SVGAnimatedNumber;
    /**
     * Returns true if the point is in the fill of the element.
     *
     * Not implemented yet.
     *
     * @param point Point.
     * @returns True if the point is in the fill of the element.
     */
    isPointInFill(point: SVGPoint): boolean;
    /**
     * Returns true if the point is in the stroke of the element.
     *
     * Not implemented yet.
     *
     * @param point Point.
     * @returns True if the point is in the stroke of the element.
     */
    isPointInStroke(point: SVGPoint): boolean;
    /**
     * Returns total length.
     *
     * Not implemented yet.
     *
     * @returns Total length.
     */
    getTotalLength(): number;
    /**
     * Returns point at length.
     *
     * Not implemented yet.
     *
     * @param _distance Distance.
     * @returns Point at length.
     */
    getPointAtLength(_distance: number): SVGPoint;
}
//# sourceMappingURL=SVGGeometryElement.d.ts.map