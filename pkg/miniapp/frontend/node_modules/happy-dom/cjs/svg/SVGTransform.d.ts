import * as PropertySymbol from '../PropertySymbol.cjs';
import BrowserWindow from '../window/BrowserWindow.cjs';
import SVGMatrix from './SVGMatrix.cjs';
import SVGTransformTypeEnum from './SVGTransformTypeEnum.cjs';
/**
 * SVG transform.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGTransform
 */
export default class SVGTransform {
    static SVG_TRANSFORM_UNKNOWN: SVGTransformTypeEnum;
    static SVG_TRANSFORM_MATRIX: SVGTransformTypeEnum;
    static SVG_TRANSFORM_TRANSLATE: SVGTransformTypeEnum;
    static SVG_TRANSFORM_SCALE: SVGTransformTypeEnum;
    static SVG_TRANSFORM_ROTATE: SVGTransformTypeEnum;
    static SVG_TRANSFORM_SKEWX: SVGTransformTypeEnum;
    static SVG_TRANSFORM_SKEWY: SVGTransformTypeEnum;
    [PropertySymbol.window]: BrowserWindow;
    [PropertySymbol.getAttribute]: () => string;
    [PropertySymbol.setAttribute]: (value: string) => void;
    [PropertySymbol.attributeValue]: string | null;
    [PropertySymbol.readOnly]: boolean;
    [PropertySymbol.matrix]: SVGMatrix | null;
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
     * Returns type.
     *
     * @returns Type.
     */
    get type(): number;
    /**
     * Returns angle.
     *
     * @returns Angle.
     */
    get angle(): number;
    /**
     * Returns matrix.
     *
     * @returns Matrix.
     */
    get matrix(): SVGMatrix;
    /**
     * Set matrix.
     *
     * @param matrix Matrix.
     */
    setMatrix(matrix: SVGMatrix): void;
    /**
     * Set translate.
     *
     * @param x X.
     * @param y Y.
     */
    setTranslate(x: number, y: number): void;
    /**
     * Set scale.
     *
     * @param x X.
     * @param y Y.
     */
    setScale(x: number, y: number): void;
    /**
     * Set rotate.
     *
     * @param angle Angle.
     * @param x X.
     * @param y Y.
     */
    setRotate(angle: number, x: number, y: number): void;
    /**
     * Set skew x.
     *
     * @param angle Angle.
     */
    setSkewX(angle: number): void;
    /**
     * Set skew y.
     *
     * @param angle Angle.
     */
    setSkewY(angle: number): void;
}
//# sourceMappingURL=SVGTransform.d.ts.map