import * as PropertySymbol from '../PropertySymbol.js';
import BrowserWindow from '../window/BrowserWindow.js';

/**
 * Rect object.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGRect
 */
export default class SVGRect {
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
	 * Returns x value.
	 *
	 * @returns X value.
	 */
	public get x(): number {
		const attributeValue = this[PropertySymbol.getAttribute]
			? this[PropertySymbol.getAttribute]()
			: this[PropertySymbol.attributeValue];
		if (!attributeValue) {
			return 0;
		}
		const parts = attributeValue.split(/\s+/);
		const value = Number(parts[0]);
		return isNaN(value) ? 0 : value;
	}

	/**
	 * Sets x value.
	 *
	 * @param value X value.
	 */
	public set x(value: number) {
		if (this[PropertySymbol.readOnly]) {
			throw new this[PropertySymbol.window].TypeError(
				`Failed to set the 'x' property on 'SVGRect': The object is read-only.`
			);
		}

		this[PropertySymbol.attributeValue] = `${String(
			typeof value === 'number' ? value : parseFloat(value)
		)} ${this.y} ${this.width} ${this.height}`;

		if (this[PropertySymbol.setAttribute]) {
			this[PropertySymbol.setAttribute](this[PropertySymbol.attributeValue]);
		}
	}

	/**
	 * Returns y value.
	 *
	 * @returns Y value.
	 */
	public get y(): number {
		const attributeValue = this[PropertySymbol.getAttribute]
			? this[PropertySymbol.getAttribute]()
			: this[PropertySymbol.attributeValue];
		if (!attributeValue) {
			return 0;
		}
		const parts = attributeValue.split(/\s+/);
		const value = Number(parts[1]);
		return isNaN(value) ? 0 : value;
	}

	/**
	 * Sets y value.
	 *
	 * @param value Y value.
	 */
	public set y(value: number) {
		if (this[PropertySymbol.readOnly]) {
			throw new this[PropertySymbol.window].TypeError(
				`Failed to set the 'y' property on 'SVGRect': The object is read-only.`
			);
		}

		this[PropertySymbol.attributeValue] = `${this.x} ${String(
			typeof value === 'number' ? value : parseFloat(value)
		)} ${this.width} ${this.height}`;

		if (this[PropertySymbol.setAttribute]) {
			this[PropertySymbol.setAttribute](this[PropertySymbol.attributeValue]);
		}
	}

	/**
	 * Returns width value.
	 *
	 * @returns Width value.
	 */
	public get width(): number {
		const attributeValue = this[PropertySymbol.getAttribute]
			? this[PropertySymbol.getAttribute]()
			: this[PropertySymbol.attributeValue];
		if (!attributeValue) {
			return 0;
		}
		const parts = attributeValue.split(/\s+/);
		const value = Number(parts[2]);
		return isNaN(value) ? 0 : value;
	}

	/**
	 * Sets width value.
	 *
	 * @param value Width value.
	 */
	public set width(value: number) {
		if (this[PropertySymbol.readOnly]) {
			throw new this[PropertySymbol.window].TypeError(
				`Failed to set the 'width' property on 'SVGRect': The object is read-only.`
			);
		}

		this[PropertySymbol.attributeValue] = `${this.x} ${this.y} ${String(
			typeof value === 'number' ? value : parseFloat(value)
		)} ${this.height}`;

		if (this[PropertySymbol.setAttribute]) {
			this[PropertySymbol.setAttribute](this[PropertySymbol.attributeValue]);
		}
	}

	/**
	 * Returns height value.
	 *
	 * @returns Height value.
	 */
	public get height(): number {
		const attributeValue = this[PropertySymbol.getAttribute]
			? this[PropertySymbol.getAttribute]()
			: this[PropertySymbol.attributeValue];
		if (!attributeValue) {
			return 0;
		}
		const parts = attributeValue.split(/\s+/);
		const value = Number(parts[3]);
		return isNaN(value) ? 0 : value;
	}

	/**
	 * Sets height value.
	 *
	 * @param value Height value.
	 */
	public set height(value: number) {
		if (this[PropertySymbol.readOnly]) {
			throw new this[PropertySymbol.window].TypeError(
				`Failed to set the 'height' property on 'SVGRect': The object is read-only.`
			);
		}

		this[PropertySymbol.attributeValue] = `${this.x} ${this.y} ${this.width} ${String(
			typeof value === 'number' ? value : parseFloat(value)
		)}`;

		if (this[PropertySymbol.setAttribute]) {
			this[PropertySymbol.setAttribute](this[PropertySymbol.attributeValue]);
		}
	}
}
