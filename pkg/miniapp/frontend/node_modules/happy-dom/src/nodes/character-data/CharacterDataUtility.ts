import CharacterData from './CharacterData.js';

/**
 * Child node utility.
 */
export default class CharacterDataUtility {
	/**
	 * Appends the given DOMString to the CharacterData.data string; when this method returns, data contains the concatenated DOMString.
	 *
	 * @param characterData Character data.
	 * @param data Data.
	 */
	public static appendData(characterData: CharacterData, data: string): void {
		characterData.data += data;
	}

	/**
	 * Removes the specified amount of characters, starting at the specified offset, from the CharacterData.data string; when this method returns, data contains the shortened DOMString.
	 *
	 * @param characterData Character data.
	 * @param offset Offset.
	 * @param count Count.
	 */
	public static deleteData(characterData: CharacterData, offset: number, count: number): void {
		characterData.data =
			characterData.data.substring(0, offset) + characterData.data.substring(offset + count);
	}

	/**
	 * Inserts the specified characters, at the specified offset, in the CharacterData.data string; when this method returns, data contains the modified DOMString.
	 *
	 * @param characterData Character data.
	 * @param offset Offset.
	 * @param data Data.
	 */
	public static insertData(characterData: CharacterData, offset: number, data: string): void {
		characterData.data =
			characterData.data.substring(0, offset) + data + characterData.data.substring(offset);
	}
	/**
	 * Replaces the specified amount of characters, starting at the specified offset, with the specified DOMString; when this method returns, data contains the modified DOMString.
	 *
	 * @param characterData Character data.
	 * @param offset Offset.
	 * @param count Count.
	 * @param data Data.
	 */
	public static replaceData(
		characterData: CharacterData,
		offset: number,
		count: number,
		data: string
	): void {
		characterData.data =
			characterData.data.substring(0, offset) + data + characterData.data.substring(offset + count);
	}

	/**
	 * Returns a DOMString containing the part of CharacterData.data of the specified length and starting at the specified offset.
	 *
	 * @param characterData Character data.
	 * @param offset Offset.
	 * @param count Count.
	 */
	public static substringData(characterData: CharacterData, offset: number, count: number): string {
		return characterData.data.substring(offset, offset + count);
	}
}
