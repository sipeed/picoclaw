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
Object.defineProperty(exports, "__esModule", { value: true });
const Event_js_1 = __importDefault(require("../../event/Event.cjs"));
const HTMLElement_js_1 = __importDefault(require("../html-element/HTMLElement.cjs"));
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
const EventPhaseEnum_js_1 = __importDefault(require("../../event/EventPhaseEnum.cjs"));
const MouseEvent_js_1 = __importDefault(require("../../event/events/MouseEvent.cjs"));
/**
 * HTMLDetailsElement
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLDetailsElement
 */
class HTMLDetailsElement extends HTMLElement_js_1.default {
    // Events
    ontoggle = null;
    /**
     * Returns the open attribute.
     */
    get open() {
        return this.getAttribute('open') !== null;
    }
    /**
     * Sets the open attribute.
     *
     * @param open New value.
     */
    set open(open) {
        if (open) {
            this.setAttribute('open', '');
        }
        else {
            this.removeAttribute('open');
        }
    }
    /**
     * @override
     */
    [PropertySymbol.onSetAttribute](attribute, replacedAttribute) {
        super[PropertySymbol.onSetAttribute](attribute, replacedAttribute);
        if (attribute[PropertySymbol.name] === 'open') {
            if (attribute[PropertySymbol.value] !== replacedAttribute?.[PropertySymbol.value]) {
                this.dispatchEvent(new Event_js_1.default('toggle'));
            }
        }
    }
    /**
     * @override
     */
    [PropertySymbol.onRemoveAttribute](removedAttribute) {
        super[PropertySymbol.onRemoveAttribute](removedAttribute);
        if (removedAttribute && removedAttribute[PropertySymbol.name] === 'open') {
            this.dispatchEvent(new Event_js_1.default('toggle'));
        }
    }
    /**
     * @override
     */
    dispatchEvent(event) {
        const returnValue = super.dispatchEvent(event);
        if (!event[PropertySymbol.defaultPrevented] &&
            event[PropertySymbol.target]?.[PropertySymbol.localName] === 'summary' &&
            event.type === 'click' &&
            event.eventPhase === EventPhaseEnum_js_1.default.bubbling &&
            event instanceof MouseEvent_js_1.default) {
            this.open = !this.open;
        }
        return returnValue;
    }
}
exports.default = HTMLDetailsElement;
//# sourceMappingURL=HTMLDetailsElement.cjs.map