import File from '../file/File.js';
import DataTransferItemList from './DataTransferItemList.js';
/**
 * DataTransfer.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/DataTransfer
 */
export default class DataTransfer {
    dropEffect: 'none' | 'copy' | 'link' | 'move';
    effectAllowed: 'none' | 'copy' | 'copyLink' | 'copyMove' | 'link' | 'linkMove' | 'move' | 'all' | 'uninitialized';
    readonly items: DataTransferItemList;
    /**
     * Returns files.
     *
     * @returns Files.
     */
    get files(): File[];
    /**
     * Returns types.
     *
     * @returns Types.
     */
    get types(): string[];
    /**
     * Clears the data.
     */
    clearData(): void;
    /**
     * Sets the data.
     *
     * @param format Format.
     * @param data Data.
     */
    setData(format: string, data: string): void;
    /**
     * Gets the data.
     *
     * @param format Format.
     * @returns Data.
     */
    getData(format: string): string;
    /**
     * Sets drag image.
     *
     * TODO: Implement.
     */
    setDragImage(): void;
}
//# sourceMappingURL=DataTransfer.d.ts.map