"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
var _a;
Object.defineProperty(exports, "__esModule", { value: true });
const HTMLElement_js_1 = __importDefault(require("../html-element/HTMLElement.cjs"));
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
const Element_js_1 = __importDefault(require("../element/Element.cjs"));
const Node_js_1 = __importDefault(require("../node/Node.cjs"));
const Event_js_1 = __importDefault(require("../../event/Event.cjs"));
const NodeTypeEnum_js_1 = __importDefault(require("../node/NodeTypeEnum.cjs"));
/**
 * HTML Slot Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLSlotElement.
 */
class HTMLSlotElement extends HTMLElement_js_1.default {
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
            if (node instanceof Node_js_1.default) {
                if (host[PropertySymbol.nodeArray].includes(node) &&
                    node[PropertySymbol.nodeType] !== NodeTypeEnum_js_1.default.commentNode) {
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
                this.dispatchEvent(new Event_js_1.default('slotchange', { bubbles: true }));
            }
            else {
                for (let i = 0, max = assignedNodes.length; i < max; i++) {
                    if (replacedAssignedNodes[i] !== assignedNodes[i]) {
                        this.dispatchEvent(new Event_js_1.default('slotchange', { bubbles: true }));
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
            this.dispatchEvent(new Event_js_1.default('slotchange', { bubbles: true }));
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
                if (node instanceof Element_js_1.default) {
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
exports.default = HTMLSlotElement;
//# sourceMappingURL=HTMLSlotElement.cjs.map