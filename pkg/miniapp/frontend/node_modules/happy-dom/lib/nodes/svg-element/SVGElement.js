import CSSStyleDeclaration from '../../css/declaration/CSSStyleDeclaration.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import Element from '../element/Element.js';
import HTMLElementUtility from '../html-element/HTMLElementUtility.js';
import DOMStringMap from '../../dom/DOMStringMap.js';
/**
 * SVG Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/SVGElement.
 */
export default class SVGElement extends Element {
    // Events
    onabort = null;
    onerror = null;
    onload = null;
    onresize = null;
    onscroll = null;
    onunload = null;
    // Internal properties
    [PropertySymbol.style] = null;
    // Private properties
    #dataset = null;
    /**
     * Returns current translate.
     *
     * @returns Element.
     */
    get ownerSVGElement() {
        let parent = this[PropertySymbol.parentNode];
        while (parent) {
            if (parent[PropertySymbol.localName] === 'svg') {
                return parent;
            }
            parent = parent[PropertySymbol.parentNode];
        }
        return null;
    }
    /**
     * Returns the SVGElement which established the current viewport. Often the nearest ancestor <svg> element. null if the given element is the outermost <svg> element.
     *
     * @returns SVG element.
     */
    get viewportElement() {
        return this.ownerSVGElement;
    }
    /**
     * Returns data set.
     *
     * @returns Data set.
     */
    get dataset() {
        return (this.#dataset ??= new DOMStringMap(PropertySymbol.illegalConstructor, this));
    }
    /**
     * Returns style.
     *
     * @returns Style.
     */
    get style() {
        if (!this[PropertySymbol.style]) {
            this[PropertySymbol.style] = new CSSStyleDeclaration(PropertySymbol.illegalConstructor, this[PropertySymbol.window], { element: this });
        }
        return this[PropertySymbol.style];
    }
    /**
     * Returns tab index.
     *
     * @returns Tab index.
     */
    get tabIndex() {
        const tabIndex = this.getAttribute('tabindex');
        return tabIndex !== null ? Number(tabIndex) : -1;
    }
    /**
     * Returns tab index.
     *
     * @param tabIndex Tab index.
     */
    set tabIndex(tabIndex) {
        if (tabIndex === -1) {
            this.removeAttribute('tabindex');
        }
        else {
            this.setAttribute('tabindex', String(tabIndex));
        }
    }
    /**
     * Triggers a blur event.
     */
    blur() {
        HTMLElementUtility.blur(this);
    }
    /**
     * Triggers a focus event.
     */
    focus() {
        HTMLElementUtility.focus(this);
    }
}
//# sourceMappingURL=SVGElement.js.map