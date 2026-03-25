import HTMLElement from '../html-element/HTMLElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import TextTrackKindEnum from '../html-media-element/TextTrackKindEnum.js';
/**
 * HTMLTrackElement
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLTrackElement
 */
export default class HTMLTrackElement extends HTMLElement {
    // Events
    oncuechange = null;
    /**
     * Returns kind.
     *
     * @returns Kind.
     */
    get kind() {
        const kind = this.getAttribute('kind');
        if (kind === null) {
            return TextTrackKindEnum.subtitles;
        }
        if (!TextTrackKindEnum[kind]) {
            return TextTrackKindEnum.metadata;
        }
        return kind;
    }
    /**
     * Sets kind.
     *
     * @param value Value.
     */
    set kind(value) {
        if (!TextTrackKindEnum[value]) {
            value = TextTrackKindEnum.metadata;
        }
        this.setAttribute('kind', value);
    }
    /**
     * Returns source.
     *
     * @returns Source.
     */
    get src() {
        if (!this.hasAttribute('src')) {
            return '';
        }
        try {
            return new URL(this.getAttribute('src'), this[PropertySymbol.ownerDocument].location.href)
                .href;
        }
        catch (e) {
            return this.getAttribute('src');
        }
    }
    /**
     * Sets source.
     *
     * @param src Source.
     */
    set src(src) {
        this.setAttribute('src', src);
    }
    /**
     * Returns source language.
     *
     * @returns Source language.
     */
    get srclang() {
        return this.getAttribute('srclang') || '';
    }
    /**
     * Sets source language.
     *
     * @param value Value.
     */
    set srclang(value) {
        this.setAttribute('srclang', value);
    }
    /**
     * Returns label.
     *
     * @returns Label.
     */
    get label() {
        return this.getAttribute('label') || '';
    }
    /**
     * Sets label.
     *
     * @param value Value.
     */
    set label(value) {
        this.setAttribute('label', value);
    }
    /**
     * Returns default.
     *
     * @returns Default.
     */
    get default() {
        return this.hasAttribute('default');
    }
    /**
     * Sets default.
     *
     * @param value Value.
     */
    set default(value) {
        if (value) {
            this.setAttribute('default', '');
        }
        else {
            this.removeAttribute('default');
        }
    }
    /**
     * Returns ready state.
     *
     * @returns Ready state.
     */
    get readyState() {
        return 0;
    }
    /**
     * Returns the TextTrack object corresponding to the track element.
     *
     * @returns TextTrack
     */
    get track() {
        const textTrack = new this[PropertySymbol.window].TextTrack(PropertySymbol.illegalConstructor);
        textTrack[PropertySymbol.kind] = this.kind;
        textTrack[PropertySymbol.label] = this.label;
        textTrack[PropertySymbol.language] = this.srclang;
        textTrack[PropertySymbol.mode] = this.default ? 'showing' : 'disabled';
        textTrack[PropertySymbol.id] = this.id;
        return textTrack;
    }
}
//# sourceMappingURL=HTMLTrackElement.js.map