import CSSRule from '../CSSRule.cjs';
import CSSStyleSheet from '../CSSStyleSheet.cjs';
/**
 * CSS parser.
 */
export default class CSSParser {
    /**
     * Parses HTML and returns a root element.
     *
     * @param parentStyleSheet Parent style sheet.
     * @param cssText CSS code.
     * @returns Root element.
     */
    static parseFromString(parentStyleSheet: CSSStyleSheet, cssText: string): CSSRule[];
    /**
     * Validates a selector text.
     *
     * @see https://www.w3.org/TR/CSS21/syndata.html#rule-sets
     * @param selectorText Selector text.
     * @returns True if valid, false otherwise.
     */
    private static validateSelectorText;
}
//# sourceMappingURL=CSSParser.d.ts.map