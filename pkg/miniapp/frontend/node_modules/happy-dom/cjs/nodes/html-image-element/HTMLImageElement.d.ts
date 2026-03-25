import HTMLElement from '../html-element/HTMLElement.cjs';
import * as PropertySymbol from '../../PropertySymbol.cjs';
/**
 * HTML Image Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLImageElement.
 */
export default class HTMLImageElement extends HTMLElement {
    [PropertySymbol.tagName]: string;
    [PropertySymbol.complete]: boolean;
    [PropertySymbol.naturalHeight]: number;
    [PropertySymbol.naturalWidth]: number;
    [PropertySymbol.loading]: string;
    [PropertySymbol.x]: number;
    [PropertySymbol.y]: number;
    cloneNode: (deep?: boolean) => HTMLImageElement;
    /**
     * Returns complete.
     *
     * @returns Complete.
     */
    get complete(): boolean;
    /**
     * Returns natural height.
     *
     * @returns Natural height.
     */
    get naturalHeight(): number;
    /**
     * Returns natural width.
     *
     * @returns Natural width.
     */
    get naturalWidth(): number;
    /**
     * Returns loading.
     *
     * @returns Loading.
     */
    get loading(): string;
    /**
     * Sets loading.
     *
     * @param loading Loading.
     */
    set loading(loading: string);
    /**
     * Returns x.
     */
    get x(): number;
    /**
     * Returns y.
     */
    get y(): number;
    /**
     * Returns decoding.
     *
     * @returns Decoding.
     */
    get decoding(): string;
    /**
     * Sets decoding.
     *
     * @param decoding Decoding.
     */
    set decoding(decoding: string);
    /**
     * Returns cross origin.
     *
     * @returns Cross origin.
     */
    get crossOrigin(): string | null;
    /**
     * Sets cross origin.
     *
     * @param crossOrigin Cross origin.
     */
    set crossOrigin(crossOrigin: string | null);
    /**
     * Returns alt.
     *
     * @returns Alt.
     */
    get alt(): string;
    /**
     * Sets alt.
     *
     * @param alt Alt.
     */
    set alt(alt: string);
    /**
     * Returns current src.
     *
     * @returns Current src.
     */
    get currentSrc(): string;
    /**
     * Returns width.
     *
     * @returns Width.
     */
    get width(): number;
    /**
     * Sets width.
     *
     * @param width Width.
     */
    set width(width: number);
    /**
     * Returns height.
     *
     * @returns Height.
     */
    get height(): number;
    /**
     * Sets height.
     *
     * @param height Height.
     */
    set height(height: number);
    /**
     * Returns is map.
     *
     * @returns Is map.
     */
    get isMap(): boolean;
    /**
     * Sets is map.
     *
     * @param ismap Is map.
     */
    set isMap(isMap: boolean);
    /**
     * Returns referrer policy.
     *
     * @returns Referrer policy.
     */
    get referrerPolicy(): string;
    /**
     * Sets referrer policy.
     *
     * @param referrerPolicy Referrer policy.
     */
    set referrerPolicy(referrerPolicy: string);
    /**
     * Returns sizes.
     *
     * @returns Sizes.
     */
    get sizes(): string;
    /**
     * Sets sizes.
     *
     * @param sizes Sizes.
     */
    set sizes(sizes: string);
    /**
     * Returns source.
     *
     * @returns Source.
     */
    get src(): string;
    /**
     * Sets source.
     *
     * @param src Source.
     */
    set src(src: string);
    /**
     * Returns srcset.
     *
     * @returns Source.
     */
    get srcset(): string;
    /**
     * Sets src set.
     *
     * @param srcset Src set.
     */
    set srcset(srcset: string);
    /**
     * Returns use map.
     *
     * @returns Use map.
     */
    get useMap(): string;
    /**
     * Sets is map.
     *
     * @param useMap Is map.
     */
    set useMap(useMap: string);
    /**
     * The decode() method of the HTMLImageElement interface returns a Promise that resolves when the image is decoded and it is safe to append the image to the DOM.
     *
     * @returns Promise.
     */
    decode(): Promise<void>;
    /**
     * @override
     */
    [PropertySymbol.cloneNode](deep?: boolean): HTMLImageElement;
}
//# sourceMappingURL=HTMLImageElement.d.ts.map