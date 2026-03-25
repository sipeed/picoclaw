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
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const PropertySymbol = __importStar(require("../PropertySymbol.cjs"));
const Window_js_1 = __importDefault(require("./Window.cjs"));
const buffer_1 = require("buffer");
/**
 * Browser window.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/Window.
 */
class GlobalWindow extends Window_js_1.default {
    // Node.js Globals
    Array = globalThis.Array;
    ArrayBuffer = globalThis.ArrayBuffer;
    Boolean = globalThis.Boolean;
    Buffer = buffer_1.Buffer;
    DataView = globalThis.DataView;
    Date = globalThis.Date;
    Error = globalThis.Error;
    EvalError = globalThis.EvalError;
    Float32Array = globalThis.Float32Array;
    Float64Array = globalThis.Float64Array;
    Function = globalThis.Function;
    Infinity = globalThis.Infinity;
    Int16Array = globalThis.Int16Array;
    Int32Array = globalThis.Int32Array;
    Int8Array = globalThis.Int8Array;
    Intl = globalThis.Intl;
    JSON = globalThis.JSON;
    Map = globalThis.Map;
    Math = globalThis.Math;
    NaN = globalThis.NaN;
    Number = globalThis.Number;
    Object = globalThis.Object;
    Promise = globalThis.Promise;
    RangeError = globalThis.RangeError;
    ReferenceError = globalThis.ReferenceError;
    RegExp = globalThis.RegExp;
    Set = globalThis.Set;
    String = globalThis.String;
    Symbol = globalThis.Symbol;
    SyntaxError = globalThis.SyntaxError;
    TypeError = globalThis.TypeError;
    URIError = globalThis.URIError;
    Uint16Array = globalThis.Uint16Array;
    Uint32Array = globalThis.Uint32Array;
    Uint8Array = globalThis.Uint8Array;
    Uint8ClampedArray = globalThis.Uint8ClampedArray;
    WeakMap = globalThis.WeakMap;
    WeakSet = globalThis.WeakSet;
    decodeURI = globalThis.decodeURI;
    decodeURIComponent = globalThis.decodeURIComponent;
    encodeURI = globalThis.encodeURI;
    encodeURIComponent = globalThis.encodeURIComponent;
    eval = globalThis.eval;
    /**
     * @deprecated
     */
    escape = globalThis.escape;
    global = globalThis;
    isFinite = globalThis.isFinite;
    isNaN = globalThis.isNaN;
    parseFloat = globalThis.parseFloat;
    parseInt = globalThis.parseInt;
    undefined = globalThis.undefined;
    /**
     * @deprecated
     */
    unescape = globalThis.unescape;
    gc = globalThis.gc;
    v8debug = globalThis.v8debug;
    /**
     * Setup of VM context.
     */
    [PropertySymbol.setupVMContext]() {
        // Do nothing
    }
}
exports.default = GlobalWindow;
//# sourceMappingURL=GlobalWindow.cjs.map