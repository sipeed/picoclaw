"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const CookieSameSiteEnum_js_1 = __importDefault(require("./enums/CookieSameSiteEnum.cjs"));
exports.default = {
    // Required
    key: null,
    originURL: null,
    // Optional
    value: null,
    domain: '',
    path: '',
    expires: null,
    httpOnly: false,
    secure: false,
    sameSite: CookieSameSiteEnum_js_1.default.lax
};
//# sourceMappingURL=DefaultCookie.cjs.map