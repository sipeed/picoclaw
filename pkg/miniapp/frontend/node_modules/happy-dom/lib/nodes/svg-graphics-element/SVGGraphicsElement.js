import SVGElement from '../svg-element/SVGElement.js';
import DOMRect from '../../dom/DOMRect.js';
import DOMMatrix from '../../dom/dom-matrix/DOMMatrix.js';
import SVGStringList from '../../svg/SVGStringList.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedTransformList from '../../svg/SVGAnimatedTransformList.js';
/**
 * SVG Graphics Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGGraphicsElement
 */
export default class SVGGraphicsElement extends SVGElement {
    // Internal properties
    [PropertySymbol.requiredExtensions] = null;
    [PropertySymbol.systemLanguage] = null;
    [PropertySymbol.transform] = null;
    // Events
    oncopy = null;
    oncut = null;
    onpaste = null;
    /**
     * Returns required extensions.
     *
     * @returns Required extensions.
     */
    get requiredExtensions() {
        if (!this[PropertySymbol.requiredExtensions]) {
            this[PropertySymbol.requiredExtensions] = new SVGStringList(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('requiredExtensions'),
                setAttribute: (value) => this.setAttribute('requiredExtensions', value)
            });
        }
        return this[PropertySymbol.requiredExtensions];
    }
    /**
     * Returns system language.
     *
     * @returns System language.
     */
    get systemLanguage() {
        if (!this[PropertySymbol.systemLanguage]) {
            this[PropertySymbol.systemLanguage] = new SVGStringList(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('systemLanguage'),
                setAttribute: (value) => this.setAttribute('systemLanguage', value)
            });
        }
        return this[PropertySymbol.systemLanguage];
    }
    /**
     * Returns transform.
     *
     * @returns Transform.
     */
    get transform() {
        if (!this[PropertySymbol.transform]) {
            this[PropertySymbol.transform] = new SVGAnimatedTransformList(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('transform'),
                setAttribute: (value) => this.setAttribute('transform', value)
            });
        }
        return this[PropertySymbol.transform];
    }
    /**
     * Returns DOM rect.
     *
     * @returns DOM rect.
     */
    getBBox() {
        return new DOMRect();
    }
    /**
     * Returns CTM.
     *
     * @returns CTM.
     */
    getCTM() {
        return new DOMMatrix();
    }
    /**
     * Returns screen CTM.
     *
     * @returns Screen CTM.
     */
    getScreenCTM() {
        return new DOMMatrix();
    }
}
//# sourceMappingURL=SVGGraphicsElement.js.map