import * as PropertySymbol from '../PropertySymbol.js';
import BrowserWindow from '../window/BrowserWindow.js';
import DOMMatrix from '../dom/dom-matrix/DOMMatrix.js';

const TRANSFORM_REGEXP = /([a-zA-Z0-9]+)\(([^)]+)\)/;
const TRANSFORM_PARAMETER_SPLIT_REGEXP = /[\s,]+/;

/**
 * SVG Matrix.
 *
 * Documentation missing at developer.mozilla.org.
 */
export default class SVGMatrix {
	// Internal properties
	public [PropertySymbol.window]: BrowserWindow;
	public [PropertySymbol.getAttribute]: (() => string | null) | null = null;
	public [PropertySymbol.setAttribute]: ((value: string) => void) | null = null;
	public [PropertySymbol.attributeValue]: string | null = null;
	public [PropertySymbol.readOnly]: boolean = false;

	/**
	 * Constructor.
	 *
	 * @param illegalConstructorSymbol Illegal constructor symbol.
	 * @param window Window.
	 * @param [options] Options.
	 * @param [options.readOnly] Read only.
	 * @param [options.getAttribute] Get attribute.
	 * @param [options.setAttribute] Set attribute.
	 */
	constructor(
		illegalConstructorSymbol: symbol,
		window: BrowserWindow,
		options?: {
			readOnly?: boolean;
			getAttribute?: () => string | null;
			setAttribute?: (value: string) => void;
		}
	) {
		if (illegalConstructorSymbol !== PropertySymbol.illegalConstructor) {
			throw new TypeError('Illegal constructor');
		}

		this[PropertySymbol.window] = window;

		if (options) {
			this[PropertySymbol.readOnly] = !!options.readOnly;
			this[PropertySymbol.getAttribute] = options.getAttribute || null;
			this[PropertySymbol.setAttribute] = options.setAttribute || null;
		}
	}

	/**
	 * Returns the `a` value of the matrix.
	 */
	public get a(): number {
		return this[PropertySymbol.getDOMMatrix]().a;
	}

	/**
	 * Sets the `a` value of the matrix.
	 */
	public set a(value: number) {
		if (this[PropertySymbol.readOnly]) {
			return;
		}
		const domMatrix = this[PropertySymbol.getDOMMatrix]();
		domMatrix.a = value;
		this[PropertySymbol.setDOMMatrix](domMatrix);
	}

	/**
	 * Returns the `b` value of the matrix.
	 */
	public get b(): number {
		return this[PropertySymbol.getDOMMatrix]().b;
	}

	/**
	 * Sets the `b` value of the matrix.
	 */
	public set b(value: number) {
		if (this[PropertySymbol.readOnly]) {
			return;
		}
		const domMatrix = this[PropertySymbol.getDOMMatrix]();
		domMatrix.b = value;
		this[PropertySymbol.setDOMMatrix](domMatrix);
	}

	/**
	 * Returns the `c` value of the matrix.
	 */
	public get c(): number {
		return this[PropertySymbol.getDOMMatrix]().c;
	}

	/**
	 * Sets the `c` value of the matrix.
	 */
	public set c(value: number) {
		if (this[PropertySymbol.readOnly]) {
			return;
		}
		const domMatrix = this[PropertySymbol.getDOMMatrix]();
		domMatrix.c = value;
		this[PropertySymbol.setDOMMatrix](domMatrix);
	}

	/**
	 * Returns the `d` value of the matrix.
	 */
	public get d(): number {
		return this[PropertySymbol.getDOMMatrix]().d;
	}

	/**
	 * Sets the `d` value of the matrix.
	 */
	public set d(value: number) {
		if (this[PropertySymbol.readOnly]) {
			return;
		}
		const domMatrix = this[PropertySymbol.getDOMMatrix]();
		domMatrix.d = value;
		this[PropertySymbol.setDOMMatrix](domMatrix);
	}

	/**
	 * Returns the `e` value of the matrix.
	 */
	public get e(): number {
		return this[PropertySymbol.getDOMMatrix]().e;
	}

	/**
	 * Sets the `e` value of the matrix.
	 */
	public set e(value: number) {
		if (this[PropertySymbol.readOnly]) {
			return;
		}
		const domMatrix = this[PropertySymbol.getDOMMatrix]();
		domMatrix.e = value;
		this[PropertySymbol.setDOMMatrix](domMatrix);
	}

	/**
	 * Returns the `f` value of the matrix.
	 */
	public get f(): number {
		return this[PropertySymbol.getDOMMatrix]().f;
	}

	/**
	 * Sets the `f` value of the matrix.
	 */
	public set f(value: number) {
		if (this[PropertySymbol.readOnly]) {
			return;
		}
		const domMatrix = this[PropertySymbol.getDOMMatrix]();
		domMatrix.f = value;
		this[PropertySymbol.setDOMMatrix](domMatrix);
	}

	/**
	 * Returns a new SVGMatrix instance which is the result of this matrix multiplied by the passed matrix.
	 *
	 * @param secondMatrix Matrix object.
	 * @returns A new SVGMatrix object.
	 */
	public multiply(secondMatrix: SVGMatrix): SVGMatrix {
		if (!(secondMatrix instanceof SVGMatrix)) {
			throw new this[PropertySymbol.window].TypeError(
				"Failed to execute 'multiply' on 'SVGMatrix': parameter 1 is not of type 'SVGMatrix'."
			);
		}
		const domMatrix = this[PropertySymbol.getDOMMatrix]();
		const svgMatrix = new SVGMatrix(PropertySymbol.illegalConstructor, this[PropertySymbol.window]);
		domMatrix.multiplySelf(secondMatrix[PropertySymbol.getDOMMatrix]());
		svgMatrix[PropertySymbol.setDOMMatrix](domMatrix);
		return svgMatrix;
	}

	/**
	 * Returns a new SVGMatrix instance which is this matrix post multiplied by a translation matrix containing the passed values.
	 *
	 * @param [x=0] X component of the translation value.
	 * @param [y=0] Y component of the translation value.
	 * @returns The resulted matrix
	 */
	public translate(x: number = 0, y: number = 0): SVGMatrix {
		const domMatrix = this[PropertySymbol.getDOMMatrix]();
		const svgMatrix = new SVGMatrix(PropertySymbol.illegalConstructor, this[PropertySymbol.window]);
		domMatrix.translateSelf(x, y);
		svgMatrix[PropertySymbol.setDOMMatrix](domMatrix);
		return svgMatrix;
	}

	/**
	 * Returns a new SVGMatrix instance which is this matrix post multiplied by a scale 2D matrix containing the passed values.
	 *
	 * @param scale The scale factor.
	 * @returns The resulted matrix
	 */
	public scale(scale: number): SVGMatrix {
		const domMatrix = this[PropertySymbol.getDOMMatrix]();
		const svgMatrix = new SVGMatrix(PropertySymbol.illegalConstructor, this[PropertySymbol.window]);
		domMatrix.scaleSelf(scale);
		svgMatrix[PropertySymbol.setDOMMatrix](domMatrix);
		return svgMatrix;
	}

	/**
	 * Returns a new SVGMatrix instance which is this matrix post multiplied by a scale 3D matrix containing the passed values.
	 *
	 * @param [scaleX] X-Axis scale.
	 * @param [scaleY] Y-Axis scale.
	 * @returns The resulted matrix
	 */
	public scaleNonUniform(scaleX = 1, scaleY = 1): SVGMatrix {
		const domMatrix = this[PropertySymbol.getDOMMatrix]();
		const svgMatrix = new SVGMatrix(PropertySymbol.illegalConstructor, this[PropertySymbol.window]);
		domMatrix.scaleNonUniformSelf(scaleX, scaleY);
		svgMatrix[PropertySymbol.setDOMMatrix](domMatrix);
		return svgMatrix;
	}

	/**
	 * Returns a new SVGMatrix instance which is this matrix post multiplied by each of 3 rotation matrices about the major axes, first X, then Y, then Z.
	 *
	 * @param angle Angle of rotation in degrees.
	 * @returns The resulted matrix
	 */
	public rotate(angle: number): SVGMatrix {
		const domMatrix = this[PropertySymbol.getDOMMatrix]();
		const svgMatrix = new SVGMatrix(PropertySymbol.illegalConstructor, this[PropertySymbol.window]);
		domMatrix.rotateSelf(angle);
		svgMatrix[PropertySymbol.setDOMMatrix](domMatrix);
		return svgMatrix;
	}

	/**
	 * Returns a new SVGMatrix instance which is this matrix post multiplied by a skew matrix along the X axis by the given angle.
	 *
	 * Not implemented in Happy DOM yet.
	 *
	 * @param [x] X-Axis skew.
	 * @param [y] Y-Axis skew.
	 */
	public rotateFromVector(x: number = 0, y: number = 0): SVGMatrix {
		const domMatrix = this[PropertySymbol.getDOMMatrix]();
		const svgMatrix = new SVGMatrix(PropertySymbol.illegalConstructor, this[PropertySymbol.window]);
		domMatrix.rotateFromVectorSelf(x, y);
		svgMatrix[PropertySymbol.setDOMMatrix](domMatrix);
		return svgMatrix;
	}

	/**
	 * Returns a new SVGMatrix instance that specifies a skew transformation along X-Axis by the given angle.
	 *
	 * @param angle Angle amount in degrees to skew.
	 * @returns The resulted matrix
	 */
	public skewX(angle: number): SVGMatrix {
		const domMatrix = this[PropertySymbol.getDOMMatrix]();
		const svgMatrix = new SVGMatrix(PropertySymbol.illegalConstructor, this[PropertySymbol.window]);
		domMatrix.skewXSelf(angle);
		svgMatrix[PropertySymbol.setDOMMatrix](domMatrix);
		return svgMatrix;
	}

	/**
	 * Returns a new SVGMatrix instance that specifies a skew transformation along Y-Axis by the given angle.
	 *
	 * @param angle Angle amount in degrees to skew.
	 * @returns The resulted matrix
	 */
	public skewY(angle: number): SVGMatrix {
		const domMatrix = this[PropertySymbol.getDOMMatrix]();
		const svgMatrix = new SVGMatrix(PropertySymbol.illegalConstructor, this[PropertySymbol.window]);
		domMatrix.skewYSelf(angle);
		svgMatrix[PropertySymbol.setDOMMatrix](domMatrix);
		return svgMatrix;
	}

	/**
	 * Returns a new SVGMatrix instance which is this matrix flipped on X-axis.
	 */
	public flipX(): SVGMatrix {
		const domMatrix = this[PropertySymbol.getDOMMatrix]();
		const svgMatrix = new SVGMatrix(PropertySymbol.illegalConstructor, this[PropertySymbol.window]);
		domMatrix.flipXSelf();
		svgMatrix[PropertySymbol.setDOMMatrix](domMatrix);
		return svgMatrix;
	}

	/**
	 * Returns a new SVGMatrix instance which is this matrix flipped on Y-axis.
	 */
	public flipY(): SVGMatrix {
		const domMatrix = this[PropertySymbol.getDOMMatrix]();
		const svgMatrix = new SVGMatrix(PropertySymbol.illegalConstructor, this[PropertySymbol.window]);
		domMatrix.flipYSelf();
		svgMatrix[PropertySymbol.setDOMMatrix](domMatrix);
		return svgMatrix;
	}

	/**
	 * Returns a new SVGMatrix instance which is this matrix inverted.
	 */
	public inverse(): SVGMatrix {
		const domMatrix = this[PropertySymbol.getDOMMatrix]();
		const svgMatrix = new SVGMatrix(PropertySymbol.illegalConstructor, this[PropertySymbol.window]);
		domMatrix.invertSelf();
		svgMatrix[PropertySymbol.setDOMMatrix](domMatrix);
		return svgMatrix;
	}

	/**
	 * Returns DOM matrix.
	 *
	 * @returns DOM matrix.
	 */
	public [PropertySymbol.getDOMMatrix](): DOMMatrix {
		const attribute = this[PropertySymbol.getAttribute]
			? this[PropertySymbol.getAttribute]()
			: this[PropertySymbol.attributeValue];

		if (!attribute) {
			return new DOMMatrix();
		}

		const match = attribute.match(TRANSFORM_REGEXP);

		if (!match) {
			return new DOMMatrix();
		}
		const parameters: number[] = [];

		for (const parameter of match[2].trim().split(TRANSFORM_PARAMETER_SPLIT_REGEXP)) {
			const value = Number(parameter);
			if (isNaN(value)) {
				throw new this[PropertySymbol.window].TypeError(
					`Failed to parse transform attribute: Expected number, but got "${parameter}" in "${attribute}".`
				);
			}
			parameters.push(value);
		}

		switch (match[1]) {
			case 'matrix':
				if (parameters.length !== 6) {
					throw new this[PropertySymbol.window].TypeError(
						`Failed to parse transform attribute: Expected 6 parameters in "${attribute}".`
					);
				}
				return <DOMMatrix>DOMMatrix[PropertySymbol.fromString](attribute);
			case 'scale':
			case 'translate':
				if (parameters.length !== 1 && parameters.length !== 2) {
					throw new this[PropertySymbol.window].TypeError(
						`Failed to parse transform attribute: Expected 1 or 2 parameters in "${attribute}".`
					);
				}
				return <DOMMatrix>DOMMatrix[PropertySymbol.fromString](attribute);
			case 'skewY':
			case 'skewX':
				if (parameters.length !== 1) {
					throw new this[PropertySymbol.window].TypeError(
						`Failed to parse transform attribute: Expected 1 parameter in "${attribute}".`
					);
				}
				return <DOMMatrix>DOMMatrix[PropertySymbol.fromString](attribute);
			case 'rotate':
				const domMatrix = new DOMMatrix();

				if (parameters.length !== 1 && parameters.length !== 3) {
					throw new this[PropertySymbol.window].TypeError(
						`Failed to parse transform attribute: Expected 1 or 3 parameters in "${attribute}".`
					);
				}

				const [angle, x, y] = parameters;

				if (x || y) {
					domMatrix.translateSelf(x, y);
				}

				const radian = (angle * Math.PI) / 180;

				/**
				 * @see https://www.w3.org/TR/SVG11/coords.html#TransformAttribute
				 **/
				domMatrix.multiplySelf(
					// prettier-ignore
					new DOMMatrix([
						Math.cos(radian), Math.sin(radian), -Math.sin(radian),
						Math.cos(radian), 0,                 0
					])
				);

				if (x || y) {
					domMatrix.translateSelf(-x, -y);
				}

				return domMatrix;
			default:
				throw new this[PropertySymbol.window].TypeError(
					`Failed to parse transform attribute: Unknown transformation "${attribute}".`
				);
		}
	}

	/**
	 * Sets DOM matrix.
	 *
	 * @param domMatrix DOM matrix.
	 */
	public [PropertySymbol.setDOMMatrix](domMatrix: DOMMatrix): void {
		this[PropertySymbol.attributeValue] = domMatrix.toString().replace(/, /g, ' ');

		if (this[PropertySymbol.setAttribute]) {
			this[PropertySymbol.setAttribute](this[PropertySymbol.attributeValue]);
		}
	}
}
