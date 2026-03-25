import MimeType from './MimeType.cjs';
/**
 * MimeTypeArray.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/MimeTypeArray.
 */
export default class MimeTypeArray {
    [n: number]: MimeType;
    readonly length: number;
    /**
     * Constructor.
     *
     * @param mimeTypes
     */
    constructor(mimeTypes: MimeType[]);
    /**
     * @param index
     */
    item(index: number): MimeType;
    /**
     * @param name
     */
    namedItem(name: string): MimeType;
    /**
     * Returns the object as a string.
     *
     * @returns String.
     */
    toString(): string;
}
//# sourceMappingURL=MimeTypeArray.d.ts.map