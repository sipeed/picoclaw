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
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
const DOMMatrixReadOnly_js_1 = __importDefault(require("./DOMMatrixReadOnly.cjs"));
/**
 * DOM Matrix.
 *
 * Based on:
 * - https://github.com/thednp/dommatrix/tree/master
 * - https://github.com/trusktr/geometry-interfaces
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/DOMMatrix
 */
class DOMMatrix extends DOMMatrixReadOnly_js_1.default {
    /**
     * Returns the `a` value of the matrix.
     */
    get a() {
        return this[PropertySymbol.m11];
    }
    /**
     * Sets the `a` value of the matrix.
     */
    set a(value) {
        this[PropertySymbol.m11] = value;
    }
    /**
     * Returns the `b` value of the matrix.
     */
    get b() {
        return this[PropertySymbol.m12];
    }
    /**
     * Sets the `b` value of the matrix.
     */
    set b(value) {
        this[PropertySymbol.m12] = value;
    }
    /**
     * Returns the `c` value of the matrix.
     */
    get c() {
        return this[PropertySymbol.m21];
    }
    /**
     * Sets the `c` value of the matrix.
     */
    set c(value) {
        this[PropertySymbol.m21] = value;
    }
    /**
     * Returns the `d` value of the matrix.
     */
    get d() {
        return this[PropertySymbol.m22];
    }
    /**
     * Sets the `d` value of the matrix.
     */
    set d(value) {
        this[PropertySymbol.m22] = value;
    }
    /**
     * Returns the `e` value of the matrix.
     */
    get e() {
        return this[PropertySymbol.m41];
    }
    /**
     * Sets the `e` value of the matrix.
     */
    set e(value) {
        this[PropertySymbol.m41] = value;
    }
    /**
     * Returns the `f` value of the matrix.
     */
    get f() {
        return this[PropertySymbol.m42];
    }
    /**
     * Sets the `f` value of the matrix.
     */
    set f(value) {
        this[PropertySymbol.m42] = value;
    }
    /**
     * Returns the `m11` value of the matrix.
     */
    get m11() {
        return this[PropertySymbol.m11];
    }
    /**
     * Sets the `m11` value of the matrix.
     */
    set m11(value) {
        this[PropertySymbol.m11] = value;
    }
    /**
     * Returns the `m12` value of the matrix.
     */
    get m12() {
        return this[PropertySymbol.m12];
    }
    /**
     * Sets the `m12` value of the matrix.
     */
    set m12(value) {
        this[PropertySymbol.m12] = value;
    }
    /**
     * Returns the `m13` value of the matrix.
     */
    get m13() {
        return this[PropertySymbol.m13];
    }
    /**
     * Sets the `m13` value of the matrix.
     */
    set m13(value) {
        this[PropertySymbol.m13] = value;
    }
    /**
     * Returns the `m14` value of the matrix.
     */
    get m14() {
        return this[PropertySymbol.m14];
    }
    /**
     * Sets the `m14` value of the matrix.
     */
    set m14(value) {
        this[PropertySymbol.m14] = value;
    }
    /**
     * Returns the `m21` value of the matrix.
     */
    get m21() {
        return this[PropertySymbol.m21];
    }
    /**
     * Sets the `m21` value of the matrix.
     */
    set m21(value) {
        this[PropertySymbol.m21] = value;
    }
    /**
     * Returns the `m22` value of the matrix.
     */
    get m22() {
        return this[PropertySymbol.m22];
    }
    /**
     * Sets the `m22` value of the matrix.
     */
    set m22(value) {
        this[PropertySymbol.m22] = value;
    }
    /**
     * Returns the `m23` value of the matrix.
     */
    get m23() {
        return this[PropertySymbol.m23];
    }
    /**
     * Sets the `m23` value of the matrix.
     */
    set m23(value) {
        this[PropertySymbol.m23] = value;
    }
    /**
     * Returns the `m24` value of the matrix.
     */
    get m24() {
        return this[PropertySymbol.m24];
    }
    /**
     * Sets the `m24` value of the matrix.
     */
    set m24(value) {
        this[PropertySymbol.m24] = value;
    }
    /**
     * Returns the `m31` value of the matrix.
     */
    get m31() {
        return this[PropertySymbol.m31];
    }
    /**
     * Sets the `m31` value of the matrix.
     */
    set m31(value) {
        this[PropertySymbol.m31] = value;
    }
    /**
     * Returns the `m32` value of the matrix.
     */
    get m32() {
        return this[PropertySymbol.m32];
    }
    /**
     * Sets the `m32` value of the matrix.
     */
    set m32(value) {
        this[PropertySymbol.m32] = value;
    }
    /**
     * Returns the `m33` value of the matrix.
     */
    get m33() {
        return this[PropertySymbol.m33];
    }
    /**
     * Sets the `m33` value of the matrix.
     */
    set m33(value) {
        this[PropertySymbol.m33] = value;
    }
    /**
     * Returns the `m34` value of the matrix.
     */
    get m34() {
        return this[PropertySymbol.m34];
    }
    /**
     * Sets the `m34` value of the matrix.
     */
    set m34(value) {
        this[PropertySymbol.m34] = value;
    }
    /**
     * Returns the `m41` value of the matrix.
     */
    get m41() {
        return this[PropertySymbol.m41];
    }
    /**
     * Sets the `m41` value of the matrix.
     */
    set m41(value) {
        this[PropertySymbol.m41] = value;
    }
    /**
     * Returns the `m42` value of the matrix.
     */
    get m42() {
        return this[PropertySymbol.m42];
    }
    /**
     * Sets the `m42` value of the matrix.
     */
    set m42(value) {
        this[PropertySymbol.m42] = value;
    }
    /**
     * Returns the `m43` value of the matrix.
     */
    get m43() {
        return this[PropertySymbol.m43];
    }
    /**
     * Sets the `m43` value of the matrix.
     */
    set m43(value) {
        this[PropertySymbol.m43] = value;
    }
    /**
     * Returns the `m44` value of the matrix.
     */
    get m44() {
        return this[PropertySymbol.m44];
    }
    /**
     * Sets the `m44` value of the matrix.
     */
    set m44(value) {
        this[PropertySymbol.m44] = value;
    }
    /**
     * The `setMatrixValue` method replaces the existing matrix with one computed in the browser (e.g.`matrix(1,0.25,-0.25,1,0,0)`).
     *
     * @param source A `DOMMatrix`, `Float32Array`, `Float64Array`, `Array`, or DOMMatrix compatible object to set the matrix values from.
     * @returns Self.
     */
    setMatrixValue(source) {
        this[PropertySymbol.setMatrixValue](source);
        return this;
    }
    /**
     * Sets self to be multiplied by the passed matrix.
     *
     * @param secondMatrix DOMMatrix
     * @returns Self.
     */
    multiplySelf(secondMatrix) {
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
    translateSelf(x = 0, y = 0, z = 0) {
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
    scaleSelf(scaleX, scaleY, scaleZ = 1, originX = 0, originY = 0, originZ = 0) {
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
    scale3dSelf(scale = 1, originX = 0, originY = 0, originZ = 0) {
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
    scaleNonUniformSelf(scaleX = 1, scaleY = 1) {
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
    rotateAxisAngleSelf(x = 0, y = 0, z = 0, angle = 0) {
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
    rotateSelf(x = 0, y, z) {
        this[PropertySymbol.rotateSelf](x, y, z);
        return this;
    }
    /**
     * Sets self to be multiplied by a skew matrix along the X axis by the given angle.
     *
     * @param [x] X-Axis skew.
     * @param [y] Y-Axis skew.
     */
    rotateFromVectorSelf(x = 0, y = 0) {
        this[PropertySymbol.rotateFromVectorSelf](x, y);
        return this;
    }
    /**
     * Set self to be specified as a skew transformation along X-Axis by the given angle.
     *
     * @param angle Angle amount in degrees to skew.
     * @returns Self.
     */
    skewXSelf(angle) {
        this[PropertySymbol.skewXSelf](angle);
        return this;
    }
    /**
     * Set self to be specified as a skew transformation along Y-Axis by the given angle.
     *
     * @param angle Angle amount in degrees to skew.
     * @returns Self.
     */
    skewYSelf(angle) {
        this[PropertySymbol.skewYSelf](angle);
        return this;
    }
    /**
     * Set self to be specified as matrix flipped on X-axis.
     */
    flipXSelf() {
        this[PropertySymbol.flipXSelf]();
        return this;
    }
    /**
     * Set self to be specified as matrix flipped on Y-axis.
     */
    flipYSelf() {
        this[PropertySymbol.flipYSelf]();
        return this;
    }
    /**
     * Set self to be specified as matrix inverted.
     */
    invertSelf() {
        this[PropertySymbol.invertSelf]();
        return this;
    }
}
exports.default = DOMMatrix;
//# sourceMappingURL=DOMMatrix.cjs.map