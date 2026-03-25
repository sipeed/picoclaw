/**
 * Node utility.
 */
export default class ClassMethodBinder {
    target;
    classes;
    cache = new Map();
    /**
     * Constructor.
     *
     * @param target Target.
     * @param classes Classes.
     */
    constructor(target, classes) {
        this.target = target;
        this.classes = classes;
    }
    /**
     * Binds method, getters and setters to a target.
     *
     * @param name Method name.
     */
    bind(name) {
        // We should never bind the Symbol.iterator method as it can cause problems with Array.from()
        if (this.cache.has(name) || name === Symbol.iterator || name === 'constructor') {
            return;
        }
        this.cache.set(name, true);
        const target = this.target;
        if (!(name in target)) {
            return;
        }
        for (const _class of this.classes) {
            const descriptor = Object.getOwnPropertyDescriptor(_class.prototype, name);
            if (descriptor) {
                if (typeof descriptor.value === 'function') {
                    if (descriptor.value.toString().startsWith('class ')) {
                        // Do not bind classes
                        return;
                    }
                    Object.defineProperty(target, name, {
                        ...descriptor,
                        value: descriptor.value.bind(target)
                    });
                }
                else if (descriptor.get !== undefined) {
                    Object.defineProperty(target, name, {
                        ...descriptor,
                        get: descriptor.get?.bind(target),
                        set: descriptor.set?.bind(target)
                    });
                }
                return;
            }
        }
    }
    /**
     * Prevents a method, getter or setter from being bound.
     *
     * @param name Method name.
     */
    preventBinding(name) {
        this.cache.set(name, true);
    }
}
//# sourceMappingURL=ClassMethodBinder.js.map