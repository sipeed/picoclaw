import DocumentFragment from '../document-fragment/DocumentFragment.cjs';
import TextTrackCue from './TextTrackCue.cjs';
import VTTRegion from './VTTRegion.cjs';
/**
 * VTTCue.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/VTTCue
 */
export default class VTTCue extends TextTrackCue {
    region: VTTRegion | null;
    vertical: string;
    snapToLines: boolean;
    line: number;
    lineAlign: string;
    position: string;
    positionAlign: string;
    size: number;
    align: string;
    text: string;
    /**
     * Constructor.
     *
     * @param startTime The start time for the cue.
     * @param endTime The end time for the cue.
     * @param text The text of the cue.
     */
    constructor(startTime: number, endTime: number, text: string);
    /**
     * Returns the cue as HTML.
     *
     * @returns DocumentFragment
     */
    getCueAsHTML(): DocumentFragment;
}
//# sourceMappingURL=VTTCue.d.ts.map