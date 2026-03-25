import HTMLElement from '../html-element/HTMLElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
/**
 * HTMLSourceElement
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLSourceElement
 */
export default class HTMLSourceElement extends HTMLElement {
    /**
     * Returns height.
     *
     * @returns Height.
     */
    get height() {
        const value = Number(this.getAttribute('height'));
        return isNaN(value) || value < 0 ? 0 : value;
    }
    /**
     * Sets height.
     *
     * @param value Height.
     */
    set height(value) {
        const parsedValue = Number(value);
        this.setAttribute('height', isNaN(parsedValue) || parsedValue < 0 ? '0' : String(parsedValue));
    }
    /**
     * Returns width.
     *
     * @returns Width.
     */
    get width() {
        const value = Number(this.getAttribute('width'));
        return isNaN(value) || value < 0 ? 0 : value;
    }
    /**
     * Sets width.
     *
     * @param value Width.
     */
    set width(value) {
        const parsedValue = Number(value);
        this.setAttribute('width', isNaN(parsedValue) || parsedValue < 0 ? '0' : String(parsedValue));
    }
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
     * @param value Media.
     */
    set media(value) {
        this.setAttribute('media', value);
    }
    /**
     * Returns sizes.
     *
     * @returns Sizes.
     */
    get sizes() {
        return this.getAttribute('sizes') || '';
    }
    /**
     * Sets sizes.
     *
     * @param value Sizes.
     */
    set sizes(value) {
        this.setAttribute('sizes', value);
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
     * Returns source set.
     *
     * @returns Source set.
     */
    get srcset() {
        return this.getAttribute('srcset') || '';
    }
    /**
     * Sets source set.
     *
     * @param value Source set.
     */
    set srcset(value) {
        this.setAttribute('srcset', value);
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
//# sourceMappingURL=HTMLSourceElement.js.map