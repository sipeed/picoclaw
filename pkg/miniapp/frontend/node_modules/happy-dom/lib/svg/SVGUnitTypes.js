import * as PropertySymbol from '../PropertySymbol.js';
/**
 * SVG Unit Types.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGUnitTypes
 */
export default class SVGUnitTypes {
    static SVG_UNIT_TYPE_UNKNOWN = 0;
    static SVG_UNIT_TYPE_USERSPACEONUSE = 1;
    static SVG_UNIT_TYPE_OBJECTBOUNDINGBOX = 2;
    SVG_UNIT_TYPE_UNKNOWN = 0;
    SVG_UNIT_TYPE_USERSPACEONUSE = 1;
    SVG_UNIT_TYPE_OBJECTBOUNDINGBOX = 2;
    /**
     * Constructor.
     *
     * @param illegalConstructorSymbol Illegal constructor symbol.
     */
    constructor(illegalConstructorSymbol) {
        if (illegalConstructorSymbol !== PropertySymbol.illegalConstructor) {
            throw new TypeError('Illegal constructor');
        }
    }
}
//# sourceMappingURL=SVGUnitTypes.js.map