import CSSStyleSheet from '../../css/CSSStyleSheet.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import HTMLElement from '../html-element/HTMLElement.js';
/**
 * HTML Style Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLStyleElement.
 */
export default class HTMLStyleElement extends HTMLElement {
    private [PropertySymbol.sheet];
    [PropertySymbol.styleNode]: this;
    [PropertySymbol.disabled]: boolean;
    /**
     * Returns media.
     *
     * @returns Media.
     */
    get media(): string;
    /**
     * Sets media.
     *
     * @param media Media.
     */
    set media(media: string);
    /**
     * Returns type.
     *
     * @deprecated
     * @returns Type.
     */
    get type(): string;
    /**
     * Sets type.
     *
     * @deprecated
     * @param type Type.
     */
    set type(type: string);
    /**
     * Returns disabled.
     *
     * @returns Disabled.
     */
    get disabled(): boolean;
    /**
     * Sets disabled.
     *
     * @param disabled Disabled.
     */
    set disabled(disabled: boolean);
    /**
     * Returns CSS style sheet.
     *
     * @returns CSS style sheet.
     */
    get sheet(): CSSStyleSheet;
    /**
     * @override
     */
    [PropertySymbol.disconnectedFromDocument](): void;
    /**
     * Updates the CSSStyleSheet with the text content.
     */
    [PropertySymbol.updateSheet](): void;
}
//# sourceMappingURL=HTMLStyleElement.d.ts.map