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
const SVGGraphicsElement_js_1 = __importDefault(require("../svg-graphics-element/SVGGraphicsElement.cjs"));
const SVGAnimatedNumber_js_1 = __importDefault(require("../../svg/SVGAnimatedNumber.cjs"));
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
const SVGPoint_js_1 = __importDefault(require("../../svg/SVGPoint.cjs"));
/**
 * SVG Geometry Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGGeometryElement
 */
class SVGGeometryElement extends SVGGraphicsElement_js_1.default {
    // Internal properties
    [PropertySymbol.pathLength] = null;
    /**
     * Returns path length.
     *
     * @returns Path length.
     */
    get pathLength() {
        if (!this[PropertySymbol.pathLength]) {
            this[PropertySymbol.pathLength] = new SVGAnimatedNumber_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
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
        if (!(point instanceof SVGPoint_js_1.default)) {
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
        if (!(point instanceof SVGPoint_js_1.default)) {
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
        return new SVGPoint_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window]);
    }
}
exports.default = SVGGeometryElement;
//# sourceMappingURL=SVGGeometryElement.cjs.map