import ICSSStyleDeclarationPropertyValue from './ICSSStyleDeclarationPropertyValue.js';
/**
 * Computed this.properties property parser.
 */
export default class CSSStyleDeclarationPropertyManager {
    properties: {
        [k: string]: ICSSStyleDeclarationPropertyValue;
    };
    private definedPropertyNames;
    /**
     * Class construtor.
     *
     * @param [options] Options.
     * @param [options.cssText] CSS string.
     */
    constructor(options?: {
        cssText?: string;
    });
    /**
     * Returns property value.
     *
     * @param name Property name.
     * @returns Property value.
     */
    get(name: string): ICSSStyleDeclarationPropertyValue | null;
    /**
     * Removes a property.
     *
     * @param name Property name.
     */
    remove(name: string): void;
    /**
     * Sets a property
     *
     * @param name Name.
     * @param value Value.
     * @param important Important.
     */
    set(name: string, value: string, important: boolean): void;
    /**
     * Returns a clone.
     *
     * @returns Clone.
     */
    clone(): CSSStyleDeclarationPropertyManager;
    /**
     * Returns size.
     *
     * @returns Size.
     */
    size(): number;
    /**
     * Returns property name.
     *
     * @param index Index.
     * @returns Property name.
     */
    item(index: number): string;
    /**
     * Converts properties to string.
     *
     * @returns String.
     */
    toString(): string;
}
//# sourceMappingURL=CSSStyleDeclarationPropertyManager.d.ts.map