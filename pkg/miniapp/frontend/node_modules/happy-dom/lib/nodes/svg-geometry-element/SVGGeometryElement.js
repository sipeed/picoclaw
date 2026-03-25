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
    // Internal properties
    [PropertySymbol.pathLength] = null;
    /**
     * Returns path length.
     *
     * @returns Path length.
     */
    get pathLength() {
        if (!this[PropertySymbol.pathLength]) {
            this[PropertySymbol.pathLength] = new SVGAnimatedNumber(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('pathLength'),
                setAttribute: (value) => this.setAttribute('pathLength', value)
            });
        }
        return this[PropertySymbol.pathLength];
    }
    /**
     * Returns true if the point is in the fill of the element.
     *
     * Not implemented yet.
     *
     * @param point Point.
     * @returns True if the point is in the fill of the element.
     */
    isPointInFill(point) {
        if (!(point instanceof SVGPoint)) {
            throw new TypeError(`Failed to execute 'isPointInFill' on 'SVGGeometryElement': parameter 1 is not of type 'SVGPoint'.`);
        }
        // TODO: Implement isPointInFill()
        return false;
    }
    /**
     * Returns true if the point is in the stroke of the element.
     *
     * Not implemented yet.
     *
     * @param point Point.
     * @returns True if the point is in the stroke of the element.
     */
    isPointInStroke(point) {
        if (!(point instanceof SVGPoint)) {
            throw new TypeError(`Failed to execute 'isPointInFill' on 'SVGGeometryElement': parameter 1 is not of type 'SVGPoint'.`);
        }
        // TODO: Implement isPointInStroke()
        return false;
    }
    /**
     * Returns total length.
     *
     * Not implemented yet.
     *
     * @returns Total length.
     */
    getTotalLength() {
        // TODO: Implement getTotalLength()
        return 0;
    }
    /**
     * Returns point at length.
     *
     * Not implemented yet.
     *
     * @param _distance Distance.
     * @returns Point at length.
     */
    getPointAtLength(_distance) {
        // TODO: Implement getPointAtLength()
        return new SVGPoint(PropertySymbol.illegalConstructor, this[PropertySymbol.window]);
    }
}
//# sourceMappingURL=SVGGeometryElement.js.map