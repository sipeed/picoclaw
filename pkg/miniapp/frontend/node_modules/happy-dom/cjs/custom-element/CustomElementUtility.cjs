"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
const PCEN_CHAR = '[-_.]|[0-9]|[a-z]|\u{B7}|[\u{C0}-\u{D6}]|[\u{D8}-\u{F6}]' +
    '|[\u{F8}-\u{37D}]|[\u{37F}-\u{1FFF}]' +
    '|[\u{200C}-\u{200D}]|[\u{203F}-\u{2040}]|[\u{2070}-\u{218F}]' +
    '|[\u{2C00}-\u{2FEF}]|[\u{3001}-\u{D7FF}]' +
    '|[\u{F900}-\u{FDCF}]|[\u{FDF0}-\u{FFFD}]|[\u{10000}-\u{EFFFF}]';
const PCEN_REGEXP = new RegExp(`^[a-z](${PCEN_CHAR})*-(${PCEN_CHAR})*$`, 'u');
const RESERVED_NAMES = [
    'annotation-xml',
    'color-profile',
    'font-face',
    'font-face-src',
    'font-face-uri',
    'font-face-format',
    'font-face-name',
    'missing-glyph'
];
/**
 * Custom element utility.
 */
class CustomElementUtility {
    /**
     * Returns true if the tag name is a valid custom element name.
     *
     * @see https://html.spec.whatwg.org/multipage/custom-elements.html#valid-custom-element-name
     * @param name Tag name.
     * @returns True if valid.
     */
    static isValidCustomElementName(name) {
        return PCEN_REGEXP.test(name) && !RESERVED_NAMES.includes(name);
    }
}
exports.default = CustomElementUtility;
//# sourceMappingURL=CustomElementUtility.cjs.map