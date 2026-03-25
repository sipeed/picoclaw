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
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
/**
 *
 */
class TextTrackCueList extends Array {
    /**
     * Constructor.
     *
     * @param [illegalConstructorSymbol] Illegal constructor symbol.
     */
    constructor(illegalConstructorSymbol) {
        super();
        // "illegalConstructorSymbol" can be "1" when calling the "splice()" method
        if (illegalConstructorSymbol !== 1 &&
            illegalConstructorSymbol !== PropertySymbol.illegalConstructor) {
            throw new TypeError('Illegal constructor');
        }
    }
    /**
     * Returns the first TextTrackCue object with the identifier passed to it.
     *
     * @param id Text track cue identifier.
     */
    getCueById(id) {
        for (const cue of this) {
            if (cue.id === id) {
                return cue;
            }
        }
        return null;
    }
}
exports.default = TextTrackCueList;
//# sourceMappingURL=TextTrackCueList.cjs.map