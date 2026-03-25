import * as PropertySymbol from '../PropertySymbol.js';
import BrowserWindow from '../window/BrowserWindow.js';

/**
 * SVG Animated Enumaration.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGAnimatedEnumeration
 */
export default class SVGAnimatedEnumeration {
	// Internal properties
	public [PropertySymbol.window]: BrowserWindow;
	public [PropertySymbol.getAttribute]: () => string;
	public [PropertySymbol.setAttribute]: (value: string) => void;
	public [PropertySymbol.values]: Array<string | null>;
	public [PropertySymbol.defaultValue]: string;

	/**
	 * Constructor.
	 *
	 * @param illegalConstructorSymbol Illegal constructor symbol.
	 * @param window Window.
	 * @param options Options.
	 * @param options.getAttribute Get attribute.
	 * @param options.setAttribute Set attribute.
	 * @param options.values Values.
	 * @param options.defaultValue Default value.
	 */
	constructor(
		illegalConstructorSymbol: symbol,
		window: BrowserWindow,
		options: {
			getAttribute: () => string | null;
			setAttribute: (value: string) => void;
			values: Array<string | null>;
			defaultValue: string;
		}
	) {
		if (illegalConstructorSymbol !== PropertySymbol.illegalConstructor) {
			throw new TypeError('Illegal constructor');
		}

		this[PropertySymbol.window] = window;

		this[PropertySymbol.getAttribute] = options.getAttribute;
		this[PropertySymbol.setAttribute] = options.setAttribute;
		this[PropertySymbol.values] = options.values;
		this[PropertySymbol.defaultValue] = options.defaultValue;
	}

	/**
	 * Returns animated value.
	 *
	 * @returns Animated value.
	 */
	public get animVal(): number {
		return this.baseVal;
	}

	/**
	 * Returns animated value.
	 *
	 * @param _value Animated value.
	 */
	public set animVal(_value) {
		// Do nothing
	}

	/**
	 * Returns base value.
	 *
	 * @returns Base value.
	 */
	public get baseVal(): number {
		const value = this[PropertySymbol.getAttribute]();
		if (!value) {
			return this[PropertySymbol.values].indexOf(this[PropertySymbol.defaultValue]) + 1;
		}
		const index = this[PropertySymbol.values].indexOf(value);
		if (index === -1) {
			const anyValueIndex = this[PropertySymbol.values].indexOf(null);
			return anyValueIndex !== -1 ? anyValueIndex + 1 : 0;
		}
		return index + 1;
	}

	/**
	 * Sets base value.
	 *
	 * @param value Base value.
	 */
	public set baseVal(value: number) {
		let parsedValue = Number(value);
		if (isNaN(parsedValue)) {
			parsedValue = 0;
		}
		if (parsedValue < 1) {
			throw new TypeError(
				`Failed to set the 'baseVal' property on 'SVGAnimatedEnumeration': The enumeration value provided is ${parsedValue}, which is not settable.`
			);
		}
		if (parsedValue > this[PropertySymbol.values].length) {
			throw new TypeError(
				`Failed to set the 'baseVal' property on 'SVGAnimatedEnumeration': The enumeration value provided (${parsedValue}) is larger than the largest allowed value (${
					this[PropertySymbol.values].length
				}).`
			);
		}
		const currentValue = this[PropertySymbol.getAttribute]();
		const isAnyValue = this[PropertySymbol.values][parsedValue - 1] === null;
		const newValue = isAnyValue ? '0' : this[PropertySymbol.values][parsedValue - 1];
		if (
			!currentValue ||
			(isAnyValue && this[PropertySymbol.values].includes(currentValue)) ||
			(!isAnyValue && currentValue !== newValue)
		) {
			this[PropertySymbol.setAttribute](newValue);
		}
	}
}
