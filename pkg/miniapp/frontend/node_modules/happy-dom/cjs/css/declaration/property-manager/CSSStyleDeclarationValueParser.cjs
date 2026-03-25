"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
const COLOR_REGEXP = /^#([0-9a-fA-F]{3,4}){1,2}$|^rgb\(([^)]*)\)$|^rgba\(([^)]*)\)$|^hsla?\(\s*(-?\d+|-?\d*.\d+)\s*,\s*(-?\d+|-?\d*.\d+)%\s*,\s*(-?\d+|-?\d*.\d+)%\s*(,\s*(-?\d+|-?\d*.\d+)\s*)?\)/;
const LENGTH_REGEXP = /^(0|[-+]?[0-9]*\.?[0-9]+)(in|cm|em|mm|pt|pc|px|ex|rem|vh|vw|ch|vw|vh|vmin|vmax|Q)$/;
const PERCENTAGE_REGEXP = /^[-+]?[0-9]*\.?[0-9]+%$/;
const DEGREE_REGEXP = /^[0-9]+deg$/;
const URL_REGEXP = /^url\(\s*([^)]*)\s*\)$/;
const INTEGER_REGEXP = /^[0-9]+$/;
const FLOAT_REGEXP = /^[0-9.]+$/;
const CALC_REGEXP = /^calc\([^^)]+\)$/;
const CSS_VARIABLE_REGEXP = /^var\(\s*(--[^)\s]+)\)$/;
const FIT_CONTENT_REGEXP = /^fit-content\([^^)]+\)$/;
const GRADIENT_REGEXP = /^((repeating-linear|linear|radial|repeating-radial|conic|repeating-conic)-gradient)\(([^)]+)\)$/;
const GLOBALS = ['inherit', 'initial', 'unset', 'revert'];
const COLORS = [
    'none',
    'currentcolor',
    'transparent',
    'silver',
    'gray',
    'white',
    'maroon',
    'red',
    'purple',
    'fuchsia',
    'green',
    'lime',
    'olive',
    'yellow',
    'navy',
    'blue',
    'teal',
    'aliceblue',
    'aqua',
    'antiquewhite',
    'aquamarine',
    'azure',
    'beige',
    'bisque',
    'black',
    'blanchedalmond',
    'blueviolet',
    'brown',
    'burlywood',
    'cadetblue',
    'chartreuse',
    'chocolate',
    'coral',
    'cornflowerblue',
    'cornsilk',
    'crimson',
    'cyan',
    'darkblue',
    'darkcyan',
    'darkgoldenrod',
    'darkgray',
    'darkgreen',
    'darkgrey',
    'darkkhaki',
    'darkmagenta',
    'darkolivegreen',
    'darkorange',
    'darkorchid',
    'darkred',
    'darksalmon',
    'darkseagreen',
    'darkslateblue',
    'darkslategray',
    'darkslategrey',
    'darkturquoise',
    'darkviolet',
    'deeppink',
    'deepskyblue',
    'dimgray',
    'dimgrey',
    'dodgerblue',
    'firebrick',
    'floralwhite',
    'forestgreen',
    'gainsboro',
    'ghostwhite',
    'gold',
    'goldenrod',
    'greenyellow',
    'grey',
    'honeydew',
    'hotpink',
    'indianred',
    'indigo',
    'ivory',
    'khaki',
    'lavender',
    'lavenderblush',
    'lawngreen',
    'lemonchiffon',
    'lightblue',
    'lightcoral',
    'lightcyan',
    'lightgoldenrodyellow',
    'lightgray',
    'lightgreen',
    'lightgrey',
    'lightpink',
    'lightsalmon',
    'lightseagreen',
    'lightskyblue',
    'lightslategray',
    'lightslategrey',
    'lightsteelblue',
    'lightyellow',
    'limegreen',
    'linen',
    'magenta',
    'mediumaquamarine',
    'mediumblue',
    'mediumorchid',
    'mediumpurple',
    'mediumseagreen',
    'mediumslateblue',
    'mediumspringgreen',
    'mediumturquoise',
    'mediumvioletred',
    'midnightblue',
    'mintcream',
    'mistyrose',
    'moccasin',
    'navajowhite',
    'oldlace',
    'olivedrab',
    'orange',
    'orangered',
    'orchid',
    'palegoldenrod',
    'palegreen',
    'paleturquoise',
    'palevioletred',
    'papayawhip',
    'peachpuff',
    'peru',
    'pink',
    'plum',
    'powderblue',
    'rebeccapurple',
    'rosybrown',
    'royalblue',
    'saddlebrown',
    'salmon',
    'sandybrown',
    'seagreen',
    'seashell',
    'sienna',
    'skyblue',
    'slateblue',
    'slategray',
    'slategrey',
    'snow',
    'springgreen',
    'steelblue',
    'tan',
    'thistle',
    'tomato',
    'turquoise',
    'violet',
    'wheat',
    'whitesmoke',
    'yellowgreen'
];
/**
 * Style declaration value parser.
 */
class CSSStyleDeclarationValueParser {
    /**
     * Returns length.
     *
     * @param value Value.
     * @returns Parsed value.
     */
    static getLength(value) {
        if (value === '0') {
            return '0px';
        }
        const match = value.match(LENGTH_REGEXP);
        if (match) {
            const number = parseFloat(match[1]);
            if (isNaN(number)) {
                return null;
            }
            const unit = match[2];
            return `${Math.round(number * 1000000) / 1000000}${unit}`;
        }
        return null;
    }
    /**
     * Returns percentance.
     *
     * @param value Value.
     * @returns Parsed value.
     */
    static getPercentage(value) {
        if (value === '0') {
            return '0%';
        }
        if (PERCENTAGE_REGEXP.test(value)) {
            return value;
        }
        return null;
    }
    /**
     * Returns degree.
     *
     * @param value Value.
     * @returns Parsed value.
     */
    static getDegree(value) {
        if (value === '0') {
            return '0deg';
        }
        if (DEGREE_REGEXP.test(value)) {
            return value;
        }
        return null;
    }
    /**
     * Returns calc.
     *
     * @param value Value.
     * @returns Parsed value.
     */
    static getCalc(value) {
        if (CALC_REGEXP.test(value)) {
            return value;
        }
        return null;
    }
    /**
     * Returns fit content.
     *
     * @param value Value.
     * @returns Parsed value.
     */
    static getFitContent(value) {
        const lowerValue = value.toLowerCase();
        if (lowerValue === 'auto' ||
            lowerValue === 'max-content' ||
            lowerValue === 'min-content' ||
            lowerValue === 'fit-content') {
            return lowerValue;
        }
        if (FIT_CONTENT_REGEXP.test(lowerValue)) {
            return lowerValue;
        }
        return null;
    }
    /**
     * Returns measurement.
     *
     * @param value Value.
     * @returns Parsed value.
     */
    static getMeasurement(value) {
        return this.getLength(value) || this.getPercentage(value) || this.getCalc(value);
    }
    /**
     * Returns measurement or auto, min-content, max-content or fit-content.
     *
     * @param value Value.
     * @returns Parsed value.
     */
    static getContentMeasurement(value) {
        return this.getFitContent(value) || this.getMeasurement(value);
    }
    /**
     * Returns measurement or auto, min-content, max-content or fit-content.
     *
     * @param value Value.
     * @returns Parsed value.
     */
    static getAutoMeasurement(value) {
        if (value.toLocaleLowerCase() === 'auto') {
            return 'auto';
        }
        return this.getMeasurement(value);
    }
    /**
     * Returns integer.
     *
     * @param value Value.
     * @returns Parsed value.
     */
    static getInteger(value) {
        if (INTEGER_REGEXP.test(value)) {
            return value;
        }
        return null;
    }
    /**
     * Returns float.
     *
     * @param value Value.
     * @returns Parsed value.
     */
    static getFloat(value) {
        if (FLOAT_REGEXP.test(value)) {
            const number = parseFloat(value);
            if (isNaN(number)) {
                return null;
            }
            return String(Math.round(number * 1000000) / 1000000);
        }
        return null;
    }
    /**
     * Returns gradient.
     *
     * @param value Value.
     * @returns Parsed value.
     */
    static getGradient(value) {
        const match = value.match(GRADIENT_REGEXP);
        if (match) {
            return `${match[1]}(${match[3]
                .trim()
                .split(/\s*,\s*/)
                .join(', ')})`;
        }
        return null;
    }
    /**
     * Returns color.
     *
     * @param value Value.
     * @returns Parsed value.
     */
    static getColor(value) {
        const lowerValue = value.toLowerCase();
        if (COLORS.includes(lowerValue)) {
            return lowerValue;
        }
        if (COLOR_REGEXP.test(value)) {
            return value.replace(/,([^ ])/g, ', $1');
        }
        return null;
    }
    /**
     * Returns URL.
     *
     * Based on:
     * https://github.com/jsdom/cssstyle/blob/master/lib/parsers.js#L222
     *
     * @param value Value.
     * @returns Parsed value.
     */
    static getURL(value) {
        if (!value) {
            return null;
        }
        if (value.toLowerCase() === 'none') {
            return 'none';
        }
        const result = URL_REGEXP.exec(value);
        if (!result) {
            return null;
        }
        let url = result[1].trim();
        if ((url[0] === '"' || url[0] === "'") && url[0] !== url[url.length - 1]) {
            return null;
        }
        if (url[0] === '"' || url[0] === "'") {
            url = url.substring(1, url.length - 1);
        }
        for (let i = 0; i < url.length; i++) {
            switch (url[i]) {
                case '(':
                case ')':
                case ' ':
                case '\t':
                case '\n':
                case "'":
                case '"':
                    return null;
                case '\\':
                    i++;
                    break;
            }
        }
        return `url("${url}")`;
    }
    /**
     * Returns global initial value.
     *
     * @param value Value.
     * @returns Parsed value.
     */
    static getInitial(value) {
        return value.toLowerCase() === 'initial' ? 'initial' : null;
    }
    /**
     * Returns CSS variable.
     *
     * @param value Value.
     * @returns Parsed value.
     */
    static getVariable(value) {
        const cssVariableMatch = value.match(CSS_VARIABLE_REGEXP);
        if (cssVariableMatch) {
            return `var(${cssVariableMatch[1]})`;
        }
        return null;
    }
    /**
     * Returns global.
     *
     * @param value Value.
     * @returns Parsed value.
     */
    static getGlobal(value) {
        const lowerValue = value.toLowerCase();
        return GLOBALS.includes(lowerValue) ? lowerValue : null;
    }
    /**
     * Returns global, unless it is not set to 'initial' as it is sometimes treated different.
     *
     * @param value Value.
     * @returns Parsed value.
     */
    static getGlobalExceptInitial(value) {
        const lowerValue = value.toLowerCase();
        return lowerValue !== 'initial' && GLOBALS.includes(lowerValue) ? lowerValue : null;
    }
}
exports.default = CSSStyleDeclarationValueParser;
//# sourceMappingURL=CSSStyleDeclarationValueParser.cjs.map