/**
 * DOMStringMap utility.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLElement/dataset
 */
export default class DOMStringMapUtility {
    /**
     * Transforms a kebab cased string to camel case.
     *
     * @param text Text string.
     * @returns Camel cased string.
     */
    static kebabToCamelCase(text) {
        const parts = text.split('-');
        for (let i = 0, max = parts.length; i < max; i++) {
            parts[i] = i > 0 ? parts[i].charAt(0).toUpperCase() + parts[i].slice(1) : parts[i];
        }
        return parts.join('');
    }
    /**
     * Transforms a camel cased string to kebab case.
     *
     * @param text Text string.
     * @returns Kebab cased string.
     */
    static camelCaseToKebab(text) {
        return text
            .toString()
            .replace(/[A-Z]+(?![a-z])|[A-Z]/g, ($, ofs) => (ofs ? '-' : '') + $.toLowerCase());
    }
}
//# sourceMappingURL=DOMStringMapUtility.js.map