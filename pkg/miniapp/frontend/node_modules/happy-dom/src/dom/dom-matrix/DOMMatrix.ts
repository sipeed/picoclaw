import * as PropertySymbol from '../../PropertySymbol.js';
import DOMMatrixReadOnly from './DOMMatrixReadOnly.js';
import IDOMMatrixCompatibleObject from './IDOMMatrixCompatibleObject.js';
import TDOMMatrixInit from './TDOMMatrixInit.js';

/**
 * DOM Matrix.
 *
 * Based on:
 * - https://github.com/thednp/dommatrix/tree/master
 * - https://github.com/trusktr/geometry-interfaces
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/DOMMatrix
 */
export default class DOMMatrix extends DOMMatrixReadOnly {
	/**
	 * Returns the `a` value of the matrix.
	 */
	public get a(): number {
		return this[PropertySymbol.m11];
	}

	/**
	 * Sets the `a` value of the matrix.
	 */
	public set a(value: number) {
		this[PropertySymbol.m11] = value;
	}

	/**
	 * Returns the `b` value of the matrix.
	 */
	public get b(): number {
		return this[PropertySymbol.m12];
	}

	/**
	 * Sets the `b` value of the matrix.
	 */
	public set b(value: number) {
		this[PropertySymbol.m12] = value;
	}

	/**
	 * Returns the `c` value of the matrix.
	 */
	public get c(): number {
		return this[PropertySymbol.m21];
	}

	/**
	 * Sets the `c` value of the matrix.
	 */
	public set c(value: number) {
		this[PropertySymbol.m21] = value;
	}

	/**
	 * Returns the `d` value of the matrix.
	 */
	public get d(): number {
		return this[PropertySymbol.m22];
	}

	/**
	 * Sets the `d` value of the matrix.
	 */
	public set d(value: number) {
		this[PropertySymbol.m22] = value;
	}

	/**
	 * Returns the `e` value of the matrix.
	 */
	public get e(): number {
		return this[PropertySymbol.m41];
	}

	/**
	 * Sets the `e` value of the matrix.
	 */
	public set e(value: number) {
		this[PropertySymbol.m41] = value;
	}

	/**
	 * Returns the `f` value of the matrix.
	 */
	public get f(): number {
		return this[PropertySymbol.m42];
	}

	/**
	 * Sets the `f` value of the matrix.
	 */
	public set f(value: number) {
		this[PropertySymbol.m42] = value;
	}

	/**
	 * Returns the `m11` value of the matrix.
	 */
	public get m11(): number {
		return this[PropertySymbol.m11];
	}

	/**
	 * Sets the `m11` value of the matrix.
	 */
	public set m11(value: number) {
		this[PropertySymbol.m11] = value;
	}

	/**
	 * Returns the `m12` value of the matrix.
	 */
	public get m12(): number {
		return this[PropertySymbol.m12];
	}

	/**
	 * Sets the `m12` value of the matrix.
	 */
	public set m12(value: number) {
		this[PropertySymbol.m12] = value;
	}

	/**
	 * Returns the `m13` value of the matrix.
	 */
	public get m13(): number {
		return this[PropertySymbol.m13];
	}

	/**
	 * Sets the `m13` value of the matrix.
	 */
	public set m13(value: number) {
		this[PropertySymbol.m13] = value;
	}

	/**
	 * Returns the `m14` value of the matrix.
	 */
	public get m14(): number {
		return this[PropertySymbol.m14];
	}

	/**
	 * Sets the `m14` value of the matrix.
	 */
	public set m14(value: number) {
		this[PropertySymbol.m14] = value;
	}

	/**
	 * Returns the `m21` value of the matrix.
	 */
	public get m21(): number {
		return this[PropertySymbol.m21];
	}

	/**
	 * Sets the `m21` value of the matrix.
	 */
	public set m21(value: number) {
		this[PropertySymbol.m21] = value;
	}

	/**
	 * Returns the `m22` value of the matrix.
	 */
	public get m22(): number {
		return this[PropertySymbol.m22];
	}

	/**
	 * Sets the `m22` value of the matrix.
	 */
	public set m22(value: number) {
		this[PropertySymbol.m22] = value;
	}

	/**
	 * Returns the `m23` value of the matrix.
	 */
	public get m23(): number {
		return this[PropertySymbol.m23];
	}

	/**
	 * Sets the `m23` value of the matrix.
	 */
	public set m23(value: number) {
		this[PropertySymbol.m23] = value;
	}

	/**
	 * Returns the `m24` value of the matrix.
	 */
	public get m24(): number {
		return this[PropertySymbol.m24];
	}

	/**
	 * Sets the `m24` value of the matrix.
	 */
	public set m24(value: number) {
		this[PropertySymbol.m24] = value;
	}

	/**
	 * Returns the `m31` value of the matrix.
	 */
	public get m31(): number {
		return this[PropertySymbol.m31];
	}

	/**
	 * Sets the `m31` value of the matrix.
	 */
	public set m31(value: number) {
		this[PropertySymbol.m31] = value;
	}

	/**
	 * Returns the `m32` value of the matrix.
	 */
	public get m32(): number {
		return this[PropertySymbol.m32];
	}

	/**
	 * Sets the `m32` value of the matrix.
	 */
	public set m32(value: number) {
		this[PropertySymbol.m32] = value;
	}

	/**
	 * Returns the `m33` value of the matrix.
	 */
	public get m33(): number {
		return this[PropertySymbol.m33];
	}

	/**
	 * Sets the `m33` value of the matrix.
	 */
	public set m33(value: number) {
		this[PropertySymbol.m33] = value;
	}

	/**
	 * Returns the `m34` value of the matrix.
	 */
	public get m34(): number {
		return this[PropertySymbol.m34];
	}

	/**
	 * Sets the `m34` value of the matrix.
	 */
	public set m34(value: number) {
		this[PropertySymbol.m34] = value;
	}

	/**
	 * Returns the `m41` value of the matrix.
	 */
	public get m41(): number {
		return this[PropertySymbol.m41];
	}

	/**
	 * Sets the `m41` value of the matrix.
	 */
	public set m41(value: number) {
		this[PropertySymbol.m41] = value;
	}

	/**
	 * Returns the `m42` value of the matrix.
	 */
	public get m42(): number {
		return this[PropertySymbol.m42];
	}

	/**
	 * Sets the `m42` value of the matrix.
	 */
	public set m42(value: number) {
		this[PropertySymbol.m42] = value;
	}

	/**
	 * Returns the `m43` value of the matrix.
	 */
	public get m43(): number {
		return this[PropertySymbol.m43];
	}

	/**
	 * Sets the `m43` value of the matrix.
	 */
	public set m43(value: number) {
		this[PropertySymbol.m43] = value;
	}

	/**
	 * Returns the `m44` value of the matrix.
	 */
	public get m44(): number {
		return this[PropertySymbol.m44];
	}

	/**
	 * Sets the `m44` value of the matrix.
	 */
	public set m44(value: number) {
		this[PropertySymbol.m44] = value;
	}

	/**
	 * The `setMatrixValue` method replaces the existing matrix with one computed in the browser (e.g.`matrix(1,0.25,-0.25,1,0,0)`).
	 *
	 * @param source A `DOMMatrix`, `Float32Array`, `Float64Array`, `Array`, or DOMMatrix compatible object to set the matrix values from.
	 * @returns Self.
	 */
	public setMatrixValue(source?: TDOMMatrixInit): DOMMatrix {
		this[PropertySymbol.setMatrixValue](source);
		return this;
	}

	/**
	 * Sets self to be multiplied by the passed matrix.
	 *
	 * @param secondMatrix DOMMatrix
	 * @returns Self.
	 */
	public multiplySelf(secondMatrix: IDOMMatrixCompatibleObject): DOMMatrix {
		this[PropertySymbol.multiplySelf](secondMatrix);
		return this;
	}

	/**
	 * Sets self to multiplied by a translation matrix containing the passed values.
	 *
	 * @param [x=0] X component of the translation value.
	 * @param [y=0] Y component of the translation value.
	 * @param [z=0] Z component of the translation value.
	 * @returns Self.
	 */
	public translateSelf(x: number = 0, y: number = 0, z: number = 0): DOMMatrix {
		this[PropertySymbol.translateSelf](x, y, z);
		return this;
	}

	/**
	 * Sets self to be multiplied by a scale matrix containing the passed values.
	 *
	 * @param [scaleX] X-Axis scale.
	 * @param [scaleY] Y-Axis scale.
	 * @param [scaleZ] Z-Axis scale.
	 * @param [originX] X-Axis scale.
	 * @param [originY] Y-Axis scale.
	 * @param [originZ] Z-Axis scale.
	 * @returns Self.
	 */
	public scaleSelf(
		scaleX?: number,
		scaleY?: number,
		scaleZ = 1,
		originX = 0,
		originY = 0,
		originZ = 0
	): DOMMatrix {
		this[PropertySymbol.scaleSelf](scaleX, scaleY, scaleZ, originX, originY, originZ);
		return this;
	}

	/**
	 * Sets self to be multiplied by a scale matrix containing the passed values.
	 *
	 * @param [scale] The scale factor.
	 * @param [originX] X-Axis scale.
	 * @param [originY] Y-Axis scale.
	 * @param [originZ] Z-Axis scale.
	 * @returns Self.
	 */
	public scale3dSelf(scale = 1, originX = 0, originY = 0, originZ = 0): DOMMatrix {
		this[PropertySymbol.scale3dSelf](scale, originX, originY, originZ);
		return this;
	}

	/**
	 * Sets self to be multiplied by a scale matrix containing the passed values.
	 *
	 * @param [scaleX] X-Axis scale.
	 * @param [scaleY] Y-Axis scale.
	 * @returns Self.
	 */
	public scaleNonUniformSelf(scaleX = 1, scaleY = 1): DOMMatrix {
		this[PropertySymbol.scaleNonUniformSelf](scaleX, scaleY);
		return this;
	}

	/**
	 * Sets self to be multiplied by a rotation matrix with the given axis and `angle`.
	 *
	 * @param [x] The X component of the axis vector.
	 * @param [y] The Y component of the axis vector.
	 * @param [z] The Z component of the axis vector.
	 * @param [angle] Angle of rotation about the axis vector, in degrees.
	 * @returns Self.
	 */
	public rotateAxisAngleSelf(x = 0, y = 0, z = 0, angle = 0): DOMMatrixReadOnly {
		this[PropertySymbol.rotateAxisAngleSelf](x, y, z, angle);
		return this;
	}

	/**
	 * Set self to be multiplied by each of 3 rotation matrices about the major axes, first X, then Y, then Z.
	 *
	 * @param [x] X component of the rotation, or Z if Y and Z are null.
	 * @param [y] Y component of the rotation value.
	 * @param [z] Z component of the rotation value.
	 * @returns Self.
	 */
	public rotateSelf(x = 0, y?: number, z?: number): DOMMatrixReadOnly {
		this[PropertySymbol.rotateSelf](x, y, z);
		return this;
	}

	/**
	 * Sets self to be multiplied by a skew matrix along the X axis by the given angle.
	 *
	 * @param [x] X-Axis skew.
	 * @param [y] Y-Axis skew.
	 */
	public rotateFromVectorSelf(x = 0, y = 0): DOMMatrixReadOnly {
		this[PropertySymbol.rotateFromVectorSelf](x, y);
		return this;
	}

	/**
	 * Set self to be specified as a skew transformation along X-Axis by the given angle.
	 *
	 * @param angle Angle amount in degrees to skew.
	 * @returns Self.
	 */
	public skewXSelf(angle: number): DOMMatrixReadOnly {
		this[PropertySymbol.skewXSelf](angle);
		return this;
	}

	/**
	 * Set self to be specified as a skew transformation along Y-Axis by the given angle.
	 *
	 * @param angle Angle amount in degrees to skew.
	 * @returns Self.
	 */
	public skewYSelf(angle: number): DOMMatrixReadOnly {
		this[PropertySymbol.skewYSelf](angle);
		return this;
	}

	/**
	 * Set self to be specified as matrix flipped on X-axis.
	 */
	public flipXSelf(): DOMMatrixReadOnly {
		this[PropertySymbol.flipXSelf]();
		return this;
	}

	/**
	 * Set self to be specified as matrix flipped on Y-axis.
	 */
	public flipYSelf(): DOMMatrixReadOnly {
		this[PropertySymbol.flipYSelf]();
		return this;
	}

	/**
	 * Set self to be specified as matrix inverted.
	 */
	public invertSelf(): DOMMatrixReadOnly {
		this[PropertySymbol.invertSelf]();
		return this;
	}
}
