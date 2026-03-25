import HTMLElement from '../html-element/HTMLElement.cjs';
import HTMLLabelElement from '../html-label-element/HTMLLabelElement.cjs';
import NodeList from '../node/NodeList.cjs';
/**
 * HTMLMeterElement
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLMeterElement
 */
export default class HTMLMeterElement extends HTMLElement {
    /**
     * Returns high.
     *
     * @returns High.
     */
    get high(): number;
    /**
     * Sets high.
     *
     * @param high High.
     */
    set high(high: number);
    /**
     * Returns low.
     *
     * @returns Low.
     */
    get low(): number;
    /**
     * Sets low.
     *
     * @param low Low.
     */
    set low(low: number);
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
     * Returns min.
     *
     * @returns Min.
     */
    get min(): number;
    /**
     * Sets min.
     *
     * @param min Min.
     */
    set min(min: number);
    /**
     * Returns optimum.
     *
     * @returns Optimum.
     */
    get optimum(): number;
    /**
     * Sets optimum.
     *
     * @param optimum Optimum.
     */
    set optimum(optimum: number);
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
     * Returns the associated label elements.
     *
     * @returns Label elements.
     */
    get labels(): NodeList<HTMLLabelElement>;
}
//# sourceMappingURL=HTMLMeterElement.d.ts.map