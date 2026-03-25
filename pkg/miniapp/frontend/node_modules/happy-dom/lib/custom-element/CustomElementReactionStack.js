import * as PropertySymbol from '../PropertySymbol.js';
import WindowBrowserContext from '../window/WindowBrowserContext.js';
/**
 * Custom element reaction stack.
 *
 * @see https://html.spec.whatwg.org/multipage/custom-elements.html#custom-element-reactions-stack
 */
export default class CustomElementReactionStack {
    window;
    /**
     * Constructor.
     *
     * @param window Window.
     */
    constructor(window) {
        this.window = window;
    }
    /**
     * Enqueues a custom element reaction.
     *
     * @see https://html.spec.whatwg.org/multipage/custom-elements.html#enqueue-a-custom-element-callback-reaction
     * @see https://html.spec.whatwg.org/multipage/custom-elements.html#enqueue-an-element-on-the-appropriate-element-queue
     * @param element Element.
     * @param callbackName Callback name.
     * @param [args] Arguments.
     */
    enqueueReaction(element, callbackName, args) {
        // If a polyfill is used, [PropertySymbol.registry] may be undefined
        const definition = this.window.customElements[PropertySymbol.registry]?.get(element.localName);
        if (!definition) {
            return;
        }
        // If the element is not connected to the main document, we should not invoke the callback.
        if (element[PropertySymbol.ownerDocument] !== this.window.document) {
            return;
        }
        // According to the spec, we should use a queue for each element and then invoke the reactions in the order they were enqueued asynchronously.
        // However, the browser seem to always invoke the reactions synchronously.
        // TODO: Can we find an example where the reactions are invoked asynchronously? In that case we should use a queue for those cases.
        switch (callbackName) {
            case 'connectedCallback':
                if (definition.livecycleCallbacks.connectedCallback) {
                    const returnValue = definition.livecycleCallbacks.connectedCallback.call(element);
                    /**
                     * It is common to import dependencies in the connectedCallback() method of web components.
                     * As Happy DOM doesn't have support for dynamic imports yet, this is a temporary solution to wait for imports in connectedCallback().
                     *
                     * @see https://github.com/capricorn86/happy-dom/issues/1442
                     */
                    if (returnValue instanceof Promise) {
                        const asyncTaskManager = new WindowBrowserContext(this.window).getAsyncTaskManager();
                        if (asyncTaskManager) {
                            const taskID = asyncTaskManager.startTask();
                            returnValue
                                .then(() => asyncTaskManager.endTask(taskID))
                                .catch(() => asyncTaskManager.endTask(taskID));
                        }
                    }
                }
                break;
            case 'disconnectedCallback':
                if (definition.livecycleCallbacks.disconnectedCallback) {
                    definition.livecycleCallbacks.disconnectedCallback.call(element);
                }
                break;
            case 'attributeChangedCallback':
                if (definition.livecycleCallbacks.attributeChangedCallback &&
                    definition.observedAttributes.has(args[0])) {
                    definition.livecycleCallbacks.attributeChangedCallback.apply(element, args);
                }
                break;
        }
    }
}
//# sourceMappingURL=CustomElementReactionStack.js.map