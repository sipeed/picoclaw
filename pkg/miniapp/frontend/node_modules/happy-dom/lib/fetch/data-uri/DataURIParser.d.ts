/**
 * Data URI parser.
 *
 * Based on:
 * https://github.com/TooTallNate/node-data-uri-to-buffer/blob/main/src/index.ts (MIT)
 */
export default class DataURIParser {
    /**
     * Returns a Buffer instance from the given data URI `uri`.
     *
     * @param uri Data URI.
     * @returns Buffer.
     */
    static parse(uri: string): {
        type: string;
        charset: string;
        buffer: Buffer;
    };
}
//# sourceMappingURL=DataURIParser.d.ts.map