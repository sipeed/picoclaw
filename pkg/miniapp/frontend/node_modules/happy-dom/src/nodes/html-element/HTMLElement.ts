import Element from '../element/Element.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import CSSStyleDeclaration from '../../css/declaration/CSSStyleDeclaration.js';
import PointerEvent from '../../event/events/PointerEvent.js';
import NodeTypeEnum from '../node/NodeTypeEnum.js';
import Event from '../../event/Event.js';
import HTMLElementUtility from './HTMLElementUtility.js';
import DOMStringMap from '../../dom/DOMStringMap.js';
import Attr from '../attr/Attr.js';

/**
 * HTML Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLElement.
 */
export default class HTMLElement extends Element {
	// Public properties
	public declare cloneNode: (deep?: boolean) => HTMLElement;
	public static observedAttributes?: string[];

	// Events
	public oncopy: ((event: Event) => void) | null = null;
	public oncut: ((event: Event) => void) | null = null;
	public onpaste: ((event: Event) => void) | null = null;
	public oninvalid: ((event: Event) => void) | null = null;
	public onanimationcancel: ((event: Event) => void) | null = null;
	public onanimationend: ((event: Event) => void) | null = null;
	public onanimationiteration: ((event: Event) => void) | null = null;
	public onanimationstart: ((event: Event) => void) | null = null;
	public onbeforeinput: ((event: Event) => void) | null = null;
	public oninput: ((event: Event) => void) | null = null;
	public onchange: ((event: Event) => void) | null = null;
	public ongotpointercapture: ((event: Event) => void) | null = null;
	public onlostpointercapture: ((event: Event) => void) | null = null;
	public onpointercancel: ((event: Event) => void) | null = null;
	public onpointerdown: ((event: Event) => void) | null = null;
	public onpointerenter: ((event: Event) => void) | null = null;
	public onpointerleave: ((event: Event) => void) | null = null;
	public onpointermove: ((event: Event) => void) | null = null;
	public onpointerout: ((event: Event) => void) | null = null;
	public onpointerover: ((event: Event) => void) | null = null;
	public onpointerup: ((event: Event) => void) | null = null;
	public ontransitioncancel: ((event: Event) => void) | null = null;
	public ontransitionend: ((event: Event) => void) | null = null;
	public ontransitionrun: ((event: Event) => void) | null = null;
	public ontransitionstart: ((event: Event) => void) | null = null;

	// Internal properties
	public [PropertySymbol.accessKey] = '';
	public [PropertySymbol.contentEditable] = 'inherit';
	public [PropertySymbol.isContentEditable] = false;
	public [PropertySymbol.offsetHeight] = 0;
	public [PropertySymbol.offsetWidth] = 0;
	public [PropertySymbol.offsetLeft] = 0;
	public [PropertySymbol.offsetTop] = 0;
	public [PropertySymbol.clientHeight] = 0;
	public [PropertySymbol.clientWidth] = 0;
	public [PropertySymbol.clientLeft] = 0;
	public [PropertySymbol.clientTop] = 0;
	public [PropertySymbol.style]: CSSStyleDeclaration = null;
	public [PropertySymbol.dataset]: DOMStringMap | null = null;

	// Private properties
	#customElementDefineCallback: () => void = null;

	/**
	 * Returns access key.
	 *
	 * @returns Access key.
	 */
	public get accessKey(): string {
		return this[PropertySymbol.accessKey];
	}

	/**
	 * Sets access key.
	 *
	 * @param accessKey Access key.
	 */
	public set accessKey(accessKey: string) {
		this[PropertySymbol.accessKey] = accessKey;
	}

	/**
	 * Returns content editable.
	 *
	 * @returns Content editable.
	 */
	public get contentEditable(): string {
		return this[PropertySymbol.contentEditable];
	}

	/**
	 * Sets content editable.
	 *
	 * @param contentEditable Content editable.
	 */
	public set contentEditable(contentEditable: string) {
		this[PropertySymbol.contentEditable] = contentEditable;
	}

	/**
	 * Returns is content editable.
	 *
	 * @returns Is content editable.
	 */
	public get isContentEditable(): boolean {
		return this[PropertySymbol.isContentEditable];
	}

	/**
	 * Returns offset height.
	 *
	 * @returns Offset height.
	 */
	public get offsetHeight(): number {
		return this[PropertySymbol.offsetHeight];
	}

	/**
	 * Returns offset width.
	 *
	 * @returns Offset width.
	 */
	public get offsetWidth(): number {
		return this[PropertySymbol.offsetWidth];
	}

	/**
	 * Returns offset left.
	 *
	 * @returns Offset left.
	 */
	public get offsetLeft(): number {
		return this[PropertySymbol.offsetLeft];
	}

	/**
	 * Returns offset top.
	 *
	 * @returns Offset top.
	 */
	public get offsetTop(): number {
		return this[PropertySymbol.offsetTop];
	}

	/**
	 * Returns client height.
	 *
	 * @returns Client height.
	 */
	public get clientHeight(): number {
		return this[PropertySymbol.clientHeight];
	}

	/**
	 * Returns client width.
	 *
	 * @returns Client width.
	 */
	public get clientWidth(): number {
		return this[PropertySymbol.clientWidth];
	}

	/**
	 * Returns client left.
	 *
	 * @returns Client left.
	 */
	public get clientLeft(): number {
		return this[PropertySymbol.clientLeft];
	}

	/**
	 * Returns client top.
	 *
	 * @returns Client top.
	 */
	public get clientTop(): number {
		return this[PropertySymbol.clientTop];
	}

	/**
	 * Returns tab index.
	 *
	 * @returns Tab index.
	 */
	public get tabIndex(): number {
		const tabIndex = this.getAttribute('tabindex');
		return tabIndex !== null ? Number(tabIndex) : -1;
	}

	/**
	 * Returns tab index.
	 *
	 * @param tabIndex Tab index.
	 */
	public set tabIndex(tabIndex: number) {
		if (tabIndex === -1) {
			this.removeAttribute('tabindex');
		} else {
			this.setAttribute('tabindex', String(tabIndex));
		}
	}

	/**
	 * Returns inner text, which is the rendered appearance of text.
	 *
	 * @see https://html.spec.whatwg.org/multipage/dom.html#the-innertext-idl-attribute
	 * @returns Inner text.
	 */
	public get innerText(): string {
		if (!this[PropertySymbol.isConnected]) {
			return this.textContent;
		}

		let result = '';

		for (const childNode of this[PropertySymbol.nodeArray]) {
			if (childNode[PropertySymbol.nodeType] === NodeTypeEnum.elementNode) {
				const childElement = <HTMLElement>childNode;
				const computedStyle = this[PropertySymbol.window].getComputedStyle(childElement);

				if (
					childElement[PropertySymbol.tagName] !== 'SCRIPT' &&
					childElement[PropertySymbol.tagName] !== 'STYLE'
				) {
					const display = computedStyle.display;
					if (display !== 'none') {
						const textTransform = computedStyle.textTransform;

						if ((display === 'block' || display === 'flex') && result) {
							result += '\n';
						}

						let text = childElement.innerText;

						switch (textTransform) {
							case 'uppercase':
								text = text.toUpperCase();
								break;
							case 'lowercase':
								text = text.toLowerCase();
								break;
							case 'capitalize':
								text = text.replace(/(^|\s)\S/g, (l) => l.toUpperCase());
								break;
						}

						result += text;
					}
				}
			} else if (childNode[PropertySymbol.nodeType] === NodeTypeEnum.textNode) {
				result += childNode.textContent.replace(/[\n\r]/, '');
			}
		}

		return result;
	}

	/**
	 * Sets the inner text, which is the rendered appearance of text.
	 *
	 * @see https://html.spec.whatwg.org/multipage/dom.html#the-innertext-idl-attribute
	 * @param innerText Inner text.
	 */
	public set innerText(text: string) {
		const childNodes = this[PropertySymbol.nodeArray];

		while (childNodes.length) {
			this.removeChild(childNodes[0]);
		}

		const texts = text.split(/[\n\r]/);
		const ownerDocument = this[PropertySymbol.ownerDocument];

		for (let i = 0, max = texts.length; i < max; i++) {
			if (i !== 0) {
				this.appendChild(ownerDocument.createElement('br'));
			}
			this.appendChild(ownerDocument.createTextNode(texts[i]));
		}
	}

	/**
	 * Returns outer text.
	 *
	 * @see https://html.spec.whatwg.org/multipage/dom.html#the-innertext-idl-attribute
	 * @returns HTML.
	 */
	public get outerText(): string {
		return this.innerText;
	}

	/**
	 * Sets outer text.
	 *
	 * @see https://html.spec.whatwg.org/multipage/dom.html#the-innertext-idl-attribute
	 * @param text Text.
	 */
	public set outerText(text: string) {
		if (!this[PropertySymbol.parentNode]) {
			throw new this[PropertySymbol.window].DOMException(
				"Failed to set the 'outerHTML' property on 'Element': This element has no parent node."
			);
		}

		const texts = text.split(/[\n\r]/);

		for (let i = 0, max = texts.length; i < max; i++) {
			if (i !== 0) {
				this[PropertySymbol.parentNode].insertBefore(
					this[PropertySymbol.ownerDocument].createElement('br'),
					this
				);
			}
			this[PropertySymbol.parentNode].insertBefore(
				this[PropertySymbol.ownerDocument].createTextNode(texts[i]),
				this
			);
		}

		this[PropertySymbol.parentNode].removeChild(this);
	}

	/**
	 * Returns style.
	 *
	 * @returns Style.
	 */
	public get style(): CSSStyleDeclaration {
		if (!this[PropertySymbol.style]) {
			this[PropertySymbol.style] = new CSSStyleDeclaration(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{ element: this }
			);
		}
		return this[PropertySymbol.style];
	}

	/**
	 * Sets style.
	 *
	 * @param cssText Style as text.
	 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLElement/style#setting_styles
	 */
	public set style(cssText: string | CSSStyleDeclaration | null) {
		this.style.cssText = typeof cssText === 'string' ? <string>cssText : '';
	}

	/**
	 * Returns data set.
	 *
	 * @returns Data set.
	 */
	public get dataset(): DOMStringMap {
		return (this[PropertySymbol.dataset] ??= new DOMStringMap(
			PropertySymbol.illegalConstructor,
			this
		));
	}

	/**
	 * Returns direction.
	 *
	 * @returns Direction.
	 */
	public get dir(): string {
		return this.getAttribute('dir') || '';
	}

	/**
	 * Returns direction.
	 *
	 * @param direction Direction.
	 */
	public set dir(direction: string) {
		this.setAttribute('dir', direction);
	}

	/**
	 * Returns hidden.
	 *
	 * @returns Hidden.
	 */
	public get hidden(): boolean {
		return this.getAttribute('hidden') !== null;
	}

	/**
	 * Returns hidden.
	 *
	 * @param hidden Hidden.
	 */
	public set hidden(hidden: boolean) {
		if (!hidden) {
			this.removeAttribute('hidden');
		} else {
			this.setAttribute('hidden', '');
		}
	}

	/**
	 * Returns inert.
	 *
	 * @returns Inert.
	 */
	public get inert(): boolean {
		return this.getAttribute('inert') !== null;
	}

	/**
	 * Returns inert.
	 *
	 * @param inert Inert.
	 */
	public set inert(inert: boolean) {
		if (!inert) {
			this.removeAttribute('inert');
		} else {
			this.setAttribute('inert', '');
		}
	}

	/**
	 * Returns language.
	 *
	 * @returns Language.
	 */
	public get lang(): string {
		return this.getAttribute('lang') || '';
	}

	/**
	 * Returns language.
	 *
	 * @param language Language.
	 */
	public set lang(lang: string) {
		this.setAttribute('lang', lang);
	}

	/**
	 * Returns title.
	 *
	 * @returns Title.
	 */
	public get title(): string {
		return this.getAttribute('title') || '';
	}

	/**
	 * Returns title.
	 *
	 * @param title Title.
	 */
	public set title(title: string) {
		this.setAttribute('title', title);
	}

	/**
	 * Returns popover.
	 *
	 * @returns Popover.
	 */
	public get popover(): string | null {
		const value = this.getAttribute('popover');
		switch (value) {
			case null:
				return null;
			case '':
			case 'auto':
				return 'auto';
			case 'manual':
				return 'manual';
			default:
				return 'manual';
		}
	}

	/**
	 * Sets popover.
	 *
	 * @param value Value.
	 */
	public set popover(value: string | null) {
		if (value === null) {
			this.removeAttribute('popover');
			return;
		}
		this.setAttribute('popover', value);
	}

	/**
	 * Connected callback.
	 */
	public connectedCallback?(): void;

	/**
	 * Disconnected callback.
	 */
	public disconnectedCallback?(): void;

	/**
	 * Attribute changed callback.
	 *
	 * @param name Name.
	 * @param oldValue Old value.
	 * @param newValue New value.
	 */
	public attributeChangedCallback?(name: string, oldValue: string, newValue: string): void;

	/**
	 * Triggers a click event.
	 */
	public click(): void {
		this.dispatchEvent(
			new PointerEvent('click', {
				bubbles: true,
				composed: true,
				cancelable: true
			})
		);
	}

	/**
	 * Triggers a blur event.
	 */
	public blur(): void {
		HTMLElementUtility.blur(this);
	}

	/**
	 * Triggers a focus event.
	 */
	public focus(): void {
		HTMLElementUtility.focus(this);
	}

	/**
	 * @override
	 */
	public override [PropertySymbol.cloneNode](deep = false): HTMLElement {
		const clone = <HTMLElement>super[PropertySymbol.cloneNode](deep);

		clone[PropertySymbol.accessKey] = this[PropertySymbol.accessKey];
		clone[PropertySymbol.contentEditable] = this[PropertySymbol.contentEditable];
		clone[PropertySymbol.isContentEditable] = this[PropertySymbol.isContentEditable];

		return clone;
	}

	/**
	 * @override
	 * @see https://html.spec.whatwg.org/multipage/dom.html#htmlelement
	 */
	public override [PropertySymbol.connectedToNode](): void {
		const window = this[PropertySymbol.window];
		const localName = this[PropertySymbol.localName];
		const allCallbacks = window.customElements[PropertySymbol.callbacks];

		// This element can potentially be a custom element that has not been defined yet
		// Therefore we need to register a callback for when it is defined in CustomElementRegistry and replace it with the registered element (see #404)
		if (this.constructor === window.HTMLElement && localName.includes('-') && allCallbacks) {
			if (!this.#customElementDefineCallback) {
				const callback = this.#onCustomElementConnected.bind(this);
				const callbacks = allCallbacks.get(localName);
				if (callbacks) {
					callbacks.unshift(callback);
				} else {
					allCallbacks.set(localName, [callback]);
				}
				this.#customElementDefineCallback = callback;
			}
		}

		super[PropertySymbol.connectedToNode]();
	}

	/**
	 * @override
	 */
	public override [PropertySymbol.disconnectedFromNode](): void {
		const window = this[PropertySymbol.window];
		const localName = this[PropertySymbol.localName];
		const allCallbacks = window.customElements[PropertySymbol.callbacks];

		// This element can potentially be a custom element that has not been defined yet
		// Therefore we need to register a callback for when it is defined in CustomElementRegistry and replace it with the registered element (see #404)
		if (this.constructor === window.HTMLElement && localName.includes('-') && allCallbacks) {
			const callbacks = allCallbacks.get(localName);

			if (callbacks && this.#customElementDefineCallback) {
				const index = callbacks.indexOf(this.#customElementDefineCallback);
				if (index !== -1) {
					callbacks.splice(index, 1);
				}
				if (!callbacks.length) {
					allCallbacks.delete(localName);
				}
				this.#customElementDefineCallback = null;
			}
		}

		super[PropertySymbol.disconnectedFromNode]();
	}

	/**
	 * @override
	 */
	public override [PropertySymbol.connectedToDocument](): void {
		super[PropertySymbol.connectedToDocument]();

		this[PropertySymbol.window][PropertySymbol.customElementReactionStack].enqueueReaction(
			this,
			'connectedCallback'
		);
	}

	/**
	 * @override
	 */
	public override [PropertySymbol.disconnectedFromDocument](): void {
		super[PropertySymbol.disconnectedFromDocument]();

		this[PropertySymbol.window][PropertySymbol.customElementReactionStack].enqueueReaction(
			this,
			'disconnectedCallback'
		);
	}

	/**
	 * @override
	 */
	public override [PropertySymbol.onSetAttribute](
		attribute: Attr,
		replacedAttribute: Attr | null
	): void {
		super[PropertySymbol.onSetAttribute](attribute, replacedAttribute);

		this[PropertySymbol.window][PropertySymbol.customElementReactionStack].enqueueReaction(
			this,
			'attributeChangedCallback',
			[attribute.name, replacedAttribute?.value ?? null, attribute.value]
		);
	}

	/**
	 * @override
	 */
	public override [PropertySymbol.onRemoveAttribute](removedAttribute: Attr): void {
		super[PropertySymbol.onRemoveAttribute](removedAttribute);

		this[PropertySymbol.window][PropertySymbol.customElementReactionStack].enqueueReaction(
			this,
			'attributeChangedCallback',
			[removedAttribute.name, removedAttribute.value, null]
		);
	}

	/**
	 * Triggered when a custom element is connected to the DOM.
	 */
	#onCustomElementConnected(): void {
		if (!this[PropertySymbol.parentNode]) {
			return;
		}

		const window = this[PropertySymbol.window];
		const localName = this[PropertySymbol.localName];
		const newElement = <HTMLElement>this[PropertySymbol.ownerDocument].createElement(localName);
		const newCache = newElement[PropertySymbol.cache];

		newElement[PropertySymbol.nodeArray] = this[PropertySymbol.nodeArray];
		newElement[PropertySymbol.elementArray] = this[PropertySymbol.elementArray];
		newElement[PropertySymbol.childNodes] = null;
		newElement[PropertySymbol.children] = null;
		newElement[PropertySymbol.isConnected] = this[PropertySymbol.isConnected];
		newElement[PropertySymbol.rootNode] = this[PropertySymbol.rootNode];
		newElement[PropertySymbol.formNode] = this[PropertySymbol.formNode];
		newElement[PropertySymbol.parentNode] = this[PropertySymbol.parentNode];
		newElement[PropertySymbol.selectNode] = this[PropertySymbol.selectNode];
		newElement[PropertySymbol.textAreaNode] = this[PropertySymbol.textAreaNode];
		newElement[PropertySymbol.mutationListeners] = this[PropertySymbol.mutationListeners];
		newElement[PropertySymbol.isValue] = this[PropertySymbol.isValue];
		newElement[PropertySymbol.cache] = this[PropertySymbol.cache];
		newElement[PropertySymbol.affectsCache] = this[PropertySymbol.affectsCache];
		newElement[PropertySymbol.attributes][PropertySymbol.namedItems] =
			this[PropertySymbol.attributes][PropertySymbol.namedItems];
		newElement[PropertySymbol.attributes][PropertySymbol.namespaceItems] =
			this[PropertySymbol.attributes][PropertySymbol.namespaceItems];

		for (const attr of newElement[PropertySymbol.attributes][
			PropertySymbol.namespaceItems
		].values()) {
			attr[PropertySymbol.ownerElement] = newElement;
		}

		this[PropertySymbol.clearCache]();

		this[PropertySymbol.nodeArray] = [];
		this[PropertySymbol.elementArray] = [];
		this[PropertySymbol.childNodes] = null;
		this[PropertySymbol.children] = null;

		this[PropertySymbol.parentNode] = null;
		this[PropertySymbol.rootNode] = null;
		this[PropertySymbol.formNode] = null;
		this[PropertySymbol.selectNode] = null;
		this[PropertySymbol.textAreaNode] = null;

		this[PropertySymbol.mutationListeners] = [];
		this[PropertySymbol.isValue] = null;
		this[PropertySymbol.cache] = newCache;
		this[PropertySymbol.affectsCache] = [];
		this[PropertySymbol.attributes][PropertySymbol.namedItems] = new Map();
		this[PropertySymbol.attributes][PropertySymbol.namespaceItems] = new Map();

		for (const node of newElement[PropertySymbol.nodeArray]) {
			node[PropertySymbol.parentNode] = newElement;
		}

		const parentChildNodes = newElement[PropertySymbol.parentNode][PropertySymbol.nodeArray];
		const parentChildElements = newElement[PropertySymbol.parentNode][PropertySymbol.elementArray];

		parentChildNodes[parentChildNodes.indexOf(this)] = newElement;
		parentChildElements[parentChildElements.indexOf(this)] = newElement;

		const allCallbacks = window.customElements[PropertySymbol.callbacks];
		const callbacks = allCallbacks.get(localName);

		if (callbacks && this.#customElementDefineCallback) {
			const index = callbacks.indexOf(this.#customElementDefineCallback);

			if (index !== -1) {
				callbacks.splice(index, 1);
			}

			if (!callbacks.length) {
				allCallbacks.delete(localName);
			}

			this.#customElementDefineCallback = null;
		}

		if (newElement[PropertySymbol.isConnected]) {
			newElement[PropertySymbol.connectedToDocument]();
		}
	}
}
