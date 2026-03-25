import Plugin from './Plugin.cjs';
/**
 * MimeType.
 */
export default class MimeType {
    readonly description: string;
    readonly enabledPlugin: Plugin;
    readonly suffixes: string;
    readonly type: string;
    /**
     * Constructor.
     *
     * @param description
     * @param enabledPlugin
     * @param suffixes
     * @param type
     */
    constructor(description: string, enabledPlugin: Plugin, suffixes: string, type: string);
    /**
     * Returns the object as a string.
     *
     * @returns String.
     */
    toString(): string;
}
//# sourceMappingURL=MimeType.d.ts.map