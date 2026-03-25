import CSSStyleDeclarationValueParser from './CSSStyleDeclarationValueParser.js';
import ICSSStyleDeclarationPropertyValue from './ICSSStyleDeclarationPropertyValue.js';

const RECT_REGEXP = /^rect\((.*)\)$/i;
const SPLIT_COMMA_SEPARATED_WITH_PARANTHESES_REGEXP = /,(?=(?:(?:(?!\))[\s\S])*\()|[^\(\)]*$)/; // Split on commas that are outside of parentheses
const SPLIT_SPACE_SEPARATED_WITH_PARANTHESES_REGEXP = /\s+(?=(?:(?:(?!\))[\s\S])*\()|[^\(\)]*$)/; // Split on spaces that are outside of parentheses
const WHITE_SPACE_GLOBAL_REGEXP = /\s+/gm;
const BORDER_STYLE = [
	'none',
	'hidden',
	'dotted',
	'dashed',
	'solid',
	'double',
	'groove',
	'ridge',
	'inset',
	'outset'
];
const BORDER_WIDTH = ['thin', 'medium', 'thick'];
const BORDER_COLLAPSE = ['separate', 'collapse'];
const BACKGROUND_REPEAT = ['repeat', 'repeat-x', 'repeat-y', 'no-repeat'];
const BACKGROUND_ORIGIN = ['border-box', 'padding-box', 'content-box'];
const BACKGROUND_CLIP = ['border-box', 'padding-box', 'content-box'];
const BACKGROUND_ATTACHMENT = ['scroll', 'fixed'];
const FLEX_BASIS = ['auto', 'fill', 'content'];
const CLEAR = ['none', 'left', 'right', 'both'];
const FLOAT = ['none', 'left', 'right', 'inline-start', 'inline-end'];
const SYSTEM_FONT = ['caption', 'icon', 'menu', 'message-box', 'small-caption', 'status-bar'];
const FONT_WEIGHT = ['normal', 'bold', 'bolder', 'lighter'];
const FONT_STYLE = ['normal', 'italic', 'oblique'];
const FONT_SIZE = [
	'xx-small',
	'x-small',
	'small',
	'medium',
	'large',
	'x-large',
	'xx-large',
	'xxx-large',
	'smaller',
	'larger'
];
const FONT_STRETCH = [
	'ultra-condensed',
	'extra-condensed',
	'condensed',
	'semi-condensed',
	'normal',
	'semi-expanded',
	'expanded',
	'extra-expanded',
	'ultra-expanded'
];

const DISPLAY = [
	/* Legacy values */
	'block',
	'inline',
	'inline-block',
	'flex',
	'inline-flex',
	'grid',
	'inline-grid',
	'flow-root',

	/* Box generation */
	'none',
	'contents',

	/* Two-value syntax */
	'block flow',
	'inline flow',
	'inline flow-root',
	'block flex',
	'inline flex',
	'block grid',
	'inline grid',
	'block flow-root',

	/* Other values */
	'table',
	'table-row',
	'list-item'
];
const BORDER_IMAGE_REPEAT = ['stretch', 'repeat', 'round', 'space'];
const TEXT_TRANSFORM = [
	'capitalize',
	'uppercase',
	'lowercase',
	'none',
	'full-width',
	'full-size-kana'
];
const VISIBILITY = ['visible', 'hidden', 'collapse'];

/**
 * Computed style property parser.
 */
export default class CSSStyleDeclarationPropertySetParser {
	/**
	 * Returns border collapse.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getBorderCollapse(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'border-collapse': { value: variable, important } };
		}
		const lowerValue = value.toLowerCase();
		if (
			CSSStyleDeclarationValueParser.getGlobal(lowerValue) ||
			BORDER_COLLAPSE.includes(lowerValue)
		) {
			return { 'border-collapse': { value: lowerValue, important } };
		}
		return null;
	}

	/**
	 * Returns display.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getDisplay(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { display: { value: variable, important } };
		}
		const lowerValue = value.toLowerCase();
		if (CSSStyleDeclarationValueParser.getGlobal(lowerValue) || DISPLAY.includes(lowerValue)) {
			return { display: { value: lowerValue, important } };
		}
		return null;
	}

	/**
	 * Returns direction.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getDirection(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { direction: { value: variable, important } };
		}
		const lowerValue = value.toLowerCase();
		if (
			CSSStyleDeclarationValueParser.getGlobal(lowerValue) ||
			lowerValue === 'ltr' ||
			lowerValue === 'rtl'
		) {
			return { direction: { value: lowerValue, important } };
		}
		return null;
	}

	/**
	 * Returns letter spacing.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getLetterSpacing(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const parsedValue =
			CSSStyleDeclarationValueParser.getVariable(value) ||
			CSSStyleDeclarationValueParser.getGlobal(value) ||
			CSSStyleDeclarationValueParser.getContentMeasurement(value);
		return parsedValue ? { 'letter-spacing': { value: parsedValue, important } } : null;
	}

	/**
	 * Returns word spacing.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getWordSpacing(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const parsedValue =
			CSSStyleDeclarationValueParser.getVariable(value) ||
			CSSStyleDeclarationValueParser.getGlobal(value) ||
			CSSStyleDeclarationValueParser.getContentMeasurement(value);
		return parsedValue ? { 'word-spacing': { value: parsedValue, important } } : null;
	}

	/**
	 * Returns text indent.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getTextIndent(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const parsedValue =
			CSSStyleDeclarationValueParser.getVariable(value) ||
			CSSStyleDeclarationValueParser.getGlobal(value) ||
			CSSStyleDeclarationValueParser.getContentMeasurement(value);
		return parsedValue ? { 'text-indent': { value: parsedValue, important } } : null;
	}

	/**
	 * Returns width.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getWidth(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const parsedValue =
			CSSStyleDeclarationValueParser.getVariable(value) ||
			CSSStyleDeclarationValueParser.getGlobal(value) ||
			CSSStyleDeclarationValueParser.getContentMeasurement(value);
		return parsedValue ? { width: { value: parsedValue, important } } : null;
	}

	/**
	 * Returns height.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getHeight(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const parsedValue =
			CSSStyleDeclarationValueParser.getVariable(value) ||
			CSSStyleDeclarationValueParser.getGlobal(value) ||
			CSSStyleDeclarationValueParser.getContentMeasurement(value);
		return parsedValue ? { height: { value: parsedValue, important } } : null;
	}

	/**
	 * Returns top.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getTop(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const parsedValue =
			CSSStyleDeclarationValueParser.getVariable(value) ||
			CSSStyleDeclarationValueParser.getGlobal(value) ||
			CSSStyleDeclarationValueParser.getContentMeasurement(value);
		return parsedValue ? { top: { value: parsedValue, important } } : null;
	}

	/**
	 * Returns top.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getRight(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const parsedValue =
			CSSStyleDeclarationValueParser.getVariable(value) ||
			CSSStyleDeclarationValueParser.getGlobal(value) ||
			CSSStyleDeclarationValueParser.getContentMeasurement(value);
		return parsedValue ? { right: { value: parsedValue, important } } : null;
	}

	/**
	 * Returns top.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getBottom(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const parsedValue =
			CSSStyleDeclarationValueParser.getVariable(value) ||
			CSSStyleDeclarationValueParser.getGlobal(value) ||
			CSSStyleDeclarationValueParser.getContentMeasurement(value);
		return parsedValue ? { bottom: { value: parsedValue, important } } : null;
	}

	/**
	 * Returns top.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getLeft(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const parsedValue =
			CSSStyleDeclarationValueParser.getVariable(value) ||
			CSSStyleDeclarationValueParser.getGlobal(value) ||
			CSSStyleDeclarationValueParser.getContentMeasurement(value);
		return parsedValue ? { left: { value: parsedValue, important } } : null;
	}

	/**
	 * Returns clear.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getClear(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { clear: { value: variable, important } };
		}
		const lowerValue = value.toLowerCase();
		if (CSSStyleDeclarationValueParser.getGlobal(lowerValue) || CLEAR.includes(lowerValue)) {
			return { clear: { value: lowerValue, important } };
		}
		return null;
	}

	/**
	 * Returns clip
	 *
	 * Based on:
	 * https://github.com/jsdom/cssstyle/blob/master/lib/properties/clip.js
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getClip(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { clip: { value: variable, important } };
		}
		const lowerValue = value.toLowerCase();
		if (CSSStyleDeclarationValueParser.getGlobal(lowerValue) || lowerValue === 'auto') {
			return { clip: { value: lowerValue, important } };
		}
		const matches = lowerValue.match(RECT_REGEXP);
		if (!matches) {
			return null;
		}
		const parts = matches[1].split(/\s*,\s*/);
		if (parts.length !== 4) {
			return null;
		}
		for (const part of parts) {
			if (!CSSStyleDeclarationValueParser.getMeasurement(part)) {
				return null;
			}
		}
		return { clip: { value, important } };
	}

	/**
	 * Returns float.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getFloat(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { float: { value: variable, important } };
		}
		const lowerValue = value.toLowerCase();
		if (CSSStyleDeclarationValueParser.getGlobal(lowerValue) || FLOAT.includes(lowerValue)) {
			return { float: { value: lowerValue, important } };
		}
		return null;
	}

	/**
	 * Returns float.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getCSSFloat(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'css-float': { value: variable, important } };
		}
		const float = this.getFloat(value, important);
		return float ? { 'css-float': float['float'] } : null;
	}

	/**
	 * Returns outline.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values.
	 */
	public static getOutline(
		value: string,
		important: boolean
	): { [key: string]: ICSSStyleDeclarationPropertyValue } {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { outline: { value: variable, important } };
		}

		const globalValue = CSSStyleDeclarationValueParser.getGlobal(value);

		if (globalValue) {
			return {
				...this.getOutlineColor(globalValue, important),
				...this.getOutlineStyle(globalValue, important),
				...this.getOutlineWidth(globalValue, important)
			};
		}

		const properties = {
			...this.getOutlineColor('initial', important),
			...this.getOutlineStyle('initial', important),
			...this.getOutlineWidth('initial', important)
		};

		const parts = value.split(SPLIT_SPACE_SEPARATED_WITH_PARANTHESES_REGEXP);

		for (const part of parts) {
			const width = this.getOutlineWidth(part, important);
			const style = this.getOutlineStyle(part, important);
			const color = this.getOutlineColor(part, important);

			if (width === null && style === null && color === null) {
				return null;
			}

			Object.assign(properties, width, style, color);
		}

		return properties;
	}

	/**
	 * Returns outline color.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getOutlineColor(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const color =
			CSSStyleDeclarationValueParser.getVariable(value) ||
			CSSStyleDeclarationValueParser.getGlobal(value) ||
			CSSStyleDeclarationValueParser.getColor(value);
		return color
			? {
					'outline-color': { value: color, important }
				}
			: null;
	}

	/**
	 * Returns outline offset.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getOutlineOffset(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const parsedValue =
			CSSStyleDeclarationValueParser.getVariable(value) ||
			CSSStyleDeclarationValueParser.getLength(value);
		return parsedValue ? { 'outline-offset': { value: parsedValue, important } } : null;
	}

	/**
	 * Returns outline style.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getOutlineStyle(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'outline-style': { value: variable, important } };
		}

		const lowerValue = value.toLowerCase();
		if (CSSStyleDeclarationValueParser.getGlobal(lowerValue) || BORDER_STYLE.includes(lowerValue)) {
			return {
				'outline-style': { value: lowerValue, important }
			};
		}
		return null;
	}

	/**
	 * Returns outline width.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getOutlineWidth(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'outline-width': { value: variable, important } };
		}

		const lowerValue = value.toLowerCase();
		const parsedValue =
			BORDER_WIDTH.includes(lowerValue) || CSSStyleDeclarationValueParser.getGlobal(lowerValue)
				? lowerValue
				: CSSStyleDeclarationValueParser.getLength(value);
		if (parsedValue) {
			return {
				'outline-width': { value: parsedValue, important }
			};
		}
		return null;
	}

	/**
	 * Returns border.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values.
	 */
	public static getBorder(
		value: string,
		important: boolean
	): { [key: string]: ICSSStyleDeclarationPropertyValue } {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { border: { value: variable, important } };
		}

		const globalValue = CSSStyleDeclarationValueParser.getGlobal(value);

		if (globalValue) {
			return {
				...this.getBorderWidth(globalValue, important),
				...this.getBorderStyle(globalValue, important),
				...this.getBorderColor(globalValue, important),
				...this.getBorderImage(globalValue, important)
			};
		}

		const properties = {
			...this.getBorderWidth('initial', important),
			...this.getBorderStyle('initial', important),
			...this.getBorderColor('initial', important),
			...this.getBorderImage('initial', important)
		};

		const parts = value
			.replace(/\s*,\s*/g, ',')
			.split(SPLIT_SPACE_SEPARATED_WITH_PARANTHESES_REGEXP);

		for (const part of parts) {
			const width = this.getBorderWidth(part, important);
			const style = this.getBorderStyle(part, important);
			const color = this.getBorderColor(part, important);

			if (width === null && style === null && color === null) {
				return null;
			}

			Object.assign(properties, width, style, color);
		}

		return properties;
	}

	/**
	 * Returns border width.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getBorderWidth(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'border-width': { value: variable, important } };
		}

		const globalValue = CSSStyleDeclarationValueParser.getGlobal(value);

		if (globalValue) {
			return {
				...this.getBorderTopWidth(globalValue, important),
				...this.getBorderRightWidth(globalValue, important),
				...this.getBorderBottomWidth(globalValue, important),
				...this.getBorderLeftWidth(globalValue, important)
			};
		}

		const parts = value.split(SPLIT_SPACE_SEPARATED_WITH_PARANTHESES_REGEXP);
		const top = this.getBorderTopWidth(parts[0], important);
		const right = this.getBorderRightWidth(parts[1] || parts[0], important);
		const bottom = this.getBorderBottomWidth(parts[2] || parts[0], important);
		const left = this.getBorderLeftWidth(parts[3] || parts[1] || parts[0], important);

		if (!top || !right || !bottom || !left) {
			return null;
		}

		return {
			...top,
			...right,
			...bottom,
			...left
		};
	}
	/**
	 * Returns border style.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getBorderStyle(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'border-style': { value: variable, important } };
		}

		const globalValue = CSSStyleDeclarationValueParser.getGlobal(value);

		if (globalValue) {
			return {
				...this.getBorderTopStyle(globalValue, important),
				...this.getBorderRightStyle(globalValue, important),
				...this.getBorderBottomStyle(globalValue, important),
				...this.getBorderLeftStyle(globalValue, important)
			};
		}

		const parts = value.split(SPLIT_SPACE_SEPARATED_WITH_PARANTHESES_REGEXP);
		const top = this.getBorderTopStyle(parts[0], important);
		const right = this.getBorderRightStyle(parts[1] || parts[0], important);
		const bottom = this.getBorderBottomStyle(parts[2] || parts[0], important);
		const left = this.getBorderLeftStyle(parts[3] || parts[1] || parts[0], important);

		if (!top || !right || !bottom || !left) {
			return null;
		}

		return {
			...top,
			...right,
			...bottom,
			...left
		};
	}

	/**
	 * Returns border color.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getBorderColor(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'border-color': { value: variable, important } };
		}

		const globalValue = CSSStyleDeclarationValueParser.getGlobal(value);

		if (globalValue) {
			return {
				...this.getBorderTopColor(globalValue, important),
				...this.getBorderRightColor(globalValue, important),
				...this.getBorderBottomColor(globalValue, important),
				...this.getBorderLeftColor(globalValue, important)
			};
		}

		const parts = value.split(SPLIT_SPACE_SEPARATED_WITH_PARANTHESES_REGEXP);
		const top = this.getBorderTopColor(parts[0], important);
		const right = this.getBorderRightColor(parts[1] || parts[0], important);
		const bottom = this.getBorderBottomColor(parts[2] || parts[0], important);
		const left = this.getBorderLeftColor(parts[3] || parts[1] || parts[0], important);

		if (!top || !right || !bottom || !left) {
			return null;
		}

		return {
			...top,
			...right,
			...bottom,
			...left
		};
	}

	/**
	 * Returns border image.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getBorderImage(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'border-image': { value: variable, important } };
		}

		const globalValue = CSSStyleDeclarationValueParser.getGlobal(value);

		if (globalValue) {
			return {
				...this.getBorderImageSource(globalValue, important),
				...this.getBorderImageSlice(globalValue, important),
				...this.getBorderImageWidth(globalValue, important),
				...this.getBorderImageOutset(globalValue, important),
				...this.getBorderImageRepeat(globalValue, important)
			};
		}

		let parsedValue = value.replace(/\s\/\s/g, '/');
		const sourceMatch = parsedValue.match(/\s*([a-zA-Z-]+\([^)]*\))\s*/);

		if (sourceMatch) {
			parsedValue = parsedValue.replace(sourceMatch[0], '');
		}

		const parts = parsedValue.split(SPLIT_SPACE_SEPARATED_WITH_PARANTHESES_REGEXP);

		if (sourceMatch) {
			parts.push(sourceMatch[1]);
		}

		const properties = {
			...this.getBorderImageSource('none', important),
			...this.getBorderImageSlice('100%', important),
			...this.getBorderImageWidth('1', important),
			...this.getBorderImageOutset('0', important),
			...this.getBorderImageRepeat('stretch', important)
		};

		for (let i = 0, max = parts.length; i < max; i++) {
			const part = parts[i];
			const previousPart = i > 0 ? parts[i - 1] : '';

			if (!part.startsWith('url') && part.includes('/')) {
				const [slice, width, outset] = part.split('/');
				const borderImageSlice =
					this.getBorderImageSlice(`${previousPart} ${slice}`, important) ||
					this.getBorderImageSlice(slice, important);
				const borderImageWidth = this.getBorderImageWidth(width, important);
				const borderImageOutset = outset && this.getBorderImageOutset(outset, important);

				if (!borderImageSlice || !borderImageWidth || borderImageOutset === null) {
					return null;
				}

				Object.assign(properties, borderImageSlice, borderImageWidth, borderImageOutset);
			} else {
				const slice =
					this.getBorderImageSlice(`${previousPart} ${part}`, important) ||
					this.getBorderImageSlice(part, important);
				const source = this.getBorderImageSource(part, important);
				const repeat = this.getBorderImageRepeat(part, important);

				if (!slice && !source && !repeat) {
					return null;
				}

				Object.assign(properties, slice, source, repeat);
			}
		}

		return properties;
	}

	/**
	 * Returns border source.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getBorderImageSource(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'border-image-source': { value: variable, important } };
		}

		const lowerValue = value.toLowerCase();

		if (CSSStyleDeclarationValueParser.getGlobal(lowerValue) || lowerValue === 'none') {
			return {
				'border-image-source': {
					important,
					value: lowerValue
				}
			};
		}

		const parsedValue =
			CSSStyleDeclarationValueParser.getURL(value) ||
			CSSStyleDeclarationValueParser.getGradient(value);

		if (!parsedValue) {
			return null;
		}

		return {
			'border-image-source': {
				important,
				value: parsedValue
			}
		};
	}

	/**
	 * Returns border slice.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getBorderImageSlice(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'border-image-slice': { value: variable, important } };
		}

		const lowerValue = value.toLowerCase();

		if (CSSStyleDeclarationValueParser.getGlobal(lowerValue)) {
			return {
				'border-image-slice': {
					important,
					value: lowerValue
				}
			};
		}

		if (lowerValue !== lowerValue.trim()) {
			return null;
		}

		const regexp = /(fill)|(calc\([^^)]+\))|([0-9]+%)|([0-9]+)/g;
		const values = [];
		let match;

		while ((match = regexp.exec(lowerValue))) {
			const previousCharacter = lowerValue[match.index - 1];
			const nextCharacter = lowerValue[match.index + match[0].length];

			if (
				(previousCharacter && previousCharacter !== ' ') ||
				(nextCharacter && nextCharacter !== ' ')
			) {
				return null;
			}

			const fill = match[1] && 'fill';
			const calc = match[2] && CSSStyleDeclarationValueParser.getCalc(match[2]);
			const percentage = match[3] && CSSStyleDeclarationValueParser.getPercentage(match[3]);
			const integer = match[4] && CSSStyleDeclarationValueParser.getInteger(match[4]);

			if (!fill && !calc && !percentage && !integer) {
				return null;
			}

			values.push(fill || calc || percentage || integer);
		}

		if (!values.length || values.length > 4) {
			return null;
		}

		return {
			'border-image-slice': {
				important,
				value: values.join(' ')
			}
		};
	}

	/**
	 * Returns border width.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getBorderImageWidth(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'border-image-width': { value: variable, important } };
		}

		const lowerValue = value.toLowerCase();

		if (CSSStyleDeclarationValueParser.getGlobal(lowerValue)) {
			return {
				'border-image-width': {
					important,
					value: lowerValue
				}
			};
		}

		const parts = lowerValue.split(SPLIT_SPACE_SEPARATED_WITH_PARANTHESES_REGEXP);

		if (parts.length > 4) {
			return null;
		}

		for (const part of parts) {
			if (
				!CSSStyleDeclarationValueParser.getInteger(part) &&
				!CSSStyleDeclarationValueParser.getAutoMeasurement(part)
			) {
				return null;
			}
		}

		return {
			'border-image-width': {
				important,
				value
			}
		};
	}

	/**
	 * Returns border outset.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getBorderImageOutset(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		if (value === '0') {
			return {
				'border-image-outset': {
					important,
					value
				}
			};
		}

		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'border-image-outset': { value: variable, important } };
		}

		const lowerValue = value.toLowerCase();

		if (CSSStyleDeclarationValueParser.getGlobal(lowerValue)) {
			return {
				'border-image-outset': {
					important,
					value: lowerValue
				}
			};
		}

		const parts = value.split(SPLIT_SPACE_SEPARATED_WITH_PARANTHESES_REGEXP);

		if (parts.length > 4) {
			return null;
		}

		const newParts = [];

		for (const part of parts) {
			const parsedValue =
				CSSStyleDeclarationValueParser.getLength(part) ||
				CSSStyleDeclarationValueParser.getFloat(part);
			if (!parsedValue) {
				return null;
			}
			newParts.push(parsedValue === '0px' ? '0' : parsedValue);
		}

		return {
			'border-image-outset': {
				important,
				value: newParts.join(' ')
			}
		};
	}

	/**
	 * Returns border repeat.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getBorderImageRepeat(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'border-image-repeat': { value: variable, important } };
		}

		const lowerValue = value.toLowerCase();

		if (CSSStyleDeclarationValueParser.getGlobal(lowerValue)) {
			return {
				'border-image-repeat': {
					important,
					value: lowerValue
				}
			};
		}

		const parts = lowerValue.split(SPLIT_SPACE_SEPARATED_WITH_PARANTHESES_REGEXP);

		if (parts.length > 2) {
			return null;
		}

		for (const part of parts) {
			if (!BORDER_IMAGE_REPEAT.includes(part)) {
				return null;
			}
		}

		return {
			'border-image-repeat': {
				important,
				value
			}
		};
	}

	/**
	 * Returns border width.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getBorderTopWidth(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'border-top-width': { value: variable, important } };
		}

		const lowerValue = value.toLowerCase();
		const parsedValue =
			BORDER_WIDTH.includes(lowerValue) || CSSStyleDeclarationValueParser.getGlobal(lowerValue)
				? lowerValue
				: CSSStyleDeclarationValueParser.getLength(value);
		if (parsedValue) {
			return {
				'border-top-width': { value: parsedValue, important }
			};
		}
		return null;
	}

	/**
	 * Returns border width.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getBorderRightWidth(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'border-right-width': { value: variable, important } };
		}

		const lowerValue = value.toLowerCase();
		const parsedValue =
			BORDER_WIDTH.includes(lowerValue) || CSSStyleDeclarationValueParser.getGlobal(lowerValue)
				? lowerValue
				: CSSStyleDeclarationValueParser.getLength(value);
		if (parsedValue) {
			return {
				'border-right-width': { value: parsedValue, important }
			};
		}
		return null;
	}

	/**
	 * Returns border width.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getBorderBottomWidth(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'border-bottom-width': { value: variable, important } };
		}

		const lowerValue = value.toLowerCase();
		const parsedValue =
			BORDER_WIDTH.includes(lowerValue) || CSSStyleDeclarationValueParser.getGlobal(lowerValue)
				? lowerValue
				: CSSStyleDeclarationValueParser.getLength(value);
		if (parsedValue) {
			return {
				'border-bottom-width': { value: parsedValue, important }
			};
		}
		return null;
	}

	/**
	 * Returns border width.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getBorderLeftWidth(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'border-left-width': { value: variable, important } };
		}

		const lowerValue = value.toLowerCase();
		const parsedValue =
			BORDER_WIDTH.includes(lowerValue) || CSSStyleDeclarationValueParser.getGlobal(lowerValue)
				? lowerValue
				: CSSStyleDeclarationValueParser.getLength(value);
		if (parsedValue) {
			return {
				'border-left-width': { value: parsedValue, important }
			};
		}
		return null;
	}

	/**
	 * Returns border style.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getBorderTopStyle(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'border-top-style': { value: variable, important } };
		}

		const lowerValue = value.toLowerCase();
		if (CSSStyleDeclarationValueParser.getGlobal(lowerValue) || BORDER_STYLE.includes(lowerValue)) {
			return {
				'border-top-style': { value: lowerValue, important }
			};
		}
		return null;
	}

	/**
	 * Returns border style.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getBorderRightStyle(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'border-right-style': { value: variable, important } };
		}

		const lowerValue = value.toLowerCase();
		if (CSSStyleDeclarationValueParser.getGlobal(lowerValue) || BORDER_STYLE.includes(lowerValue)) {
			return {
				'border-right-style': { value: lowerValue, important }
			};
		}
		return null;
	}

	/**
	 * Returns border style.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getBorderBottomStyle(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'border-bottom-style': { value: variable, important } };
		}

		const lowerValue = value.toLowerCase();
		if (CSSStyleDeclarationValueParser.getGlobal(lowerValue) || BORDER_STYLE.includes(lowerValue)) {
			return {
				'border-bottom-style': { value: lowerValue, important }
			};
		}
		return null;
	}

	/**
	 * Returns border style.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getBorderLeftStyle(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'border-left-style': { value: variable, important } };
		}

		const lowerValue = value.toLowerCase();
		if (CSSStyleDeclarationValueParser.getGlobal(lowerValue) || BORDER_STYLE.includes(lowerValue)) {
			return {
				'border-left-style': { value: lowerValue, important }
			};
		}
		return null;
	}

	/**
	 * Returns border color.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getBorderTopColor(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'border-top-color': { value: variable, important } };
		}

		const color =
			CSSStyleDeclarationValueParser.getGlobal(value) ||
			CSSStyleDeclarationValueParser.getColor(value);
		return color
			? {
					'border-top-color': { value: color, important }
				}
			: null;
	}

	/**
	 * Returns border color.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getBorderRightColor(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'border-right-color': { value: variable, important } };
		}

		const color =
			CSSStyleDeclarationValueParser.getGlobal(value) ||
			CSSStyleDeclarationValueParser.getColor(value);
		return color
			? {
					'border-right-color': { value: color, important }
				}
			: null;
	}

	/**
	 * Returns border color.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getBorderBottomColor(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'border-bottom-color': { value: variable, important } };
		}

		const color =
			CSSStyleDeclarationValueParser.getGlobal(value) ||
			CSSStyleDeclarationValueParser.getColor(value);
		return color
			? {
					'border-bottom-color': { value: color, important }
				}
			: null;
	}

	/**
	 * Returns border color.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getBorderLeftColor(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'border-left-color': { value: variable, important } };
		}

		const color =
			CSSStyleDeclarationValueParser.getGlobal(value) ||
			CSSStyleDeclarationValueParser.getColor(value);
		return color
			? {
					'border-left-color': { value: color, important }
				}
			: null;
	}

	/**
	 * Returns border radius.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values.
	 */
	public static getBorderRadius(
		value: string,
		important: boolean
	): { [key: string]: ICSSStyleDeclarationPropertyValue } {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'border-radius': { value: variable, important } };
		}

		const globalValue = CSSStyleDeclarationValueParser.getGlobal(value);

		if (globalValue) {
			return {
				...this.getBorderTopLeftRadius(globalValue, important),
				...this.getBorderTopRightRadius(globalValue, important),
				...this.getBorderBottomRightRadius(globalValue, important),
				...this.getBorderBottomLeftRadius(globalValue, important)
			};
		}

		const parts = value.split(SPLIT_SPACE_SEPARATED_WITH_PARANTHESES_REGEXP);
		const topLeft = this.getBorderTopLeftRadius(parts[0], important);
		const topRight = this.getBorderTopRightRadius(parts[1] || parts[0], important);
		const bottomRight = this.getBorderBottomRightRadius(parts[2] || parts[0], important);
		const bottomLeft = this.getBorderBottomLeftRadius(parts[3] || parts[1] || parts[0], important);

		if (!topLeft || !topRight || !bottomRight || !bottomLeft) {
			return null;
		}

		return {
			...topLeft,
			...topRight,
			...bottomRight,
			...bottomLeft
		};
	}

	/**
	 * Returns border radius.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values.
	 */
	public static getBorderTopLeftRadius(
		value: string,
		important: boolean
	): { [key: string]: ICSSStyleDeclarationPropertyValue } {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'border-top-left-radius': { value: variable, important } };
		}

		const radius =
			CSSStyleDeclarationValueParser.getGlobal(value) ||
			CSSStyleDeclarationValueParser.getMeasurement(value);
		return radius ? { 'border-top-left-radius': { important, value: radius } } : null;
	}

	/**
	 * Returns border radius.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values.
	 */
	public static getBorderTopRightRadius(
		value: string,
		important: boolean
	): { [key: string]: ICSSStyleDeclarationPropertyValue } {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'border-top-right-radius': { value: variable, important } };
		}

		const radius =
			CSSStyleDeclarationValueParser.getGlobal(value) ||
			CSSStyleDeclarationValueParser.getMeasurement(value);
		return radius ? { 'border-top-right-radius': { important, value: radius } } : null;
	}

	/**
	 * Returns border radius.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values.
	 */
	public static getBorderBottomRightRadius(
		value: string,
		important: boolean
	): { [key: string]: ICSSStyleDeclarationPropertyValue } {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'border-bottom-right-radius': { value: variable, important } };
		}

		const radius =
			CSSStyleDeclarationValueParser.getGlobal(value) ||
			CSSStyleDeclarationValueParser.getMeasurement(value);
		return radius ? { 'border-bottom-right-radius': { important, value: radius } } : null;
	}

	/**
	 * Returns border radius.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values.
	 */
	public static getBorderBottomLeftRadius(
		value: string,
		important: boolean
	): { [key: string]: ICSSStyleDeclarationPropertyValue } {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'border-bottom-left-radius': { value: variable, important } };
		}

		const radius =
			CSSStyleDeclarationValueParser.getGlobal(value) ||
			CSSStyleDeclarationValueParser.getMeasurement(value);
		return radius ? { 'border-bottom-left-radius': { important, value: radius } } : null;
	}

	/**
	 * Returns border top, right, bottom or left.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values.
	 */
	public static getBorderTop(
		value: string,
		important: boolean
	): { [key: string]: ICSSStyleDeclarationPropertyValue } {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'border-top': { value: variable, important } };
		}

		const globalValue = CSSStyleDeclarationValueParser.getGlobal(value);

		if (globalValue) {
			return {
				...this.getBorderTopWidth(globalValue, important),
				...this.getBorderTopStyle(globalValue, important),
				...this.getBorderTopColor(globalValue, important)
			};
		}

		const properties = {
			...this.getBorderTopWidth('initial', important),
			...this.getBorderTopStyle('initial', important),
			...this.getBorderTopColor('initial', important)
		};

		const parts = value.split(SPLIT_SPACE_SEPARATED_WITH_PARANTHESES_REGEXP);

		for (const part of parts) {
			const width = this.getBorderTopWidth(part, important);
			const style = this.getBorderTopStyle(part, important);
			const color = this.getBorderTopColor(part, important);

			if (width === null && style === null && color === null) {
				return null;
			}

			Object.assign(properties, width, style, color);
		}

		return properties;
	}

	/**
	 * Returns border top, right, bottom or left.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values.
	 */
	public static getBorderRight(
		value: string,
		important: boolean
	): { [key: string]: ICSSStyleDeclarationPropertyValue } {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'border-right': { value: variable, important } };
		}

		const globalValue = CSSStyleDeclarationValueParser.getGlobal(value);

		if (globalValue) {
			return {
				...this.getBorderRightWidth(globalValue, important),
				...this.getBorderRightStyle(globalValue, important),
				...this.getBorderRightColor(globalValue, important)
			};
		}

		const properties = {
			...this.getBorderRightWidth('initial', important),
			...this.getBorderRightStyle('initial', important),
			...this.getBorderRightColor('initial', important)
		};

		const parts = value.split(SPLIT_SPACE_SEPARATED_WITH_PARANTHESES_REGEXP);

		for (const part of parts) {
			const width = this.getBorderRightWidth(part, important);
			const style = this.getBorderRightStyle(part, important);
			const color = this.getBorderRightColor(part, important);

			if (width === null && style === null && color === null) {
				return null;
			}

			Object.assign(properties, width, style, color);
		}

		return properties;
	}

	/**
	 * Returns border top, right, bottom or left.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values.
	 */
	public static getBorderBottom(
		value: string,
		important: boolean
	): { [key: string]: ICSSStyleDeclarationPropertyValue } {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'border-bottom': { value: variable, important } };
		}

		const globalValue = CSSStyleDeclarationValueParser.getGlobal(value);

		if (globalValue) {
			return {
				...this.getBorderBottomWidth(globalValue, important),
				...this.getBorderBottomStyle(globalValue, important),
				...this.getBorderBottomColor(globalValue, important)
			};
		}

		const properties = {
			...this.getBorderBottomWidth('initial', important),
			...this.getBorderBottomStyle('initial', important),
			...this.getBorderBottomColor('initial', important)
		};

		const parts = value.split(SPLIT_SPACE_SEPARATED_WITH_PARANTHESES_REGEXP);

		for (const part of parts) {
			const width = this.getBorderBottomWidth(part, important);
			const style = this.getBorderBottomStyle(part, important);
			const color = this.getBorderBottomColor(part, important);

			if (width === null && style === null && color === null) {
				return null;
			}

			Object.assign(properties, width, style, color);
		}

		return properties;
	}

	/**
	 * Returns border top, right, bottom or left.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values.
	 */
	public static getBorderLeft(
		value: string,
		important: boolean
	): { [key: string]: ICSSStyleDeclarationPropertyValue } {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'border-left': { value: variable, important } };
		}

		const globalValue = CSSStyleDeclarationValueParser.getGlobal(value);

		if (globalValue) {
			return {
				...this.getBorderLeftWidth(globalValue, important),
				...this.getBorderLeftStyle(globalValue, important),
				...this.getBorderLeftColor(globalValue, important)
			};
		}

		const properties = {
			...this.getBorderLeftWidth('initial', important),
			...this.getBorderLeftStyle('initial', important),
			...this.getBorderLeftColor('initial', important)
		};

		const parts = value.split(SPLIT_SPACE_SEPARATED_WITH_PARANTHESES_REGEXP);

		for (const part of parts) {
			const width = this.getBorderLeftWidth(part, important);
			const style = this.getBorderLeftStyle(part, important);
			const color = this.getBorderLeftColor(part, important);

			if (width === null && style === null && color === null) {
				return null;
			}

			Object.assign(properties, width, style, color);
		}

		return properties;
	}

	/**
	 * Returns padding.
	 *
	 * @param value Value.
	 * @param important Important.
	 */
	public static getPadding(
		value: string,
		important: boolean
	): { [key: string]: ICSSStyleDeclarationPropertyValue } {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { padding: { value: variable, important } };
		}

		const globalValue = CSSStyleDeclarationValueParser.getGlobal(value);

		if (globalValue) {
			return {
				...this.getPaddingTop(globalValue, important),
				...this.getPaddingRight(globalValue, important),
				...this.getPaddingBottom(globalValue, important),
				...this.getPaddingLeft(globalValue, important)
			};
		}

		const parts = value.split(SPLIT_SPACE_SEPARATED_WITH_PARANTHESES_REGEXP);
		const top = this.getPaddingTop(parts[0], important);
		const right = this.getPaddingRight(parts[1] || parts[0], important);
		const bottom = this.getPaddingBottom(parts[2] || parts[0], important);
		const left = this.getPaddingLeft(parts[3] || parts[1] || parts[0], important);

		if (!top || !right || !bottom || !left) {
			return null;
		}

		return {
			...top,
			...right,
			...bottom,
			...left
		};
	}

	/**
	 * Returns padding top.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values.
	 */
	public static getPaddingTop(
		value: string,
		important: boolean
	): { [key: string]: ICSSStyleDeclarationPropertyValue } {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'padding-top': { value: variable, important } };
		}

		const padding =
			CSSStyleDeclarationValueParser.getGlobal(value) ||
			CSSStyleDeclarationValueParser.getMeasurement(value);
		return padding ? { 'padding-top': { value: padding, important } } : null;
	}

	/**
	 * Returns padding right.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values.
	 */
	public static getPaddingRight(
		value: string,
		important: boolean
	): { [key: string]: ICSSStyleDeclarationPropertyValue } {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'padding-right': { value: variable, important } };
		}

		const padding =
			CSSStyleDeclarationValueParser.getGlobal(value) ||
			CSSStyleDeclarationValueParser.getMeasurement(value);
		return padding ? { 'padding-right': { value: padding, important } } : null;
	}

	/**
	 * Returns padding bottom.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values.
	 */
	public static getPaddingBottom(
		value: string,
		important: boolean
	): { [key: string]: ICSSStyleDeclarationPropertyValue } {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'padding-bottom': { value: variable, important } };
		}

		const padding =
			CSSStyleDeclarationValueParser.getGlobal(value) ||
			CSSStyleDeclarationValueParser.getMeasurement(value);
		return padding ? { 'padding-bottom': { value: padding, important } } : null;
	}

	/**
	 * Returns padding left.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values.
	 */
	public static getPaddingLeft(
		value: string,
		important: boolean
	): { [key: string]: ICSSStyleDeclarationPropertyValue } {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'padding-left': { value: variable, important } };
		}

		const padding =
			CSSStyleDeclarationValueParser.getGlobal(value) ||
			CSSStyleDeclarationValueParser.getMeasurement(value);
		return padding ? { 'padding-left': { value: padding, important } } : null;
	}

	/**
	 * Returns margin.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values.
	 */
	public static getMargin(
		value: string,
		important: boolean
	): { [key: string]: ICSSStyleDeclarationPropertyValue } {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { margin: { value: variable, important } };
		}

		const globalValue = CSSStyleDeclarationValueParser.getGlobal(value);

		if (globalValue) {
			return {
				...this.getMarginTop(globalValue, important),
				...this.getMarginRight(globalValue, important),
				...this.getMarginBottom(globalValue, important),
				...this.getMarginLeft(globalValue, important)
			};
		}

		const parts = value.split(SPLIT_SPACE_SEPARATED_WITH_PARANTHESES_REGEXP);
		const top = this.getMarginTop(parts[0], important);
		const right = this.getMarginRight(parts[1] || parts[0], important);
		const bottom = this.getMarginBottom(parts[2] || parts[0], important);
		const left = this.getMarginLeft(parts[3] || parts[1] || parts[0], important);

		if (!top || !right || !bottom || !left) {
			return null;
		}

		return {
			...top,
			...right,
			...bottom,
			...left
		};
	}

	/**
	 * Returns margin top.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values.
	 */
	public static getMarginTop(
		value: string,
		important: boolean
	): { [key: string]: ICSSStyleDeclarationPropertyValue } {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'margin-top': { value: variable, important } };
		}

		const margin =
			CSSStyleDeclarationValueParser.getGlobal(value) ||
			CSSStyleDeclarationValueParser.getAutoMeasurement(value);
		return margin ? { 'margin-top': { value: margin, important } } : null;
	}

	/**
	 * Returns margin right.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values.
	 */
	public static getMarginRight(
		value: string,
		important: boolean
	): { [key: string]: ICSSStyleDeclarationPropertyValue } {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'margin-right': { value: variable, important } };
		}

		const margin =
			CSSStyleDeclarationValueParser.getGlobal(value) ||
			CSSStyleDeclarationValueParser.getAutoMeasurement(value);
		return margin ? { 'margin-right': { value: margin, important } } : null;
	}

	/**
	 * Returns margin right.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values.
	 */
	public static getMarginBottom(
		value: string,
		important: boolean
	): { [key: string]: ICSSStyleDeclarationPropertyValue } {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'margin-bottom': { value: variable, important } };
		}

		const margin =
			CSSStyleDeclarationValueParser.getGlobal(value) ||
			CSSStyleDeclarationValueParser.getAutoMeasurement(value);
		return margin ? { 'margin-bottom': { value: margin, important } } : null;
	}

	/**
	 * Returns margin left.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values.
	 */
	public static getMarginLeft(
		value: string,
		important: boolean
	): { [key: string]: ICSSStyleDeclarationPropertyValue } {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'margin-left': { value: variable, important } };
		}

		const margin =
			CSSStyleDeclarationValueParser.getGlobal(value) ||
			CSSStyleDeclarationValueParser.getAutoMeasurement(value);
		return margin ? { 'margin-left': { value: margin, important } } : null;
	}

	/**
	 * Returns flex.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values.
	 */
	public static getFlex(
		value: string,
		important: boolean
	): { [key: string]: ICSSStyleDeclarationPropertyValue } {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { flex: { value: variable, important } };
		}

		const lowerValue = value.trim().toLowerCase();
		const globalValue = CSSStyleDeclarationValueParser.getGlobal(value);

		if (globalValue) {
			return {
				...this.getFlexGrow(globalValue, important),
				...this.getFlexShrink(globalValue, important),
				...this.getFlexBasis(globalValue, important)
			};
		}

		switch (lowerValue) {
			case 'none':
				return {
					...this.getFlexGrow('0', important),
					...this.getFlexShrink('0', important),
					...this.getFlexBasis('auto', important)
				};
			case 'auto':
				return {
					...this.getFlexGrow('1', important),
					...this.getFlexShrink('1', important),
					...this.getFlexBasis('auto', important)
				};
		}

		const measurement = CSSStyleDeclarationValueParser.getContentMeasurement(lowerValue);

		if (measurement) {
			return {
				...this.getFlexGrow('1', important),
				...this.getFlexShrink('1', important),
				...this.getFlexBasis(measurement, important)
			};
		}

		const parts = value.split(SPLIT_SPACE_SEPARATED_WITH_PARANTHESES_REGEXP);
		const flexGrow = this.getFlexGrow(parts[0], important);
		const flexShrink = this.getFlexShrink(parts[1] || '1', important);
		const flexBasis = this.getFlexBasis(parts[2] || '0%', important);

		if (!flexGrow || !flexShrink || !flexBasis) {
			return null;
		}

		return {
			...flexGrow,
			...flexShrink,
			...flexBasis
		};
	}

	/**
	 * Returns flex basis.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getFlexBasis(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'flex-basis': { value: variable, important } };
		}

		const lowerValue = value.toLowerCase();
		if (CSSStyleDeclarationValueParser.getGlobal(lowerValue) || FLEX_BASIS.includes(lowerValue)) {
			return { 'flex-basis': { value: lowerValue, important } };
		}
		const measurement = CSSStyleDeclarationValueParser.getContentMeasurement(lowerValue);
		return measurement ? { 'flex-basis': { value: measurement, important } } : null;
	}

	/**
	 * Returns flex shrink.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getFlexShrink(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'flex-shrink': { value: variable, important } };
		}

		const parsedValue =
			CSSStyleDeclarationValueParser.getGlobal(value) ||
			CSSStyleDeclarationValueParser.getFloat(value);
		return parsedValue ? { 'flex-shrink': { value: parsedValue, important } } : null;
	}

	/**
	 * Returns flex grow.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getFlexGrow(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'flex-grow': { value: variable, important } };
		}

		const parsedValue =
			CSSStyleDeclarationValueParser.getGlobal(value) ||
			CSSStyleDeclarationValueParser.getFloat(value);
		return parsedValue ? { 'flex-grow': { value: parsedValue, important } } : null;
	}

	/**
	 * Returns background.
	 *
	 * @param name Name.
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values.
	 */
	public static getBackground(
		value: string,
		important: boolean
	): { [key: string]: ICSSStyleDeclarationPropertyValue } {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { background: { value: variable, important } };
		}

		const globalValue = CSSStyleDeclarationValueParser.getGlobal(value);

		if (globalValue) {
			return {
				...this.getBackgroundImage(globalValue, important),
				...this.getBackgroundPosition(globalValue, important),
				...this.getBackgroundSize(globalValue, important),
				...this.getBackgroundRepeat(globalValue, important),
				...this.getBackgroundAttachment(globalValue, important),
				...this.getBackgroundOrigin(globalValue, important),
				...this.getBackgroundClip(globalValue, important),
				...this.getBackgroundColor(globalValue, important)
			};
		}

		const properties = {
			...this.getBackgroundImage('initial', important),
			...this.getBackgroundPosition('initial', important),
			...this.getBackgroundSize('initial', important),
			...this.getBackgroundRepeat('initial', important),
			...this.getBackgroundAttachment('initial', important),
			...this.getBackgroundOrigin('initial', important),
			...this.getBackgroundClip('initial', important),
			...this.getBackgroundColor('initial', important)
		};

		const parts = value
			.replace(/\s,\s/g, ',')
			.replace(/\s\/\s/g, '/')
			.split(SPLIT_SPACE_SEPARATED_WITH_PARANTHESES_REGEXP);

		const backgroundPositions = [];

		for (const part of parts) {
			if (!part.startsWith('url') && part.includes('/')) {
				const [position, size] = part.split('/');
				const backgroundPositionX = this.getBackgroundPositionX(position, important);
				const backgroundPositionY = this.getBackgroundPositionY(position, important);

				const backgroundSize = this.getBackgroundSize(size, important);

				if ((!backgroundPositionX && !backgroundPositionY) || !backgroundSize) {
					return null;
				}

				if (backgroundPositionY) {
					backgroundPositions.push(backgroundPositionY['background-position-y'].value);
				} else if (backgroundPositionX) {
					backgroundPositions.push(backgroundPositionX['background-position-x'].value);
				}

				Object.assign(properties, backgroundSize);
			} else {
				const backgroundImage = this.getBackgroundImage(part, important);
				const backgroundRepeat = this.getBackgroundRepeat(part, important);
				const backgroundAttachment = this.getBackgroundAttachment(part, important);
				const backgroundPositionX = this.getBackgroundPositionX(part, important);
				const backgroundPositionY = this.getBackgroundPositionY(part, important);
				const backgroundColor = this.getBackgroundColor(part, important);
				const backgroundOrigin = this.getBackgroundOrigin(part, important);
				const backgroundClip = this.getBackgroundClip(part, important);

				if (
					!backgroundImage &&
					!backgroundRepeat &&
					!backgroundAttachment &&
					!backgroundPositionX &&
					!backgroundPositionY &&
					!backgroundColor &&
					!backgroundOrigin &&
					!backgroundClip
				) {
					return null;
				}

				if (backgroundPositionX) {
					backgroundPositions.push(backgroundPositionX['background-position-x'].value);
				} else if (backgroundPositionY) {
					backgroundPositions.push(backgroundPositionY['background-position-y'].value);
				}

				Object.assign(
					properties,
					backgroundImage,
					backgroundRepeat,
					backgroundAttachment,
					backgroundColor,
					backgroundOrigin,
					backgroundClip
				);
			}
		}

		if (backgroundPositions.length) {
			Object.assign(
				properties,
				this.getBackgroundPosition(backgroundPositions.join(' '), important)
			);
		}

		return properties;
	}

	/**
	 * Returns background size.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getBackgroundSize(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'background-size': { value: variable, important } };
		}

		const lowerValue = value.toLowerCase();
		if (CSSStyleDeclarationValueParser.getGlobal(lowerValue)) {
			return { 'background-size': { value: lowerValue, important } };
		}

		const imageParts = lowerValue.split(SPLIT_COMMA_SEPARATED_WITH_PARANTHESES_REGEXP);
		const parsed = [];

		for (const imagePart of imageParts) {
			const parts = imagePart.trim().split(' ');
			if (parts.length !== 1 && parts.length !== 2) {
				return null;
			}
			if (parts.length === 1) {
				if (
					parts[0] !== 'cover' &&
					parts[0] !== 'contain' &&
					!CSSStyleDeclarationValueParser.getAutoMeasurement(parts[0])
				) {
					return null;
				}
				parsed.push(parts[0]);
			} else {
				if (
					!CSSStyleDeclarationValueParser.getAutoMeasurement(parts[0]) ||
					!CSSStyleDeclarationValueParser.getAutoMeasurement(parts[1])
				) {
					return null;
				}
				parsed.push(`${parts[0]} ${parts[1]}`);
			}
		}
		if (parsed.length === 1) {
			return { 'background-size': { value: parsed.join(', '), important } };
		}
		return null;
	}

	/**
	 * Returns background origin.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getBackgroundOrigin(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'background-origin': { value: variable, important } };
		}

		const lowerValue = value.toLowerCase();
		if (
			CSSStyleDeclarationValueParser.getGlobal(lowerValue) ||
			BACKGROUND_ORIGIN.includes(lowerValue)
		) {
			return { 'background-origin': { value: lowerValue, important } };
		}
		return null;
	}

	/**
	 * Returns background clip.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getBackgroundClip(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'background-clip': { value: variable, important } };
		}

		const lowerValue = value.toLowerCase();
		if (
			CSSStyleDeclarationValueParser.getGlobal(lowerValue) ||
			BACKGROUND_CLIP.includes(lowerValue)
		) {
			return { 'background-clip': { value: lowerValue, important } };
		}
		return null;
	}

	/**
	 * Returns background repeat.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getBackgroundRepeat(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'background-repeat': { value: variable, important } };
		}

		const lowerValue = value.toLowerCase();
		if (
			CSSStyleDeclarationValueParser.getGlobal(lowerValue) ||
			BACKGROUND_REPEAT.includes(lowerValue)
		) {
			return { 'background-repeat': { value: lowerValue, important } };
		}
		return null;
	}

	/**
	 * Returns background attachment.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getBackgroundAttachment(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'background-attachment': { value: variable, important } };
		}

		const lowerValue = value.toLowerCase();
		if (
			CSSStyleDeclarationValueParser.getGlobal(lowerValue) ||
			BACKGROUND_ATTACHMENT.includes(lowerValue)
		) {
			return { 'background-attachment': { value: lowerValue, important } };
		}
		return null;
	}

	/**
	 * Returns background position.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getBackgroundPosition(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'background-position': { value: variable, important } };
		}

		const globalValue = CSSStyleDeclarationValueParser.getGlobal(value);
		if (globalValue) {
			return {
				...this.getBackgroundPositionX(globalValue, important),
				...this.getBackgroundPositionY(globalValue, important)
			};
		}

		const imageParts = value.split(SPLIT_COMMA_SEPARATED_WITH_PARANTHESES_REGEXP);
		let x = '';
		let y = '';

		for (const imagePart of imageParts) {
			const parts = imagePart.trim().split(SPLIT_SPACE_SEPARATED_WITH_PARANTHESES_REGEXP);

			if (x) {
				x += ',';
				y += ',';
			}

			switch (parts.length) {
				case 1:
					if (parts[0] === 'top' || parts[0] === 'bottom') {
						x += 'center';
						y += parts[0];
					} else if (parts[0] === 'left' || parts[0] === 'right') {
						x += parts[0];
						y += 'center';
					} else if (parts[0] === 'center') {
						x += 'center';
						y += 'center';
					}
					break;
				case 2:
					x += parts[0] === 'top' || parts[0] === 'bottom' ? parts[1] : parts[0];
					y += parts[0] === 'top' || parts[0] === 'bottom' ? parts[0] : parts[1];
					break;
				case 3:
					if (
						parts[0] === 'top' ||
						parts[0] === 'bottom' ||
						parts[1] === 'left' ||
						parts[1] === 'right' ||
						parts[2] === 'left' ||
						parts[2] === 'right'
					) {
						if (CSSStyleDeclarationValueParser.getMeasurement(parts[1])) {
							x += parts[2];
							y += `${parts[0]} ${parts[1]}`;
						} else {
							x += `${parts[1]} ${parts[2]}`;
							y += parts[0];
						}
					} else {
						if (CSSStyleDeclarationValueParser.getMeasurement(parts[1])) {
							x += `${parts[0]} ${parts[1]}`;
							y += parts[2];
						} else {
							x += parts[0];
							y += `${parts[1]} ${parts[2]}`;
						}
					}
					break;
				case 4:
					x +=
						parts[0] === 'top' ||
						parts[0] === 'bottom' ||
						parts[1] === 'top' ||
						parts[1] === 'bottom'
							? `${parts[2]} ${parts[3]}`
							: `${parts[0]} ${parts[1]}`;
					y +=
						parts[0] === 'top' ||
						parts[0] === 'bottom' ||
						parts[1] === 'top' ||
						parts[1] === 'bottom'
							? `${parts[0]} ${parts[1]}`
							: `${parts[2]} ${parts[3]}`;
					break;
				default:
					return null;
			}
		}

		const xValue = this.getBackgroundPositionX(x, important);
		const yValue = this.getBackgroundPositionY(y, important);

		if (xValue && yValue) {
			return {
				...xValue,
				...yValue
			};
		}

		return null;
	}

	/**
	 * Returns background position.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getBackgroundPositionX(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'background-position-x': { value: variable, important } };
		}

		const lowerValue = value.toLowerCase();

		if (CSSStyleDeclarationValueParser.getGlobal(lowerValue)) {
			return { 'background-position-x': { value: lowerValue, important } };
		}

		const imageParts = lowerValue.split(SPLIT_COMMA_SEPARATED_WITH_PARANTHESES_REGEXP);
		let parsedValue = '';

		for (const imagePart of imageParts) {
			const parts = imagePart.trim().split(SPLIT_SPACE_SEPARATED_WITH_PARANTHESES_REGEXP);

			if (parsedValue) {
				parsedValue += ',';
			}

			for (const part of parts) {
				const measurement = CSSStyleDeclarationValueParser.getMeasurement(part);
				if (!measurement && part !== 'left' && part !== 'right' && part !== 'center') {
					return null;
				}

				if (parsedValue) {
					parsedValue += ' ';
				}

				parsedValue += measurement || part;
			}
		}

		return { 'background-position-x': { value: parsedValue, important } };
	}

	/**
	 * Returns background position.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getBackgroundPositionY(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'background-position-y': { value: variable, important } };
		}

		const lowerValue = value.toLowerCase();

		if (CSSStyleDeclarationValueParser.getGlobal(lowerValue)) {
			return { 'background-position-y': { value: lowerValue, important } };
		}

		const imageParts = lowerValue.split(SPLIT_COMMA_SEPARATED_WITH_PARANTHESES_REGEXP);
		let parsedValue = '';

		for (const imagePart of imageParts) {
			const parts = imagePart.trim().split(SPLIT_SPACE_SEPARATED_WITH_PARANTHESES_REGEXP);

			if (parsedValue) {
				parsedValue += ',';
			}

			for (const part of parts) {
				const measurement = CSSStyleDeclarationValueParser.getMeasurement(part);
				if (!measurement && part !== 'top' && part !== 'bottom' && part !== 'center') {
					return null;
				}

				if (parsedValue) {
					parsedValue += ' ';
				}

				parsedValue += measurement || part;
			}
		}

		return { 'background-position-y': { value: parsedValue, important } };
	}

	/**
	 * Returns background color.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property value.
	 */
	public static getBackgroundColor(
		value: string,
		important: boolean
	): { [key: string]: ICSSStyleDeclarationPropertyValue } {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'background-color': { value: variable, important } };
		}

		const color =
			CSSStyleDeclarationValueParser.getGlobal(value) ||
			CSSStyleDeclarationValueParser.getColor(value);

		return color
			? {
					['background-color']: { important, value: color }
				}
			: null;
	}

	/**
	 * Returns background image.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property value.
	 */
	public static getBackgroundImage(
		value: string,
		important: boolean
	): { [key: string]: ICSSStyleDeclarationPropertyValue } {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'background-image': { value: variable, important } };
		}

		const lowerValue = value.toLowerCase();

		if (CSSStyleDeclarationValueParser.getGlobal(lowerValue) || lowerValue === 'none') {
			return { 'background-image': { value: lowerValue, important } };
		}

		const parts = value.split(SPLIT_COMMA_SEPARATED_WITH_PARANTHESES_REGEXP);
		const parsed = [];

		for (const part of parts) {
			const parsedValue =
				CSSStyleDeclarationValueParser.getURL(part.trim()) ||
				CSSStyleDeclarationValueParser.getGradient(part.trim());
			if (!parsedValue) {
				return null;
			}
			parsed.push(parsedValue);
		}

		if (parsed.length) {
			return { 'background-image': { value: parsed.join(', '), important } };
		}

		return null;
	}

	/**
	 * Returns color.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property value.
	 */
	public static getColor(
		value: string,
		important: boolean
	): { [key: string]: ICSSStyleDeclarationPropertyValue } {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { color: { value: variable, important } };
		}

		const color =
			CSSStyleDeclarationValueParser.getGlobal(value) ||
			CSSStyleDeclarationValueParser.getColor(value);

		return color ? { color: { important, value: color } } : null;
	}

	/**
	 * Returns color.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property value.
	 */
	public static getFloodColor(
		value: string,
		important: boolean
	): { [key: string]: ICSSStyleDeclarationPropertyValue } {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'flood-color': { value: variable, important } };
		}
		const color =
			CSSStyleDeclarationValueParser.getGlobal(value) ||
			CSSStyleDeclarationValueParser.getColor(value);

		return color ? { 'flood-color': { important, value: color } } : null;
	}

	/**
	 * Returns font.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getFont(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { font: { value: variable, important } };
		}

		const lowerValue = value.toLowerCase();

		if (CSSStyleDeclarationValueParser.getGlobal(lowerValue)) {
			return {
				...this.getFontStyle(lowerValue, important),
				...this.getFontVariant(lowerValue, important),
				...this.getFontWeight(lowerValue, important),
				...this.getFontStretch(lowerValue, important),
				...this.getFontSize(lowerValue, important),
				...this.getLineHeight(lowerValue, important),
				...this.getFontFamily(lowerValue, important)
			};
		}

		if (SYSTEM_FONT.includes(lowerValue)) {
			return { font: { value: lowerValue, important } };
		}

		const properties = {
			...this.getFontStyle('normal', important),
			...this.getFontVariant('normal', important),
			...this.getFontWeight('normal', important),
			...this.getFontStretch('normal', important),
			...this.getLineHeight('normal', important)
		};

		const parts = value
			.replace(/\s*\/\s*/g, '/')
			.split(SPLIT_SPACE_SEPARATED_WITH_PARANTHESES_REGEXP);

		for (let i = 0, max = parts.length; i < max; i++) {
			const part = parts[i];
			if (part.includes('/')) {
				const [size, height] = part.split('/');
				const fontSize = this.getFontSize(size, important);
				const lineHeight = this.getLineHeight(height, important);

				if (!fontSize || !lineHeight) {
					return null;
				}

				Object.assign(properties, fontSize, lineHeight);
			} else {
				const fontStyle = this.getFontStyle(part, important);
				const fontVariant = this.getFontVariant(part, important);
				const fontWeight = this.getFontWeight(part, important);
				const fontSize = this.getFontSize(part, important);
				const fontStretch = this.getFontStretch(part, important);

				if (fontStyle) {
					Object.assign(properties, fontStyle);
				} else if (fontVariant) {
					Object.assign(properties, fontVariant);
				} else if (fontWeight) {
					Object.assign(properties, fontWeight);
				} else if (fontSize) {
					Object.assign(properties, fontSize);
				} else if (fontStretch) {
					Object.assign(properties, fontStretch);
				} else {
					const fontFamilyValue = parts.slice(i).join(' ');
					const fontFamily = this.getFontFamily(fontFamilyValue, important);
					if (!fontFamily) {
						return null;
					}
					Object.assign(properties, fontFamily);
					break;
				}
			}
		}

		return properties;
	}

	/**
	 * Returns font style.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getFontStyle(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'font-style': { value: variable, important } };
		}

		const lowerValue = value.toLowerCase();
		if (CSSStyleDeclarationValueParser.getGlobal(lowerValue) || FONT_STYLE.includes(lowerValue)) {
			return { 'font-style': { value: lowerValue, important } };
		}
		const parts = value.split(SPLIT_SPACE_SEPARATED_WITH_PARANTHESES_REGEXP);
		if (parts.length === 2 && parts[0] === 'oblique') {
			const degree = CSSStyleDeclarationValueParser.getDegree(parts[1]);
			return degree ? { 'font-style': { value: lowerValue, important } } : null;
		}
		return null;
	}

	/**
	 * Returns font variant.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getFontVariant(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'font-variant': { value: variable, important } };
		}

		const lowerValue = value.toLowerCase();
		return CSSStyleDeclarationValueParser.getGlobal(lowerValue) ||
			lowerValue === 'normal' ||
			lowerValue === 'small-caps'
			? { 'font-variant': { value: lowerValue, important } }
			: null;
	}

	/**
	 * Returns font strech.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getFontStretch(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'font-stretch': { value: variable, important } };
		}

		const lowerValue = value.toLowerCase();
		if (CSSStyleDeclarationValueParser.getGlobal(lowerValue) || FONT_STRETCH.includes(lowerValue)) {
			return { 'font-stretch': { value: lowerValue, important } };
		}
		const percentage = CSSStyleDeclarationValueParser.getPercentage(value);
		return percentage ? { 'font-stretch': { value: percentage, important } } : null;
	}

	/**
	 * Returns font weight.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getFontWeight(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'font-weight': { value: variable, important } };
		}

		const lowerValue = value.toLowerCase();
		if (CSSStyleDeclarationValueParser.getGlobal(lowerValue) || FONT_WEIGHT.includes(lowerValue)) {
			return { 'font-weight': { value: lowerValue, important } };
		}
		const integer = CSSStyleDeclarationValueParser.getInteger(value);
		return integer ? { 'font-weight': { value: integer, important } } : null;
	}

	/**
	 * Returns font size.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getFontSize(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'font-size': { value: variable, important } };
		}

		const lowerValue = value.toLowerCase();
		if (CSSStyleDeclarationValueParser.getGlobal(lowerValue) || FONT_SIZE.includes(lowerValue)) {
			return { 'font-size': { value: lowerValue, important } };
		}
		const measurement = CSSStyleDeclarationValueParser.getMeasurement(value);
		return measurement ? { 'font-size': { value: measurement, important } } : null;
	}

	/**
	 * Returns line height.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getLineHeight(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'line-height': { value: variable, important } };
		}

		const lowerValue = value.toLowerCase();
		if (CSSStyleDeclarationValueParser.getGlobal(lowerValue) || lowerValue === 'normal') {
			return { 'line-height': { value: lowerValue, important } };
		}
		const lineHeight =
			CSSStyleDeclarationValueParser.getFloat(value) ||
			CSSStyleDeclarationValueParser.getMeasurement(value);
		return lineHeight ? { 'line-height': { value: lineHeight, important } } : null;
	}

	/**
	 * Returns font family.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getFontFamily(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'font-family': { value: variable, important } };
		}

		const globalValue = CSSStyleDeclarationValueParser.getGlobal(value);

		if (globalValue) {
			return { 'font-family': { value: globalValue, important } };
		}

		const parts = value.split(',');
		let parsedValue = '';
		let endWithApostroph = false;

		for (let i = 0, max = parts.length; i < max; i++) {
			let trimmedPart = parts[i].trim().replace(/'/g, '"');

			if (!trimmedPart) {
				return null;
			}

			if (trimmedPart.includes(' ')) {
				const apostrophCount = (trimmedPart.match(/"/g) || []).length;
				if ((trimmedPart[0] !== '"' || i !== 0) && apostrophCount !== 2 && apostrophCount !== 0) {
					return null;
				}
				if (trimmedPart[0] === '"' && trimmedPart[trimmedPart.length - 1] !== '"') {
					endWithApostroph = true;
				} else if (trimmedPart[0] !== '"' && trimmedPart[trimmedPart.length - 1] !== '"') {
					trimmedPart = `"${trimmedPart}"`;
				}
			} else {
				trimmedPart = trimmedPart.replace(/"/g, '');
			}

			if (i > 0) {
				parsedValue += ', ';
			}

			parsedValue += trimmedPart;
		}

		if (endWithApostroph) {
			parsedValue += '"';
		}

		if (!parsedValue) {
			return null;
		}

		return {
			'font-family': {
				important,
				value: parsedValue
			}
		};
	}

	/**
	 * Returns font family.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property values
	 */
	public static getTextTransform(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'text-transform': { value: variable, important } };
		}

		const lowerValue = value.toLowerCase();
		const parsedValue =
			CSSStyleDeclarationValueParser.getGlobal(lowerValue) ||
			(TEXT_TRANSFORM.includes(lowerValue) && lowerValue);
		if (parsedValue) {
			return {
				'text-transform': { value: parsedValue, important }
			};
		}
		return null;
	}

	/**
	 * Returns visibility.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property
	 */
	public static getVisibility(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { visibility: { value: variable, important } };
		}

		const lowerValue = value.toLowerCase();
		const parsedValue =
			CSSStyleDeclarationValueParser.getGlobal(lowerValue) ||
			(VISIBILITY.includes(lowerValue) && lowerValue);
		if (parsedValue) {
			return {
				visibility: { value: parsedValue, important }
			};
		}
		return null;
	}

	/**
	 * Returns aspect ratio.
	 *
	 * @param value Value.
	 * @param important Important.
	 * @returns Property
	 */
	public static getAspectRatio(
		value: string,
		important: boolean
	): {
		[key: string]: ICSSStyleDeclarationPropertyValue;
	} {
		const variable = CSSStyleDeclarationValueParser.getVariable(value);
		if (variable) {
			return { 'aspect-ratio': { value: variable, important } };
		}

		const lowerValue = value.toLowerCase();

		if (CSSStyleDeclarationValueParser.getGlobal(lowerValue)) {
			return { 'aspect-ratio': { value: lowerValue, important } };
		}

		let parsedValue = value;

		const hasAuto = parsedValue.includes('auto');

		if (hasAuto) {
			parsedValue = parsedValue.replace('auto', '');
		}

		parsedValue = parsedValue.replace(WHITE_SPACE_GLOBAL_REGEXP, '');

		if (!parsedValue) {
			return { 'aspect-ratio': { value: 'auto', important } };
		}

		const aspectRatio = parsedValue.split('/');

		if (aspectRatio.length > 3) {
			return null;
		}

		const width = Number(aspectRatio[0]);
		const height = aspectRatio[1] ? Number(aspectRatio[1]) : 1;

		if (isNaN(width) || isNaN(height)) {
			return null;
		}

		if (hasAuto) {
			return { 'aspect-ratio': { value: `auto ${width} / ${height}`, important } };
		}

		return { 'aspect-ratio': { value: `${width} / ${height}`, important } };
	}
}
