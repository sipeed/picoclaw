import HTMLElement from '../nodes/html-element/HTMLElement.cjs';
export default interface ICustomElementDefinition {
    elementClass: typeof HTMLElement;
    extends: string;
    observedAttributes: Set<string>;
    livecycleCallbacks: {
        connectedCallback: () => void;
        disconnectedCallback: () => void;
        attributeChangedCallback: (name: string, oldValue: string | null, newValue: string | null) => void;
    };
}
//# sourceMappingURL=ICustomElementDefinition.d.ts.map