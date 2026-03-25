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
const SVGGeometryElement_js_1 = __importDefault(require("../svg-geometry-element/SVGGeometryElement.cjs"));
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
const SVGPointList_js_1 = __importDefault(require("../../svg/SVGPointList.cjs"));
/**
 * SVG Polygon Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGPolygonElement
 */
class SVGPolygonElement extends SVGGeometryElement_js_1.default {
    // Internal properties
    [PropertySymbol.animatedPoints] = null;
    [PropertySymbol.points] = null;
    /**
     * Returns animated points.
     *
     * @returns Animated points.
     */
    get animatedPoints() {
        if (!this[PropertySymbol.animatedPoints]) {
            this[PropertySymbol.animatedPoints] = new SVGPointList_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                readOnly: true,
                getAttribute: () => this.getAttribute('points'),
                setAttribute: () => { }
            });
        }
        return this[PropertySymbol.animatedPoints];
    }
    /**
     * Returns points.
     *
     * @returns Points.
     */
    get points() {
        if (!this[PropertySymbol.points]) {
            this[PropertySymbol.points] = new SVGPointList_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('points'),
                setAttribute: (value) => this.setAttribute('points', value)
            });
        }
        return this[PropertySymbol.points];
    }
}
exports.default = SVGPolygonElement;
//# sourceMappingURL=SVGPolygonElement.cjs.map