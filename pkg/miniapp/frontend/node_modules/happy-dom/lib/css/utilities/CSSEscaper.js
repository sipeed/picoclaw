/**
 * CSS escaper.
 */
export default class CSSEscaper {
    /**
     * Escapes CSS.
     *
     * Based on:
     * https://github.com/mathiasbynens/CSS.escape
     *
     * @param cssText CSS.
     * @returns Escaped CSS.
     */
    static escape(cssText) {
        if (arguments.length == 0) {
            throw new TypeError('`CSS.escape` requires an argument.');
        }
        const returnValue = String(cssText);
        const length = returnValue.length;
        let index = -1;
        let codeUnit;
        let result = '';
        const firstCodeUnit = returnValue.charCodeAt(0);
        if (
        // If the character is the first character and is a `-` (U+002D), and
        // There is no second character, […]
        length == 1 &&
            firstCodeUnit == 0x002d) {
            return '\\' + returnValue;
        }
        while (++index < length) {
            codeUnit = returnValue.charCodeAt(index);
            // Note: there’s no need to special-case astral symbols, surrogate
            // Pairs, or lone surrogates.
            // If the character is NULL (U+0000), then the REPLACEMENT CHARACTER
            // (U+FFFD).
            if (codeUnit == 0x0000) {
                result += '\uFFFD';
                continue;
            }
            if (
            // If the character is in the range [\1-\1F] (U+0001 to U+001F) or is
            // U+007F, […]
            (codeUnit >= 0x0001 && codeUnit <= 0x001f) ||
                codeUnit == 0x007f ||
                // If the character is the first character and is in the range [0-9]
                // (U+0030 to U+0039), […]
                (index == 0 && codeUnit >= 0x0030 && codeUnit <= 0x0039) ||
                // If the character is the second character and is in the range [0-9]
                // (U+0030 to U+0039) and the first character is a `-` (U+002D), […]
                (index == 1 && codeUnit >= 0x0030 && codeUnit <= 0x0039 && firstCodeUnit == 0x002d)) {
                // https://drafts.csswg.org/cssom/#escape-a-character-as-code-point
                result += '\\' + codeUnit.toString(16) + ' ';
                continue;
            }
            // If the character is not handled by one of the above rules and is
            // Greater than or equal to U+0080, is `-` (U+002D) or `_` (U+005F), or
            // Is in one of the ranges [0-9] (U+0030 to U+0039), [A-Z] (U+0041 to
            // U+005A), or [a-z] (U+0061 to U+007A), […]
            if (codeUnit >= 0x0080 ||
                codeUnit == 0x002d ||
                codeUnit == 0x005f ||
                (codeUnit >= 0x0030 && codeUnit <= 0x0039) ||
                (codeUnit >= 0x0041 && codeUnit <= 0x005a) ||
                (codeUnit >= 0x0061 && codeUnit <= 0x007a)) {
                // The character itself
                result += returnValue.charAt(index);
                continue;
            }
            // Otherwise, the escaped character.
            // https://drafts.csswg.org/cssom/#escape-a-character
            result += '\\' + returnValue.charAt(index);
        }
        return result;
    }
}
//# sourceMappingURL=CSSEscaper.js.map