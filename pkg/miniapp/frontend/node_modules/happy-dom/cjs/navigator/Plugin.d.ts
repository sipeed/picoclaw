import MimeType from './MimeType.cjs';
/**
 * Plugin.
 */
export default class Plugin {
    [n: number]: MimeType;
    readonly length: number;
    readonly description: string;
    readonly filename: string;
    readonly name: string;
    /**
     * Constructor.
     *
     * @param mimeTypes Mime types.
     * @param description Description.
     * @param filename Filename.
     * @param name Name.
     */
    constructor(mimeTypes: MimeType[], description: string, filename: string, name: string);
    /**
     * Item.
     *
     * @param index Number.
     * @returns IMimeType.
     */
    item(index: number): MimeType;
    /**
     * NamedItem.
     *
     * @param name String.
     * @returns IMimeType.
     */
    namedItem(name: string): MimeType;
    /**
     * Returns the object as a string.
     *
     * @returns String.
     */
    toString(): string;
}
//# sourceMappingURL=Plugin.d.ts.map