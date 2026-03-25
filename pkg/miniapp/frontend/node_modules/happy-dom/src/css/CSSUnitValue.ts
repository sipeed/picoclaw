import CSSUnits from './CSSUnits.js';

/**
 * CSS unit value.
 */
export default class CSSUnitValue {
	public unit: string = null;
	public value: number = null;

	/**
	 * Constructor.
	 *
	 * @param value Value.
	 * @param unit Unit.
	 */
	constructor(value: number, unit: string) {
		if (typeof value !== 'number') {
			throw new TypeError('The provided double value is non-finite');
		}
		if (!CSSUnits.includes(unit)) {
			throw new TypeError('Invalid unit: ' + unit);
		}
		this.value = value;
		this.unit = unit;
	}
}
