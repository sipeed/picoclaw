"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
/**
 * MediaList interface.
 */
class MediaList {
    length = 0;
    /**
     * Media text.
     *
     * @returns Media text.
     */
    get mediaText() {
        const mediaText = [];
        for (let i = 0; i < this.length; i++) {
            mediaText.push(this[i]);
        }
        return mediaText.join(', ');
    }
    /**
     * Returns item.
     *
     * @param index Index.
     * @returns Item.
     */
    item(index) {
        return this[index] || '';
    }
    /**
     * Appends a medium.
     *
     * @param medium Medium.
     */
    appendMedium(medium) {
        this[this.length] = medium;
        this.length++;
    }
    /**
     * Deletes a medium.
     *
     * @param medium Medium.
     */
    deleteMedium(medium) {
        let isDeleted = false;
        for (let i = 0; i < this.length; i++) {
            if (isDeleted) {
                this[i - 1] = this[i];
            }
            if (this[i] === medium) {
                isDeleted = true;
            }
        }
        if (isDeleted) {
            this.length--;
        }
    }
}
exports.default = MediaList;
//# sourceMappingURL=MediaList.cjs.map