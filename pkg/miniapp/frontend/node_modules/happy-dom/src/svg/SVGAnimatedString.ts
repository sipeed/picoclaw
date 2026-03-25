import * as PropertySymbol from '../PropertySymbol.js';
import BrowserWindow from '../window/BrowserWindow.js';

/**
 * SVG Animated String.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGAnimatedString
 */
export default class SVGAnimatedString {
	// Internal properties
	public [PropertySymbol.window]: BrowserWindow;
	public [PropertySymbol.getAttribute]: (() => string | null) | null = null;
	public [PropertySymbol.setAttribute]: ((value: string) => void) | null = null;

	/**
	 * Constructor.
	 *
	 * @param illegalConstructorSymbol Illegal constructor symbol.
	 * @param window Window.
	 * @param options Options.
	 * @param options.getAttribute Get attribute.
	 * @param options.setAttribute Set attribute.
	 */
	constructor(
		illegalConstructorSymbol: symbol,
		window: BrowserWindow,
		options: {
			getAttribute: () => string | null;
			setAttribute: (value: string) => void;
		}
	) {
		if (illegalConstructorSymbol !== PropertySymbol.illegalConstructor) {
			throw new TypeError('Illegal constructor');
		}

		this[PropertySymbol.window] = window;

		this[PropertySymbol.getAttribute] = options.getAttribute;
		this[PropertySymbol.setAttribute] = options.setAttribute;
	}

	/**
	 * Returns animated value.
	 *
	 * @returns Animated value.
	 */
	public get animVal(): string {
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
	public get baseVal(): string {
		const attributeValue = this[PropertySymbol.getAttribute]();

		if (!attributeValue) {
			return '';
		}

		return attributeValue;
	}

	/**
	 * Sets base value.
	 *
	 * @param value Base value.
	 */
	public set baseVal(value: string) {
		this[PropertySymbol.setAttribute](String(value));
	}
}
