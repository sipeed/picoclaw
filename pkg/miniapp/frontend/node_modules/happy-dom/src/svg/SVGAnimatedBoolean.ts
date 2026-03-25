import * as PropertySymbol from '../PropertySymbol.js';
import BrowserWindow from '../window/BrowserWindow.js';

/**
 * SVG Animated Boolean.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGAnimatedBoolean
 */
export default class SVGAnimatedBoolean {
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
	public get animVal(): boolean {
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
	public get baseVal(): boolean {
		const attributeValue = this[PropertySymbol.getAttribute]();
		return attributeValue === 'true';
	}

	/**
	 * Sets base value.
	 *
	 * @param value Base value.
	 */
	public set baseVal(value: boolean) {
		this[PropertySymbol.setAttribute](
			typeof value !== 'boolean' ? String(Boolean(value)) : String(value)
		);
	}
}
