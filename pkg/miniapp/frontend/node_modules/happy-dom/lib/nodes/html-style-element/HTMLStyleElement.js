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
    [PropertySymbol.sheet] = null;
    [PropertySymbol.styleNode] = this;
    [PropertySymbol.disabled] = false;
    /**
     * Returns media.
     *
     * @returns Media.
     */
    get media() {
        return this.getAttribute('media') || '';
    }
    /**
     * Sets media.
     *
     * @param media Media.
     */
    set media(media) {
        this.setAttribute('media', media);
    }
    /**
     * Returns type.
     *
     * @deprecated
     * @returns Type.
     */
    get type() {
        return this.getAttribute('type') || '';
    }
    /**
     * Sets type.
     *
     * @deprecated
     * @param type Type.
     */
    set type(type) {
        this.setAttribute('type', type);
    }
    /**
     * Returns disabled.
     *
     * @returns Disabled.
     */
    get disabled() {
        return this[PropertySymbol.disabled];
    }
    /**
     * Sets disabled.
     *
     * @param disabled Disabled.
     */
    set disabled(disabled) {
        this[PropertySymbol.disabled] = Boolean(disabled);
    }
    /**
     * Returns CSS style sheet.
     *
     * @returns CSS style sheet.
     */
    get sheet() {
        if (!this[PropertySymbol.isConnected]) {
            return null;
        }
        if (!this[PropertySymbol.sheet]) {
            this[PropertySymbol.sheet] = new CSSStyleSheet();
            this[PropertySymbol.sheet].replaceSync(this.textContent);
        }
        return this[PropertySymbol.sheet];
    }
    /**
     * @override
     */
    [PropertySymbol.disconnectedFromDocument]() {
        super[PropertySymbol.disconnectedFromDocument]();
        this[PropertySymbol.sheet] = null;
    }
    /**
     * Updates the CSSStyleSheet with the text content.
     */
    [PropertySymbol.updateSheet]() {
        if (this[PropertySymbol.sheet]) {
            this[PropertySymbol.sheet].replaceSync(this.textContent);
        }
    }
}
//# sourceMappingURL=HTMLStyleElement.js.map