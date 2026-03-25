import File from '../file/File.cjs';
/**
 * Data transfer item.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/DataTransferItem.
 */
export default class DataTransferItem {
    #private;
    readonly kind: 'string' | 'file';
    readonly type: string;
    /**
     * Constructor.
     *
     * @param item Item.
     * @param type Type.
     */
    constructor(item: string | File, type?: string);
    /**
     * Returns file.
     */
    getAsFile(): File;
    /**
     * Returns string.
     *
     * @param callback Callback.
     */
    getAsString(callback: (text: string) => void): void;
}
//# sourceMappingURL=DataTransferItem.d.ts.map