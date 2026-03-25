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
	public Array: typeof Array = globalThis.Array;
	public ArrayBuffer: typeof ArrayBuffer = globalThis.ArrayBuffer;
	public Boolean: typeof Boolean = globalThis.Boolean;
	public Buffer: typeof Buffer = Buffer;
	public DataView: typeof DataView = globalThis.DataView;
	public Date: typeof Date = globalThis.Date;
	public Error: typeof Error = globalThis.Error;
	public EvalError: typeof EvalError = globalThis.EvalError;
	public Float32Array: typeof Float32Array = globalThis.Float32Array;
	public Float64Array: typeof Float64Array = globalThis.Float64Array;
	public Function: typeof Function = globalThis.Function;
	public Infinity: typeof Infinity = globalThis.Infinity;
	public Int16Array: typeof Int16Array = globalThis.Int16Array;
	public Int32Array: typeof Int32Array = globalThis.Int32Array;
	public Int8Array: typeof Int8Array = globalThis.Int8Array;
	public Intl: typeof Intl = globalThis.Intl;
	public JSON: typeof JSON = globalThis.JSON;
	public Map: MapConstructor = globalThis.Map;
	public Math: typeof Math = globalThis.Math;
	public NaN: typeof NaN = globalThis.NaN;
	public Number: typeof Number = globalThis.Number;
	public Object: typeof Object = globalThis.Object;
	public Promise: typeof Promise = globalThis.Promise;
	public RangeError: typeof RangeError = globalThis.RangeError;
	public ReferenceError: typeof ReferenceError = globalThis.ReferenceError;
	public RegExp: typeof RegExp = globalThis.RegExp;
	public Set: SetConstructor = globalThis.Set;
	public String: typeof String = globalThis.String;
	public Symbol: Function = globalThis.Symbol;
	public SyntaxError: typeof SyntaxError = globalThis.SyntaxError;
	public TypeError: typeof TypeError = globalThis.TypeError;
	public URIError: typeof URIError = globalThis.URIError;
	public Uint16Array: typeof Uint16Array = globalThis.Uint16Array;
	public Uint32Array: typeof Uint32Array = globalThis.Uint32Array;
	public Uint8Array: typeof Uint8Array = globalThis.Uint8Array;
	public Uint8ClampedArray: typeof Uint8ClampedArray = globalThis.Uint8ClampedArray;
	public WeakMap: WeakMapConstructor = globalThis.WeakMap;
	public WeakSet: WeakSetConstructor = globalThis.WeakSet;
	public decodeURI: typeof decodeURI = globalThis.decodeURI;
	public decodeURIComponent: typeof decodeURIComponent = globalThis.decodeURIComponent;
	public encodeURI: typeof encodeURI = globalThis.encodeURI;
	public encodeURIComponent: typeof encodeURIComponent = globalThis.encodeURIComponent;
	public eval: typeof eval = globalThis.eval;
	/**
	 * @deprecated
	 */
	public escape: (str: string) => string = globalThis.escape;
	public global: typeof globalThis = globalThis;
	public isFinite: typeof isFinite = globalThis.isFinite;
	public isNaN: typeof isNaN = globalThis.isNaN;
	public parseFloat: typeof parseFloat = globalThis.parseFloat;
	public parseInt: typeof parseInt = globalThis.parseInt;
	public undefined: typeof undefined = globalThis.undefined;
	/**
	 * @deprecated
	 */
	public unescape: (str: string) => string = globalThis.unescape;
	public gc: () => void = globalThis.gc;
	public v8debug?: unknown = globalThis.v8debug;

	/**
	 * Setup of VM context.
	 */
	protected override [PropertySymbol.setupVMContext](): void {
		// Do nothing
	}
}
