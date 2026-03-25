import HTMLElement from '../html-element/HTMLElement.cjs';
/**
 * HTML Opt Group Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLOptGroupElement
 */
export default class HTMLOptGroupElement extends HTMLElement {
    /**
     * Returns label.
     *
     * @returns Label.
     */
    get label(): string;
    /**
     * Sets label.
     *
     * @param label Label.
     */
    set label(label: string);
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
}
//# sourceMappingURL=HTMLOptGroupElement.d.ts.map