import * as PropertySymbol from '../../PropertySymbol.js';
import QuerySelector from '../../query-selector/QuerySelector.js';
import HTMLElement from '../html-element/HTMLElement.js';
/**
 * HTML Option Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLOptionElement.
 */
export default class HTMLOptionElement extends HTMLElement {
    [PropertySymbol.selectedness] = false;
    [PropertySymbol.dirtyness] = false;
    [PropertySymbol.selectNode] = null;
    /**
     * Returns inner text, which is the rendered appearance of text.
     *
     * @returns Inner text.
     */
    get text() {
        return this.innerText;
    }
    /**
     * Sets the inner text, which is the rendered appearance of text.
     *
     * @param innerText Inner text.
     */
    set text(text) {
        this.innerText = text;
    }
    /**
     * Returns index.
     *
     * @returns Index.
     */
    get index() {
        if (!this[PropertySymbol.selectNode]) {
            return 0;
        }
        const options = QuerySelector.querySelectorAll(this[PropertySymbol.selectNode], 'option')[PropertySymbol.elements];
        return options.indexOf(this);
    }
    /**
     * Returns the parent form element.
     *
     * @returns Form.
     */
    get form() {
        return this[PropertySymbol.selectNode]?.form || null;
    }
    /**
     * Returns selected.
     *
     * @returns Selected.
     */
    get selected() {
        return this[PropertySymbol.selectedness];
    }
    /**
     * Sets selected.
     *
     * @param selected Selected.
     */
    set selected(selected) {
        const selectNode = this[PropertySymbol.selectNode];
        this[PropertySymbol.dirtyness] = true;
        this[PropertySymbol.selectedness] = Boolean(selected);
        if (selectNode) {
            selectNode[PropertySymbol.updateSelectedness](this[PropertySymbol.selectedness] ? this : null);
        }
    }
    /**
     * Returns disabled.
     *
     * @returns Disabled.
     */
    get disabled() {
        return this.getAttribute('disabled') !== null;
    }
    /**
     * Sets disabled.
     *
     * @param disabled Disabled.
     */
    set disabled(disabled) {
        if (!disabled) {
            this.removeAttribute('disabled');
        }
        else {
            this.setAttribute('disabled', '');
        }
    }
    /**
     * Returns value.
     *
     * @returns Value.
     */
    get value() {
        return this.getAttribute('value') ?? this.textContent;
    }
    /**
     * Sets value.
     *
     * @param value Value.
     */
    set value(value) {
        this.setAttribute('value', value);
    }
    /**
     * @override
     */
    [PropertySymbol.onSetAttribute](attribute, replacedAttribute) {
        super[PropertySymbol.onSetAttribute](attribute, replacedAttribute);
        if (!this[PropertySymbol.dirtyness] &&
            attribute[PropertySymbol.name] === 'selected' &&
            replacedAttribute?.[PropertySymbol.value] !== attribute[PropertySymbol.value]) {
            const selectNode = this[PropertySymbol.selectNode];
            this[PropertySymbol.selectedness] = true;
            if (selectNode) {
                selectNode[PropertySymbol.updateSelectedness](this);
            }
        }
    }
    /**
     * @override
     */
    [PropertySymbol.onRemoveAttribute](removedAttribute) {
        super[PropertySymbol.onRemoveAttribute](removedAttribute);
        if (removedAttribute &&
            !this[PropertySymbol.dirtyness] &&
            removedAttribute[PropertySymbol.name] === 'selected') {
            const selectNode = this[PropertySymbol.selectNode];
            this[PropertySymbol.selectedness] = false;
            if (selectNode) {
                selectNode[PropertySymbol.updateSelectedness]();
            }
        }
    }
    /**
     * @override
     */
    [PropertySymbol.connectedToNode]() {
        super[PropertySymbol.connectedToNode]();
        if (this[PropertySymbol.selectNode]) {
            this[PropertySymbol.selectNode][PropertySymbol.updateSelectedness]();
        }
    }
    /**
     * @override
     */
    [PropertySymbol.disconnectedFromNode]() {
        if (this[PropertySymbol.selectNode]) {
            this[PropertySymbol.selectNode][PropertySymbol.updateSelectedness]();
        }
        super[PropertySymbol.disconnectedFromNode]();
    }
}
//# sourceMappingURL=HTMLOptionElement.js.map