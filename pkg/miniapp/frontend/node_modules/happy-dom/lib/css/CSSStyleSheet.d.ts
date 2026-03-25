import CSSRule from './CSSRule.js';
import MediaList from './MediaList.js';
import BrowserWindow from '../window/BrowserWindow.js';
import * as PropertySymbol from '../PropertySymbol.js';
/**
 * CSS StyleSheet.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/CSSStyleSheet.
 */
export default class CSSStyleSheet {
    #private;
    protected [PropertySymbol.window]: BrowserWindow;
    value: string;
    name: string;
    namespaceURI: string;
    readonly cssRules: CSSRule[];
    media: MediaList | string;
    title: string;
    alternate: boolean;
    disabled: boolean;
    /**
     * Constructor.
     *
     * @param [options] Options.
     * @param [options.media] Media.
     * @param [options.title] Title.
     * @param [options.alternate] Alternate.
     * @param [options.disabled] Disabled.
     */
    constructor(options?: {
        media?: MediaList | string;
        title?: string;
        alternate?: boolean;
        disabled?: boolean;
    });
    /**
     * Inserts a rule.
     *
     * @see https://developer.mozilla.org/en-US/docs/Web/API/CSSStyleSheet/insertRule
     * @param rule Rule.
     * @param [index] Index.
     * @returns The newly inserterted rule's index.
     */
    insertRule(rule: string, index?: number): number;
    /**
     * Removes a rule.
     *
     * @see https://developer.mozilla.org/en-US/docs/Web/API/CSSStyleSheet/deleteRule
     * @param index Index.
     */
    deleteRule(index: number): void;
    /**
     * Replaces all CSS rules.
     *
     * @see https://developer.mozilla.org/en-US/docs/Web/API/CSSStyleSheet/replace
     * @param text CSS text.
     * @returns Promise.
     */
    replace(text: string): Promise<void>;
    /**
     * Replaces all CSS rules.
     *
     * @see https://developer.mozilla.org/en-US/docs/Web/API/CSSStyleSheet/replaceSync
     * @param text CSS text.
     */
    replaceSync(text: string): void;
}
//# sourceMappingURL=CSSStyleSheet.d.ts.map