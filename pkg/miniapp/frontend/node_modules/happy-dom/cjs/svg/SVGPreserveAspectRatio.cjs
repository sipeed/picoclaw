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
const PropertySymbol = __importStar(require("../PropertySymbol.cjs"));
const SVGPreserveAspectRatioMeetOrSliceEnum_js_1 = __importDefault(require("./SVGPreserveAspectRatioMeetOrSliceEnum.cjs"));
const SVGPreserveAspectRatioAlignEnum_js_1 = __importDefault(require("./SVGPreserveAspectRatioAlignEnum.cjs"));
const ALIGN_KEYS = Object.values(SVGPreserveAspectRatioAlignEnum_js_1.default);
ALIGN_KEYS.length = ALIGN_KEYS.indexOf(0);
const MEET_OR_SLICE_KEYS = Object.values(SVGPreserveAspectRatioMeetOrSliceEnum_js_1.default);
MEET_OR_SLICE_KEYS.length = MEET_OR_SLICE_KEYS.indexOf(0);
/**
 * SVG preserve aspect ratio.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGPreserveAspectRatio
 */
class SVGPreserveAspectRatio {
    // Static properties
    static SVG_MEETORSLICE_UNKNOWN = SVGPreserveAspectRatioMeetOrSliceEnum_js_1.default.unknown;
    static SVG_MEETORSLICE_MEET = SVGPreserveAspectRatioMeetOrSliceEnum_js_1.default.meet;
    static SVG_MEETORSLICE_SLICE = SVGPreserveAspectRatioMeetOrSliceEnum_js_1.default.slice;
    static SVG_PRESERVEASPECTRATIO_UNKNOWN = SVGPreserveAspectRatioAlignEnum_js_1.default.unknown;
    static SVG_PRESERVEASPECTRATIO_NONE = SVGPreserveAspectRatioAlignEnum_js_1.default.none;
    static SVG_PRESERVEASPECTRATIO_XMINYMIN = SVGPreserveAspectRatioAlignEnum_js_1.default.xMinYMin;
    static SVG_PRESERVEASPECTRATIO_XMIDYMIN = SVGPreserveAspectRatioAlignEnum_js_1.default.xMidYMin;
    static SVG_PRESERVEASPECTRATIO_XMAXYMIN = SVGPreserveAspectRatioAlignEnum_js_1.default.xMaxYMin;
    static SVG_PRESERVEASPECTRATIO_XMINYMID = SVGPreserveAspectRatioAlignEnum_js_1.default.xMinYMid;
    static SVG_PRESERVEASPECTRATIO_XMIDYMID = SVGPreserveAspectRatioAlignEnum_js_1.default.xMidYMid;
    static SVG_PRESERVEASPECTRATIO_XMAXYMID = SVGPreserveAspectRatioAlignEnum_js_1.default.xMaxYMid;
    static SVG_PRESERVEASPECTRATIO_XMINYMAX = SVGPreserveAspectRatioAlignEnum_js_1.default.xMinYMax;
    static SVG_PRESERVEASPECTRATIO_XMIDYMAX = SVGPreserveAspectRatioAlignEnum_js_1.default.xMidYMax;
    static SVG_PRESERVEASPECTRATIO_XMAXYMAX = SVGPreserveAspectRatioAlignEnum_js_1.default.xMaxYMax;
    // Internal properties
    [PropertySymbol.window];
    [PropertySymbol.getAttribute] = null;
    [PropertySymbol.setAttribute] = null;
    [PropertySymbol.attributeValue] = null;
    [PropertySymbol.readOnly] = false;
    /**
     * Constructor.
     *
     * @param illegalConstructorSymbol Illegal constructor symbol.
     * @param window Window.
     * @param [options] Options.
     * @param [options.readOnly] Read only.
     * @param [options.getAttribute] Get attribute.
     * @param [options.setAttribute] Set attribute.
     */
    constructor(illegalConstructorSymbol, window, options) {
        if (illegalConstructorSymbol !== PropertySymbol.illegalConstructor) {
            throw new TypeError('Illegal constructor');
        }
        this[PropertySymbol.window] = window;
        if (options) {
            this[PropertySymbol.readOnly] = !!options.readOnly;
            this[PropertySymbol.getAttribute] = options.getAttribute || null;
            this[PropertySymbol.setAttribute] = options.setAttribute || null;
        }
    }
    /**
     * Returns align.
     *
     * @returns Align.
     */
    get align() {
        const attributeValue = this[PropertySymbol.getAttribute]
            ? this[PropertySymbol.getAttribute]()
            : this[PropertySymbol.attributeValue];
        if (!attributeValue) {
            return SVGPreserveAspectRatioAlignEnum_js_1.default.xMidYMid;
        }
        const align = attributeValue.split(/\s+/)[0];
        if (SVGPreserveAspectRatioAlignEnum_js_1.default[align] === undefined) {
            return SVGPreserveAspectRatioAlignEnum_js_1.default.xMidYMid;
        }
        return SVGPreserveAspectRatioAlignEnum_js_1.default[align];
    }
    /**
     * Sets align.
     *
     * @param value Align.
     */
    set align(value) {
        if (this[PropertySymbol.readOnly]) {
            throw new this[PropertySymbol.window].TypeError(`Failed to set the 'align' property on 'SVGPreserveAspectRatio': The object is read-only.`);
        }
        const parsedValue = Number(value);
        if (isNaN(parsedValue) || parsedValue < 1 || parsedValue > ALIGN_KEYS.length) {
            throw new this[PropertySymbol.window].TypeError(`Failed to set the 'align' property on 'SVGPreserveAspectRatio': The alignment provided is invalid.`);
        }
        this[PropertySymbol.attributeValue] = `${ALIGN_KEYS[parsedValue]} ${MEET_OR_SLICE_KEYS[this.meetOrSlice]}`;
        if (this[PropertySymbol.setAttribute]) {
            this[PropertySymbol.setAttribute](this[PropertySymbol.attributeValue]);
        }
    }
    /**
     * Returns meet or slice.
     *
     * @returns Meet or slice.
     */
    get meetOrSlice() {
        const attributeValue = this[PropertySymbol.getAttribute]
            ? this[PropertySymbol.getAttribute]()
            : this[PropertySymbol.attributeValue];
        if (!attributeValue) {
            return SVGPreserveAspectRatioMeetOrSliceEnum_js_1.default.meet;
        }
        const meetOrSlice = attributeValue.split(/\s+/)[1];
        if (!meetOrSlice || SVGPreserveAspectRatioMeetOrSliceEnum_js_1.default[meetOrSlice] === undefined) {
            return SVGPreserveAspectRatioMeetOrSliceEnum_js_1.default.meet;
        }
        return SVGPreserveAspectRatioMeetOrSliceEnum_js_1.default[meetOrSlice];
    }
    /**
     * Sets meet or slice.
     *
     * @param value Meet or slice.
     */
    set meetOrSlice(value) {
        if (this[PropertySymbol.readOnly]) {
            throw new this[PropertySymbol.window].TypeError(`Failed to set the 'meetOrSlice' property on 'SVGPreserveAspectRatio': The object is read-only.`);
        }
        const parsedValue = Number(value);
        if (isNaN(parsedValue) || parsedValue < 1 || parsedValue > 2) {
            throw new this[PropertySymbol.window].TypeError(`Failed to set the 'meetOrSlice' property on 'SVGPreserveAspectRatio': The meetOrSlice provided is invalid.`);
        }
        this[PropertySymbol.attributeValue] = `${ALIGN_KEYS[this.align]} ${MEET_OR_SLICE_KEYS[parsedValue]}`;
        if (this[PropertySymbol.setAttribute]) {
            this[PropertySymbol.setAttribute](this[PropertySymbol.attributeValue]);
        }
    }
}
exports.default = SVGPreserveAspectRatio;
//# sourceMappingURL=SVGPreserveAspectRatio.cjs.map