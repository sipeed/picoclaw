import * as PropertySymbol from '../PropertySymbol.js';
import SVGPreserveAspectRatioMeetOrSliceEnum from './SVGPreserveAspectRatioMeetOrSliceEnum.js';
import SVGPreserveAspectRatioAlignEnum from './SVGPreserveAspectRatioAlignEnum.js';
const ALIGN_KEYS = Object.values(SVGPreserveAspectRatioAlignEnum);
ALIGN_KEYS.length = ALIGN_KEYS.indexOf(0);
const MEET_OR_SLICE_KEYS = Object.values(SVGPreserveAspectRatioMeetOrSliceEnum);
MEET_OR_SLICE_KEYS.length = MEET_OR_SLICE_KEYS.indexOf(0);
/**
 * SVG preserve aspect ratio.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGPreserveAspectRatio
 */
export default class SVGPreserveAspectRatio {
    // Static properties
    static SVG_MEETORSLICE_UNKNOWN = SVGPreserveAspectRatioMeetOrSliceEnum.unknown;
    static SVG_MEETORSLICE_MEET = SVGPreserveAspectRatioMeetOrSliceEnum.meet;
    static SVG_MEETORSLICE_SLICE = SVGPreserveAspectRatioMeetOrSliceEnum.slice;
    static SVG_PRESERVEASPECTRATIO_UNKNOWN = SVGPreserveAspectRatioAlignEnum.unknown;
    static SVG_PRESERVEASPECTRATIO_NONE = SVGPreserveAspectRatioAlignEnum.none;
    static SVG_PRESERVEASPECTRATIO_XMINYMIN = SVGPreserveAspectRatioAlignEnum.xMinYMin;
    static SVG_PRESERVEASPECTRATIO_XMIDYMIN = SVGPreserveAspectRatioAlignEnum.xMidYMin;
    static SVG_PRESERVEASPECTRATIO_XMAXYMIN = SVGPreserveAspectRatioAlignEnum.xMaxYMin;
    static SVG_PRESERVEASPECTRATIO_XMINYMID = SVGPreserveAspectRatioAlignEnum.xMinYMid;
    static SVG_PRESERVEASPECTRATIO_XMIDYMID = SVGPreserveAspectRatioAlignEnum.xMidYMid;
    static SVG_PRESERVEASPECTRATIO_XMAXYMID = SVGPreserveAspectRatioAlignEnum.xMaxYMid;
    static SVG_PRESERVEASPECTRATIO_XMINYMAX = SVGPreserveAspectRatioAlignEnum.xMinYMax;
    static SVG_PRESERVEASPECTRATIO_XMIDYMAX = SVGPreserveAspectRatioAlignEnum.xMidYMax;
    static SVG_PRESERVEASPECTRATIO_XMAXYMAX = SVGPreserveAspectRatioAlignEnum.xMaxYMax;
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
            return SVGPreserveAspectRatioAlignEnum.xMidYMid;
        }
        const align = attributeValue.split(/\s+/)[0];
        if (SVGPreserveAspectRatioAlignEnum[align] === undefined) {
            return SVGPreserveAspectRatioAlignEnum.xMidYMid;
        }
        return SVGPreserveAspectRatioAlignEnum[align];
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
            return SVGPreserveAspectRatioMeetOrSliceEnum.meet;
        }
        const meetOrSlice = attributeValue.split(/\s+/)[1];
        if (!meetOrSlice || SVGPreserveAspectRatioMeetOrSliceEnum[meetOrSlice] === undefined) {
            return SVGPreserveAspectRatioMeetOrSliceEnum.meet;
        }
        return SVGPreserveAspectRatioMeetOrSliceEnum[meetOrSlice];
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
//# sourceMappingURL=SVGPreserveAspectRatio.js.map