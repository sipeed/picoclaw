import BrowserWindow from '../window/BrowserWindow.cjs';
import IMediaQueryRange from './IMediaQueryRange.cjs';
import IMediaQueryRule from './IMediaQueryRule.cjs';
import MediaQueryTypeEnum from './MediaQueryTypeEnum.cjs';
/**
 * Media query this.
 */
export default class MediaQueryItem {
    mediaTypes: MediaQueryTypeEnum[];
    not: boolean;
    rules: IMediaQueryRule[];
    ranges: IMediaQueryRange[];
    private rootFontSize;
    private window;
    /**
     * Constructor.
     *
     * @param options Options.
     * @param options.window Owner window.
     * @param [options.rootFontSize] Root font size.
     * @param [options.mediaTypes] Media types.
     * @param [options.not] Not.
     * @param [options.rules] Rules.
     * @param [options.ranges] Ranges.
     */
    constructor(options: {
        window: BrowserWindow;
        rootFontSize?: string | number | null;
        mediaTypes?: MediaQueryTypeEnum[];
        not?: boolean;
        rules?: IMediaQueryRule[];
        ranges?: IMediaQueryRange[];
    });
    /**
     * Returns media string.
     */
    toString(): string;
    /**
     * Returns "true" if the item matches.
     */
    matches(): boolean;
    /**
     * Returns "true" if all matches.
     *
     * @returns "true" if all matches.
     */
    private matchesAll;
    /**
     * Returns "true" if the mediaType matches.
     *
     * @param mediaType Media type.
     * @returns "true" if the mediaType matches.
     */
    private matchesMediaType;
    /**
     * Returns "true" if the range matches.
     *
     * @param range Range.
     * @returns "true" if the range matches.
     */
    private matchesRange;
    /**
     * Returns "true" if the rule matches.
     *
     * @param rule Rule.
     * @returns "true" if the rule matches.
     */
    private matchesRule;
    /**
     * Convert to pixels.
     *
     * @param value Value.
     * @returns Value in pixels.
     */
    private toPixels;
}
//# sourceMappingURL=MediaQueryItem.d.ts.map