"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
Object.defineProperty(exports, "__esModule", { value: true });
const PropertySymbol = __importStar(require("../PropertySymbol.cjs"));
/**
 * SVG Unit Types.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGUnitTypes
 */
class SVGUnitTypes {
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
exports.default = SVGUnitTypes;
//# sourceMappingURL=SVGUnitTypes.cjs.map