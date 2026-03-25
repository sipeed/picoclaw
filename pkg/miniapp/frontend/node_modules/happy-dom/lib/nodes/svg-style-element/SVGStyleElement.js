import CSSStyleSheet from '../../css/CSSStyleSheet.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import SVGElement from '../svg-element/SVGElement.js';
/**
 * SVG Style Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGStyleElement
 */
export default class SVGStyleElement extends SVGElement {
    // Internal properties
    [PropertySymbol.sheet] = null;
    [PropertySymbol.styleNode] = this;
    [PropertySymbol.disabled] = false;
    /**
     * Returns media.
     *
     * @returns Media.
     */
    get media() {
        return this.getAttribute('media') || 'all';
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
        return this.getAttribute('type') || 'text/css';
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
     * Returns title.
     *
     * @returns Title.
     */
    get title() {
        return this.getAttribute('title') || '';
    }
    /**
     * Sets title.
     *
     * @param title Title.
     */
    set title(title) {
        this.setAttribute('title', title);
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
     * Returns style.
     *
     * @returns Style.
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
//# sourceMappingURL=SVGStyleElement.js.map