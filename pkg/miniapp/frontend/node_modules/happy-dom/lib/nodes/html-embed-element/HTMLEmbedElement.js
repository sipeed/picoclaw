import HTMLElement from '../html-element/HTMLElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
/**
 * HTMLEmbedElement
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLEmbedElement
 */
export default class HTMLEmbedElement extends HTMLElement {
    /**
     * Returns height.
     *
     * @returns Height.
     */
    get height() {
        return this.getAttribute('height') || '';
    }
    /**
     * Sets height.
     *
     * @param height Height.
     */
    set height(height) {
        this.setAttribute('height', height);
    }
    /**
     * Returns width.
     *
     * @returns Width.
     */
    get width() {
        return this.getAttribute('width') || '';
    }
    /**
     * Sets width.
     *
     * @param width Width.
     */
    set width(width) {
        this.setAttribute('width', width);
    }
    /**
     * Returns source.
     *
     * @returns Source.
     */
    get src() {
        if (!this.hasAttribute('src')) {
            return '';
        }
        try {
            return new URL(this.getAttribute('src'), this[PropertySymbol.ownerDocument].location.href)
                .href;
        }
        catch (e) {
            return this.getAttribute('src');
        }
    }
    /**
     * Sets source.
     *
     * @param src Source.
     */
    set src(src) {
        this.setAttribute('src', src);
    }
    /**
     * Returns type.
     *
     * @returns Type.
     */
    get type() {
        return this.getAttribute('type') || '';
    }
    /**
     * Sets type.
     *
     * @param type Type.
     */
    set type(type) {
        this.setAttribute('type', type);
    }
}
//# sourceMappingURL=HTMLEmbedElement.js.map