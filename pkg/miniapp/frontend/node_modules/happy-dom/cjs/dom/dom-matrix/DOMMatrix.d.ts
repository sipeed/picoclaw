import DOMMatrixReadOnly from './DOMMatrixReadOnly.cjs';
import IDOMMatrixCompatibleObject from './IDOMMatrixCompatibleObject.cjs';
import TDOMMatrixInit from './TDOMMatrixInit.cjs';
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
    get a(): number;
    /**
     * Sets the `a` value of the matrix.
     */
    set a(value: number);
    /**
     * Returns the `b` value of the matrix.
     */
    get b(): number;
    /**
     * Sets the `b` value of the matrix.
     */
    set b(value: number);
    /**
     * Returns the `c` value of the matrix.
     */
    get c(): number;
    /**
     * Sets the `c` value of the matrix.
     */
    set c(value: number);
    /**
     * Returns the `d` value of the matrix.
     */
    get d(): number;
    /**
     * Sets the `d` value of the matrix.
     */
    set d(value: number);
    /**
     * Returns the `e` value of the matrix.
     */
    get e(): number;
    /**
     * Sets the `e` value of the matrix.
     */
    set e(value: number);
    /**
     * Returns the `f` value of the matrix.
     */
    get f(): number;
    /**
     * Sets the `f` value of the matrix.
     */
    set f(value: number);
    /**
     * Returns the `m11` value of the matrix.
     */
    get m11(): number;
    /**
     * Sets the `m11` value of the matrix.
     */
    set m11(value: number);
    /**
     * Returns the `m12` value of the matrix.
     */
    get m12(): number;
    /**
     * Sets the `m12` value of the matrix.
     */
    set m12(value: number);
    /**
     * Returns the `m13` value of the matrix.
     */
    get m13(): number;
    /**
     * Sets the `m13` value of the matrix.
     */
    set m13(value: number);
    /**
     * Returns the `m14` value of the matrix.
     */
    get m14(): number;
    /**
     * Sets the `m14` value of the matrix.
     */
    set m14(value: number);
    /**
     * Returns the `m21` value of the matrix.
     */
    get m21(): number;
    /**
     * Sets the `m21` value of the matrix.
     */
    set m21(value: number);
    /**
     * Returns the `m22` value of the matrix.
     */
    get m22(): number;
    /**
     * Sets the `m22` value of the matrix.
     */
    set m22(value: number);
    /**
     * Returns the `m23` value of the matrix.
     */
    get m23(): number;
    /**
     * Sets the `m23` value of the matrix.
     */
    set m23(value: number);
    /**
     * Returns the `m24` value of the matrix.
     */
    get m24(): number;
    /**
     * Sets the `m24` value of the matrix.
     */
    set m24(value: number);
    /**
     * Returns the `m31` value of the matrix.
     */
    get m31(): number;
    /**
     * Sets the `m31` value of the matrix.
     */
    set m31(value: number);
    /**
     * Returns the `m32` value of the matrix.
     */
    get m32(): number;
    /**
     * Sets the `m32` value of the matrix.
     */
    set m32(value: number);
    /**
     * Returns the `m33` value of the matrix.
     */
    get m33(): number;
    /**
     * Sets the `m33` value of the matrix.
     */
    set m33(value: number);
    /**
     * Returns the `m34` value of the matrix.
     */
    get m34(): number;
    /**
     * Sets the `m34` value of the matrix.
     */
    set m34(value: number);
    /**
     * Returns the `m41` value of the matrix.
     */
    get m41(): number;
    /**
     * Sets the `m41` value of the matrix.
     */
    set m41(value: number);
    /**
     * Returns the `m42` value of the matrix.
     */
    get m42(): number;
    /**
     * Sets the `m42` value of the matrix.
     */
    set m42(value: number);
    /**
     * Returns the `m43` value of the matrix.
     */
    get m43(): number;
    /**
     * Sets the `m43` value of the matrix.
     */
    set m43(value: number);
    /**
     * Returns the `m44` value of the matrix.
     */
    get m44(): number;
    /**
     * Sets the `m44` value of the matrix.
     */
    set m44(value: number);
    /**
     * The `setMatrixValue` method replaces the existing matrix with one computed in the browser (e.g.`matrix(1,0.25,-0.25,1,0,0)`).
     *
     * @param source A `DOMMatrix`, `Float32Array`, `Float64Array`, `Array`, or DOMMatrix compatible object to set the matrix values from.
     * @returns Self.
     */
    setMatrixValue(source?: TDOMMatrixInit): DOMMatrix;
    /**
     * Sets self to be multiplied by the passed matrix.
     *
     * @param secondMatrix DOMMatrix
     * @returns Self.
     */
    multiplySelf(secondMatrix: IDOMMatrixCompatibleObject): DOMMatrix;
    /**
     * Sets self to multiplied by a translation matrix containing the passed values.
     *
     * @param [x=0] X component of the translation value.
     * @param [y=0] Y component of the translation value.
     * @param [z=0] Z component of the translation value.
     * @returns Self.
     */
    translateSelf(x?: number, y?: number, z?: number): DOMMatrix;
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
    scaleSelf(scaleX?: number, scaleY?: number, scaleZ?: number, originX?: number, originY?: number, originZ?: number): DOMMatrix;
    /**
     * Sets self to be multiplied by a scale matrix containing the passed values.
     *
     * @param [scale] The scale factor.
     * @param [originX] X-Axis scale.
     * @param [originY] Y-Axis scale.
     * @param [originZ] Z-Axis scale.
     * @returns Self.
     */
    scale3dSelf(scale?: number, originX?: number, originY?: number, originZ?: number): DOMMatrix;
    /**
     * Sets self to be multiplied by a scale matrix containing the passed values.
     *
     * @param [scaleX] X-Axis scale.
     * @param [scaleY] Y-Axis scale.
     * @returns Self.
     */
    scaleNonUniformSelf(scaleX?: number, scaleY?: number): DOMMatrix;
    /**
     * Sets self to be multiplied by a rotation matrix with the given axis and `angle`.
     *
     * @param [x] The X component of the axis vector.
     * @param [y] The Y component of the axis vector.
     * @param [z] The Z component of the axis vector.
     * @param [angle] Angle of rotation about the axis vector, in degrees.
     * @returns Self.
     */
    rotateAxisAngleSelf(x?: number, y?: number, z?: number, angle?: number): DOMMatrixReadOnly;
    /**
     * Set self to be multiplied by each of 3 rotation matrices about the major axes, first X, then Y, then Z.
     *
     * @param [x] X component of the rotation, or Z if Y and Z are null.
     * @param [y] Y component of the rotation value.
     * @param [z] Z component of the rotation value.
     * @returns Self.
     */
    rotateSelf(x?: number, y?: number, z?: number): DOMMatrixReadOnly;
    /**
     * Sets self to be multiplied by a skew matrix along the X axis by the given angle.
     *
     * @param [x] X-Axis skew.
     * @param [y] Y-Axis skew.
     */
    rotateFromVectorSelf(x?: number, y?: number): DOMMatrixReadOnly;
    /**
     * Set self to be specified as a skew transformation along X-Axis by the given angle.
     *
     * @param angle Angle amount in degrees to skew.
     * @returns Self.
     */
    skewXSelf(angle: number): DOMMatrixReadOnly;
    /**
     * Set self to be specified as a skew transformation along Y-Axis by the given angle.
     *
     * @param angle Angle amount in degrees to skew.
     * @returns Self.
     */
    skewYSelf(angle: number): DOMMatrixReadOnly;
    /**
     * Set self to be specified as matrix flipped on X-axis.
     */
    flipXSelf(): DOMMatrixReadOnly;
    /**
     * Set self to be specified as matrix flipped on Y-axis.
     */
    flipYSelf(): DOMMatrixReadOnly;
    /**
     * Set self to be specified as matrix inverted.
     */
    invertSelf(): DOMMatrixReadOnly;
}
//# sourceMappingURL=DOMMatrix.d.ts.map