import * as PropertySymbol from '../PropertySymbol.js';
import Window from './Window.js';
import { Buffer } from 'buffer';
/**
 * Browser window.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/Window.
 */
export default class GlobalWindow extends Window {
    // Node.js Globals
    Array = globalThis.Array;
    ArrayBuffer = globalThis.ArrayBuffer;
    Boolean = globalThis.Boolean;
    Buffer = Buffer;
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
//# sourceMappingURL=GlobalWindow.js.map