"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const MediaQueryItem_js_1 = __importDefault(require("./MediaQueryItem.cjs"));
const MediaQueryTypeEnum_js_1 = __importDefault(require("./MediaQueryTypeEnum.cjs"));
/**
 * Media query RegExp.
 *
 * Group 1: "not", "only", "all", "screen", "print".
 * Group 2: Rule.
 * Group 3: Rule end paranthesis (must be present if no value).
 * Group 4: Comma (,).
 * Group 5: "or", "and".
 */
const MEDIA_QUERY_REGEXP = /(not|only|all|screen|print)|\(([^\)]+)(\)){0,1}|(,)| +(or|and) +/g;
/**
 * Check if resolution RegExp.
 */
const IS_RESOLUTION_REGEXP = /[<>]/;
/**
 * Resolution RegExp.
 *
 * Group 1: First resolution value.
 * Group 2: First resolution operator.
 * Group 3: Resolution type.
 * Group 4: Second resolution operator.
 * Group 5: Second resolution value.
 */
const RESOLUTION_REGEXP = /(?:([0-9]+[a-z]+) *(<|<=|>|=>)){0,1} *(width|height) *(?:(<|<=|>|=>) *([0-9]+[a-z]+)){0,1}/;
/**
 * Utility for parsing a query string.
 */
class MediaQueryParser {
    /**
     * Parses a media query string.
     *
     * @param options Options.
     * @param options.window Owner window.
     * @param options.mediaQuery Media query string.
     * @param [options.rootFontSize] Root font size.
     * @returns Media query items.
     */
    static parse(options) {
        let currentMediaQueryItem = new MediaQueryItem_js_1.default({
            window: options.window,
            rootFontSize: options.rootFontSize
        });
        const mediaQueryItems = [currentMediaQueryItem];
        const regexp = new RegExp(MEDIA_QUERY_REGEXP);
        let match = null;
        while ((match = regexp.exec(options.mediaQuery.toLowerCase()))) {
            if (match[4] === ',' || match[5] === 'or') {
                currentMediaQueryItem = new MediaQueryItem_js_1.default({
                    window: options.window,
                    rootFontSize: options.rootFontSize
                });
                mediaQueryItems.push(currentMediaQueryItem);
            }
            else if (match[1] === 'all' || match[1] === 'screen' || match[1] === 'print') {
                currentMediaQueryItem.mediaTypes.push(match[1]);
            }
            else if (match[1] === 'not') {
                currentMediaQueryItem.not = true;
            }
            else if (match[2]) {
                const resolutionMatch = IS_RESOLUTION_REGEXP.test(match[2])
                    ? match[2].match(RESOLUTION_REGEXP)
                    : null;
                if (resolutionMatch && (resolutionMatch[1] || resolutionMatch[5])) {
                    currentMediaQueryItem.ranges.push({
                        before: resolutionMatch[1]
                            ? {
                                value: resolutionMatch[1],
                                operator: resolutionMatch[2]
                            }
                            : null,
                        type: resolutionMatch[3],
                        after: resolutionMatch[5]
                            ? {
                                value: resolutionMatch[5],
                                operator: resolutionMatch[4]
                            }
                            : null
                    });
                }
                else {
                    const [name, value] = match[2].split(':');
                    const trimmedValue = value ? value.trim() : null;
                    if (!trimmedValue && !match[3]) {
                        return [
                            new MediaQueryItem_js_1.default({
                                window: options.window,
                                rootFontSize: options.rootFontSize,
                                not: true,
                                mediaTypes: [MediaQueryTypeEnum_js_1.default.all]
                            })
                        ];
                    }
                    currentMediaQueryItem.rules.push({
                        name: name.trim(),
                        value: trimmedValue
                    });
                }
            }
        }
        return mediaQueryItems;
    }
}
exports.default = MediaQueryParser;
//# sourceMappingURL=MediaQueryParser.cjs.map