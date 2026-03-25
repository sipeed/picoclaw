import * as PropertySymbol from '../PropertySymbol.js';
import SVGLengthTypeEnum from './SVGLengthTypeEnum.js';
import BrowserWindow from '../window/BrowserWindow.js';

const ATTRIBUTE_REGEXP = /^(\d+|\d+\.\d+)(px|em|ex|cm|mm|in|pt|pc|%|)$/;

/**
 * SVG length.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGLength
 */
export default class SVGLength {
	// Static properties
	public static SVG_LENGTHTYPE_UNKNOWN = SVGLengthTypeEnum.unknown;
	public static SVG_LENGTHTYPE_NUMBER = SVGLengthTypeEnum.number;
	public static SVG_LENGTHTYPE_PERCENTAGE = SVGLengthTypeEnum.percentage;
	public static SVG_LENGTHTYPE_EMS = SVGLengthTypeEnum.ems;
	public static SVG_LENGTHTYPE_EXS = SVGLengthTypeEnum.exs;
	public static SVG_LENGTHTYPE_PX = SVGLengthTypeEnum.px;
	public static SVG_LENGTHTYPE_CM = SVGLengthTypeEnum.cm;
	public static SVG_LENGTHTYPE_MM = SVGLengthTypeEnum.mm;
	public static SVG_LENGTHTYPE_IN = SVGLengthTypeEnum.in;
	public static SVG_LENGTHTYPE_PT = SVGLengthTypeEnum.pt;
	public static SVG_LENGTHTYPE_PC = SVGLengthTypeEnum.pc;

	// Internal properties
	public [PropertySymbol.window]: BrowserWindow;
	public [PropertySymbol.getAttribute]: (() => string | null) | null = null;
	public [PropertySymbol.setAttribute]: ((value: string) => void) | null = null;
	public [PropertySymbol.attributeValue]: string | null = null;
	public [PropertySymbol.readOnly]: boolean = false;

	/**
	 * Constructor.
	 *
	 * @param illegalConstructorSymbol Illegal constructor symbol.
	 * @param window Window.
	 * @param [options] Options.
	 * @param [options.readOnly] Read only.
	 * @param [options.getAttribute] Get attribute.
	 * @param [options.setAttribute] Set attribute.
	 */
	constructor(
		illegalConstructorSymbol: symbol,
		window: BrowserWindow,
		options?: {
			readOnly?: boolean;
			getAttribute?: () => string | null;
			setAttribute?: (value: string) => void;
		}
	) {
		if (illegalConstructorSymbol !== PropertySymbol.illegalConstructor) {
			throw new TypeError('Illegal constructor');
		}

		this[PropertySymbol.window] = window;

		if (options) {
			this[PropertySymbol.readOnly] = !!options.readOnly;
			this[PropertySymbol.getAttribute] = options.getAttribute || null;
			this[PropertySymbol.setAttribute] = options.setAttribute || null;
		}
	}

	/**
	 * Returns unit type.
	 *
	 * @returns Unit type.
	 */
	public get unitType(): SVGLengthTypeEnum {
		const attributeValue = this[PropertySymbol.getAttribute]
			? this[PropertySymbol.getAttribute]() || ''
			: this[PropertySymbol.attributeValue] || '';
		const match = attributeValue.match(ATTRIBUTE_REGEXP);

		if (!match) {
			return SVGLengthTypeEnum.unknown;
		}

		if (isNaN(parseFloat(match[1]))) {
			return SVGLengthTypeEnum.unknown;
		}

		switch (match[2]) {
			case '':
				return SVGLengthTypeEnum.number;
			case 'px':
				return SVGLengthTypeEnum.px;
			case 'cm':
				return SVGLengthTypeEnum.cm;
			case 'mm':
				return SVGLengthTypeEnum.mm;
			case 'in':
				return SVGLengthTypeEnum.in;
			case 'pt':
				return SVGLengthTypeEnum.pt;
			case 'pc':
				return SVGLengthTypeEnum.pc;
			case 'em':
			case 'ex':
			case '%':
				throw new this[PropertySymbol.window].TypeError(
					`Failed to execute 'value' on 'SVGLength': Could not resolve relative length.`
				);
			default:
				return SVGLengthTypeEnum.unknown;
		}
	}

	/**
	 * Returns value.
	 *
	 * @returns Value.
	 */
	public get value(): number {
		const attributeValue = this[PropertySymbol.getAttribute]
			? this[PropertySymbol.getAttribute]() || ''
			: this[PropertySymbol.attributeValue] || '';
		const match = attributeValue.match(ATTRIBUTE_REGEXP);

		if (!match) {
			return 0;
		}

		const parsedValue = parseFloat(match[1]);

		if (isNaN(parsedValue)) {
			return 0;
		}

		switch (match[2]) {
			case '':
				return parsedValue;
			case 'px':
				return parsedValue;
			case 'cm':
				return (parsedValue / 2.54) * 96;
			case 'mm':
				return (parsedValue / 25.4) * 96;
			case 'in':
				return parsedValue * 96;
			case 'pt':
				return parsedValue * 72;
			case 'pc':
				return parsedValue * 6;
			case 'em':
			case 'ex':
			case '%':
				throw new this[PropertySymbol.window].TypeError(
					`Failed to execute 'value' on 'SVGLength': Could not resolve relative length.`
				);
			default:
				return 0;
		}
	}

	/**
	 * Sets value.
	 *
	 * @param value Value in pixels.
	 */
	public set value(value: number) {
		if (this[PropertySymbol.readOnly]) {
			throw new this[PropertySymbol.window].TypeError(
				`Failed to set the 'value' property on 'SVGLength': The object is read-only.`
			);
		}

		// Value in pixels
		value = typeof value !== 'number' ? parseFloat(String(value)) : value;
		if (isNaN(value)) {
			throw new this[PropertySymbol.window].TypeError(
				`Failed to set the 'value' property on 'SVGLength': The provided float value is non-finite.`
			);
		}
		let unitType = '';
		let valueInSpecifiedUnits = value;
		switch (this.unitType) {
			case SVGLengthTypeEnum.number:
				valueInSpecifiedUnits = value;
				unitType = '';
				break;
			case SVGLengthTypeEnum.px:
				valueInSpecifiedUnits = value;
				unitType = 'px';
				break;
			case SVGLengthTypeEnum.cm:
				valueInSpecifiedUnits = (value / 96) * 2.54;
				unitType = 'cm';
				break;
			case SVGLengthTypeEnum.mm:
				valueInSpecifiedUnits = (value / 96) * 25.4;
				unitType = 'mm';
				break;
			case SVGLengthTypeEnum.in:
				valueInSpecifiedUnits = value / 96;
				unitType = 'in';
				break;
			case SVGLengthTypeEnum.pt:
				valueInSpecifiedUnits = value / 72;
				unitType = 'pt';
				break;
			case SVGLengthTypeEnum.pc:
				valueInSpecifiedUnits = value / 6;
				unitType = 'pc';
				break;
			case SVGLengthTypeEnum.percentage:
			case SVGLengthTypeEnum.ems:
			case SVGLengthTypeEnum.exs:
				throw new this[PropertySymbol.window].TypeError(
					`Failed to set the 'value' property on 'SVGLength': Could not resolve relative length.`
				);
			default:
				break;
		}

		this[PropertySymbol.attributeValue] = String(valueInSpecifiedUnits) + unitType;

		if (this[PropertySymbol.setAttribute]) {
			this[PropertySymbol.setAttribute](this[PropertySymbol.attributeValue]);
		}
	}

	/**
	 * Returns value as string.
	 *
	 * @returns Value as string.
	 */
	public get valueAsString(): string {
		return this[PropertySymbol.getAttribute]
			? this[PropertySymbol.getAttribute]() || '0'
			: this[PropertySymbol.attributeValue] || '0';
	}

	/**
	 * Returns value in specified units.
	 *
	 * @returns Value in specified units.
	 */
	public get valueInSpecifiedUnits(): number {
		const attributeValue = this.valueAsString;
		return parseFloat(attributeValue) || 0;
	}

	/**
	 * New value specific units.
	 *
	 * @param unitType
	 * @param value
	 */
	public newValueSpecifiedUnits(unitType: number, value: number): void {
		if (this[PropertySymbol.readOnly]) {
			throw new this[PropertySymbol.window].TypeError(
				`Failed to execute 'newValueSpecifiedUnits' on 'SVGLength': The object is read-only.`
			);
		}

		if (typeof unitType !== 'number') {
			throw new this[PropertySymbol.window].TypeError(
				`Failed to execute 'newValueSpecifiedUnits' on 'SVGLength': parameter 1 ('unitType') is not of type 'number'.`
			);
		}

		value = typeof value !== 'number' ? parseFloat(String(value)) : value;

		if (isNaN(value)) {
			throw new this[PropertySymbol.window].TypeError(
				`Failed to execute 'newValueSpecifiedUnits' on 'SVGLength': The provided float value is non-finite.`
			);
		}

		let unit = '';
		switch (unitType) {
			case SVGLengthTypeEnum.number:
				unit = '';
				break;
			case SVGLengthTypeEnum.px:
				unit = 'px';
				break;
			case SVGLengthTypeEnum.cm:
				unit = 'cm';
				break;
			case SVGLengthTypeEnum.mm:
				unit = 'mm';
				break;
			case SVGLengthTypeEnum.in:
				unit = 'in';
				break;
			case SVGLengthTypeEnum.pt:
				unit = 'pt';
				break;
			case SVGLengthTypeEnum.pc:
				unit = 'pc';
				break;
			case SVGLengthTypeEnum.ems:
			case SVGLengthTypeEnum.exs:
			case SVGLengthTypeEnum.percentage:
				throw new this[PropertySymbol.window].TypeError(
					`Failed to execute 'newValueSpecifiedUnits' on 'SVGLength': Could not resolve relative length.`
				);
			default:
				break;
		}

		this[PropertySymbol.attributeValue] = String(value) + unit;

		if (this[PropertySymbol.setAttribute]) {
			this[PropertySymbol.setAttribute](this[PropertySymbol.attributeValue]);
		}
	}

	/**
	 * Convert to specific units.
	 * @param unitType
	 */
	public convertToSpecifiedUnits(unitType: SVGLengthTypeEnum): void {
		if (this[PropertySymbol.readOnly]) {
			throw new this[PropertySymbol.window].TypeError(
				`Failed to execute 'convertToSpecifiedUnits' on 'SVGLength': The object is read-only.`
			);
		}

		if (typeof unitType !== 'number') {
			throw new this[PropertySymbol.window].TypeError(
				`Failed to execute 'convertToSpecifiedUnits' on 'SVGLength': parameter 1 ('unitType') is not of type 'number'.`
			);
		}

		let value = this.value;
		let unit = '';

		switch (unitType) {
			case SVGLengthTypeEnum.number:
				unit = '';
				break;
			case SVGLengthTypeEnum.px:
				unit = 'px';
				break;
			case SVGLengthTypeEnum.cm:
				value = (value / 96) * 2.54;
				unit = 'cm';
				break;
			case SVGLengthTypeEnum.mm:
				value = (value / 96) * 25.4;
				unit = 'mm';
				break;
			case SVGLengthTypeEnum.in:
				value = value / 96;
				unit = 'in';
				break;
			case SVGLengthTypeEnum.pt:
				value = value / 72;
				unit = 'pt';
				break;
			case SVGLengthTypeEnum.pc:
				value = value / 6;
				unit = 'pc';
				break;
			case SVGLengthTypeEnum.percentage:
			case SVGLengthTypeEnum.ems:
			case SVGLengthTypeEnum.exs:
				throw new this[PropertySymbol.window].TypeError(
					`Failed to execute 'convertToSpecifiedUnits' on 'SVGLength': Could not resolve relative length.`
				);
			default:
				break;
		}

		this[PropertySymbol.attributeValue] = String(value) + unit;

		if (this[PropertySymbol.setAttribute]) {
			this[PropertySymbol.setAttribute](this[PropertySymbol.attributeValue]);
		}
	}
}
