"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
/**
 * Utility for encoding.
 */
class XMLEncodeUtility {
    /**
     * Encodes attribute value.
     *
     * @param value Value.
     * @returns Escaped value.
     */
    static encodeXMLAttributeValue(value) {
        if (value === null) {
            return '';
        }
        return value
            .replace(/&/gu, '&amp;')
            .replace(/"/gu, '&quot;')
            .replace(/</gu, '&lt;')
            .replace(/>/gu, '&gt;')
            .replace(/\t/gu, '&#x9;')
            .replace(/\n/gu, '&#xA;')
            .replace(/\r/gu, '&#xD;');
    }
    /**
     * Decodes attribute value.
     *
     * @param value Value.
     * @returns Decoded value.
     */
    static decodeXMLAttributeValue(value) {
        if (value === null) {
            return '';
        }
        return value
            .replace(/&quot;/gu, '"')
            .replace(/&lt;/gu, '<')
            .replace(/&gt;/gu, '>')
            .replace(/&#x9;/gu, '\t')
            .replace(/&#xA;/gu, '\n')
            .replace(/&#xD;/gu, '\r')
            .replace(/&amp;/gu, '&');
    }
    /**
     * Encodes attribute value.
     *
     * @param value Value.
     * @returns Escaped value.
     */
    static encodeHTMLAttributeValue(value) {
        if (value === null) {
            return '';
        }
        return value.replace(/&/gu, '&amp;').replace(/"/gu, '&quot;');
    }
    /**
     * Decodes attribute value.
     *
     * @param value Value.
     * @returns Decoded value.
     */
    static decodeHTMLAttributeValue(value) {
        if (value === null) {
            return '';
        }
        return value.replace(/&quot;/gu, '"').replace(/&amp;/gu, '&');
    }
    /**
     * Encodes text content.
     *
     * @param text Value.
     * @returns Escaped value.
     */
    static encodeTextContent(text) {
        if (text === null) {
            return '';
        }
        return text
            .replace(/&/gu, '&amp;')
            .replace(/\xA0/gu, '&nbsp;')
            .replace(/</gu, '&lt;')
            .replace(/>/gu, '&gt;');
    }
    /**
     * Decodes text content.
     *
     * @param text Value.
     * @returns Decoded value.
     */
    static decodeTextContent(text) {
        if (text === null) {
            return '';
        }
        return text
            .replace(/&nbsp;/gu, String.fromCharCode(160))
            .replace(/&lt;/gu, '<')
            .replace(/&gt;/gu, '>')
            .replace(/&amp;/gu, '&');
    }
    /**
     * Decodes HTML entities.
     *
     * @param value Value.
     * @returns Decoded value.
     */
    static decodeHTMLEntities(value) {
        if (value === null) {
            return '';
        }
        return value
            .replace(/&lt;/gu, '<')
            .replace(/&gt;/gu, '>')
            .replace(/&nbsp;/gu, String.fromCharCode(160))
            .replace(/&quot;/gu, '"')
            .replace(/&apos;/gu, "'")
            .replace(/&#(\d+);/gu, (_match, dec) => String.fromCharCode(parseInt(dec, 10)))
            .replace(/&#x([A-Fa-f\d]+);/gu, (_match, hex) => String.fromCharCode(parseInt(hex, 16)))
            .replace(/&amp;/gu, '&');
    }
    /**
     * Decodes XML entities.
     *
     * @param value Value.
     * @returns Decoded value.
     */
    static decodeXMLEntities(value) {
        if (value === null) {
            return '';
        }
        return (value
            .replace(/&lt;/gu, '<')
            .replace(/&gt;/gu, '>')
            // "&nbsp;" Should not be supported in XML.
            .replace(/&quot;/gu, '"')
            .replace(/&apos;/gu, "'")
            .replace(/&#(\d+);/gu, (_match, dec) => String.fromCharCode(parseInt(dec, 10)))
            .replace(/&#x([A-Fa-f\d]+);/gu, (_match, hex) => String.fromCharCode(parseInt(hex, 16)))
            .replace(/&amp;/gu, '&'));
    }
}
exports.default = XMLEncodeUtility;
//# sourceMappingURL=XMLEncodeUtility.cjs.map