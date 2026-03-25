var BrowserNavigationCrossOriginPolicyEnum;
(function (BrowserNavigationCrossOriginPolicyEnum) {
    /** The browser can navigate to any origin. */
    BrowserNavigationCrossOriginPolicyEnum["anyOrigin"] = "anyOrigin";
    /** The browser can only navigate to the same origin as the current page or its parent. */
    BrowserNavigationCrossOriginPolicyEnum["sameOrigin"] = "sameOrigin";
    /** The browser can never navigate from a secure protocol (https) to an unsecure protocol (http), but it can always navigate to a secure (https). */
    BrowserNavigationCrossOriginPolicyEnum["strictOrigin"] = "strictOrigin";
})(BrowserNavigationCrossOriginPolicyEnum || (BrowserNavigationCrossOriginPolicyEnum = {}));
export default BrowserNavigationCrossOriginPolicyEnum;
//# sourceMappingURL=BrowserNavigationCrossOriginPolicyEnum.js.map