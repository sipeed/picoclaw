/**
 * MediaList interface.
 */
export default class MediaList {
	public readonly length = 0;

	/**
	 * Media text.
	 *
	 * @returns Media text.
	 */
	public get mediaText(): string {
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
	public item(index: number): string {
		return this[index] || '';
	}

	/**
	 * Appends a medium.
	 *
	 * @param medium Medium.
	 */
	public appendMedium(medium: string): void {
		this[this.length] = medium;
		(<number>this.length)++;
	}

	/**
	 * Deletes a medium.
	 *
	 * @param medium Medium.
	 */
	public deleteMedium(medium: string): void {
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
			(<number>this.length)--;
		}
	}
}
