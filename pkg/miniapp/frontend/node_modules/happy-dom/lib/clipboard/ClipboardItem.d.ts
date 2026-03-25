import Blob from '../file/Blob.js';
/**
 * Clipboard Item API.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/ClipboardItem.
 */
export default class ClipboardItem {
    #private;
    readonly presentationStyle: 'unspecified' | 'inline' | 'attachment';
    /**
     * Constructor.
     *
     * @param data Data.
     * @param [options] Options.
     * @param [options.presentationStyle] Presentation style.
     */
    constructor(data: {
        [mimeType: string]: Blob | string | Promise<Blob | string>;
    }, options?: {
        presentationStyle?: 'unspecified' | 'inline' | 'attachment';
    });
    /**
     * Returns types.
     *
     * @returns Types.
     */
    get types(): string[];
    /**
     * Returns data by type.
     *
     * @param type Type.
     * @returns Data.
     */
    getType(type: string): Promise<Blob>;
}
//# sourceMappingURL=ClipboardItem.d.ts.map