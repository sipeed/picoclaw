// Groups:
// Property name   => \s*([^:;]+?)\s*:
// Property value  => \s*((?:[^(;]*?(?:\([^)]*\))?)*?) <- will match any non ';' char except inside (), nested parentheses are not supported
// Important ("!important") => \s*(!important)?
// End of rule  => \s*(?:$|;)
const SPLIT_RULES_REGEXP = /\s*([^:;]+?)\s*:\s*((?:[^(;]*?(?:\([^)]*\))?)*?)\s*(!important)?\s*(?:$|;)/g;
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
    static parse(cssText) {
        const properties = {};
        const rules = [];
        const regexp = new RegExp(SPLIT_RULES_REGEXP);
        let match;
        while ((match = regexp.exec(cssText))) {
            const name = (match[1] ?? '').trim();
            const value = (match[2] ?? '').trim();
            const important = match[3] ? true : false;
            if (name && value) {
                if (name.startsWith('--')) {
                    properties[name] = value;
                }
                rules.push({ name, value, important });
            }
        }
        return { rules, properties };
    }
}
//# sourceMappingURL=CSSStyleDeclarationCSSParser.js.map