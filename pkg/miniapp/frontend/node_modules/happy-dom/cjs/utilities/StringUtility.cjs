"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
const ASCII_LOWER_CASE_CACHE = new Map();
const ASCII_UPPER_CASE_CACHE = new Map();
/**
 * String utility.
 */
class StringUtility {
    /**
     * ASCII lowercase.
     *
     * @see https://infra.spec.whatwg.org/#ascii-lowercase
     * @param text Text.
     * @returns Lowercase text.
     */
    static asciiLowerCase(text) {
        const cached = ASCII_LOWER_CASE_CACHE.get(text);
        if (cached) {
            return cached;
        }
        let newText = '';
        for (const char of text) {
            const value = char.charCodeAt(0);
            if (value >= 65 && value <= 90) {
                newText += String.fromCharCode(value + 32);
            }
            else {
                newText += char;
            }
        }
        ASCII_LOWER_CASE_CACHE.set(text, newText);
        return newText;
    }
    /**
     * ASCII uppercase.
     *
     * @see https://infra.spec.whatwg.org/#ascii-uppercase
     * @param text Text.
     * @returns Uppercase text.
     */
    static asciiUpperCase(text) {
        const cached = ASCII_UPPER_CASE_CACHE.get(text);
        if (cached) {
            return cached;
        }
        let newText = '';
        for (const char of text) {
            const value = char.charCodeAt(0);
            if (value >= 97 && value <= 122) {
                newText += String.fromCharCode(value - 32);
            }
            else {
                newText += char;
            }
        }
        ASCII_UPPER_CASE_CACHE.set(text, newText);
        return newText;
    }
}
exports.default = StringUtility;
//# sourceMappingURL=StringUtility.cjs.map