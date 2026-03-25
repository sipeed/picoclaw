import HTMLElement from '../html-element/HTMLElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import Event from '../../event/Event.js';
import ErrorEvent from '../../event/events/ErrorEvent.js';
import Attr from '../attr/Attr.js';
/**
 * HTML Script Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLScriptElement.
 */
export default class HTMLScriptElement extends HTMLElement {
    #private;
    cloneNode: (deep?: boolean) => HTMLScriptElement;
    onerror: (event: ErrorEvent) => void;
    onload: (event: Event) => void;
    [PropertySymbol.evaluateScript]: boolean;
    /**
     * Returns type.
     *
     * @returns Type.
     */
    get type(): string;
    /**
     * Sets type.
     *
     * @param type Type.
     */
    set type(type: string);
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
     * Returns charset.
     *
     * @returns Charset.
     */
    get charset(): string;
    /**
     * Sets charset.
     *
     * @param charset Charset.
     */
    set charset(charset: string);
    /**
     * Returns lang.
     *
     * @returns Lang.
     */
    get lang(): string;
    /**
     * Sets lang.
     *
     * @param lang Lang.
     */
    set lang(lang: string);
    /**
     * Returns async.
     *
     * @returns Async.
     */
    get async(): boolean;
    /**
     * Sets async.
     *
     * @param async Async.
     */
    set async(async: boolean);
    /**
     * Returns defer.
     *
     * @returns Defer.
     */
    get defer(): boolean;
    /**
     * Sets defer.
     *
     * @param defer Defer.
     */
    set defer(defer: boolean);
    /**
     * Returns text.
     *
     * @returns Text.
     */
    get text(): string;
    /**
     * Sets text.
     *
     * @param text Text.
     */
    set text(text: string);
    /**
     * @override
     */
    [PropertySymbol.cloneNode](deep?: boolean): HTMLScriptElement;
    /**
     * @override
     */
    [PropertySymbol.connectedToDocument](): void;
    /**
     * @override
     */
    [PropertySymbol.onSetAttribute](attribute: Attr, replacedAttribute: Attr | null): void;
}
//# sourceMappingURL=HTMLScriptElement.d.ts.map