import CSSStyleDeclarationValueParser from './CSSStyleDeclarationValueParser.js';
/**
 * Computed style property parser.
 */
export default class CSSStyleDeclarationPropertyGetParser {
    /**
     * Returns margin.
     *
     * @param properties Properties.
     * @returns Property value
     */
    static getMargin(properties) {
        return this.getPaddingLikeProperty(['margin-top', 'margin-right', 'margin-bottom', 'margin-left'], properties);
    }
    /**
     * Returns padding.
     *
     * @param properties Properties.
     * @returns Property value
     */
    static getPadding(properties) {
        return this.getPaddingLikeProperty(['padding-top', 'padding-right', 'padding-bottom', 'padding-left'], properties);
    }
    /**
     * Returns outline.
     *
     * @param properties Properties.
     * @returns Property value
     */
    static getOutline(properties) {
        if (!properties['outline-color']?.value ||
            !properties['outline-style']?.value ||
            !properties['outline-width']?.value) {
            return null;
        }
        const important = properties['outline-color'].important &&
            properties['outline-style'].important &&
            properties['outline-width'].important;
        if (CSSStyleDeclarationValueParser.getGlobalExceptInitial(properties['outline-width'].value) &&
            properties['outline-width'].value === properties['outline-style'].value &&
            properties['outline-width'].value === properties['outline-color'].value) {
            return {
                important,
                value: properties['outline-width'].value
            };
        }
        const values = [];
        if (!CSSStyleDeclarationValueParser.getInitial(properties['outline-color']?.value)) {
            values.push(properties['outline-color'].value);
        }
        if (!CSSStyleDeclarationValueParser.getInitial(properties['outline-style']?.value)) {
            values.push(properties['outline-style'].value);
        }
        if (!CSSStyleDeclarationValueParser.getInitial(properties['outline-width'].value)) {
            values.push(properties['outline-width'].value);
        }
        return {
            important,
            value: values.join(' ')
        };
    }
    /**
     * Returns border.
     *
     * @param properties Properties.
     * @returns Property value
     */
    static getBorder(properties) {
        if (!properties['border-top-width']?.value ||
            properties['border-top-width']?.value !== properties['border-right-width']?.value ||
            properties['border-top-width']?.value !== properties['border-bottom-width']?.value ||
            properties['border-top-width']?.value !== properties['border-left-width']?.value ||
            !properties['border-top-style']?.value ||
            properties['border-top-style']?.value !== properties['border-right-style']?.value ||
            properties['border-top-style']?.value !== properties['border-bottom-style']?.value ||
            properties['border-top-style']?.value !== properties['border-left-style']?.value ||
            !properties['border-top-color']?.value ||
            properties['border-top-color']?.value !== properties['border-right-color']?.value ||
            properties['border-top-color']?.value !== properties['border-bottom-color']?.value ||
            properties['border-top-color']?.value !== properties['border-left-color']?.value ||
            !properties['border-image-source']?.value ||
            !properties['border-image-slice']?.value ||
            !properties['border-image-width']?.value ||
            !properties['border-image-outset']?.value ||
            !properties['border-image-repeat']?.value) {
            return null;
        }
        const important = properties['border-top-width'].important &&
            properties['border-right-width'].important &&
            properties['border-bottom-width'].important &&
            properties['border-left-width'].important &&
            properties['border-top-style'].important &&
            properties['border-right-style'].important &&
            properties['border-bottom-style'].important &&
            properties['border-left-style'].important &&
            properties['border-top-color'].important &&
            properties['border-right-color'].important &&
            properties['border-bottom-color'].important &&
            properties['border-left-color'].important &&
            properties['border-image-source'].important &&
            properties['border-image-slice'].important &&
            properties['border-image-width'].important &&
            properties['border-image-outset'].important &&
            properties['border-image-repeat'].important;
        if (CSSStyleDeclarationValueParser.getGlobalExceptInitial(properties['border-top-width'].value) ||
            CSSStyleDeclarationValueParser.getGlobalExceptInitial(properties['border-top-style'].value) ||
            CSSStyleDeclarationValueParser.getGlobalExceptInitial(properties['border-top-color'].value) ||
            CSSStyleDeclarationValueParser.getGlobalExceptInitial(properties['border-image-source'].value) ||
            CSSStyleDeclarationValueParser.getGlobalExceptInitial(properties['border-image-slice'].value) ||
            CSSStyleDeclarationValueParser.getGlobalExceptInitial(properties['border-image-width'].value) ||
            CSSStyleDeclarationValueParser.getGlobalExceptInitial(properties['border-image-outset'].value) ||
            CSSStyleDeclarationValueParser.getGlobalExceptInitial(properties['border-image-repeat'].value)) {
            if (properties['border-top-width'].value !== properties['border-top-style'].value ||
                properties['border-top-width'].value !== properties['border-top-color'].value ||
                properties['border-top-width'].value !== properties['border-image-source'].value ||
                properties['border-top-width'].value !== properties['border-image-slice'].value ||
                properties['border-top-width'].value !== properties['border-image-width'].value ||
                properties['border-top-width'].value !== properties['border-image-outset'].value ||
                properties['border-top-width'].value !== properties['border-image-repeat'].value) {
                return null;
            }
            return {
                important,
                value: properties['border-top-width'].value
            };
        }
        const values = [];
        if (!CSSStyleDeclarationValueParser.getInitial(properties['border-top-width'].value)) {
            values.push(properties['border-top-width'].value);
        }
        if (!CSSStyleDeclarationValueParser.getInitial(properties['border-top-style'].value)) {
            values.push(properties['border-top-style'].value);
        }
        if (!CSSStyleDeclarationValueParser.getInitial(properties['border-top-color'].value)) {
            values.push(properties['border-top-color'].value);
        }
        return {
            important,
            value: values.join(' ')
        };
    }
    /**
     * Returns border.
     *
     * @param properties Properties.
     * @returns Property value
     */
    static getBorderTop(properties) {
        return this.getBorderTopRightBottomLeft('top', properties);
    }
    /**
     * Returns border.
     *
     * @param properties Properties.
     * @returns Property value
     */
    static getBorderRight(properties) {
        return this.getBorderTopRightBottomLeft('right', properties);
    }
    /**
     * Returns border.
     *
     * @param properties Properties.
     * @returns Property value
     */
    static getBorderBottom(properties) {
        return this.getBorderTopRightBottomLeft('bottom', properties);
    }
    /**
     * Returns border.
     *
     * @param properties Properties.
     * @returns Property value
     */
    static getBorderLeft(properties) {
        return this.getBorderTopRightBottomLeft('left', properties);
    }
    /**
     * Returns border.
     *
     * @param properties Properties.
     * @returns Property value
     */
    static getBorderColor(properties) {
        return this.getPaddingLikeProperty(['border-top-color', 'border-right-color', 'border-bottom-color', 'border-left-color'], properties);
    }
    /**
     * Returns border.
     *
     * @param properties Properties.
     * @returns Property value
     */
    static getBorderWidth(properties) {
        return this.getPaddingLikeProperty(['border-top-width', 'border-right-width', 'border-bottom-width', 'border-left-width'], properties);
    }
    /**
     * Returns border.
     *
     * @param properties Properties.
     * @returns Property value
     */
    static getBorderStyle(properties) {
        return this.getPaddingLikeProperty(['border-top-style', 'border-right-style', 'border-bottom-style', 'border-left-style'], properties);
    }
    /**
     * Returns border radius.
     *
     * @param properties Properties.
     * @returns Property value
     */
    static getBorderRadius(properties) {
        return this.getPaddingLikeProperty([
            'border-top-left-radius',
            'border-top-right-radius',
            'border-bottom-right-radius',
            'border-bottom-left-radius'
        ], properties);
    }
    /**
     * Returns border image.
     *
     * @param properties Properties.
     * @returns Property value
     */
    static getBorderImage(properties) {
        if (!properties['border-image-source']?.value ||
            !properties['border-image-slice']?.value ||
            !properties['border-image-width']?.value ||
            !properties['border-image-outset']?.value ||
            !properties['border-image-repeat']?.value) {
            return null;
        }
        const important = properties['border-image-source'].important &&
            properties['border-image-slice'].important &&
            properties['border-image-width'].important &&
            properties['border-image-outset'].important &&
            properties['border-image-repeat'].important;
        if (CSSStyleDeclarationValueParser.getGlobal(properties['border-image-source'].value) ||
            CSSStyleDeclarationValueParser.getGlobal(properties['border-image-slice'].value) ||
            CSSStyleDeclarationValueParser.getGlobal(properties['border-image-width'].value) ||
            CSSStyleDeclarationValueParser.getGlobal(properties['border-image-outset'].value) ||
            CSSStyleDeclarationValueParser.getGlobal(properties['border-image-repeat'].value)) {
            if (properties['border-image-source'].value !== properties['border-image-slice'].value ||
                properties['border-image-source'].value !== properties['border-image-width'].value ||
                properties['border-image-source'].value !== properties['border-image-outset'].value ||
                properties['border-image-source'].value !== properties['border-image-repeat'].value) {
                return null;
            }
            return {
                important,
                value: properties['border-image-source'].value
            };
        }
        return {
            important,
            value: `${properties['border-image-source'].value} ${properties['border-image-slice'].value} / ${properties['border-image-width'].value} / ${properties['border-image-outset'].value} ${properties['border-image-repeat'].value}`
        };
    }
    /**
     * Returns background.
     *
     * @param properties Properties.
     * @returns Property value
     */
    static getBackground(properties) {
        if (!properties['background-image']?.value ||
            !properties['background-repeat']?.value ||
            !properties['background-attachment']?.value ||
            !properties['background-position-x']?.value ||
            !properties['background-position-y']?.value ||
            !properties['background-color']?.value ||
            !properties['background-size']?.value ||
            !properties['background-origin']?.value ||
            !properties['background-clip']?.value) {
            return null;
        }
        const important = properties['background-image'].important &&
            properties['background-repeat'].important &&
            properties['background-attachment'].important &&
            properties['background-position-x'].important &&
            properties['background-position-y'].important &&
            properties['background-color'].important &&
            properties['background-size'].important &&
            properties['background-origin'].important &&
            properties['background-clip'].important;
        if (CSSStyleDeclarationValueParser.getGlobalExceptInitial(properties['background-image'].value) ||
            CSSStyleDeclarationValueParser.getGlobalExceptInitial(properties['background-repeat'].value) ||
            CSSStyleDeclarationValueParser.getGlobalExceptInitial(properties['background-attachment'].value) ||
            CSSStyleDeclarationValueParser.getGlobalExceptInitial(properties['background-position-x'].value) ||
            CSSStyleDeclarationValueParser.getGlobalExceptInitial(properties['background-position-y'].value) ||
            CSSStyleDeclarationValueParser.getGlobalExceptInitial(properties['background-color'].value) ||
            CSSStyleDeclarationValueParser.getGlobalExceptInitial(properties['background-size'].value) ||
            CSSStyleDeclarationValueParser.getGlobalExceptInitial(properties['background-origin'].value) ||
            CSSStyleDeclarationValueParser.getGlobalExceptInitial(properties['background-clip'].value)) {
            if (properties['background-image'].value !== properties['background-repeat'].value ||
                properties['background-image'].value !== properties['background-attachment'].value ||
                properties['background-image'].value !== properties['background-position-x'].value ||
                properties['background-image'].value !== properties['background-position-y'].value ||
                properties['background-image'].value !== properties['background-color'].value ||
                properties['background-image'].value !== properties['background-size'].value ||
                properties['background-image'].value !== properties['background-origin'].value ||
                properties['background-image'].value !== properties['background-clip'].value) {
                return null;
            }
            return {
                important,
                value: properties['background-image'].value
            };
        }
        const values = [];
        if (!CSSStyleDeclarationValueParser.getInitial(properties['background-image'].value)) {
            values.push(properties['background-image'].value);
        }
        if (!CSSStyleDeclarationValueParser.getInitial(properties['background-position-x'].value) &&
            !CSSStyleDeclarationValueParser.getInitial(properties['background-position-y'].value) &&
            !CSSStyleDeclarationValueParser.getInitial(properties['background-size'].value)) {
            values.push(`${properties['background-position-x'].value} ${properties['background-position-y'].value} / ${properties['background-size'].value}`);
        }
        else if (!CSSStyleDeclarationValueParser.getInitial(properties['background-position-x'].value) &&
            !CSSStyleDeclarationValueParser.getInitial(properties['background-position-y'].value)) {
            values.push(`${properties['background-position-x'].value} ${properties['background-position-y'].value}`);
        }
        if (!CSSStyleDeclarationValueParser.getInitial(properties['background-repeat'].value)) {
            values.push(properties['background-repeat'].value);
        }
        if (!CSSStyleDeclarationValueParser.getInitial(properties['background-attachment'].value)) {
            values.push(properties['background-attachment'].value);
        }
        if (!CSSStyleDeclarationValueParser.getInitial(properties['background-origin'].value)) {
            values.push(properties['background-origin'].value);
        }
        if (!CSSStyleDeclarationValueParser.getInitial(properties['background-clip'].value)) {
            values.push(properties['background-clip'].value);
        }
        if (!CSSStyleDeclarationValueParser.getInitial(properties['background-color'].value)) {
            values.push(properties['background-color'].value);
        }
        return {
            important,
            value: values.join(' ')
        };
    }
    /**
     * Returns background position.
     *
     * @param properties Properties.
     * @returns Property value
     */
    static getBackgroundPosition(properties) {
        if (!properties['background-position-x']?.value ||
            !properties['background-position-y']?.value) {
            return null;
        }
        const important = properties['background-position-x'].important &&
            properties['background-position-y'].important;
        if (CSSStyleDeclarationValueParser.getGlobal(properties['background-position-x'].value) ||
            CSSStyleDeclarationValueParser.getGlobal(properties['background-position-y'].value)) {
            if (properties['background-position-x'].value !== properties['background-position-y'].value) {
                return null;
            }
            return {
                important,
                value: properties['background-position-x'].value
            };
        }
        const positionX = properties['background-position-x'].value.replace(/ *, */g, ',').split(',');
        const positionY = properties['background-position-y'].value.replace(/ *, */g, ',').split(',');
        const parts = [];
        for (let i = 0; i < positionX.length; i++) {
            parts.push(`${positionX[i]} ${positionY[i]}`);
        }
        return {
            important,
            value: parts.join(', ')
        };
    }
    /**
     * Returns flex.
     *
     * @param properties Properties.
     * @returns Property value
     */
    static getFlex(properties) {
        if (!properties['flex-grow']?.value ||
            !properties['flex-shrink']?.value ||
            !properties['flex-basis']?.value) {
            return null;
        }
        const important = properties['flex-grow'].important &&
            properties['flex-shrink'].important &&
            properties['flex-basis'].important;
        if (CSSStyleDeclarationValueParser.getGlobal(properties['flex-grow'].value) ||
            CSSStyleDeclarationValueParser.getGlobal(properties['flex-shrink'].value) ||
            CSSStyleDeclarationValueParser.getGlobal(properties['flex-basis'].value)) {
            if (properties['flex-grow'].value !== properties['flex-shrink'].value ||
                properties['flex-grow'].value !== properties['flex-basis'].value) {
                return null;
            }
            return {
                important,
                value: properties['flex-grow'].value
            };
        }
        return {
            important,
            value: `${properties['flex-grow'].value} ${properties['flex-shrink'].value} ${properties['flex-basis'].value}`
        };
    }
    /**
     * Returns flex.
     *
     * @param properties Properties.
     * @returns Property value
     */
    static getFont(properties) {
        if (!properties['font-size']?.value ||
            !properties['font-family']?.value ||
            !properties['font-weight']?.value ||
            !properties['font-style']?.value ||
            !properties['font-variant']?.value ||
            !properties['font-stretch']?.value ||
            !properties['line-height']?.value) {
            return null;
        }
        const important = properties['font-size'].important &&
            properties['font-family'].important &&
            properties['font-weight'].important &&
            properties['font-style'].important &&
            properties['font-variant'].important &&
            properties['font-stretch'].important &&
            properties['line-height'].important;
        if (CSSStyleDeclarationValueParser.getGlobal(properties['font-size'].value) ||
            CSSStyleDeclarationValueParser.getGlobal(properties['font-family'].value) ||
            CSSStyleDeclarationValueParser.getGlobal(properties['font-weight'].value) ||
            CSSStyleDeclarationValueParser.getGlobal(properties['font-style'].value) ||
            CSSStyleDeclarationValueParser.getGlobal(properties['font-variant'].value) ||
            CSSStyleDeclarationValueParser.getGlobal(properties['font-stretch'].value) ||
            CSSStyleDeclarationValueParser.getGlobal(properties['line-height'].value)) {
            if (properties['font-size'].value !== properties['font-family'].value ||
                properties['font-size'].value !== properties['font-weight'].value ||
                properties['font-size'].value !== properties['font-style'].value ||
                properties['font-size'].value !== properties['font-variant'].value ||
                properties['font-size'].value !== properties['font-stretch'].value ||
                properties['font-size'].value !== properties['line-height'].value) {
                return null;
            }
            return {
                important,
                value: properties['font-size'].value
            };
        }
        const values = [];
        if (properties['font-style'].value !== 'normal') {
            values.push(properties['font-style'].value);
        }
        if (properties['font-variant'].value !== 'normal') {
            values.push(properties['font-variant'].value);
        }
        if (properties['font-weight'].value !== 'normal') {
            values.push(properties['font-weight'].value);
        }
        if (properties['font-stretch'].value !== 'normal') {
            values.push(properties['font-stretch'].value);
        }
        if (properties['line-height'].value !== 'normal') {
            values.push(`${properties['font-size'].value} / ${properties['line-height'].value}`);
        }
        else {
            values.push(properties['font-size'].value);
        }
        values.push(properties['font-family'].value);
        return {
            important,
            value: values.join(' ')
        };
    }
    /**
     * Returns border.
     *
     * @param properties Properties.
     * @param position
     * @returns Property value
     */
    static getBorderTopRightBottomLeft(position, properties) {
        if (!properties[`border-${position}-width`]?.value ||
            !properties[`border-${position}-style`]?.value ||
            !properties[`border-${position}-color`]?.value) {
            return null;
        }
        const important = properties[`border-${position}-width`].important &&
            properties[`border-${position}-style`].important &&
            properties[`border-${position}-color`].important;
        if (CSSStyleDeclarationValueParser.getGlobalExceptInitial(properties[`border-${position}-width`].value) &&
            properties[`border-${position}-width`].value ===
                properties[`border-${position}-style`].value &&
            properties[`border-${position}-width`].value === properties[`border-${position}-color`].value) {
            return {
                important,
                value: properties[`border-${position}-width`].value
            };
        }
        const values = [];
        if (!CSSStyleDeclarationValueParser.getInitial(properties[`border-${position}-width`].value)) {
            values.push(properties[`border-${position}-width`].value);
        }
        if (!CSSStyleDeclarationValueParser.getInitial(properties[`border-${position}-style`]?.value)) {
            values.push(properties[`border-${position}-style`].value);
        }
        if (!CSSStyleDeclarationValueParser.getInitial(properties[`border-${position}-color`]?.value)) {
            values.push(properties[`border-${position}-color`].value);
        }
        return {
            important,
            value: values.join(' ')
        };
    }
    /**
     * Returns a padding like property.
     *
     * @param properties Properties.
     * @param position
     * @param propertyNames
     * @returns Property value
     */
    static getPaddingLikeProperty(propertyNames, properties) {
        if (!properties[propertyNames[0]]?.value ||
            !properties[propertyNames[1]]?.value ||
            !properties[propertyNames[2]]?.value ||
            !properties[propertyNames[3]]?.value) {
            return null;
        }
        const important = properties[propertyNames[0]].important &&
            properties[propertyNames[1]].important &&
            properties[propertyNames[2]].important &&
            properties[propertyNames[3]].important;
        if (CSSStyleDeclarationValueParser.getGlobal(properties[propertyNames[0]].value) ||
            CSSStyleDeclarationValueParser.getGlobal(properties[propertyNames[1]].value) ||
            CSSStyleDeclarationValueParser.getGlobal(properties[propertyNames[2]].value) ||
            CSSStyleDeclarationValueParser.getGlobal(properties[propertyNames[3]].value)) {
            if (properties[propertyNames[0]].value !== properties[propertyNames[1]].value ||
                properties[propertyNames[0]].value !== properties[propertyNames[2]].value ||
                properties[propertyNames[0]].value !== properties[propertyNames[3]].value) {
                return null;
            }
            return {
                important,
                value: properties[propertyNames[0]].value
            };
        }
        const values = [properties[propertyNames[0]].value];
        if (properties[propertyNames[1]].value !== properties[propertyNames[0]].value ||
            properties[propertyNames[2]].value !== properties[propertyNames[0]].value ||
            properties[propertyNames[3]].value !== properties[propertyNames[1]].value) {
            values.push(properties[propertyNames[1]].value);
        }
        if (properties[propertyNames[2]].value !== properties[propertyNames[0]].value ||
            properties[propertyNames[3]].value !== properties[propertyNames[1]].value) {
            values.push(properties[propertyNames[2]].value);
        }
        if (properties[propertyNames[3]].value !== properties[propertyNames[1]].value) {
            values.push(properties[propertyNames[3]].value);
        }
        return {
            important,
            value: values.join(' ')
        };
    }
}
//# sourceMappingURL=CSSStyleDeclarationPropertyGetParser.js.map