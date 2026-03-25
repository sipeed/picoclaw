import SVGElement from '../svg-element/SVGElement.js';
import SVGStringList from '../../svg/SVGStringList.js';
import * as PropertySymbol from '../../PropertySymbol.js';
/**
 * SVG Animation Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGAnimationElement
 */
export default class SVGAnimationElement extends SVGElement {
    // Internal properties
    [PropertySymbol.requiredExtensions] = null;
    [PropertySymbol.systemLanguage] = null;
    // Events
    onbegin = null;
    onend = null;
    onrepeat = null;
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
     * Returns target element.
     *
     * @returns Target element.
     */
    get targetElement() {
        // TODO: Implement targetElement
        return null;
    }
}
//# sourceMappingURL=SVGAnimationElement.js.map