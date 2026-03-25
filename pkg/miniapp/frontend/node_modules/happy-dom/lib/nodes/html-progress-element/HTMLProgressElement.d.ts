import HTMLElement from '../html-element/HTMLElement.js';
import HTMLLabelElement from '../html-label-element/HTMLLabelElement.js';
import NodeList from '../node/NodeList.js';
/**
 * HTMLProgressElement
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLProgressElement
 */
export default class HTMLProgressElement extends HTMLElement {
    /**
     * Returns max.
     *
     * @returns Max.
     */
    get max(): number;
    /**
     * Sets max.
     *
     * @param max Max.
     */
    set max(max: number);
    /**
     * Returns value.
     *
     * @returns Value.
     */
    get value(): number;
    /**
     * Sets value.
     *
     * @param value Value.
     */
    set value(value: number);
    /**
     * Returns position.
     *
     * @returns Position.
     */
    get position(): number;
    /**
     * Returns the associated label elements.
     *
     * @returns Label elements.
     */
    get labels(): NodeList<HTMLLabelElement>;
}
//# sourceMappingURL=HTMLProgressElement.d.ts.map