import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedEnumeration from '../../svg/SVGAnimatedEnumeration.js';
import SVGAnimatedLength from '../../svg/SVGAnimatedLength.js';
import SVGAnimatedAngle from '../../svg/SVGAnimatedAngle.js';
import SVGAnimatedRect from '../../svg/SVGAnimatedRect.js';
import SVGAnimatedPreserveAspectRatio from '../../svg/SVGAnimatedPreserveAspectRatio.js';
import SVGElement from '../svg-element/SVGElement.js';
/**
 * SVG Rect Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGMarkerElement
 */
export default class SVGMarkerElement extends SVGElement {
    // Static properties
    static SVG_MARKER_ORIENT_UNKNOWN = 0;
    static SVG_MARKER_ORIENT_AUTO = 1;
    static SVG_MARKER_ORIENT_ANGLE = 2;
    static SVG_MARKERUNITS_UNKNOWN = 0;
    static SVG_MARKERUNITS_USERSPACEONUSE = 1;
    static SVG_MARKERUNITS_STROKEWIDTH = 2;
    // Public properties
    SVG_MARKER_ORIENT_UNKNOWN = 0;
    SVG_MARKER_ORIENT_AUTO = 1;
    SVG_MARKER_ORIENT_ANGLE = 2;
    // Internal properties
    [PropertySymbol.markerUnits] = null;
    [PropertySymbol.markerWidth] = null;
    [PropertySymbol.markerHeight] = null;
    [PropertySymbol.orientType] = null;
    [PropertySymbol.orientAngle] = null;
    [PropertySymbol.refX] = null;
    [PropertySymbol.refY] = null;
    [PropertySymbol.viewBox] = null;
    [PropertySymbol.preserveAspectRatio] = null;
    /**
     * Returns marker units.
     *
     * @returns Marker units.
     */
    get markerUnits() {
        if (!this[PropertySymbol.markerUnits]) {
            this[PropertySymbol.markerUnits] = new SVGAnimatedEnumeration(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('markerUnits'),
                setAttribute: (value) => this.setAttribute('markerUnits', value),
                values: ['userSpaceOnUse', 'strokeWidth'],
                defaultValue: 'strokeWidth'
            });
        }
        return this[PropertySymbol.markerUnits];
    }
    /**
     * Returns marker width.
     *
     * @returns Marker width.
     */
    get markerWidth() {
        if (!this[PropertySymbol.markerWidth]) {
            this[PropertySymbol.markerWidth] = new SVGAnimatedLength(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('markerWidth'),
                setAttribute: (value) => this.setAttribute('markerWidth', value)
            });
        }
        return this[PropertySymbol.markerWidth];
    }
    /**
     * Returns marker height.
     *
     * @returns Marker height.
     */
    get markerHeight() {
        if (!this[PropertySymbol.markerHeight]) {
            this[PropertySymbol.markerHeight] = new SVGAnimatedLength(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('markerHeight'),
                setAttribute: (value) => this.setAttribute('markerHeight', value)
            });
        }
        return this[PropertySymbol.markerHeight];
    }
    /**
     * Returns orient type.
     *
     * @returns Orient type.
     */
    get orientType() {
        if (!this[PropertySymbol.orientType]) {
            this[PropertySymbol.orientType] = new SVGAnimatedEnumeration(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('orient'),
                setAttribute: (value) => this.setAttribute('orient', value),
                values: ['auto', null],
                defaultValue: 'auto'
            });
        }
        return this[PropertySymbol.orientType];
    }
    /**
     * Returns orient angle.
     *
     * @returns Orient angle.
     */
    get orientAngle() {
        if (!this[PropertySymbol.orientAngle]) {
            this[PropertySymbol.orientAngle] = new SVGAnimatedAngle(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('orient'),
                setAttribute: (value) => this.setAttribute('orient', value)
            });
        }
        return this[PropertySymbol.orientAngle];
    }
    /**
     * Returns ref x.
     *
     * @returns Ref x.
     */
    get refX() {
        if (!this[PropertySymbol.refX]) {
            this[PropertySymbol.refX] = new SVGAnimatedLength(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('refX'),
                setAttribute: (value) => this.setAttribute('refX', value)
            });
        }
        return this[PropertySymbol.refX];
    }
    /**
     * Returns ref y.
     *
     * @returns Ref y.
     */
    get refY() {
        if (!this[PropertySymbol.refY]) {
            this[PropertySymbol.refY] = new SVGAnimatedLength(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('refY'),
                setAttribute: (value) => this.setAttribute('refY', value)
            });
        }
        return this[PropertySymbol.refY];
    }
    /**
     * Returns view box.
     *
     * @returns View box.
     */
    get viewBox() {
        if (!this[PropertySymbol.viewBox]) {
            this[PropertySymbol.viewBox] = new SVGAnimatedRect(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('viewBox'),
                setAttribute: (value) => this.setAttribute('viewBox', value)
            });
        }
        return this[PropertySymbol.viewBox];
    }
    /**
     * Returns preserve aspect ratio.
     *
     * @returns Preserve aspect ratio.
     */
    get preserveAspectRatio() {
        if (!this[PropertySymbol.preserveAspectRatio]) {
            this[PropertySymbol.preserveAspectRatio] = new SVGAnimatedPreserveAspectRatio(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('preserveAspectRatio'),
                setAttribute: (value) => this.setAttribute('preserveAspectRatio', value)
            });
        }
        return this[PropertySymbol.preserveAspectRatio];
    }
    /**
     * Sets the value of the orient attribute to auto.
     */
    setOrientToAuto() {
        this.setAttribute('orient', 'auto');
    }
    /**
     * Sets the value of the orient attribute to an angle.
     *
     * @param angle Angle.
     */
    setOrientToAngle(angle) {
        this.setAttribute('orient', angle.valueAsString);
    }
}
//# sourceMappingURL=SVGMarkerElement.js.map