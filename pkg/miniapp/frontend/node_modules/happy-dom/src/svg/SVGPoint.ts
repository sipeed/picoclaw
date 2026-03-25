import * as PropertySymbol from '../PropertySymbol.js';
import BrowserWindow from '../window/BrowserWindow.js';

const ATTRIBUTE_SEPARATOR_REGEXP = /[\t\f\n\r, ]+/;

/**
 * SVG point.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGPoint
 */
export default class SVGPoint {
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
	 * @param [options.setAttribute] Set attribute
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
	 * Returns x.
	 *
	 * @returns X.
	 */
	public get x(): number {
		const attributeValue = this[PropertySymbol.getAttribute]
			? this[PropertySymbol.getAttribute]()
			: this[PropertySymbol.attributeValue];
		const parts = (attributeValue || '').split(ATTRIBUTE_SEPARATOR_REGEXP);
		return !!parts[0] ? parseFloat(parts[0]) : 0;
	}

	/**
	 * Sets x.
	 *
	 * @param value X.
	 */
	public set x(value: number) {
		if (this[PropertySymbol.readOnly]) {
			throw new this[PropertySymbol.window].TypeError(
				`Failed to set the 'x' property on 'SVGPoint': The object is read-only.`
			);
		}

		this[PropertySymbol.attributeValue] = `${value} ${this.y}`;

		if (this[PropertySymbol.setAttribute]) {
			this[PropertySymbol.setAttribute](this[PropertySymbol.attributeValue]);
		}
	}

	/**
	 * Returns y.
	 *
	 * @returns Y.
	 */
	public get y(): number {
		const attributeValue = this[PropertySymbol.getAttribute]
			? this[PropertySymbol.getAttribute]()
			: this[PropertySymbol.attributeValue];
		const parts = (attributeValue || '').split(ATTRIBUTE_SEPARATOR_REGEXP);
		return !!parts[1] ? parseFloat(parts[1]) : 0;
	}

	/**
	 * Sets y.
	 *
	 * @param value Y.
	 */
	public set y(value: number) {
		if (this[PropertySymbol.readOnly]) {
			throw new this[PropertySymbol.window].TypeError(
				`Failed to set the 'y' property on 'SVGPoint': The object is read-only.`
			);
		}

		this[PropertySymbol.attributeValue] = `${this.x} ${value}`;

		if (this[PropertySymbol.setAttribute]) {
			this[PropertySymbol.setAttribute](this[PropertySymbol.attributeValue]);
		}
	}
}
