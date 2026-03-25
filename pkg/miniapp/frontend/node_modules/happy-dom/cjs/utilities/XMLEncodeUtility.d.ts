/**
 * Utility for encoding.
 */
export default class XMLEncodeUtility {
    /**
     * Encodes attribute value.
     *
     * @param value Value.
     * @returns Escaped value.
     */
    static encodeXMLAttributeValue(value: string | null): string;
    /**
     * Decodes attribute value.
     *
     * @param value Value.
     * @returns Decoded value.
     */
    static decodeXMLAttributeValue(value: string | null): string;
    /**
     * Encodes attribute value.
     *
     * @param value Value.
     * @returns Escaped value.
     */
    static encodeHTMLAttributeValue(value: string | null): string;
    /**
     * Decodes attribute value.
     *
     * @param value Value.
     * @returns Decoded value.
     */
    static decodeHTMLAttributeValue(value: string | null): string;
    /**
     * Encodes text content.
     *
     * @param text Value.
     * @returns Escaped value.
     */
    static encodeTextContent(text: string | null): string;
    /**
     * Decodes text content.
     *
     * @param text Value.
     * @returns Decoded value.
     */
    static decodeTextContent(text: string | null): string;
    /**
     * Decodes HTML entities.
     *
     * @param value Value.
     * @returns Decoded value.
     */
    static decodeHTMLEntities(value: string): string;
    /**
     * Decodes XML entities.
     *
     * @param value Value.
     * @returns Decoded value.
     */
    static decodeXMLEntities(value: string): string;
}
//# sourceMappingURL=XMLEncodeUtility.d.ts.map