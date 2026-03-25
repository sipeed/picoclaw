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
const HTMLElement_js_1 = __importDefault(require("../html-element/HTMLElement.cjs"));
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
const EventPhaseEnum_js_1 = __importDefault(require("../../event/EventPhaseEnum.cjs"));
const MouseEvent_js_1 = __importDefault(require("../../event/events/MouseEvent.cjs"));
/**
 * HTML Label Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLLabelElement.
 */
class HTMLLabelElement extends HTMLElement_js_1.default {
    /**
     * Returns a string containing the ID of the labeled control. This reflects the "for" attribute.
     *
     * @returns ID of the labeled control.
     */
    get htmlFor() {
        const htmlFor = this.getAttribute('for');
        if (htmlFor !== null) {
            return htmlFor;
        }
        return htmlFor !== null ? htmlFor : '';
    }
    /**
     * Sets a string containing the ID of the labeled control. This reflects the "for" attribute.
     *
     * @param htmlFor ID of the labeled control.
     */
    set htmlFor(htmlFor) {
        this.setAttribute('for', htmlFor);
    }
    /**
     * Returns an HTML element representing the control with which the label is associated.
     *
     * @returns Control element.
     */
    get control() {
        const htmlFor = this.getAttribute('for');
        if (htmlFor !== null) {
            if (!htmlFor || !this[PropertySymbol.isConnected]) {
                return null;
            }
            const control = (this[PropertySymbol.rootNode].getElementById(htmlFor));
            if (control) {
                switch (control[PropertySymbol.tagName]) {
                    case 'INPUT':
                        return control.type !== 'hidden' ? control : null;
                    case 'BUTTON':
                    case 'METER':
                    case 'OUTPUT':
                    case 'PROGRESS':
                    case 'SELECT':
                    case 'TEXTAREA':
                        return control;
                    default:
                        return null;
                }
            }
        }
        return (this.querySelector('button,input:not([type="hidden"]),meter,output,progress,select,textarea'));
    }
    /**
     * Returns the parent form element.
     *
     * @returns Form.
     */
    get form() {
        return this.control?.form || null;
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
    dispatchEvent(event) {
        const returnValue = super.dispatchEvent(event);
        if (!event[PropertySymbol.defaultPrevented] &&
            event.type === 'click' &&
            event.eventPhase === EventPhaseEnum_js_1.default.none &&
            event instanceof MouseEvent_js_1.default) {
            const control = this.control;
            if (control && event.target !== control) {
                control.dispatchEvent(new MouseEvent_js_1.default('click', { bubbles: true, cancelable: true }));
            }
        }
        return returnValue;
    }
}
exports.default = HTMLLabelElement;
//# sourceMappingURL=HTMLLabelElement.cjs.map