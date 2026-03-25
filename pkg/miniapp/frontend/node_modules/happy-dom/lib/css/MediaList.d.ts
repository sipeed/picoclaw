/**
 * MediaList interface.
 */
export default class MediaList {
    readonly length = 0;
    /**
     * Media text.
     *
     * @returns Media text.
     */
    get mediaText(): string;
    /**
     * Returns item.
     *
     * @param index Index.
     * @returns Item.
     */
    item(index: number): string;
    /**
     * Appends a medium.
     *
     * @param medium Medium.
     */
    appendMedium(medium: string): void;
    /**
     * Deletes a medium.
     *
     * @param medium Medium.
     */
    deleteMedium(medium: string): void;
}
//# sourceMappingURL=MediaList.d.ts.map