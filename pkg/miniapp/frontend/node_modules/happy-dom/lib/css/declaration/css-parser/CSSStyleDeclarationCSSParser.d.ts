/**
 * CSS parser.
 */
export default class CSSStyleDeclarationCSSParser {
    /**
     * Class construtor.
     *
     * @param cssText CSS string.
     * @param callback Callback.
     */
    static parse(cssText: string): {
        rules: Array<{
            name: string;
            value: string;
            important: boolean;
        }>;
        properties: {
            [name: string]: string;
        };
    };
}
//# sourceMappingURL=CSSStyleDeclarationCSSParser.d.ts.map