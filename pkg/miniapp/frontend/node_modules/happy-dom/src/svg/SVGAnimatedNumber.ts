import * as PropertySymbol from '../PropertySymbol.js';
import BrowserWindow from '../window/BrowserWindow.js';

/**
 * SVG Animated Number.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGAnimatedNumber
 */
export default class SVGAnimatedNumber {
	// Internal properties
	public [PropertySymbol.window]: BrowserWindow;
	public [PropertySymbol.getAttribute]: (() => string | null) | null = null;
	public [PropertySymbol.setAttribute]: ((value: string) => void) | null = null;
	public [PropertySymbol.defaultValue]: number = 0;

	/**
	 * Constructor.
	 *
	 * @param illegalConstructorSymbol Illegal constructor symbol.
	 * @param window Window.
	 * @param options Options.
	 * @param options.getAttribute Get attribute.
	 * @param options.setAttribute Set attribute.
	 * @param options.defaultValue
	 */
	constructor(
		illegalConstructorSymbol: symbol,
		window: BrowserWindow,
		options: {
			getAttribute: () => string | null;
			setAttribute: (value: string) => void;
			defaultValue?: number;
		}
	) {
		if (illegalConstructorSymbol !== PropertySymbol.illegalConstructor) {
			throw new TypeError('Illegal constructor');
		}

		this[PropertySymbol.window] = window;

		this[PropertySymbol.getAttribute] = options.getAttribute;
		this[PropertySymbol.setAttribute] = options.setAttribute;
		this[PropertySymbol.defaultValue] = options.defaultValue || 0;
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
	 * @param value Animated value.
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
		const attributeValue = this[PropertySymbol.getAttribute]();

		if (!attributeValue) {
			return this[PropertySymbol.defaultValue];
		}

		const value = parseFloat(attributeValue);

		if (isNaN(value)) {
			return this[PropertySymbol.defaultValue];
		}

		return value;
	}

	/**
	 * Sets base value.
	 *
	 * @param value Base value.
	 */
	public set baseVal(value: number) {
		const parsedValue = typeof value !== 'number' ? parseFloat(<string>(<unknown>value)) : value;

		if (isNaN(parsedValue)) {
			throw new this[PropertySymbol.window].TypeError(
				`TypeError: Failed to set the 'baseVal' property on 'SVGAnimatedNumber': The provided float value is non-finite.`
			);
		}

		this[PropertySymbol.setAttribute](String(parsedValue));
	}
}
