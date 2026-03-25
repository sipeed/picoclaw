import CSSStyleSheet from '../../css/CSSStyleSheet.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import SVGElement from '../svg-element/SVGElement.js';
/**
 * SVG Style Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGStyleElement
 */
export default class SVGStyleElement extends SVGElement {
    [PropertySymbol.sheet]: CSSStyleSheet | null;
    [PropertySymbol.styleNode]: this;
    [PropertySymbol.disabled]: boolean;
    /**
     * Returns media.
     *
     * @returns Media.
     */
    get media(): string;
    /**
     * Sets media.
     *
     * @param media Media.
     */
    set media(media: string);
    /**
     * Returns type.
     *
     * @deprecated
     * @returns Type.
     */
    get type(): string;
    /**
     * Sets type.
     *
     * @deprecated
     * @param type Type.
     */
    set type(type: string);
    /**
     * Returns title.
     *
     * @returns Title.
     */
    get title(): string;
    /**
     * Sets title.
     *
     * @param title Title.
     */
    set title(title: string);
    /**
     * Returns disabled.
     *
     * @returns Disabled.
     */
    get disabled(): boolean;
    /**
     * Sets disabled.
     *
     * @param disabled Disabled.
     */
    set disabled(disabled: boolean);
    /**
     * Returns style.
     *
     * @returns Style.
     */
    get sheet(): CSSStyleSheet;
    /**
     * @override
     */
    [PropertySymbol.disconnectedFromDocument](): void;
    /**
     * Updates the CSSStyleSheet with the text content.
     */
    [PropertySymbol.updateSheet](): void;
}
//# sourceMappingURL=SVGStyleElement.d.ts.map