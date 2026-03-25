import BrowserWindow from '../../../window/BrowserWindow.js';

/**
 * CSS Measurement Converter.
 */
export default class CSSMeasurementConverter {
	/**
	 * Returns measurement in pixels.
	 *
	 * @param options Options.
	 * @param options.window Owner window.
	 * @param options.value Measurement (e.g. "10px", "10rem" or "10em").
	 * @param options.rootFontSize Root font size in pixels.
	 * @param options.parentFontSize Parent font size in pixels.
	 * @param [options.parentSize] Parent size in pixels.
	 * @returns Measurement in pixels.
	 */
	public static toPixels(options: {
		window: BrowserWindow;
		value: string;
		rootFontSize: string | number;
		parentFontSize: string | number;
		parentSize?: string | number | null;
	}): number | null {
		const value = parseFloat(options.value);
		const unit = options.value.replace(value.toString(), '');

		if (isNaN(value)) {
			return null;
		}

		switch (unit) {
			case 'px':
				return value;
			case 'rem':
				return this.round(value * parseFloat(<string>options.rootFontSize));
			case 'em':
				return this.round(value * parseFloat(<string>options.parentFontSize));
			case 'vw':
				return this.round((value * options.window.innerWidth) / 100);
			case 'vh':
				return this.round((value * options.window.innerHeight) / 100);
			case '%':
				return options.parentSize !== undefined && options.parentSize !== null
					? this.round((value * parseFloat(<string>options.parentSize)) / 100)
					: null;
			case 'vmin':
				return this.round(
					(value * Math.min(options.window.innerWidth, options.window.innerHeight)) / 100
				);
			case 'vmax':
				return (value * Math.max(options.window.innerWidth, options.window.innerHeight)) / 100;
			case 'cm':
				return this.round(value * 37.7812);
			case 'mm':
				return this.round(value * 3.7781);
			case 'in':
				return this.round(value * 96);
			case 'pt':
				return this.round(value * 1.3281);
			case 'pc':
				return this.round(value * 16);
			case 'Q':
				return this.round(value * 0.945);
			default:
				return null;
		}
	}

	/**
	 * Rounds the number with 4 decimals.
	 *
	 * @param value Value.
	 * @returns Rounded value.
	 */
	public static round(value: number): number {
		return Math.round(value * 10000) / 10000;
	}
}
