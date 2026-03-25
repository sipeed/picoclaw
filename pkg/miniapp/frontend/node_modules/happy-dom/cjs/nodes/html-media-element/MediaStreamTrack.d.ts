import Event from '../../event/Event.cjs';
import EventTarget from '../../event/EventTarget.cjs';
import * as PropertySymbol from '../../PropertySymbol.cjs';
import IMediaTrackCapabilities from './IMediaTrackCapabilities.cjs';
import IMediaTrackSettings from './IMediaTrackSettings.cjs';
/**
 * Canvas Capture Media Stream Track.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/MediaStreamTrack
 */
export default class MediaStreamTrack extends EventTarget {
    #private;
    contentHint: '' | 'speech' | 'speech-recognition' | 'music' | 'motion' | 'detail' | 'text';
    enabled: boolean;
    readonly id: string;
    muted: boolean;
    readyState: 'live' | 'ended';
    label: string;
    [PropertySymbol.label]: string;
    [PropertySymbol.kind]: 'audio' | 'video';
    [PropertySymbol.constraints]: object;
    [PropertySymbol.capabilities]: IMediaTrackCapabilities;
    [PropertySymbol.settings]: IMediaTrackSettings;
    onended: ((event: Event) => void) | null;
    onmute: ((event: Event) => void) | null;
    onunmute: ((event: Event) => void) | null;
    /**
     * Constructor.
     *
     * @param illegalConstructorSymbol Illegal constructor symbol.
     */
    constructor(illegalConstructorSymbol: symbol);
    /**
     * Returns the kind of the track.
     *
     * @returns Kind.
     */
    get kind(): 'audio' | 'video';
    /**
     * Applies constraints.
     *
     * @param _constraints Constraints.
     * @param constraints
     */
    applyConstraints(constraints: object): Promise<void>;
    /**
     * Returns constraints.
     *
     * @returns Constraints.
     */
    getConstraints(): object;
    /**
     * Returns capabilities.
     *
     * @returns Capabilities.
     */
    getCapabilities(): IMediaTrackCapabilities;
    /**
     * Returns settings.
     *
     * @returns Settings.
     */
    getSettings(): IMediaTrackSettings;
    /**
     * Clones the track.
     *
     * @returns Clone.
     */
    clone(): MediaStreamTrack;
    /**
     * Stops the track.
     */
    stop(): void;
}
//# sourceMappingURL=MediaStreamTrack.d.ts.map