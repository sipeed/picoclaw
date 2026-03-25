import HTMLElement from '../html-element/HTMLElement.js';
import TextTrack from '../html-media-element/TextTrack.js';
/**
 * HTMLTrackElement
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLTrackElement
 */
export default class HTMLTrackElement extends HTMLElement {
    oncuechange: (event: Event) => void;
    /**
     * Returns kind.
     *
     * @returns Kind.
     */
    get kind(): string;
    /**
     * Sets kind.
     *
     * @param value Value.
     */
    set kind(value: string);
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
     * Returns source language.
     *
     * @returns Source language.
     */
    get srclang(): string;
    /**
     * Sets source language.
     *
     * @param value Value.
     */
    set srclang(value: string);
    /**
     * Returns label.
     *
     * @returns Label.
     */
    get label(): string;
    /**
     * Sets label.
     *
     * @param value Value.
     */
    set label(value: string);
    /**
     * Returns default.
     *
     * @returns Default.
     */
    get default(): boolean;
    /**
     * Sets default.
     *
     * @param value Value.
     */
    set default(value: boolean);
    /**
     * Returns ready state.
     *
     * @returns Ready state.
     */
    get readyState(): number;
    /**
     * Returns the TextTrack object corresponding to the track element.
     *
     * @returns TextTrack
     */
    get track(): TextTrack;
}
//# sourceMappingURL=HTMLTrackElement.d.ts.map