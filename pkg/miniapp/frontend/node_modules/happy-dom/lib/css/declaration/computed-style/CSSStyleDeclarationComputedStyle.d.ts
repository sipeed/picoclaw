import Element from '../../../nodes/element/Element.js';
import CSSStyleDeclarationPropertyManager from '../property-manager/CSSStyleDeclarationPropertyManager.js';
/**
 * CSS Style Declaration utility
 */
export default class CSSStyleDeclarationComputedStyle {
    private element;
    /**
     * Constructor.
     *
     * @param element Element.
     * @param [computed] Computed.
     */
    constructor(element: Element);
    /**
     * Returns style sheets.
     *
     * @param element Element.
     * @returns Style sheets.
     */
    getComputedStyle(): CSSStyleDeclarationPropertyManager;
    /**
     * Applies CSS text to elements.
     *
     * @param options Options.
     * @param options.elements Elements.
     * @param options.cssRules CSS rules.
     * @param options.rootElement Root element.
     * @param [options.hostElement] Host element.
     */
    private parseCSSRules;
    /**
     * Parses CSS variables in a value.
     *
     * @param value Value.
     * @param cssVariables CSS variables.
     * @returns CSS value.
     */
    private parseCSSVariablesInValue;
    /**
     * Parses measurements in a value.
     *
     * @param options Options.
     * @param options.value Value.
     * @param options.rootFontSize Root font size.
     * @param options.parentFontSize Parent font size.
     * @param [options.parentSize] Parent width.
     * @returns CSS value.
     */
    private parseMeasurementsInValue;
}
//# sourceMappingURL=CSSStyleDeclarationComputedStyle.d.ts.map