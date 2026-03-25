import CSSMeasurementConverter from '../css/declaration/measurement-converter/CSSMeasurementConverter.js';
import WindowBrowserContext from '../window/WindowBrowserContext.js';
import MediaQueryTypeEnum from './MediaQueryTypeEnum.js';
/**
 * Media query this.
 */
export default class MediaQueryItem {
    mediaTypes;
    not;
    rules;
    ranges;
    rootFontSize = null;
    window;
    /**
     * Constructor.
     *
     * @param options Options.
     * @param options.window Owner window.
     * @param [options.rootFontSize] Root font size.
     * @param [options.mediaTypes] Media types.
     * @param [options.not] Not.
     * @param [options.rules] Rules.
     * @param [options.ranges] Ranges.
     */
    constructor(options) {
        this.window = options.window;
        this.rootFontSize = options.rootFontSize || null;
        this.mediaTypes = options.mediaTypes || [];
        this.not = options.not || false;
        this.rules = options.rules || [];
        this.ranges = options.ranges || [];
    }
    /**
     * Returns media string.
     */
    toString() {
        return `${this.not ? 'not ' : ''}${this.mediaTypes.join(', ')}${(this.not || this.mediaTypes.length > 0) && !!this.ranges.length ? ' and ' : ''}${this.ranges
            .map((range) => `(${range.before ? `${range.before.value} ${range.before.operator} ` : ''}${range.type}${range.after ? ` ${range.after.operator} ${range.after.value}` : ''})`)
            .join(' and ')}${(this.not || this.mediaTypes.length > 0) && !!this.rules.length ? ' and ' : ''}${this.rules
            .map((rule) => (rule.value ? `(${rule.name}: ${rule.value})` : `(${rule.name})`))
            .join(' and ')}`;
    }
    /**
     * Returns "true" if the item matches.
     */
    matches() {
        return this.not ? !this.matchesAll() : this.matchesAll();
    }
    /**
     * Returns "true" if all matches.
     *
     * @returns "true" if all matches.
     */
    matchesAll() {
        if (!!this.mediaTypes.length) {
            let isMediaTypeMatch = false;
            for (const mediaType of this.mediaTypes) {
                if (this.matchesMediaType(mediaType)) {
                    isMediaTypeMatch = true;
                    break;
                }
            }
            if (!isMediaTypeMatch) {
                return false;
            }
        }
        for (const rule of this.rules) {
            if (!this.matchesRule(rule)) {
                return false;
            }
        }
        for (const range of this.ranges) {
            if (!this.matchesRange(range)) {
                return false;
            }
        }
        return true;
    }
    /**
     * Returns "true" if the mediaType matches.
     *
     * @param mediaType Media type.
     * @returns "true" if the mediaType matches.
     */
    matchesMediaType(mediaType) {
        if (mediaType === MediaQueryTypeEnum.all) {
            return true;
        }
        return mediaType === new WindowBrowserContext(this.window).getSettings()?.device.mediaType;
    }
    /**
     * Returns "true" if the range matches.
     *
     * @param range Range.
     * @returns "true" if the range matches.
     */
    matchesRange(range) {
        const windowSize = range.type === 'width' ? this.window.innerWidth : this.window.innerHeight;
        if (range.before) {
            const beforeValue = this.toPixels(range.before.value);
            if (beforeValue === null) {
                return false;
            }
            switch (range.before.operator) {
                case '<':
                    if (beforeValue >= windowSize) {
                        return false;
                    }
                    break;
                case '<=':
                    if (beforeValue > windowSize) {
                        return false;
                    }
                    break;
                case '>':
                    if (beforeValue <= windowSize) {
                        return false;
                    }
                    break;
                case '>=':
                    if (beforeValue < windowSize) {
                        return false;
                    }
                    break;
            }
        }
        if (range.after) {
            const afterValue = this.toPixels(range.after.value);
            if (afterValue === null) {
                return false;
            }
            switch (range.after.operator) {
                case '<':
                    if (windowSize >= afterValue) {
                        return false;
                    }
                    break;
                case '<=':
                    if (windowSize > afterValue) {
                        return false;
                    }
                    break;
                case '>':
                    if (windowSize <= afterValue) {
                        return false;
                    }
                    break;
                case '>=':
                    if (windowSize < afterValue) {
                        return false;
                    }
                    break;
            }
        }
        return true;
    }
    /**
     * Returns "true" if the rule matches.
     *
     * @param rule Rule.
     * @returns "true" if the rule matches.
     */
    matchesRule(rule) {
        if (!rule.value) {
            switch (rule.name) {
                case 'min-width':
                case 'max-width':
                case 'min-height':
                case 'max-height':
                case 'width':
                case 'height':
                case 'orientation':
                case 'prefers-color-scheme':
                case 'hover':
                case 'any-hover':
                case 'any-pointer':
                case 'pointer':
                case 'display-mode':
                case 'min-aspect-ratio':
                case 'max-aspect-ratio':
                case 'aspect-ratio':
                    return true;
            }
            return false;
        }
        switch (rule.name) {
            case 'min-width':
                const minWidth = this.toPixels(rule.value);
                return minWidth !== null && this.window.innerWidth >= minWidth;
            case 'max-width':
                const maxWidth = this.toPixels(rule.value);
                return maxWidth !== null && this.window.innerWidth <= maxWidth;
            case 'min-height':
                const minHeight = this.toPixels(rule.value);
                return minHeight !== null && this.window.innerHeight >= minHeight;
            case 'max-height':
                const maxHeight = this.toPixels(rule.value);
                return maxHeight !== null && this.window.innerHeight <= maxHeight;
            case 'width':
                const width = this.toPixels(rule.value);
                return width !== null && this.window.innerWidth === width;
            case 'height':
                const height = this.toPixels(rule.value);
                return height !== null && this.window.innerHeight === height;
            case 'orientation':
                return rule.value === 'landscape'
                    ? this.window.innerWidth > this.window.innerHeight
                    : this.window.innerWidth < this.window.innerHeight;
            case 'prefers-color-scheme':
                return (rule.value ===
                    new WindowBrowserContext(this.window).getSettings().device.prefersColorScheme);
            case 'any-hover':
            case 'hover':
                if (rule.value === 'none') {
                    return this.window.navigator.maxTouchPoints > 0;
                }
                if (rule.value === 'hover') {
                    return this.window.navigator.maxTouchPoints === 0;
                }
                return false;
            case 'any-pointer':
            case 'pointer':
                if (rule.value === 'none') {
                    return false;
                }
                if (rule.value === 'coarse') {
                    return this.window.navigator.maxTouchPoints > 0;
                }
                if (rule.value === 'fine') {
                    return this.window.navigator.maxTouchPoints === 0;
                }
                return false;
            case 'display-mode':
                return rule.value === 'browser';
            case 'min-aspect-ratio':
            case 'max-aspect-ratio':
            case 'aspect-ratio':
                const aspectRatio = rule.value.split('/');
                const aspectRatioWidth = parseInt(aspectRatio[0], 10);
                const aspectRatioHeight = parseInt(aspectRatio[1], 10);
                if (isNaN(aspectRatioWidth) || isNaN(aspectRatioHeight)) {
                    return false;
                }
                switch (rule.name) {
                    case 'min-aspect-ratio':
                        return (aspectRatioWidth / aspectRatioHeight <=
                            this.window.innerWidth / this.window.innerHeight);
                    case 'max-aspect-ratio':
                        return (aspectRatioWidth / aspectRatioHeight >=
                            this.window.innerWidth / this.window.innerHeight);
                    case 'aspect-ratio':
                        return (aspectRatioWidth / aspectRatioHeight ===
                            this.window.innerWidth / this.window.innerHeight);
                }
        }
        return false;
    }
    /**
     * Convert to pixels.
     *
     * @param value Value.
     * @returns Value in pixels.
     */
    toPixels(value) {
        if (!new WindowBrowserContext(this.window).getSettings()?.disableComputedStyleRendering &&
            value.endsWith('em')) {
            this.rootFontSize =
                this.rootFontSize ||
                    parseFloat(this.window.getComputedStyle(this.window.document.documentElement).fontSize);
            return CSSMeasurementConverter.toPixels({
                window: this.window,
                value,
                rootFontSize: this.rootFontSize,
                parentFontSize: this.rootFontSize
            });
        }
        return CSSMeasurementConverter.toPixels({
            window: this.window,
            value,
            rootFontSize: 16,
            parentFontSize: 16
        });
    }
}
//# sourceMappingURL=MediaQueryItem.js.map