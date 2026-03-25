var _a;
import HTMLElement from '../html-element/HTMLElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import Element from '../element/Element.js';
import Node from '../node/Node.js';
import Event from '../../event/Event.js';
import NodeTypeEnum from '../node/NodeTypeEnum.js';
/**
 * HTML Slot Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLSlotElement.
 */
class HTMLSlotElement extends HTMLElement {
    // Internal properties
    [PropertySymbol.assignedNodes] = [];
    // Events
    onslotchange = null;
    /**
     * Returns name.
     *
     * @returns Name.
     */
    get name() {
        return this.getAttribute('name') || '';
    }
    /**
     * Sets name.
     *
     * @param name Name.
     */
    set name(name) {
        this.setAttribute('name', name);
    }
    /**
     * Sets the slot's manually assigned nodes to an ordered set of slottables.
     *
     * @param nodes Nodes.
     */
    assign(...nodes) {
        const shadowRoot = this.getRootNode();
        if (shadowRoot?.[PropertySymbol.slotAssignment] !== 'manual') {
            return;
        }
        const host = shadowRoot.host;
        for (const node of nodes) {
            if (node instanceof Node) {
                if (host[PropertySymbol.nodeArray].includes(node) &&
                    node[PropertySymbol.nodeType] !== NodeTypeEnum.commentNode) {
                    if (node[PropertySymbol.assignedToSlot]) {
                        const index = node[PropertySymbol.assignedToSlot][PropertySymbol.assignedNodes].indexOf(node);
                        if (index !== -1) {
                            node[PropertySymbol.assignedToSlot][PropertySymbol.assignedNodes].splice(index, 1);
                        }
                    }
                    node[PropertySymbol.assignedToSlot] = this;
                    this[PropertySymbol.assignedNodes].push(node);
                }
            }
        }
    }
    /**
     * Returns assigned nodes.
     *
     * @param [options] Options.
     * @param [options.flatten] A boolean value indicating whether to return the assigned nodes of any available child <slot> elements (true) or not (false). Defaults to false.
     * @returns Nodes.
     */
    assignedNodes(options) {
        return this.#assignedNodes(this.name, options);
    }
    /**
     * Returns assigned elements.
     *
     * @param [options] Options.
     * @param [options.flatten] A boolean value indicating whether to return the assigned elements of any available child <slot> elements (true) or not (false). Defaults to false.
     * @returns Nodes.
     */
    assignedElements(options) {
        return this.#assignedElements(this.name, options);
    }
    /**
     * @override
     */
    [PropertySymbol.cloneNode](deep = false) {
        return super[PropertySymbol.cloneNode](deep);
    }
    /**
     * @override
     */
    [PropertySymbol.onSetAttribute](attribute, replacedAttribute) {
        super[PropertySymbol.onSetAttribute](attribute, replacedAttribute);
        if (attribute[PropertySymbol.name] === 'name' &&
            attribute[PropertySymbol.value] !== replacedAttribute?.[PropertySymbol.value]) {
            const replacedAssignedNodes = this.#assignedNodes(replacedAttribute?.[PropertySymbol.value]);
            const assignedNodes = this.#assignedNodes(attribute.value);
            if (replacedAssignedNodes.length !== assignedNodes.length) {
                this.dispatchEvent(new Event('slotchange', { bubbles: true }));
            }
            else {
                for (let i = 0, max = assignedNodes.length; i < max; i++) {
                    if (replacedAssignedNodes[i] !== assignedNodes[i]) {
                        this.dispatchEvent(new Event('slotchange', { bubbles: true }));
                        break;
                    }
                }
            }
        }
    }
    /**
     * @override
     */
    [PropertySymbol.onRemoveAttribute](removedAttribute) {
        super[PropertySymbol.onRemoveAttribute](removedAttribute);
        if (removedAttribute[PropertySymbol.name] === 'name' &&
            removedAttribute[PropertySymbol.value] &&
            this.#assignedNodes(removedAttribute.value).length > 0) {
            this.dispatchEvent(new Event('slotchange', { bubbles: true }));
        }
    }
    /**
     * Returns assigned nodes.
     *
     * @param name Name.
     * @param [options] Options.
     * @param [options.flatten] A boolean value indicating whether to return the assigned nodes of any available child <slot> elements (true) or not (false). Defaults to false.
     * @returns Nodes.
     */
    #assignedNodes(name, options) {
        const shadowRoot = this.getRootNode();
        if (!shadowRoot?.host) {
            return [];
        }
        if (shadowRoot[PropertySymbol.slotAssignment] === 'manual') {
            return this[PropertySymbol.assignedNodes];
        }
        const host = shadowRoot.host;
        const flatten = !!options?.flatten;
        const assigned = [];
        for (const slotNode of host[PropertySymbol.nodeArray]) {
            const slotName = slotNode['slot'];
            if ((name && slotName && slotName === name) || (!name && !slotName)) {
                if (flatten && slotNode instanceof _a) {
                    for (const slotChild of slotNode.assignedNodes(options)) {
                        assigned.push(slotChild);
                    }
                }
                else {
                    assigned.push(slotNode);
                }
            }
        }
        return assigned;
    }
    /**
     * Returns assigned elements.
     *
     * @param name Name.
     * @param [options] Options.
     * @param [options.flatten] A boolean value indicating whether to return the assigned elements of any available child <slot> elements (true) or not (false). Defaults to false.
     * @returns Nodes.
     */
    #assignedElements(name, options) {
        const shadowRoot = this.getRootNode();
        if (!shadowRoot?.host) {
            return [];
        }
        if (shadowRoot[PropertySymbol.slotAssignment] === 'manual') {
            const elements = [];
            for (const node of this[PropertySymbol.assignedNodes]) {
                if (node instanceof Element) {
                    elements.push(node);
                }
            }
            return elements;
        }
        const host = shadowRoot.host;
        const flatten = !!options?.flatten;
        const assigned = [];
        for (const slotElement of host[PropertySymbol.elementArray]) {
            const slotName = slotElement.slot;
            if ((name && slotName === name) || (!name && !slotName)) {
                if (flatten && slotElement instanceof _a) {
                    for (const slotChild of slotElement.assignedElements(options)) {
                        assigned.push(slotChild);
                    }
                }
                else {
                    assigned.push(slotElement);
                }
            }
        }
        return assigned;
    }
}
_a = HTMLSlotElement;
export default HTMLSlotElement;
//# sourceMappingURL=HTMLSlotElement.js.map