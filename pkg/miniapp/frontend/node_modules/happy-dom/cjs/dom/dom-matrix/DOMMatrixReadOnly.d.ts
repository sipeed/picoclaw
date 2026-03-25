import DOMPoint from '../DOMPoint.cjs';
import IDOMPointInit from '../IDOMPointInit.cjs';
import * as PropertySymbol from '../../PropertySymbol.cjs';
import TDOMMatrixInit from './TDOMMatrixInit.cjs';
import TDOMMatrix2DArray from './TDOMMatrix2DArray.cjs';
import TDOMMatrix3DArray from './TDOMMatrix3DArray.cjs';
import IDOMMatrixJSON from './IDOMMatrixJSON.cjs';
import IDOMMatrixCompatibleObject from './IDOMMatrixCompatibleObject.cjs';
/**
 * DOM Matrix.
 *
 * Based on:
 * - https://github.com/trusktr/geometry-interfaces
 * - https://github.com/thednp/dommatrix/tree/master
 * - https://github.com/jarek-foksa/geometry-polyfill/blob/master/geometry-polyfill.js
 * - https://github.com/Automattic/node-canvas/blob/master/lib/DOMMatrix.js
 *
 *
 * 3D Matrix:
 * _________________________
 * | m11 | m21 | m31 | m41 |
 * | m12 | m22 | m32 | m42 |
 * | m13 | m23 | m33 | m43 |
 * | m14 | m24 | m34 | m44 |
 * -------------------------̣
 *
 * 2D Matrix:
 * _________________________
 * | m11 | m21 | 0   | m41 |
 * | m12 | m22 | 0   | m42 |
 * | 0   | 0   | 1   | 0   |
 * | 0   | 0   | 0   | 1   |
 * -------------------------
 *
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/DOMMatrixReadOnly
 */
export default class DOMMatrixReadOnly {
    #private;
    [PropertySymbol.m11]: number;
    [PropertySymbol.m12]: number;
    [PropertySymbol.m13]: number;
    [PropertySymbol.m14]: number;
    [PropertySymbol.m21]: number;
    [PropertySymbol.m22]: number;
    [PropertySymbol.m23]: number;
    [PropertySymbol.m24]: number;
    [PropertySymbol.m31]: number;
    [PropertySymbol.m32]: number;
    [PropertySymbol.m33]: number;
    [PropertySymbol.m34]: number;
    [PropertySymbol.m41]: number;
    [PropertySymbol.m42]: number;
    [PropertySymbol.m43]: number;
    [PropertySymbol.m44]: number;
    /**
     * Constructor.
     *
     * @param init Init parameter.
     */
    constructor(init?: TDOMMatrixInit);
    /**
     * Returns the `a` value of the matrix.
     */
    get a(): number;
    /**
     * Returns the `b` value of the matrix.
     */
    get b(): number;
    /**
     * Returns the `c` value of the matrix.
     */
    get c(): number;
    /**
     * Returns the `d` value of the matrix.
     */
    get d(): number;
    /**
     * Returns the `e` value of the matrix.
     */
    get e(): number;
    /**
     * Returns the `f` value of the matrix.
     */
    get f(): number;
    /**
     * Returns the `m11` value of the matrix.
     */
    get m11(): number;
    /**
     * Returns the `m12` value of the matrix.
     */
    get m12(): number;
    /**
     * Returns the `m13` value of the matrix.
     */
    get m13(): number;
    /**
     * Returns the `m14` value of the matrix.
     */
    get m14(): number;
    /**
     * Returns the `m21` value of the matrix.
     */
    get m21(): number;
    /**
     * Returns the `m22` value of the matrix.
     */
    get m22(): number;
    /**
     * Returns the `m23` value of the matrix.
     */
    get m23(): number;
    /**
     * Returns the `m24` value of the matrix.
     */
    get m24(): number;
    /**
     * Returns the `m31` value of the matrix.
     */
    get m31(): number;
    /**
     * Returns the `m32` value of the matrix.
     */
    get m32(): number;
    /**
     * Returns the `m33` value of the matrix.
     */
    get m33(): number;
    /**
     * Returns the `m34` value of the matrix.
     */
    get m34(): number;
    /**
     * Returns the `m41` value of the matrix.
     */
    get m41(): number;
    /**
     * Returns the `m42` value of the matrix.
     */
    get m42(): number;
    /**
     * Returns the `m43` value of the matrix.
     */
    get m43(): number;
    /**
     * Returns the `m44` value of the matrix.
     */
    get m44(): number;
    /**
     * A `Boolean` whose value is `true` if the matrix is the identity matrix.
     *
     * The identity matrix is one in which every value is 0 except those on the main diagonal from top-left to bottom-right corner (in other words, where the offsets in each direction are equal).
     *
     * @returns "true" if the matrix is the identity matrix.
     */
    get isIdentity(): boolean;
    /**
     * A `Boolean` flag whose value is `true` if the matrix is a 2D matrix and `false` if the matrix is 3D.
     *
     * @returns "true" if the matrix is a 2D matrix.
     */
    get is2D(): boolean;
    /**
     * Returns a *Float32Array* containing elements which comprise the matrix.
     *
     * The method can return either the 16 elements or the 6 elements depending on the value of the `is2D` parameter.
     *
     * @param [is2D] Set to `true` to return a 2D matrix.
     * @returns An *Array* representation of the matrix.
     */
    toFloat32Array(is2D?: boolean): Float32Array;
    /**
     * Returns a *Float64Array* containing elements which comprise the matrix.
     *
     * The method can return either the 16 elements or the 6 elements depending on the value of the `is2D` parameter.
     *
     * @param [is2D] Set to `true` to return a 2D matrix.
     * @returns An *Array* representation of the matrix
     */
    toFloat64Array(is2D?: boolean): Float64Array;
    /**
     * Returns a string representation of the matrix in `CSS` matrix syntax, using the appropriate `CSS` matrix notation.
     *
     * Examples:
     * - `matrix3d(m11, m12, m13, m14, m21, ...)`
     * - `matrix(a, b, c, d, e, f)`
     *
     * @returns A string representation of the matrix.
     */
    toString(): string;
    /**
     * Returns an Object that can be serialized to a JSON string.
     *
     * The result can be used as a second parameter for the `fromMatrix` static method to load values into another matrix instance.
     *
     * @returns An *Object* with matrix values.
     */
    toJSON(): IDOMMatrixJSON;
    /**
     * Returns a new DOMMatrix instance which is the result of this matrix multiplied by the passed matrix.
     *
     * @param secondMatrix DOMMatrix
     * @returns A new DOMMatrix object.
     */
    multiply(secondMatrix: IDOMMatrixCompatibleObject): DOMMatrixReadOnly;
    /**
     * Returns a new DOMMatrix instance which is this matrix post multiplied by a translation matrix containing the passed values.
     *
     * @param [x=0] X component of the translation value.
     * @param [y=0] Y component of the translation value.
     * @param [z=0] Z component of the translation value.
     * @returns The resulted matrix
     */
    translate(x?: number, y?: number, z?: number): DOMMatrixReadOnly;
    /**
     * Returns a new DOMMatrix instance which is this matrix post multiplied by a scale 2D matrix containing the passed values.
     *
     * @param [scaleX] X-Axis scale.
     * @param [scaleY] Y-Axis scale.
     * @param [scaleZ] Z-Axis scale.
     * @param [originX] X-Axis scale.
     * @param [originY] Y-Axis scale.
     * @param [originZ] Z-Axis scale.
     * @returns The resulted matrix
     */
    scale(scaleX?: number, scaleY?: number, scaleZ?: number, originX?: number, originY?: number, originZ?: number): DOMMatrixReadOnly;
    /**
     * Returns a new DOMMatrix instance which is this matrix post multiplied by a scale 3D matrix containing the passed values.
     *
     * @param [scale] The scale factor.
     * @param [originX] X-Axis scale.
     * @param [originY] Y-Axis scale.
     * @param [originZ] Z-Axis scale.
     * @returns The resulted matrix
     */
    scale3d(scale?: number, originX?: number, originY?: number, originZ?: number): DOMMatrixReadOnly;
    /**
     * Returns a new DOMMatrix instance which is this matrix post multiplied by a scale 3D matrix containing the passed values.
     *
     * @param [scaleX] X-Axis scale.
     * @param [scaleY] Y-Axis scale.
     * @returns The resulted matrix
     */
    scaleNonUniform(scaleX?: number, scaleY?: number): DOMMatrixReadOnly;
    /**
     * Returns a new DOMMatrix instance which is this matrix post multiplied by a rotation matrix with the given axis and `angle`.
     *
     * @param [x] The X component of the axis vector.
     * @param [y] The Y component of the axis vector.
     * @param [z] The Z component of the axis vector.
     * @param [angle] Angle of rotation about the axis vector, in degrees.
     * @returns The resulted matrix
     */
    rotateAxisAngle(x?: number, y?: number, z?: number, angle?: number): DOMMatrixReadOnly;
    /**
     * Returns a new DOMMatrix instance which is this matrix post multiplied by each of 3 rotation matrices about the major axes, first X, then Y, then Z.
     *
     * @param [x] X component of the rotation, or Z if Y and Z are null.
     * @param [y] Y component of the rotation value.
     * @param [z] Z component of the rotation value.
     * @returns The resulted matrix
     */
    rotate(x?: number, y?: number, z?: number): DOMMatrixReadOnly;
    /**
     * Returns a new DOMMatrix instance which is this matrix post multiplied by a rotation matrix with the angle between the specified vector and (1, 0).
     *
     * @param [x] X-Axis skew.
     * @param [y] Y-Axis skew.
     */
    rotateFromVector(x?: number, y?: number): DOMMatrixReadOnly;
    /**
     * Returns a new DOMMatrix instance that specifies a skew transformation along X-Axis by the given angle.
     *
     * @param angle Angle amount in degrees to skew.
     * @returns The resulted matrix
     */
    skewX(angle: number): DOMMatrixReadOnly;
    /**
     * Returns a new DOMMatrix instance that specifies a skew transformation along Y-Axis by the given angle.
     *
     * @param angle Angle amount in degrees to skew.
     * @returns The resulted matrix
     */
    skewY(angle: number): DOMMatrixReadOnly;
    /**
     * Returns a new DOMMatrix instance which is this matrix flipped on X-axis.
     */
    flipX(): DOMMatrixReadOnly;
    /**
     * Returns a new DOMMatrix instance which is this matrix flipped on Y-axis.
     */
    flipY(): DOMMatrixReadOnly;
    /**
     * Returns a new DOMMatrix instance which is this matrix inversed.
     */
    inverse(): DOMMatrixReadOnly;
    /**
     * Returns a new DOMPoint instance with the vector transformed using the matrix.
     *
     * @param domPoint DOM point compatible object.
     * @returns A new DOMPoint object.
     */
    transformPoint(domPoint: IDOMPointInit): DOMPoint;
    /**
     * The `setMatrixValue` method replaces the existing matrix with one computed in the browser (e.g.`matrix(1,0.25,-0.25,1,0,0)`).
     *
     * @param source A `DOMMatrix`, `Float32Array`, `Float64Array`, `Array`, or DOMMatrix compatible object to set the matrix values from.
     */
    [PropertySymbol.setMatrixValue](source?: TDOMMatrixInit): void;
    /**
     * Applies a multiply operation to the current matrix.
     *
     * @param matrix Second matrix.
     */
    [PropertySymbol.multiplySelf](matrix: IDOMMatrixCompatibleObject): void;
    /**
     * Applies translate to the matrix.
     *
     * This method is equivalent to the CSS `translate3d()` function.
     *
     * @see https://drafts.csswg.org/css-transforms-1/#TranslateDefined
     * @see https://developer.mozilla.org/en-US/docs/Web/CSS/transform-function/translate3d
     * @see https://www.w3.org/TR/css-transforms-1/#transform-functions
     * @param [x] X-Axis position.
     * @param [y] Y-Axis position.
     * @param [z] Z-Axis position.
     */
    [PropertySymbol.translateSelf](x?: number, y?: number, z?: number): void;
    /**
     * Applies a scale to the matrix.
     *
     * This method is equivalent to the CSS `scale()` function.
     *
     * @see https://drafts.csswg.org/css-transforms-1/#ScaleDefined
     * @see https://developer.mozilla.org/en-US/docs/Web/CSS/transform-function/scale
     * @see https://www.w3.org/TR/css-transforms-1/#transform-functions
     * @param [scaleX] X-Axis scale.
     * @param [scaleY] Y-Axis scale.
     * @param [scaleZ] Z-Axis scale.
     * @param [originX] X-Axis scale.
     * @param [originY] Y-Axis scale.
     * @param [originZ] Z-Axis scale.
     */
    [PropertySymbol.scaleSelf](scaleX?: number, scaleY?: number, scaleZ?: number, originX?: number, originY?: number, originZ?: number): void;
    /**
     * Applies a scale to the matrix.
     *
     * This method is equivalent to the CSS `scale()` function.
     *
     * @see https://developer.mozilla.org/en-US/docs/Web/CSS/transform-function/scale3d
     * @see https://www.w3.org/TR/css-transforms-1/#transform-functions
     * @param [scale] The scale factor.
     * @param [originX] X-Axis scale.
     * @param [originY] Y-Axis scale.
     * @param [originZ] Z-Axis scale.
     */
    [PropertySymbol.scale3dSelf](scale?: number, originX?: number, originY?: number, originZ?: number): void;
    /**
     * Applies a scale to the matrix.
     *
     * @see https://www.w3.org/TR/css-transforms-1/#transform-functions
     * @param [scaleX] X-Axis scale.
     * @param [scaleY] Y-Axis scale.
     */
    [PropertySymbol.scaleNonUniformSelf](scaleX?: number, scaleY?: number): void;
    /**
     * Applies a rotation to the matrix.
     *
     * This method is equivalent to the CSS `rotate3d()` function.
     *
     * @see https://drafts.fxtf.org/geometry/#dom-dommatrixreadonly-rotateaxisangleself
     * @see https://www.w3.org/TR/css-transforms-1/#transform-functions
     * @see https://developer.mozilla.org/en-US/docs/Web/CSS/transform-function/rotate3d
     * @param [x] X-Axis vector.
     * @param [y] Y-Axis vector.
     * @param [z] Z-Axis vector.
     * @param [angle] Angle in degrees of the rotation.
     */
    [PropertySymbol.rotateAxisAngleSelf](x?: number, y?: number, z?: number, angle?: number): void;
    /**
     * Applies a rotation to the matrix.
     *
     * @see http://en.wikipedia.org/wiki/Rotation_matrix
     * @see https://www.w3.org/TR/css-transforms-1/#transform-functions
     * @param [x] X-Axis rotation in degrees.
     * @param [y] Y-Axis rotation in degrees.
     * @param [z] Z-Axis rotation in degrees.
     */
    [PropertySymbol.rotateSelf](x?: number, y?: number, z?: number): void;
    /**
     * Modifies the matrix by rotating it by the angle between the specified vector and (1, 0).
     *
     * @param x The X component of the axis vector.
     * @param y The Y component of the axis vector.
     */
    [PropertySymbol.rotateFromVectorSelf](x?: number, y?: number): void;
    /**
     * Applies a skew operation to the matrix on the X axis.
     *
     * This method is equivalent to the CSS `skewX()` function.
     *
     * @see https://developer.mozilla.org/en-US/docs/Web/CSS/transform-function/skewX
     * @see https://www.w3.org/TR/css-transforms-1/#transform-functions
     * @param angle Angle in degrees.
     */
    [PropertySymbol.skewXSelf](angle: number): void;
    /**
     * Applies a skew operation to the matrix on the Y axis.
     *
     * This method is equivalent to the CSS `skewY()` function.
     *
     * @see https://developer.mozilla.org/en-US/docs/Web/CSS/transform-function/skewY
     * @see https://www.w3.org/TR/css-transforms-1/#transform-functions
     * @param angle Angle in degrees.
     */
    [PropertySymbol.skewYSelf](angle: number): void;
    /**
     * Applies a flip operation to the matrix on the X axis.
     */
    [PropertySymbol.flipXSelf](): void;
    /**
     * Applies a flip operation to the matrix on the Y axis.
     */
    [PropertySymbol.flipYSelf](): void;
    /**
     * Applies an inversion operation to the matrix.
     */
    [PropertySymbol.invertSelf](): void;
    /**
     * Returns an *Array* containing elements which comprise the matrix.
     *
     * @param matrix Matrix to convert.
     * @param [is2D] If the matrix is 2D.
     * @returns Array representation of the matrix.
     */
    [PropertySymbol.toArray](is2D?: boolean): TDOMMatrix2DArray | TDOMMatrix3DArray;
    /**
     * Returns a new `DOMMatrix` instance given an existing matrix.
     *
     * @param matrix Matrix.
     */
    static fromMatrix(matrix: IDOMMatrixCompatibleObject): DOMMatrixReadOnly;
    /**
     * Returns a new `DOMMatrix` instance given an array of 16/6 floating point values.
     *
     * @param array An `Array` to feed values from.
     * @returns DOMMatrix instance.
     */
    static fromFloat32Array(array: Float32Array): DOMMatrixReadOnly;
    /**
     * Returns a new `DOMMatrix` instance given an array of 16/6 floating point values.
     *
     * @param array An `Array` to feed values from.
     * @returns DOMMatrix instance.
     */
    static fromFloat64Array(array: Float64Array): DOMMatrixReadOnly;
    /**
     * Returns a new `DOMMatrix` instance given an array of 16/6 floating point values.
     *
     * Conditions:
     * - If the array has six values, the result is a 2D matrix.
     * - If the array has 16 values, the result is a 3D matrix.
     * - Otherwise, a TypeError exception is thrown.
     *
     * @param array An `Array` to feed values from.
     * @returns DOMMatrix instance.
     */
    static [PropertySymbol.fromArray](array: any[] | Float32Array | Float64Array): DOMMatrixReadOnly;
    /**
     * Returns a new `DOMMatrix` instance from a DOM transform string.
     *
     * @param source valid CSS transform string syntax.
     * @returns DOMMatrix instance.
     */
    static [PropertySymbol.fromString](source: string): DOMMatrixReadOnly;
    /**
     * Returns length.
     *
     * @param length Length to convert.
     * @returns Length.
     */
    private static [PropertySymbol.getLength];
}
//# sourceMappingURL=DOMMatrixReadOnly.d.ts.map