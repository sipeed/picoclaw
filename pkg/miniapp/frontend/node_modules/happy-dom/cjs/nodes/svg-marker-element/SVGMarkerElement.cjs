"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
const SVGAnimatedEnumeration_js_1 = __importDefault(require("../../svg/SVGAnimatedEnumeration.cjs"));
const SVGAnimatedLength_js_1 = __importDefault(require("../../svg/SVGAnimatedLength.cjs"));
const SVGAnimatedAngle_js_1 = __importDefault(require("../../svg/SVGAnimatedAngle.cjs"));
const SVGAnimatedRect_js_1 = __importDefault(require("../../svg/SVGAnimatedRect.cjs"));
const SVGAnimatedPreserveAspectRatio_js_1 = __importDefault(require("../../svg/SVGAnimatedPreserveAspectRatio.cjs"));
const SVGElement_js_1 = __importDefault(require("../svg-element/SVGElement.cjs"));
/**
 * SVG Rect Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGMarkerElement
 */
class SVGMarkerElement extends SVGElement_js_1.default {
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
            this[PropertySymbol.markerUnits] = new SVGAnimatedEnumeration_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
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
            this[PropertySymbol.markerWidth] = new SVGAnimatedLength_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
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
            this[PropertySymbol.markerHeight] = new SVGAnimatedLength_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
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
            this[PropertySymbol.orientType] = new SVGAnimatedEnumeration_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
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
            this[PropertySymbol.orientAngle] = new SVGAnimatedAngle_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
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
            this[PropertySymbol.refX] = new SVGAnimatedLength_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
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
            this[PropertySymbol.refY] = new SVGAnimatedLength_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
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
            this[PropertySymbol.viewBox] = new SVGAnimatedRect_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
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
            this[PropertySymbol.preserveAspectRatio] = new SVGAnimatedPreserveAspectRatio_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
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
exports.default = SVGMarkerElement;
//# sourceMappingURL=SVGMarkerElement.cjs.map