import * as PropertySymbol from '../PropertySymbol.cjs';
import BrowserWindow from '../window/BrowserWindow.cjs';
import DOMMatrix from '../dom/dom-matrix/DOMMatrix.cjs';
/**
 * SVG Matrix.
 *
 * Documentation missing at developer.mozilla.org.
 */
export default class SVGMatrix {
    [PropertySymbol.window]: BrowserWindow;
    [PropertySymbol.getAttribute]: (() => string | null) | null;
    [PropertySymbol.setAttribute]: ((value: string) => void) | null;
    [PropertySymbol.attributeValue]: string | null;
    [PropertySymbol.readOnly]: boolean;
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
    constructor(illegalConstructorSymbol: symbol, window: BrowserWindow, options?: {
        readOnly?: boolean;
        getAttribute?: () => string | null;
        setAttribute?: (value: string) => void;
    });
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
     * Returns a new SVGMatrix instance which is the result of this matrix multiplied by the passed matrix.
     *
     * @param secondMatrix Matrix object.
     * @returns A new SVGMatrix object.
     */
    multiply(secondMatrix: SVGMatrix): SVGMatrix;
    /**
     * Returns a new SVGMatrix instance which is this matrix post multiplied by a translation matrix containing the passed values.
     *
     * @param [x=0] X component of the translation value.
     * @param [y=0] Y component of the translation value.
     * @returns The resulted matrix
     */
    translate(x?: number, y?: number): SVGMatrix;
    /**
     * Returns a new SVGMatrix instance which is this matrix post multiplied by a scale 2D matrix containing the passed values.
     *
     * @param scale The scale factor.
     * @returns The resulted matrix
     */
    scale(scale: number): SVGMatrix;
    /**
     * Returns a new SVGMatrix instance which is this matrix post multiplied by a scale 3D matrix containing the passed values.
     *
     * @param [scaleX] X-Axis scale.
     * @param [scaleY] Y-Axis scale.
     * @returns The resulted matrix
     */
    scaleNonUniform(scaleX?: number, scaleY?: number): SVGMatrix;
    /**
     * Returns a new SVGMatrix instance which is this matrix post multiplied by each of 3 rotation matrices about the major axes, first X, then Y, then Z.
     *
     * @param angle Angle of rotation in degrees.
     * @returns The resulted matrix
     */
    rotate(angle: number): SVGMatrix;
    /**
     * Returns a new SVGMatrix instance which is this matrix post multiplied by a skew matrix along the X axis by the given angle.
     *
     * Not implemented in Happy DOM yet.
     *
     * @param [x] X-Axis skew.
     * @param [y] Y-Axis skew.
     */
    rotateFromVector(x?: number, y?: number): SVGMatrix;
    /**
     * Returns a new SVGMatrix instance that specifies a skew transformation along X-Axis by the given angle.
     *
     * @param angle Angle amount in degrees to skew.
     * @returns The resulted matrix
     */
    skewX(angle: number): SVGMatrix;
    /**
     * Returns a new SVGMatrix instance that specifies a skew transformation along Y-Axis by the given angle.
     *
     * @param angle Angle amount in degrees to skew.
     * @returns The resulted matrix
     */
    skewY(angle: number): SVGMatrix;
    /**
     * Returns a new SVGMatrix instance which is this matrix flipped on X-axis.
     */
    flipX(): SVGMatrix;
    /**
     * Returns a new SVGMatrix instance which is this matrix flipped on Y-axis.
     */
    flipY(): SVGMatrix;
    /**
     * Returns a new SVGMatrix instance which is this matrix inverted.
     */
    inverse(): SVGMatrix;
    /**
     * Returns DOM matrix.
     *
     * @returns DOM matrix.
     */
    [PropertySymbol.getDOMMatrix](): DOMMatrix;
    /**
     * Sets DOM matrix.
     *
     * @param domMatrix DOM matrix.
     */
    [PropertySymbol.setDOMMatrix](domMatrix: DOMMatrix): void;
}
//# sourceMappingURL=SVGMatrix.d.ts.map