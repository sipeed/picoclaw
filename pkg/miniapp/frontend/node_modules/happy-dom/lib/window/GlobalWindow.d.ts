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
    Array: typeof Array;
    ArrayBuffer: typeof ArrayBuffer;
    Boolean: typeof Boolean;
    Buffer: typeof Buffer;
    DataView: typeof DataView;
    Date: typeof Date;
    Error: typeof Error;
    EvalError: typeof EvalError;
    Float32Array: typeof Float32Array;
    Float64Array: typeof Float64Array;
    Function: typeof Function;
    Infinity: typeof Infinity;
    Int16Array: typeof Int16Array;
    Int32Array: typeof Int32Array;
    Int8Array: typeof Int8Array;
    Intl: typeof Intl;
    JSON: typeof JSON;
    Map: MapConstructor;
    Math: typeof Math;
    NaN: typeof NaN;
    Number: typeof Number;
    Object: typeof Object;
    Promise: typeof Promise;
    RangeError: typeof RangeError;
    ReferenceError: typeof ReferenceError;
    RegExp: typeof RegExp;
    Set: SetConstructor;
    String: typeof String;
    Symbol: Function;
    SyntaxError: typeof SyntaxError;
    TypeError: typeof TypeError;
    URIError: typeof URIError;
    Uint16Array: typeof Uint16Array;
    Uint32Array: typeof Uint32Array;
    Uint8Array: typeof Uint8Array;
    Uint8ClampedArray: typeof Uint8ClampedArray;
    WeakMap: WeakMapConstructor;
    WeakSet: WeakSetConstructor;
    decodeURI: typeof decodeURI;
    decodeURIComponent: typeof decodeURIComponent;
    encodeURI: typeof encodeURI;
    encodeURIComponent: typeof encodeURIComponent;
    eval: typeof eval;
    /**
     * @deprecated
     */
    escape: (str: string) => string;
    global: typeof globalThis;
    isFinite: typeof isFinite;
    isNaN: typeof isNaN;
    parseFloat: typeof parseFloat;
    parseInt: typeof parseInt;
    undefined: typeof undefined;
    /**
     * @deprecated
     */
    unescape: (str: string) => string;
    gc: () => void;
    v8debug?: unknown;
    /**
     * Setup of VM context.
     */
    protected [PropertySymbol.setupVMContext](): void;
}
//# sourceMappingURL=GlobalWindow.d.ts.map