import * as PropertySymbol from '../PropertySymbol.cjs';
import HTMLElement from '../nodes/html-element/HTMLElement.cjs';
import Node from '../nodes/node/Node.cjs';
import BrowserWindow from '../window/BrowserWindow.cjs';
import ICustomElementDefinition from './ICustomElementDefinition.cjs';
/**
 * Custom elements registry.
 */
export default class CustomElementRegistry {
    #private;
    [PropertySymbol.registry]: Map<string, ICustomElementDefinition>;
    [PropertySymbol.classRegistry]: Map<typeof HTMLElement, string>;
    [PropertySymbol.callbacks]: Map<string, Array<() => void>>;
    [PropertySymbol.destroyed]: boolean;
    /**
     * Constructor.
     *
     * @param window Window.
     */
    constructor(window: BrowserWindow);
    /**
     * Defines a custom element class.
     *
     * @param name Tag name of element.
     * @param elementClass Element class.
     * @param [options] Options.
     * @param [options.extends] Extends tag name.
     */
    define(name: string, elementClass: typeof HTMLElement, options?: {
        extends?: string;
    }): void;
    /**
     * Returns a defined element class.
     *
     * @param name Tag name of element.
     * @returns HTMLElement Class defined or undefined.
     */
    get(name: string): typeof HTMLElement | undefined;
    /**
     * Upgrades a custom element directly, even before it is connected to its shadow root.
     *
     * Not implemented yet.
     *
     * @param _root Root node.
     */
    upgrade(_root: Node): void;
    /**
     * When defined.
     *
     * @param name Tag name of element.
     */
    whenDefined(name: string): Promise<void>;
    /**
     * Reverse lookup searching for name by given element class.
     *
     * @param elementClass Class constructor.
     * @returns Found tag name or `null`.
     */
    getName(elementClass: typeof HTMLElement): string | null;
    /**
     * Destroys the registry.
     */
    [PropertySymbol.destroy](): void;
}
//# sourceMappingURL=CustomElementRegistry.d.ts.map