import SVGElement from '../svg-element/SVGElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedLength from '../../svg/SVGAnimatedLength.js';
import SVGAnimatedString from '../../svg/SVGAnimatedString.js';
import SVGAnimatedNumber from '../../svg/SVGAnimatedNumber.js';
import SVGAnimatedInteger from '../../svg/SVGAnimatedInteger.js';
import SVGAnimatedEnumeration from '../../svg/SVGAnimatedEnumeration.js';
/**
 * SVGFETurbulenceElement.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGFETurbulenceElement
 */
export default class SVGFETurbulenceElement extends SVGElement {
    // Static properties
    static SVG_TURBULENCE_TYPE_UNKNOWN = 0;
    static SVG_TURBULENCE_TYPE_FRACTALNOISE = 1;
    static SVG_TURBULENCE_TYPE_TURBULENCE = 2;
    static SVG_STITCHTYPE_UNKNOWN = 0;
    static SVG_STITCHTYPE_STITCH = 1;
    static SVG_STITCHTYPE_NOSTITCH = 2;
    // Internal properties
    [PropertySymbol.baseFrequencyX] = null;
    [PropertySymbol.baseFrequencyY] = null;
    [PropertySymbol.height] = null;
    [PropertySymbol.numOctaves] = null;
    [PropertySymbol.result] = null;
    [PropertySymbol.seed] = null;
    [PropertySymbol.stitchTiles] = null;
    [PropertySymbol.type] = null;
    [PropertySymbol.width] = null;
    [PropertySymbol.x] = null;
    [PropertySymbol.y] = null;
    /**
     * Returns baseFrequencyX.
     *
     * @returns Base frequency x.
     */
    get baseFrequencyX() {
        if (!this[PropertySymbol.baseFrequencyX]) {
            this[PropertySymbol.baseFrequencyX] = new SVGAnimatedNumber(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('baseFrequencyX'),
                setAttribute: (value) => this.setAttribute('baseFrequencyX', value)
            });
        }
        return this[PropertySymbol.baseFrequencyX];
    }
    /**
     * Returns baseFrequencyY.
     *
     * @returns Base frequency y.
     */
    get baseFrequencyY() {
        if (!this[PropertySymbol.baseFrequencyY]) {
            this[PropertySymbol.baseFrequencyY] = new SVGAnimatedNumber(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('baseFrequencyY'),
                setAttribute: (value) => this.setAttribute('baseFrequencyY', value)
            });
        }
        return this[PropertySymbol.baseFrequencyY];
    }
    /**
     * Returns height.
     *
     * @returns Height.
     */
    get height() {
        if (!this[PropertySymbol.height]) {
            this[PropertySymbol.height] = new SVGAnimatedLength(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('height'),
                setAttribute: (value) => this.setAttribute('height', value)
            });
        }
        return this[PropertySymbol.height];
    }
    /**
     * Returns numOctaves.
     *
     * @returns Num octaves.
     */
    get numOctaves() {
        if (!this[PropertySymbol.numOctaves]) {
            this[PropertySymbol.numOctaves] = new SVGAnimatedInteger(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('numOctaves'),
                setAttribute: (value) => this.setAttribute('numOctaves', value)
            });
        }
        return this[PropertySymbol.numOctaves];
    }
    /**
     * Returns result.
     *
     * @returns Result.
     */
    get result() {
        if (!this[PropertySymbol.result]) {
            this[PropertySymbol.result] = new SVGAnimatedString(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('result'),
                setAttribute: (value) => this.setAttribute('result', value)
            });
        }
        return this[PropertySymbol.result];
    }
    /**
     * Returns seed.
     *
     * @returns Seed.
     */
    get seed() {
        if (!this[PropertySymbol.seed]) {
            this[PropertySymbol.seed] = new SVGAnimatedNumber(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('seed'),
                setAttribute: (value) => this.setAttribute('seed', value)
            });
        }
        return this[PropertySymbol.seed];
    }
    /**
     * Returns stitchTiles.
     *
     * @returns Stitch tiles.
     */
    get stitchTiles() {
        if (!this[PropertySymbol.stitchTiles]) {
            this[PropertySymbol.stitchTiles] = new SVGAnimatedEnumeration(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('stitchTiles'),
                setAttribute: (value) => this.setAttribute('stitchTiles', value),
                values: ['stitch', 'noStitch'],
                defaultValue: 'stitch'
            });
        }
        return this[PropertySymbol.stitchTiles];
    }
    /**
     * Returns type.
     *
     * @returns Type.
     */
    get type() {
        if (!this[PropertySymbol.type]) {
            this[PropertySymbol.type] = new SVGAnimatedEnumeration(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('type'),
                setAttribute: (value) => this.setAttribute('type', value),
                values: ['fractalNoise', 'turbulence'],
                defaultValue: 'turbulence'
            });
        }
        return this[PropertySymbol.type];
    }
    /**
     * Returns width.
     *
     * @returns Width.
     */
    get width() {
        if (!this[PropertySymbol.width]) {
            this[PropertySymbol.width] = new SVGAnimatedLength(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('width'),
                setAttribute: (value) => this.setAttribute('width', value)
            });
        }
        return this[PropertySymbol.width];
    }
    /**
     * Returns x.
     *
     * @returns X.
     */
    get x() {
        if (!this[PropertySymbol.x]) {
            this[PropertySymbol.x] = new SVGAnimatedLength(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('x'),
                setAttribute: (value) => this.setAttribute('x', value)
            });
        }
        return this[PropertySymbol.x];
    }
    /**
     * Returns y.
     *
     * @returns Y.
     */
    get y() {
        if (!this[PropertySymbol.y]) {
            this[PropertySymbol.y] = new SVGAnimatedLength(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('y'),
                setAttribute: (value) => this.setAttribute('y', value)
            });
        }
        return this[PropertySymbol.y];
    }
}
//# sourceMappingURL=SVGFETurbulenceElement.js.map