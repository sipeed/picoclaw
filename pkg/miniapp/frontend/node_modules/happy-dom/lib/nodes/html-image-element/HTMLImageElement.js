import HTMLElement from '../html-element/HTMLElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
/**
 * HTML Image Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLImageElement.
 */
export default class HTMLImageElement extends HTMLElement {
    [PropertySymbol.tagName] = 'IMG';
    [PropertySymbol.complete] = false;
    [PropertySymbol.naturalHeight] = 0;
    [PropertySymbol.naturalWidth] = 0;
    [PropertySymbol.loading] = 'auto';
    [PropertySymbol.x] = 0;
    [PropertySymbol.y] = 0;
    /**
     * Returns complete.
     *
     * @returns Complete.
     */
    get complete() {
        return this[PropertySymbol.complete];
    }
    /**
     * Returns natural height.
     *
     * @returns Natural height.
     */
    get naturalHeight() {
        return this[PropertySymbol.naturalHeight];
    }
    /**
     * Returns natural width.
     *
     * @returns Natural width.
     */
    get naturalWidth() {
        return this[PropertySymbol.naturalWidth];
    }
    /**
     * Returns loading.
     *
     * @returns Loading.
     */
    get loading() {
        const loading = this.getAttribute('loading');
        return loading === 'eager' || loading === 'lazy' ? loading : 'auto';
    }
    /**
     * Sets loading.
     *
     * @param loading Loading.
     */
    set loading(loading) {
        this.setAttribute('loading', loading);
    }
    /**
     * Returns x.
     */
    get x() {
        return this[PropertySymbol.x];
    }
    /**
     * Returns y.
     */
    get y() {
        return this[PropertySymbol.y];
    }
    /**
     * Returns decoding.
     *
     * @returns Decoding.
     */
    get decoding() {
        return this.getAttribute('decoding') || 'auto';
    }
    /**
     * Sets decoding.
     *
     * @param decoding Decoding.
     */
    set decoding(decoding) {
        this.setAttribute('decoding', decoding);
    }
    /**
     * Returns cross origin.
     *
     * @returns Cross origin.
     */
    get crossOrigin() {
        return this.getAttribute('crossOrigin');
    }
    /**
     * Sets cross origin.
     *
     * @param crossOrigin Cross origin.
     */
    set crossOrigin(crossOrigin) {
        if (crossOrigin === 'anonymous' || crossOrigin === 'use-credentials') {
            this.setAttribute('crossOrigin', crossOrigin);
        }
    }
    /**
     * Returns alt.
     *
     * @returns Alt.
     */
    get alt() {
        return this.getAttribute('alt') || '';
    }
    /**
     * Sets alt.
     *
     * @param alt Alt.
     */
    set alt(alt) {
        this.setAttribute('alt', alt);
    }
    /**
     * Returns current src.
     *
     * @returns Current src.
     */
    get currentSrc() {
        return this.src;
    }
    /**
     * Returns width.
     *
     * @returns Width.
     */
    get width() {
        const width = this.getAttribute('width');
        return width !== null ? Number(width) : 0;
    }
    /**
     * Sets width.
     *
     * @param width Width.
     */
    set width(width) {
        this.setAttribute('width', String(width));
    }
    /**
     * Returns height.
     *
     * @returns Height.
     */
    get height() {
        const height = this.getAttribute('height');
        return height !== null ? Number(height) : 0;
    }
    /**
     * Sets height.
     *
     * @param height Height.
     */
    set height(height) {
        this.setAttribute('height', String(height));
    }
    /**
     * Returns is map.
     *
     * @returns Is map.
     */
    get isMap() {
        return this.getAttribute('ismap') !== null;
    }
    /**
     * Sets is map.
     *
     * @param ismap Is map.
     */
    set isMap(isMap) {
        if (!isMap) {
            this.removeAttribute('ismap');
        }
        else {
            this.setAttribute('ismap', '');
        }
    }
    /**
     * Returns referrer policy.
     *
     * @returns Referrer policy.
     */
    get referrerPolicy() {
        return this.getAttribute('referrerpolicy') || '';
    }
    /**
     * Sets referrer policy.
     *
     * @param referrerPolicy Referrer policy.
     */
    set referrerPolicy(referrerPolicy) {
        this.setAttribute('referrerpolicy', referrerPolicy);
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
     * @param sizes Sizes.
     */
    set sizes(sizes) {
        this.setAttribute('sizes', sizes);
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
     * Returns srcset.
     *
     * @returns Source.
     */
    get srcset() {
        return this.getAttribute('srcset') || '';
    }
    /**
     * Sets src set.
     *
     * @param srcset Src set.
     */
    set srcset(srcset) {
        this.setAttribute('srcset', srcset);
    }
    /**
     * Returns use map.
     *
     * @returns Use map.
     */
    get useMap() {
        return this.getAttribute('usemap') || '';
    }
    /**
     * Sets is map.
     *
     * @param useMap Is map.
     */
    set useMap(useMap) {
        this.setAttribute('usemap', useMap);
    }
    /**
     * The decode() method of the HTMLImageElement interface returns a Promise that resolves when the image is decoded and it is safe to append the image to the DOM.
     *
     * @returns Promise.
     */
    decode() {
        return Promise.resolve();
    }
    /**
     * @override
     */
    [PropertySymbol.cloneNode](deep = false) {
        return super[PropertySymbol.cloneNode](deep);
    }
}
//# sourceMappingURL=HTMLImageElement.js.map