import * as PropertySymbol from '../PropertySymbol.js';
import SVGAngle from './SVGAngle.js';
import BrowserWindow from '../window/BrowserWindow.js';

/**
 * SVG Animated Angle.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGAnimatedAngle
 */
export default class SVGAnimatedAngle {
	// Internal properties
	public [PropertySymbol.window]: BrowserWindow;
	public [PropertySymbol.getAttribute]: () => string;
	public [PropertySymbol.setAttribute]: (value: string) => void;
	public [PropertySymbol.baseVal]: SVGAngle | null = null;
	public [PropertySymbol.animVal]: SVGAngle | null = null;

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
	public get animVal(): SVGAngle {
		if (!this[PropertySymbol.animVal]) {
			this[PropertySymbol.animVal] = new SVGAngle(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					readOnly: true,
					getAttribute: this[PropertySymbol.getAttribute]
				}
			);
		}
		return this[PropertySymbol.animVal];
	}

	/**
	 * Sets animated value.
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
	public get baseVal(): SVGAngle {
		if (!this[PropertySymbol.baseVal]) {
			this[PropertySymbol.baseVal] = new SVGAngle(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: this[PropertySymbol.getAttribute],
					setAttribute: this[PropertySymbol.setAttribute]
				}
			);
		}
		return this[PropertySymbol.baseVal];
	}

	/**
	 * Sets base value.
	 *
	 * @param value Base value.
	 */
	public set baseVal(_value) {
		// Do nothing
	}
}
