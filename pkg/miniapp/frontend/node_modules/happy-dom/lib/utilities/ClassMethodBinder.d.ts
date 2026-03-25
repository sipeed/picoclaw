/**
 * Node utility.
 */
export default class ClassMethodBinder {
    private target;
    private classes;
    private cache;
    /**
     * Constructor.
     *
     * @param target Target.
     * @param classes Classes.
     */
    constructor(target: Object, classes: any[]);
    /**
     * Binds method, getters and setters to a target.
     *
     * @param name Method name.
     */
    bind(name: string | symbol): void;
    /**
     * Prevents a method, getter or setter from being bound.
     *
     * @param name Method name.
     */
    preventBinding(name: string | symbol): void;
}
//# sourceMappingURL=ClassMethodBinder.d.ts.map