import * as PropertySymbol from '../../../PropertySymbol.js';
import CSSStyleDeclarationPropertyManager from '../property-manager/CSSStyleDeclarationPropertyManager.js';
import NodeTypeEnum from '../../../nodes/node/NodeTypeEnum.js';
import CSSRuleTypeEnum from '../../CSSRuleTypeEnum.js';
import CSSStyleDeclarationElementDefaultCSS from './config/CSSStyleDeclarationElementDefaultCSS.js';
import CSSStyleDeclarationElementInheritedProperties from './config/CSSStyleDeclarationElementInheritedProperties.js';
import CSSStyleDeclarationElementMeasurementProperties from './config/CSSStyleDeclarationElementMeasurementProperties.js';
import CSSStyleDeclarationCSSParser from '../css-parser/CSSStyleDeclarationCSSParser.js';
import QuerySelector from '../../../query-selector/QuerySelector.js';
import CSSMeasurementConverter from '../measurement-converter/CSSMeasurementConverter.js';
import MediaQueryList from '../../../match-media/MediaQueryList.js';
import WindowBrowserContext from '../../../window/WindowBrowserContext.js';
const CSS_MEASUREMENT_REGEXP = /[0-9.]+(px|rem|em|vw|vh|%|vmin|vmax|cm|mm|in|pt|pc|Q)/g;
const CSS_VARIABLE_REGEXP = /var\( *(--[^), ]+)\)|var\( *(--[^), ]+), *(.+)\)/;
/**
 * CSS Style Declaration utility
 */
export default class CSSStyleDeclarationComputedStyle {
    element;
    /**
     * Constructor.
     *
     * @param element Element.
     * @param [computed] Computed.
     */
    constructor(element) {
        this.element = element;
    }
    /**
     * Returns style sheets.
     *
     * @param element Element.
     * @returns Style sheets.
     */
    getComputedStyle() {
        const documentElements = [];
        const parentElements = [];
        let styleAndElement = {
            element: this.element,
            cssTexts: []
        };
        let shadowRootElements = [];
        if (!this.element[PropertySymbol.isConnected]) {
            return new CSSStyleDeclarationPropertyManager();
        }
        const cacheResult = this.element[PropertySymbol.cache].computedStyle;
        if (cacheResult?.result) {
            const result = cacheResult.result.deref();
            if (result) {
                return result;
            }
        }
        // Walks through all parent elements and stores them in an array with element and matching CSS text.
        while (styleAndElement.element) {
            if (styleAndElement.element[PropertySymbol.nodeType] === NodeTypeEnum.elementNode) {
                const rootNode = styleAndElement.element.getRootNode();
                if (rootNode[PropertySymbol.nodeType] === NodeTypeEnum.documentNode) {
                    documentElements.unshift(styleAndElement);
                }
                else {
                    shadowRootElements.unshift(styleAndElement);
                }
                parentElements.unshift(styleAndElement);
            }
            if (styleAndElement.element === this.element[PropertySymbol.ownerDocument]) {
                const styleSheets = (this.element[PropertySymbol.ownerDocument].querySelectorAll('style,link[rel="stylesheet"]'));
                for (const styleSheet of styleSheets) {
                    const sheet = styleSheet.sheet;
                    if (sheet) {
                        this.parseCSSRules({
                            elements: documentElements,
                            rootElement: documentElements[0].element[PropertySymbol.tagName] === 'HTML'
                                ? documentElements[0]
                                : null,
                            cssRules: sheet.cssRules
                        });
                    }
                }
                for (const sheet of this.element[PropertySymbol.ownerDocument].adoptedStyleSheets) {
                    this.parseCSSRules({
                        elements: documentElements,
                        rootElement: documentElements[0].element[PropertySymbol.tagName] === 'HTML'
                            ? documentElements[0]
                            : null,
                        cssRules: sheet.cssRules
                    });
                }
                styleAndElement = { element: null, cssTexts: [] };
            }
            else if (styleAndElement.element[PropertySymbol.nodeType] === NodeTypeEnum.documentFragmentNode &&
                styleAndElement.element.host) {
                const shadowRoot = styleAndElement.element;
                const styleSheets = (shadowRoot.querySelectorAll('style,link[rel="stylesheet"]'));
                styleAndElement = {
                    element: shadowRoot.host,
                    cssTexts: []
                };
                for (const styleSheet of styleSheets) {
                    const sheet = styleSheet.sheet;
                    if (sheet) {
                        this.parseCSSRules({
                            elements: shadowRootElements,
                            cssRules: sheet.cssRules,
                            hostElement: styleAndElement
                        });
                    }
                }
                for (const sheet of shadowRoot.adoptedStyleSheets) {
                    this.parseCSSRules({
                        elements: shadowRootElements,
                        cssRules: sheet.cssRules,
                        hostElement: styleAndElement
                    });
                }
                shadowRootElements = [];
            }
            else {
                styleAndElement = {
                    element: styleAndElement.element[PropertySymbol.parentNode],
                    cssTexts: []
                };
            }
        }
        // Concatenates all parent element CSS to one string.
        const targetElement = parentElements[parentElements.length - 1];
        const propertyManager = new CSSStyleDeclarationPropertyManager();
        const cssProperties = {};
        let rootFontSize = 16;
        let parentFontSize = 16;
        for (const parentElement of parentElements) {
            parentElement.cssTexts.sort((a, b) => a.priorityWeight - b.priorityWeight);
            let elementCSSText = '';
            if (CSSStyleDeclarationElementDefaultCSS[parentElement.element[PropertySymbol.tagName]]) {
                if (typeof CSSStyleDeclarationElementDefaultCSS[parentElement.element[PropertySymbol.tagName]] === 'string') {
                    elementCSSText +=
                        CSSStyleDeclarationElementDefaultCSS[parentElement.element[PropertySymbol.tagName]];
                }
                else {
                    for (const key of Object.keys(CSSStyleDeclarationElementDefaultCSS[parentElement.element[PropertySymbol.tagName]])) {
                        if (key === 'default' || !!parentElement.element[key]) {
                            elementCSSText +=
                                CSSStyleDeclarationElementDefaultCSS[parentElement.element[PropertySymbol.tagName]][key];
                        }
                    }
                }
                elementCSSText +=
                    CSSStyleDeclarationElementDefaultCSS[parentElement.element[PropertySymbol.tagName]];
            }
            for (const cssText of parentElement.cssTexts) {
                elementCSSText += cssText.cssText;
            }
            const elementStyleAttribute = parentElement.element.getAttribute('style');
            if (elementStyleAttribute) {
                elementCSSText += elementStyleAttribute;
            }
            const rulesAndProperties = CSSStyleDeclarationCSSParser.parse(elementCSSText);
            const rules = rulesAndProperties.rules;
            Object.assign(cssProperties, rulesAndProperties.properties);
            for (const { name, value, important } of rules) {
                if (CSSStyleDeclarationElementInheritedProperties[name] ||
                    parentElement === targetElement) {
                    const parsedValue = this.parseCSSVariablesInValue(value.trim(), cssProperties);
                    if (parsedValue && (!propertyManager.get(name)?.important || important)) {
                        propertyManager.set(name, parsedValue, important);
                        if (name === 'font' || name === 'font-size') {
                            const fontSize = propertyManager.properties['font-size'];
                            if (fontSize !== null) {
                                const parsedValue = this.parseMeasurementsInValue({
                                    value: fontSize.value,
                                    rootFontSize,
                                    parentFontSize,
                                    parentSize: parentFontSize
                                });
                                if (parentElement.element[PropertySymbol.tagName] === 'HTML') {
                                    rootFontSize = parsedValue;
                                }
                                else if (parentElement !== targetElement) {
                                    parentFontSize = parsedValue;
                                }
                            }
                        }
                    }
                }
            }
        }
        for (const name of CSSStyleDeclarationElementMeasurementProperties) {
            const property = propertyManager.properties[name];
            if (property) {
                property.value = this.parseMeasurementsInValue({
                    value: property.value,
                    rootFontSize,
                    parentFontSize,
                    // TODO: Only "font-size" is supported when using percentage values. Add support for other properties.
                    parentSize: name === 'font-size' ? parentFontSize : null
                });
            }
        }
        const cachedResult = {
            result: new WeakRef(propertyManager)
        };
        this.element[PropertySymbol.cache].computedStyle = cachedResult;
        this.element[PropertySymbol.ownerDocument][PropertySymbol.affectsComputedStyleCache].push(cachedResult);
        return propertyManager;
    }
    /**
     * Applies CSS text to elements.
     *
     * @param options Options.
     * @param options.elements Elements.
     * @param options.cssRules CSS rules.
     * @param options.rootElement Root element.
     * @param [options.hostElement] Host element.
     */
    parseCSSRules(options) {
        if (!options.elements.length) {
            return;
        }
        const window = this.element[PropertySymbol.window];
        for (const rule of options.cssRules) {
            if (rule.type === CSSRuleTypeEnum.styleRule) {
                const selectorText = rule.selectorText;
                if (selectorText) {
                    if (selectorText.startsWith(':host')) {
                        if (options.hostElement) {
                            options.hostElement.cssTexts.push({
                                cssText: rule[PropertySymbol.cssText],
                                priorityWeight: 0
                            });
                        }
                    }
                    else if (selectorText.startsWith(':root')) {
                        if (options.rootElement) {
                            options.rootElement.cssTexts.push({
                                cssText: rule[PropertySymbol.cssText],
                                priorityWeight: 0
                            });
                        }
                    }
                    else {
                        for (const element of options.elements) {
                            const match = QuerySelector.matches(element.element, selectorText, {
                                ignoreErrors: true
                            });
                            if (match) {
                                element.cssTexts.push({
                                    cssText: rule[PropertySymbol.cssText],
                                    priorityWeight: match.priorityWeight
                                });
                            }
                        }
                    }
                }
            }
            else if (rule.type === CSSRuleTypeEnum.mediaRule &&
                // TODO: We need to send in a predfined root font size as it will otherwise be calculated using Window.getComputedStyle(), which will cause a never ending loop. Is there another solution?
                new MediaQueryList({
                    window,
                    media: rule.conditionText,
                    rootFontSize: this.element[PropertySymbol.tagName] === 'HTML' ? 16 : null
                }).matches) {
                this.parseCSSRules({
                    elements: options.elements,
                    cssRules: rule.cssRules,
                    hostElement: options.hostElement
                });
            }
        }
    }
    /**
     * Parses CSS variables in a value.
     *
     * @param value Value.
     * @param cssVariables CSS variables.
     * @returns CSS value.
     */
    parseCSSVariablesInValue(value, cssVariables) {
        let newValue = value;
        let match;
        while ((match = newValue.match(CSS_VARIABLE_REGEXP)) !== null) {
            // Fallback value - E.g. var(--my-var, #FFFFFF)
            if (match[2] !== undefined) {
                newValue = newValue.replace(match[0], cssVariables[match[2]] || match[3]);
            }
            else {
                newValue = newValue.replace(match[0], cssVariables[match[1]] || '');
            }
        }
        return newValue;
    }
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
    parseMeasurementsInValue(options) {
        if (new WindowBrowserContext(this.element[PropertySymbol.window]).getSettings()
            ?.disableComputedStyleRendering) {
            return options.value;
        }
        const regexp = new RegExp(CSS_MEASUREMENT_REGEXP);
        let newValue = options.value;
        let match;
        while ((match = regexp.exec(options.value)) !== null) {
            if (match[1] !== 'px') {
                const valueInPixels = CSSMeasurementConverter.toPixels({
                    window: this.element[PropertySymbol.window],
                    value: match[0],
                    rootFontSize: options.rootFontSize,
                    parentFontSize: options.parentFontSize,
                    parentSize: options.parentSize
                });
                if (valueInPixels !== null) {
                    newValue = newValue.replace(match[0], valueInPixels + 'px');
                }
            }
        }
        return newValue;
    }
}
//# sourceMappingURL=CSSStyleDeclarationComputedStyle.js.map