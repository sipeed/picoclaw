"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const CSSStyleDeclarationPropertySetParser_js_1 = __importDefault(require("./CSSStyleDeclarationPropertySetParser.cjs"));
const CSSStyleDeclarationValueParser_js_1 = __importDefault(require("./CSSStyleDeclarationValueParser.cjs"));
const CSSStyleDeclarationPropertyGetParser_js_1 = __importDefault(require("./CSSStyleDeclarationPropertyGetParser.cjs"));
const CSSStyleDeclarationCSSParser_js_1 = __importDefault(require("../css-parser/CSSStyleDeclarationCSSParser.cjs"));
const TO_STRING_SHORTHAND_PROPERTIES = [
    ['margin'],
    ['padding'],
    ['border', ['border-width', 'border-style', 'border-color', 'border-image']],
    ['border-radius'],
    ['background', 'background-position'],
    ['font']
];
/**
 * Computed this.properties property parser.
 */
class CSSStyleDeclarationPropertyManager {
    properties = {};
    definedPropertyNames = {};
    /**
     * Class construtor.
     *
     * @param [options] Options.
     * @param [options.cssText] CSS string.
     */
    constructor(options) {
        if (options?.cssText) {
            const { rules } = CSSStyleDeclarationCSSParser_js_1.default.parse(options.cssText);
            for (const rule of rules) {
                if (rule.important || !this.get(rule.name)?.important) {
                    this.set(rule.name, rule.value, rule.important);
                }
            }
        }
    }
    /**
     * Returns property value.
     *
     * @param name Property name.
     * @returns Property value.
     */
    get(name) {
        if (this.properties[name]) {
            return this.properties[name];
        }
        switch (name) {
            case 'margin':
                return CSSStyleDeclarationPropertyGetParser_js_1.default.getMargin(this.properties);
            case 'padding':
                return CSSStyleDeclarationPropertyGetParser_js_1.default.getPadding(this.properties);
            case 'border':
                return CSSStyleDeclarationPropertyGetParser_js_1.default.getBorder(this.properties);
            case 'border-top':
                return CSSStyleDeclarationPropertyGetParser_js_1.default.getBorderTop(this.properties);
            case 'border-right':
                return CSSStyleDeclarationPropertyGetParser_js_1.default.getBorderRight(this.properties);
            case 'border-bottom':
                return CSSStyleDeclarationPropertyGetParser_js_1.default.getBorderBottom(this.properties);
            case 'border-left':
                return CSSStyleDeclarationPropertyGetParser_js_1.default.getBorderLeft(this.properties);
            case 'border-color':
                return CSSStyleDeclarationPropertyGetParser_js_1.default.getBorderColor(this.properties);
            case 'border-style':
                return CSSStyleDeclarationPropertyGetParser_js_1.default.getBorderStyle(this.properties);
            case 'border-width':
                return CSSStyleDeclarationPropertyGetParser_js_1.default.getBorderWidth(this.properties);
            case 'border-radius':
                return CSSStyleDeclarationPropertyGetParser_js_1.default.getBorderRadius(this.properties);
            case 'border-image':
                return CSSStyleDeclarationPropertyGetParser_js_1.default.getBorderImage(this.properties);
            case 'outline':
                return CSSStyleDeclarationPropertyGetParser_js_1.default.getOutline(this.properties);
            case 'background':
                return CSSStyleDeclarationPropertyGetParser_js_1.default.getBackground(this.properties);
            case 'background-position':
                return CSSStyleDeclarationPropertyGetParser_js_1.default.getBackgroundPosition(this.properties);
            case 'flex':
                return CSSStyleDeclarationPropertyGetParser_js_1.default.getFlex(this.properties);
            case 'font':
                return CSSStyleDeclarationPropertyGetParser_js_1.default.getFont(this.properties);
        }
        return this.properties[name] || null;
    }
    /**
     * Removes a property.
     *
     * @param name Property name.
     */
    remove(name) {
        delete this.properties[name];
        delete this.definedPropertyNames[name];
        switch (name) {
            case 'border':
                delete this.properties['border-top-width'];
                delete this.properties['border-right-width'];
                delete this.properties['border-bottom-width'];
                delete this.properties['border-left-width'];
                delete this.properties['border-top-style'];
                delete this.properties['border-right-style'];
                delete this.properties['border-bottom-style'];
                delete this.properties['border-left-style'];
                delete this.properties['border-top-color'];
                delete this.properties['border-right-color'];
                delete this.properties['border-bottom-color'];
                delete this.properties['border-left-color'];
                delete this.properties['border-image-source'];
                delete this.properties['border-image-slice'];
                delete this.properties['border-image-width'];
                delete this.properties['border-image-outset'];
                delete this.properties['border-image-repeat'];
                break;
            case 'border-top':
                delete this.properties['border-top-width'];
                delete this.properties['border-top-style'];
                delete this.properties['border-top-color'];
                delete this.properties['border-image-source'];
                delete this.properties['border-image-slice'];
                delete this.properties['border-image-width'];
                delete this.properties['border-image-outset'];
                delete this.properties['border-image-repeat'];
                break;
            case 'border-right':
                delete this.properties['border-right-width'];
                delete this.properties['border-right-style'];
                delete this.properties['border-right-color'];
                delete this.properties['border-image-source'];
                delete this.properties['border-image-slice'];
                delete this.properties['border-image-width'];
                delete this.properties['border-image-outset'];
                delete this.properties['border-image-repeat'];
                break;
            case 'border-bottom':
                delete this.properties['border-bottom-width'];
                delete this.properties['border-bottom-style'];
                delete this.properties['border-bottom-color'];
                delete this.properties['border-image-source'];
                delete this.properties['border-image-slice'];
                delete this.properties['border-image-width'];
                delete this.properties['border-image-outset'];
                delete this.properties['border-image-repeat'];
                break;
            case 'border-left':
                delete this.properties['border-left-width'];
                delete this.properties['border-left-style'];
                delete this.properties['border-left-color'];
                delete this.properties['border-image-source'];
                delete this.properties['border-image-slice'];
                delete this.properties['border-image-width'];
                delete this.properties['border-image-outset'];
                delete this.properties['border-image-repeat'];
                break;
            case 'border-width':
                delete this.properties['border-top-width'];
                delete this.properties['border-right-width'];
                delete this.properties['border-bottom-width'];
                delete this.properties['border-left-width'];
                break;
            case 'border-style':
                delete this.properties['border-top-style'];
                delete this.properties['border-right-style'];
                delete this.properties['border-bottom-style'];
                delete this.properties['border-left-style'];
                break;
            case 'border-color':
                delete this.properties['border-top-color'];
                delete this.properties['border-right-color'];
                delete this.properties['border-bottom-color'];
                delete this.properties['border-left-color'];
                break;
            case 'border-image':
                delete this.properties['border-image-source'];
                delete this.properties['border-image-slice'];
                delete this.properties['border-image-width'];
                delete this.properties['border-image-outset'];
                delete this.properties['border-image-repeat'];
                break;
            case 'border-radius':
                delete this.properties['border-top-left-radius'];
                delete this.properties['border-top-right-radius'];
                delete this.properties['border-bottom-right-radius'];
                delete this.properties['border-bottom-left-radius'];
                break;
            case 'outline':
                delete this.properties['outline-color'];
                delete this.properties['outline-style'];
                delete this.properties['outline-width'];
                break;
            case 'background':
                delete this.properties['background-color'];
                delete this.properties['background-image'];
                delete this.properties['background-repeat'];
                delete this.properties['background-attachment'];
                delete this.properties['background-position-x'];
                delete this.properties['background-position-y'];
                delete this.properties['background-size'];
                delete this.properties['background-origin'];
                delete this.properties['background-clip'];
                break;
            case 'background-position':
                delete this.properties['background-position-x'];
                delete this.properties['background-position-y'];
                break;
            case 'flex':
                delete this.properties['flex-grow'];
                delete this.properties['flex-shrink'];
                delete this.properties['flex-basis'];
                break;
            case 'font':
                delete this.properties['font-style'];
                delete this.properties['font-variant'];
                delete this.properties['font-weight'];
                delete this.properties['font-stretch'];
                delete this.properties['font-size'];
                delete this.properties['line-height'];
                delete this.properties['font-family'];
                break;
            case 'padding':
                delete this.properties['padding-top'];
                delete this.properties['padding-right'];
                delete this.properties['padding-bottom'];
                delete this.properties['padding-left'];
                break;
            case 'margin':
                delete this.properties['margin-top'];
                delete this.properties['margin-right'];
                delete this.properties['margin-bottom'];
                delete this.properties['margin-left'];
                break;
        }
    }
    /**
     * Sets a property
     *
     * @param name Name.
     * @param value Value.
     * @param important Important.
     */
    set(name, value, important) {
        if (value === null) {
            this.remove(name);
            return;
        }
        let properties = null;
        switch (name) {
            case 'border':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getBorder(value, important);
                break;
            case 'border-top':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getBorderTop(value, important);
                break;
            case 'border-right':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getBorderRight(value, important);
                break;
            case 'border-bottom':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getBorderBottom(value, important);
                break;
            case 'border-left':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getBorderLeft(value, important);
                break;
            case 'border-width':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getBorderWidth(value, important);
                break;
            case 'border-style':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getBorderStyle(value, important);
                break;
            case 'border-color':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getBorderColor(value, important);
                break;
            case 'border-image':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getBorderImage(value, important);
                break;
            case 'border-image-source':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getBorderImageSource(value, important);
                break;
            case 'border-image-slice':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getBorderImageSlice(value, important);
                break;
            case 'border-image-width':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getBorderImageWidth(value, important);
                break;
            case 'border-image-outset':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getBorderImageOutset(value, important);
                break;
            case 'border-image-repeat':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getBorderImageRepeat(value, important);
                break;
            case 'border-top-width':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getBorderTopWidth(value, important);
                break;
            case 'border-right-width':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getBorderRightWidth(value, important);
                break;
            case 'border-bottom-width':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getBorderBottomWidth(value, important);
                break;
            case 'border-left-width':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getBorderLeftWidth(value, important);
                break;
            case 'border-top-color':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getBorderTopColor(value, important);
                break;
            case 'border-right-color':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getBorderRightColor(value, important);
                break;
            case 'border-bottom-color':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getBorderBottomColor(value, important);
                break;
            case 'border-left-color':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getBorderLeftColor(value, important);
                break;
            case 'border-top-style':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getBorderTopStyle(value, important);
                break;
            case 'border-right-style':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getBorderRightStyle(value, important);
                break;
            case 'border-bottom-style':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getBorderBottomStyle(value, important);
                break;
            case 'border-left-style':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getBorderLeftStyle(value, important);
                break;
            case 'border-radius':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getBorderRadius(value, important);
                break;
            case 'border-top-left-radius':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getBorderTopLeftRadius(value, important);
                break;
            case 'border-top-right-radius':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getBorderTopRightRadius(value, important);
                break;
            case 'border-bottom-right-radius':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getBorderBottomRightRadius(value, important);
                break;
            case 'border-bottom-left-radius':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getBorderBottomLeftRadius(value, important);
                break;
            case 'border-collapse':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getBorderCollapse(value, important);
                break;
            case 'outline':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getOutline(value, important);
                break;
            case 'outline-width':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getOutlineWidth(value, important);
                break;
            case 'outline-style':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getOutlineStyle(value, important);
                break;
            case 'outline-color':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getOutlineColor(value, important);
                break;
            case 'letter-spacing':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getLetterSpacing(value, important);
                break;
            case 'word-spacing':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getWordSpacing(value, important);
                break;
            case 'clear':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getClear(value, important);
                break;
            case 'clip':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getClip(value, important);
                break;
            case 'css-float':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getCSSFloat(value, important);
                break;
            case 'float':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getFloat(value, important);
                break;
            case 'display':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getDisplay(value, important);
                break;
            case 'direction':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getDirection(value, important);
                break;
            case 'flex':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getFlex(value, important);
                break;
            case 'flex-shrink':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getFlexShrink(value, important);
                break;
            case 'flex-grow':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getFlexGrow(value, important);
                break;
            case 'flex-basis':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getFlexBasis(value, important);
                break;
            case 'padding':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getPadding(value, important);
                break;
            case 'padding-top':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getPaddingTop(value, important);
                break;
            case 'padding-right':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getPaddingRight(value, important);
                break;
            case 'padding-bottom':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getPaddingBottom(value, important);
                break;
            case 'padding-left':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getPaddingLeft(value, important);
                break;
            case 'margin':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getMargin(value, important);
                break;
            case 'margin-top':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getMarginTop(value, important);
                break;
            case 'margin-right':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getMarginRight(value, important);
                break;
            case 'margin-bottom':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getMarginBottom(value, important);
                break;
            case 'margin-left':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getMarginLeft(value, important);
                break;
            case 'background':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getBackground(value, important);
                break;
            case 'background-image':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getBackgroundImage(value, important);
                break;
            case 'background-color':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getBackgroundColor(value, important);
                break;
            case 'background-repeat':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getBackgroundRepeat(value, important);
                break;
            case 'background-attachment':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getBackgroundAttachment(value, important);
                break;
            case 'background-position':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getBackgroundPosition(value, important);
                break;
            case 'width':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getWidth(value, important);
                break;
            case 'height':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getHeight(value, important);
                break;
            case 'top':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getTop(value, important);
                break;
            case 'right':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getRight(value, important);
                break;
            case 'bottom':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getBottom(value, important);
                break;
            case 'left':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getLeft(value, important);
                break;
            case 'font':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getFont(value, important);
                break;
            case 'font-style':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getFontStyle(value, important);
                break;
            case 'font-variant':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getFontVariant(value, important);
                break;
            case 'font-weight':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getFontWeight(value, important);
                break;
            case 'font-stretch':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getFontStretch(value, important);
                break;
            case 'font-size':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getFontSize(value, important);
                break;
            case 'line-height':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getLineHeight(value, important);
                break;
            case 'text-indent':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getTextIndent(value, important);
                break;
            case 'font-family':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getFontFamily(value, important);
                break;
            case 'color':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getColor(value, important);
                break;
            case 'flood-color':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getFloodColor(value, important);
                break;
            case 'text-transform':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getTextTransform(value, important);
                break;
            case 'visibility':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getVisibility(value, important);
                break;
            case 'aspect-ratio':
                properties = CSSStyleDeclarationPropertySetParser_js_1.default.getAspectRatio(value, important);
                break;
            default:
                const trimmedValue = value.trim();
                if (trimmedValue) {
                    const globalValue = CSSStyleDeclarationValueParser_js_1.default.getGlobal(trimmedValue);
                    properties = {
                        [name]: { value: globalValue || trimmedValue, important }
                    };
                }
                break;
        }
        if (properties !== null && Object.keys(properties).length > 0) {
            this.definedPropertyNames[name] = true;
            Object.assign(this.properties, properties);
        }
    }
    /**
     * Returns a clone.
     *
     * @returns Clone.
     */
    clone() {
        const _class = this.constructor;
        const clone = new _class();
        clone.properties = JSON.parse(JSON.stringify(this.properties));
        clone.definedPropertyNames = Object.assign({}, this.definedPropertyNames);
        return clone;
    }
    /**
     * Returns size.
     *
     * @returns Size.
     */
    size() {
        return Object.keys(this.properties).length;
    }
    /**
     * Returns property name.
     *
     * @param index Index.
     * @returns Property name.
     */
    item(index) {
        return Object.keys(this.properties)[index] || '';
    }
    /**
     * Converts properties to string.
     *
     * @returns String.
     */
    toString() {
        const result = [];
        const clone = this.clone();
        const properties = {};
        for (const shorthandPropertyGroup of TO_STRING_SHORTHAND_PROPERTIES) {
            for (const shorthandProperty of shorthandPropertyGroup) {
                if (Array.isArray(shorthandProperty)) {
                    let isMatch = false;
                    for (const childShorthandProperty of shorthandProperty) {
                        const property = clone.get(childShorthandProperty);
                        if (property) {
                            properties[childShorthandProperty] = property;
                            clone.remove(childShorthandProperty);
                            isMatch = true;
                        }
                    }
                    if (isMatch) {
                        break;
                    }
                }
                else {
                    const property = clone.get(shorthandProperty);
                    if (property) {
                        properties[shorthandProperty] = property;
                        clone.remove(shorthandProperty);
                        break;
                    }
                }
            }
        }
        for (const name of Object.keys(clone.properties)) {
            properties[name] = clone.get(name);
        }
        for (const definedPropertyName of Object.keys(this.definedPropertyNames)) {
            const property = properties[definedPropertyName];
            if (property) {
                result.push(`${definedPropertyName}: ${property.value}${property.important ? ' !important' : ''};`);
                delete properties[definedPropertyName];
            }
        }
        for (const propertyName of Object.keys(properties)) {
            const property = properties[propertyName];
            if (property) {
                result.push(`${propertyName}: ${property.value}${property.important ? ' !important' : ''};`);
            }
        }
        return result.join(' ');
    }
}
exports.default = CSSStyleDeclarationPropertyManager;
//# sourceMappingURL=CSSStyleDeclarationPropertyManager.cjs.map