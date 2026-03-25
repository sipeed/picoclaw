import * as PropertySymbol from '../PropertySymbol.js';

/**
 * SVG Unit Types.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGUnitTypes
 */
export default class SVGUnitTypes {
	public static readonly SVG_UNIT_TYPE_UNKNOWN = 0;
	public static readonly SVG_UNIT_TYPE_USERSPACEONUSE = 1;
	public static readonly SVG_UNIT_TYPE_OBJECTBOUNDINGBOX = 2;
	public readonly SVG_UNIT_TYPE_UNKNOWN = 0;
	public readonly SVG_UNIT_TYPE_USERSPACEONUSE = 1;
	public readonly SVG_UNIT_TYPE_OBJECTBOUNDINGBOX = 2;

	/**
	 * Constructor.
	 *
	 * @param illegalConstructorSymbol Illegal constructor symbol.
	 */
	constructor(illegalConstructorSymbol: Symbol) {
		if (illegalConstructorSymbol !== PropertySymbol.illegalConstructor) {
			throw new TypeError('Illegal constructor');
		}
	}
}
