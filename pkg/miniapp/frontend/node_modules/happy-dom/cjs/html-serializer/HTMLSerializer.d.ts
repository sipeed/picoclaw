import Node from '../nodes/node/Node.cjs';
import ShadowRoot from '../nodes/shadow-root/ShadowRoot.cjs';
/**
 * Serializes a node into HTML.
 */
export default class HTMLSerializer {
    private options;
    /**
     * Constructor.
     *
     * @param [options] Options.
     * @param [options.serializableShadowRoots] If shadow roots should be serialized.
     * @param [options.shadowRoots] Shadow roots to serialize.
     * @param [options.allShadowRoots] If all shadow roots should be serialized.
     */
    constructor(options?: {
        serializableShadowRoots?: boolean;
        shadowRoots?: ShadowRoot[] | null;
        allShadowRoots?: boolean;
    });
    /**
     * Renders an element as HTML.
     *
     * @param root Root element.
     * @returns Result.
     */
    serializeToString(root: Node): string;
    /**
     * Returns attributes as a string.
     *
     * @param element Element.
     * @returns Attributes.
     */
    private getAttributes;
}
//# sourceMappingURL=HTMLSerializer.d.ts.map