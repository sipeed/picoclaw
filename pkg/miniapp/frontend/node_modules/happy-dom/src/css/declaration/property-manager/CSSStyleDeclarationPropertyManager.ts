import ICSSStyleDeclarationPropertyValue from './ICSSStyleDeclarationPropertyValue.js';
import CSSStyleDeclarationPropertySetParser from './CSSStyleDeclarationPropertySetParser.js';
import CSSStyleDeclarationValueParser from './CSSStyleDeclarationValueParser.js';
import CSSStyleDeclarationPropertyGetParser from './CSSStyleDeclarationPropertyGetParser.js';
import CSSStyleDeclarationCSSParser from '../css-parser/CSSStyleDeclarationCSSParser.js';

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
export default class CSSStyleDeclarationPropertyManager {
	public properties: {
		[k: string]: ICSSStyleDeclarationPropertyValue;
	} = {};
	private definedPropertyNames: { [k: string]: boolean } = {};

	/**
	 * Class construtor.
	 *
	 * @param [options] Options.
	 * @param [options.cssText] CSS string.
	 */
	constructor(options?: { cssText?: string }) {
		if (options?.cssText) {
			const { rules } = CSSStyleDeclarationCSSParser.parse(options.cssText);
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
	public get(name: string): ICSSStyleDeclarationPropertyValue | null {
		if (this.properties[name]) {
			return this.properties[name];
		}
		switch (name) {
			case 'margin':
				return CSSStyleDeclarationPropertyGetParser.getMargin(this.properties);
			case 'padding':
				return CSSStyleDeclarationPropertyGetParser.getPadding(this.properties);
			case 'border':
				return CSSStyleDeclarationPropertyGetParser.getBorder(this.properties);
			case 'border-top':
				return CSSStyleDeclarationPropertyGetParser.getBorderTop(this.properties);
			case 'border-right':
				return CSSStyleDeclarationPropertyGetParser.getBorderRight(this.properties);
			case 'border-bottom':
				return CSSStyleDeclarationPropertyGetParser.getBorderBottom(this.properties);
			case 'border-left':
				return CSSStyleDeclarationPropertyGetParser.getBorderLeft(this.properties);
			case 'border-color':
				return CSSStyleDeclarationPropertyGetParser.getBorderColor(this.properties);
			case 'border-style':
				return CSSStyleDeclarationPropertyGetParser.getBorderStyle(this.properties);
			case 'border-width':
				return CSSStyleDeclarationPropertyGetParser.getBorderWidth(this.properties);
			case 'border-radius':
				return CSSStyleDeclarationPropertyGetParser.getBorderRadius(this.properties);
			case 'border-image':
				return CSSStyleDeclarationPropertyGetParser.getBorderImage(this.properties);
			case 'outline':
				return CSSStyleDeclarationPropertyGetParser.getOutline(this.properties);
			case 'background':
				return CSSStyleDeclarationPropertyGetParser.getBackground(this.properties);
			case 'background-position':
				return CSSStyleDeclarationPropertyGetParser.getBackgroundPosition(this.properties);
			case 'flex':
				return CSSStyleDeclarationPropertyGetParser.getFlex(this.properties);
			case 'font':
				return CSSStyleDeclarationPropertyGetParser.getFont(this.properties);
		}

		return this.properties[name] || null;
	}

	/**
	 * Removes a property.
	 *
	 * @param name Property name.
	 */
	public remove(name: string): void {
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
	public set(name: string, value: string, important: boolean): void {
		if (value === null) {
			this.remove(name);
			return;
		}

		let properties = null;

		switch (name) {
			case 'border':
				properties = CSSStyleDeclarationPropertySetParser.getBorder(value, important);
				break;
			case 'border-top':
				properties = CSSStyleDeclarationPropertySetParser.getBorderTop(value, important);
				break;
			case 'border-right':
				properties = CSSStyleDeclarationPropertySetParser.getBorderRight(value, important);
				break;
			case 'border-bottom':
				properties = CSSStyleDeclarationPropertySetParser.getBorderBottom(value, important);
				break;
			case 'border-left':
				properties = CSSStyleDeclarationPropertySetParser.getBorderLeft(value, important);
				break;
			case 'border-width':
				properties = CSSStyleDeclarationPropertySetParser.getBorderWidth(value, important);
				break;
			case 'border-style':
				properties = CSSStyleDeclarationPropertySetParser.getBorderStyle(value, important);
				break;
			case 'border-color':
				properties = CSSStyleDeclarationPropertySetParser.getBorderColor(value, important);
				break;
			case 'border-image':
				properties = CSSStyleDeclarationPropertySetParser.getBorderImage(value, important);
				break;
			case 'border-image-source':
				properties = CSSStyleDeclarationPropertySetParser.getBorderImageSource(value, important);
				break;
			case 'border-image-slice':
				properties = CSSStyleDeclarationPropertySetParser.getBorderImageSlice(value, important);
				break;
			case 'border-image-width':
				properties = CSSStyleDeclarationPropertySetParser.getBorderImageWidth(value, important);
				break;
			case 'border-image-outset':
				properties = CSSStyleDeclarationPropertySetParser.getBorderImageOutset(value, important);
				break;
			case 'border-image-repeat':
				properties = CSSStyleDeclarationPropertySetParser.getBorderImageRepeat(value, important);
				break;
			case 'border-top-width':
				properties = CSSStyleDeclarationPropertySetParser.getBorderTopWidth(value, important);
				break;
			case 'border-right-width':
				properties = CSSStyleDeclarationPropertySetParser.getBorderRightWidth(value, important);
				break;
			case 'border-bottom-width':
				properties = CSSStyleDeclarationPropertySetParser.getBorderBottomWidth(value, important);
				break;
			case 'border-left-width':
				properties = CSSStyleDeclarationPropertySetParser.getBorderLeftWidth(value, important);
				break;
			case 'border-top-color':
				properties = CSSStyleDeclarationPropertySetParser.getBorderTopColor(value, important);
				break;
			case 'border-right-color':
				properties = CSSStyleDeclarationPropertySetParser.getBorderRightColor(value, important);
				break;
			case 'border-bottom-color':
				properties = CSSStyleDeclarationPropertySetParser.getBorderBottomColor(value, important);
				break;
			case 'border-left-color':
				properties = CSSStyleDeclarationPropertySetParser.getBorderLeftColor(value, important);
				break;
			case 'border-top-style':
				properties = CSSStyleDeclarationPropertySetParser.getBorderTopStyle(value, important);
				break;
			case 'border-right-style':
				properties = CSSStyleDeclarationPropertySetParser.getBorderRightStyle(value, important);
				break;
			case 'border-bottom-style':
				properties = CSSStyleDeclarationPropertySetParser.getBorderBottomStyle(value, important);
				break;
			case 'border-left-style':
				properties = CSSStyleDeclarationPropertySetParser.getBorderLeftStyle(value, important);
				break;
			case 'border-radius':
				properties = CSSStyleDeclarationPropertySetParser.getBorderRadius(value, important);
				break;
			case 'border-top-left-radius':
				properties = CSSStyleDeclarationPropertySetParser.getBorderTopLeftRadius(value, important);
				break;
			case 'border-top-right-radius':
				properties = CSSStyleDeclarationPropertySetParser.getBorderTopRightRadius(value, important);
				break;
			case 'border-bottom-right-radius':
				properties = CSSStyleDeclarationPropertySetParser.getBorderBottomRightRadius(
					value,
					important
				);
				break;
			case 'border-bottom-left-radius':
				properties = CSSStyleDeclarationPropertySetParser.getBorderBottomLeftRadius(
					value,
					important
				);
				break;
			case 'border-collapse':
				properties = CSSStyleDeclarationPropertySetParser.getBorderCollapse(value, important);
				break;
			case 'outline':
				properties = CSSStyleDeclarationPropertySetParser.getOutline(value, important);
				break;
			case 'outline-width':
				properties = CSSStyleDeclarationPropertySetParser.getOutlineWidth(value, important);
				break;
			case 'outline-style':
				properties = CSSStyleDeclarationPropertySetParser.getOutlineStyle(value, important);
				break;
			case 'outline-color':
				properties = CSSStyleDeclarationPropertySetParser.getOutlineColor(value, important);
				break;
			case 'letter-spacing':
				properties = CSSStyleDeclarationPropertySetParser.getLetterSpacing(value, important);
				break;
			case 'word-spacing':
				properties = CSSStyleDeclarationPropertySetParser.getWordSpacing(value, important);
				break;
			case 'clear':
				properties = CSSStyleDeclarationPropertySetParser.getClear(value, important);
				break;
			case 'clip':
				properties = CSSStyleDeclarationPropertySetParser.getClip(value, important);
				break;
			case 'css-float':
				properties = CSSStyleDeclarationPropertySetParser.getCSSFloat(value, important);
				break;
			case 'float':
				properties = CSSStyleDeclarationPropertySetParser.getFloat(value, important);
				break;
			case 'display':
				properties = CSSStyleDeclarationPropertySetParser.getDisplay(value, important);
				break;
			case 'direction':
				properties = CSSStyleDeclarationPropertySetParser.getDirection(value, important);
				break;
			case 'flex':
				properties = CSSStyleDeclarationPropertySetParser.getFlex(value, important);
				break;
			case 'flex-shrink':
				properties = CSSStyleDeclarationPropertySetParser.getFlexShrink(value, important);
				break;
			case 'flex-grow':
				properties = CSSStyleDeclarationPropertySetParser.getFlexGrow(value, important);
				break;
			case 'flex-basis':
				properties = CSSStyleDeclarationPropertySetParser.getFlexBasis(value, important);
				break;
			case 'padding':
				properties = CSSStyleDeclarationPropertySetParser.getPadding(value, important);
				break;
			case 'padding-top':
				properties = CSSStyleDeclarationPropertySetParser.getPaddingTop(value, important);
				break;
			case 'padding-right':
				properties = CSSStyleDeclarationPropertySetParser.getPaddingRight(value, important);
				break;
			case 'padding-bottom':
				properties = CSSStyleDeclarationPropertySetParser.getPaddingBottom(value, important);
				break;
			case 'padding-left':
				properties = CSSStyleDeclarationPropertySetParser.getPaddingLeft(value, important);
				break;
			case 'margin':
				properties = CSSStyleDeclarationPropertySetParser.getMargin(value, important);
				break;
			case 'margin-top':
				properties = CSSStyleDeclarationPropertySetParser.getMarginTop(value, important);
				break;
			case 'margin-right':
				properties = CSSStyleDeclarationPropertySetParser.getMarginRight(value, important);
				break;
			case 'margin-bottom':
				properties = CSSStyleDeclarationPropertySetParser.getMarginBottom(value, important);
				break;
			case 'margin-left':
				properties = CSSStyleDeclarationPropertySetParser.getMarginLeft(value, important);
				break;
			case 'background':
				properties = CSSStyleDeclarationPropertySetParser.getBackground(value, important);
				break;
			case 'background-image':
				properties = CSSStyleDeclarationPropertySetParser.getBackgroundImage(value, important);
				break;
			case 'background-color':
				properties = CSSStyleDeclarationPropertySetParser.getBackgroundColor(value, important);
				break;
			case 'background-repeat':
				properties = CSSStyleDeclarationPropertySetParser.getBackgroundRepeat(value, important);
				break;
			case 'background-attachment':
				properties = CSSStyleDeclarationPropertySetParser.getBackgroundAttachment(value, important);
				break;
			case 'background-position':
				properties = CSSStyleDeclarationPropertySetParser.getBackgroundPosition(value, important);
				break;
			case 'width':
				properties = CSSStyleDeclarationPropertySetParser.getWidth(value, important);
				break;
			case 'height':
				properties = CSSStyleDeclarationPropertySetParser.getHeight(value, important);
				break;
			case 'top':
				properties = CSSStyleDeclarationPropertySetParser.getTop(value, important);
				break;
			case 'right':
				properties = CSSStyleDeclarationPropertySetParser.getRight(value, important);
				break;
			case 'bottom':
				properties = CSSStyleDeclarationPropertySetParser.getBottom(value, important);
				break;
			case 'left':
				properties = CSSStyleDeclarationPropertySetParser.getLeft(value, important);
				break;
			case 'font':
				properties = CSSStyleDeclarationPropertySetParser.getFont(value, important);
				break;
			case 'font-style':
				properties = CSSStyleDeclarationPropertySetParser.getFontStyle(value, important);
				break;
			case 'font-variant':
				properties = CSSStyleDeclarationPropertySetParser.getFontVariant(value, important);
				break;
			case 'font-weight':
				properties = CSSStyleDeclarationPropertySetParser.getFontWeight(value, important);
				break;
			case 'font-stretch':
				properties = CSSStyleDeclarationPropertySetParser.getFontStretch(value, important);
				break;
			case 'font-size':
				properties = CSSStyleDeclarationPropertySetParser.getFontSize(value, important);
				break;
			case 'line-height':
				properties = CSSStyleDeclarationPropertySetParser.getLineHeight(value, important);
				break;
			case 'text-indent':
				properties = CSSStyleDeclarationPropertySetParser.getTextIndent(value, important);
				break;
			case 'font-family':
				properties = CSSStyleDeclarationPropertySetParser.getFontFamily(value, important);
				break;
			case 'color':
				properties = CSSStyleDeclarationPropertySetParser.getColor(value, important);
				break;
			case 'flood-color':
				properties = CSSStyleDeclarationPropertySetParser.getFloodColor(value, important);
				break;
			case 'text-transform':
				properties = CSSStyleDeclarationPropertySetParser.getTextTransform(value, important);
				break;
			case 'visibility':
				properties = CSSStyleDeclarationPropertySetParser.getVisibility(value, important);
				break;
			case 'aspect-ratio':
				properties = CSSStyleDeclarationPropertySetParser.getAspectRatio(value, important);
				break;

			default:
				const trimmedValue = value.trim();
				if (trimmedValue) {
					const globalValue = CSSStyleDeclarationValueParser.getGlobal(trimmedValue);
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
	public clone(): CSSStyleDeclarationPropertyManager {
		const _class = <typeof CSSStyleDeclarationPropertyManager>this.constructor;
		const clone: CSSStyleDeclarationPropertyManager = new _class();

		clone.properties = JSON.parse(JSON.stringify(this.properties));
		clone.definedPropertyNames = Object.assign({}, this.definedPropertyNames);

		return clone;
	}

	/**
	 * Returns size.
	 *
	 * @returns Size.
	 */
	public size(): number {
		return Object.keys(this.properties).length;
	}

	/**
	 * Returns property name.
	 *
	 * @param index Index.
	 * @returns Property name.
	 */
	public item(index: number): string {
		return Object.keys(this.properties)[index] || '';
	}

	/**
	 * Converts properties to string.
	 *
	 * @returns String.
	 */
	public toString(): string {
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
				} else {
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
				result.push(
					`${definedPropertyName}: ${property.value}${property.important ? ' !important' : ''};`
				);
				delete properties[definedPropertyName];
			}
		}

		for (const propertyName of Object.keys(properties)) {
			const property = properties[propertyName];
			if (property) {
				result.push(
					`${propertyName}: ${property.value}${property.important ? ' !important' : ''};`
				);
			}
		}

		return result.join(' ');
	}
}
