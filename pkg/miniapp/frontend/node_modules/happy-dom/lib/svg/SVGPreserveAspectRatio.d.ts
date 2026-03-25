import * as PropertySymbol from '../PropertySymbol.js';
import SVGPreserveAspectRatioMeetOrSliceEnum from './SVGPreserveAspectRatioMeetOrSliceEnum.js';
import SVGPreserveAspectRatioAlignEnum from './SVGPreserveAspectRatioAlignEnum.js';
import BrowserWindow from '../window/BrowserWindow.js';
/**
 * SVG preserve aspect ratio.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGPreserveAspectRatio
 */
export default class SVGPreserveAspectRatio {
    static SVG_MEETORSLICE_UNKNOWN: SVGPreserveAspectRatioMeetOrSliceEnum;
    static SVG_MEETORSLICE_MEET: SVGPreserveAspectRatioMeetOrSliceEnum;
    static SVG_MEETORSLICE_SLICE: SVGPreserveAspectRatioMeetOrSliceEnum;
    static SVG_PRESERVEASPECTRATIO_UNKNOWN: SVGPreserveAspectRatioAlignEnum;
    static SVG_PRESERVEASPECTRATIO_NONE: SVGPreserveAspectRatioAlignEnum;
    static SVG_PRESERVEASPECTRATIO_XMINYMIN: SVGPreserveAspectRatioAlignEnum;
    static SVG_PRESERVEASPECTRATIO_XMIDYMIN: SVGPreserveAspectRatioAlignEnum;
    static SVG_PRESERVEASPECTRATIO_XMAXYMIN: SVGPreserveAspectRatioAlignEnum;
    static SVG_PRESERVEASPECTRATIO_XMINYMID: SVGPreserveAspectRatioAlignEnum;
    static SVG_PRESERVEASPECTRATIO_XMIDYMID: SVGPreserveAspectRatioAlignEnum;
    static SVG_PRESERVEASPECTRATIO_XMAXYMID: SVGPreserveAspectRatioAlignEnum;
    static SVG_PRESERVEASPECTRATIO_XMINYMAX: SVGPreserveAspectRatioAlignEnum;
    static SVG_PRESERVEASPECTRATIO_XMIDYMAX: SVGPreserveAspectRatioAlignEnum;
    static SVG_PRESERVEASPECTRATIO_XMAXYMAX: SVGPreserveAspectRatioAlignEnum;
    [PropertySymbol.window]: BrowserWindow;
    [PropertySymbol.getAttribute]: (() => string | null) | null;
    [PropertySymbol.setAttribute]: ((value: string) => void) | null;
    [PropertySymbol.attributeValue]: string | null;
    [PropertySymbol.readOnly]: boolean;
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
    constructor(illegalConstructorSymbol: symbol, window: BrowserWindow, options?: {
        readOnly?: boolean;
        getAttribute?: () => string | null;
        setAttribute?: (value: string) => void;
    });
    /**
     * Returns align.
     *
     * @returns Align.
     */
    get align(): SVGPreserveAspectRatioAlignEnum;
    /**
     * Sets align.
     *
     * @param value Align.
     */
    set align(value: SVGPreserveAspectRatioAlignEnum);
    /**
     * Returns meet or slice.
     *
     * @returns Meet or slice.
     */
    get meetOrSlice(): SVGPreserveAspectRatioMeetOrSliceEnum;
    /**
     * Sets meet or slice.
     *
     * @param value Meet or slice.
     */
    set meetOrSlice(value: SVGPreserveAspectRatioMeetOrSliceEnum);
}
//# sourceMappingURL=SVGPreserveAspectRatio.d.ts.map