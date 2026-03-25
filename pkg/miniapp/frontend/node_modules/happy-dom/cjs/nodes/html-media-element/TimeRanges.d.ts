/**
 * When loading a media resource for use by an <audio> or <video> element, the TimeRanges interface is used for representing the time ranges of the media resource that have been buffered, the time ranges that have been played, and the time ranges that are seekable.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/TimeRanges
 */
export default class TimeRanges {
    /**
     * Constructor.
     *
     * @param illegalConstructorSymbol Illegal constructor symbol.
     */
    constructor(illegalConstructorSymbol: symbol);
    /**
     * Returns length.
     */
    get length(): number;
    /**
     * Returns `Symbol.toStringTag`.
     *
     * @returns `Symbol.toStringTag`.
     */
    get [Symbol.toStringTag](): string;
    /**
     * Returns `[object NodeList]`.
     *
     * @returns `[object NodeList]`.
     */
    toLocaleString(): string;
    /**
     * Returns `[object NodeList]`.
     *
     * @returns `[object NodeList]`.
     */
    toString(): string;
    /**
     * Returns start.
     *
     * @param _index Index.
     * @returns Start.
     */
    start(_index: number): number;
    /**
     * Returns end.
     *
     * @param _index Index.
     * @returns End.
     */
    end(_index: number): number;
}
//# sourceMappingURL=TimeRanges.d.ts.map