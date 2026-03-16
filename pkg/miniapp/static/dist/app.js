(() => {
  var __create = Object.create;
  var __getProtoOf = Object.getPrototypeOf;
  var __defProp = Object.defineProperty;
  var __getOwnPropNames = Object.getOwnPropertyNames;
  var __hasOwnProp = Object.prototype.hasOwnProperty;
  function __accessProp(key) {
    return this[key];
  }
  var __toESMCache_node;
  var __toESMCache_esm;
  var __toESM = (mod, isNodeMode, target) => {
    var canCache = mod != null && typeof mod === "object";
    if (canCache) {
      var cache = isNodeMode ? __toESMCache_node ??= new WeakMap : __toESMCache_esm ??= new WeakMap;
      var cached = cache.get(mod);
      if (cached)
        return cached;
    }
    target = mod != null ? __create(__getProtoOf(mod)) : {};
    const to = isNodeMode || !mod || !mod.__esModule ? __defProp(target, "default", { value: mod, enumerable: true }) : target;
    for (let key of __getOwnPropNames(mod))
      if (!__hasOwnProp.call(to, key))
        __defProp(to, key, {
          get: __accessProp.bind(mod, key),
          enumerable: true
        });
    if (canCache)
      cache.set(mod, to);
    return to;
  };
  var __commonJS = (cb, mod) => () => (mod || cb((mod = { exports: {} }).exports, mod), mod.exports);

  // node_modules/highlight.js/lib/core.js
  var require_core = __commonJS((exports, module) => {
    function deepFreeze(obj) {
      if (obj instanceof Map) {
        obj.clear = obj.delete = obj.set = function() {
          throw new Error("map is read-only");
        };
      } else if (obj instanceof Set) {
        obj.add = obj.clear = obj.delete = function() {
          throw new Error("set is read-only");
        };
      }
      Object.freeze(obj);
      Object.getOwnPropertyNames(obj).forEach((name) => {
        const prop = obj[name];
        const type = typeof prop;
        if ((type === "object" || type === "function") && !Object.isFrozen(prop)) {
          deepFreeze(prop);
        }
      });
      return obj;
    }

    class Response {
      constructor(mode) {
        if (mode.data === undefined)
          mode.data = {};
        this.data = mode.data;
        this.isMatchIgnored = false;
      }
      ignoreMatch() {
        this.isMatchIgnored = true;
      }
    }
    function escapeHTML(value) {
      return value.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;").replace(/'/g, "&#x27;");
    }
    function inherit$1(original, ...objects) {
      const result = Object.create(null);
      for (const key in original) {
        result[key] = original[key];
      }
      objects.forEach(function(obj) {
        for (const key in obj) {
          result[key] = obj[key];
        }
      });
      return result;
    }
    var SPAN_CLOSE = "</span>";
    var emitsWrappingTags = (node) => {
      return !!node.scope;
    };
    var scopeToCSSClass = (name, { prefix }) => {
      if (name.startsWith("language:")) {
        return name.replace("language:", "language-");
      }
      if (name.includes(".")) {
        const pieces = name.split(".");
        return [
          `${prefix}${pieces.shift()}`,
          ...pieces.map((x2, i) => `${x2}${"_".repeat(i + 1)}`)
        ].join(" ");
      }
      return `${prefix}${name}`;
    };

    class HTMLRenderer {
      constructor(parseTree, options) {
        this.buffer = "";
        this.classPrefix = options.classPrefix;
        parseTree.walk(this);
      }
      addText(text) {
        this.buffer += escapeHTML(text);
      }
      openNode(node) {
        if (!emitsWrappingTags(node))
          return;
        const className = scopeToCSSClass(node.scope, { prefix: this.classPrefix });
        this.span(className);
      }
      closeNode(node) {
        if (!emitsWrappingTags(node))
          return;
        this.buffer += SPAN_CLOSE;
      }
      value() {
        return this.buffer;
      }
      span(className) {
        this.buffer += `<span class="${className}">`;
      }
    }
    var newNode = (opts = {}) => {
      const result = { children: [] };
      Object.assign(result, opts);
      return result;
    };

    class TokenTree {
      constructor() {
        this.rootNode = newNode();
        this.stack = [this.rootNode];
      }
      get top() {
        return this.stack[this.stack.length - 1];
      }
      get root() {
        return this.rootNode;
      }
      add(node) {
        this.top.children.push(node);
      }
      openNode(scope) {
        const node = newNode({ scope });
        this.add(node);
        this.stack.push(node);
      }
      closeNode() {
        if (this.stack.length > 1) {
          return this.stack.pop();
        }
        return;
      }
      closeAllNodes() {
        while (this.closeNode())
          ;
      }
      toJSON() {
        return JSON.stringify(this.rootNode, null, 4);
      }
      walk(builder) {
        return this.constructor._walk(builder, this.rootNode);
      }
      static _walk(builder, node) {
        if (typeof node === "string") {
          builder.addText(node);
        } else if (node.children) {
          builder.openNode(node);
          node.children.forEach((child) => this._walk(builder, child));
          builder.closeNode(node);
        }
        return builder;
      }
      static _collapse(node) {
        if (typeof node === "string")
          return;
        if (!node.children)
          return;
        if (node.children.every((el) => typeof el === "string")) {
          node.children = [node.children.join("")];
        } else {
          node.children.forEach((child) => {
            TokenTree._collapse(child);
          });
        }
      }
    }

    class TokenTreeEmitter extends TokenTree {
      constructor(options) {
        super();
        this.options = options;
      }
      addText(text) {
        if (text === "") {
          return;
        }
        this.add(text);
      }
      startScope(scope) {
        this.openNode(scope);
      }
      endScope() {
        this.closeNode();
      }
      __addSublanguage(emitter, name) {
        const node = emitter.root;
        if (name)
          node.scope = `language:${name}`;
        this.add(node);
      }
      toHTML() {
        const renderer = new HTMLRenderer(this, this.options);
        return renderer.value();
      }
      finalize() {
        this.closeAllNodes();
        return true;
      }
    }
    function source(re2) {
      if (!re2)
        return null;
      if (typeof re2 === "string")
        return re2;
      return re2.source;
    }
    function lookahead(re2) {
      return concat("(?=", re2, ")");
    }
    function anyNumberOfTimes(re2) {
      return concat("(?:", re2, ")*");
    }
    function optional(re2) {
      return concat("(?:", re2, ")?");
    }
    function concat(...args) {
      const joined = args.map((x2) => source(x2)).join("");
      return joined;
    }
    function stripOptionsFromArgs(args) {
      const opts = args[args.length - 1];
      if (typeof opts === "object" && opts.constructor === Object) {
        args.splice(args.length - 1, 1);
        return opts;
      } else {
        return {};
      }
    }
    function either(...args) {
      const opts = stripOptionsFromArgs(args);
      const joined = "(" + (opts.capture ? "" : "?:") + args.map((x2) => source(x2)).join("|") + ")";
      return joined;
    }
    function countMatchGroups(re2) {
      return new RegExp(re2.toString() + "|").exec("").length - 1;
    }
    function startsWith(re2, lexeme) {
      const match = re2 && re2.exec(lexeme);
      return match && match.index === 0;
    }
    var BACKREF_RE = /\[(?:[^\\\]]|\\.)*\]|\(\??|\\([1-9][0-9]*)|\\./;
    function _rewriteBackreferences(regexps, { joinWith }) {
      let numCaptures = 0;
      return regexps.map((regex) => {
        numCaptures += 1;
        const offset = numCaptures;
        let re2 = source(regex);
        let out = "";
        while (re2.length > 0) {
          const match = BACKREF_RE.exec(re2);
          if (!match) {
            out += re2;
            break;
          }
          out += re2.substring(0, match.index);
          re2 = re2.substring(match.index + match[0].length);
          if (match[0][0] === "\\" && match[1]) {
            out += "\\" + String(Number(match[1]) + offset);
          } else {
            out += match[0];
            if (match[0] === "(") {
              numCaptures++;
            }
          }
        }
        return out;
      }).map((re2) => `(${re2})`).join(joinWith);
    }
    var MATCH_NOTHING_RE = /\b\B/;
    var IDENT_RE = "[a-zA-Z]\\w*";
    var UNDERSCORE_IDENT_RE = "[a-zA-Z_]\\w*";
    var NUMBER_RE = "\\b\\d+(\\.\\d+)?";
    var C_NUMBER_RE = "(-?)(\\b0[xX][a-fA-F0-9]+|(\\b\\d+(\\.\\d*)?|\\.\\d+)([eE][-+]?\\d+)?)";
    var BINARY_NUMBER_RE = "\\b(0b[01]+)";
    var RE_STARTERS_RE = "!|!=|!==|%|%=|&|&&|&=|\\*|\\*=|\\+|\\+=|,|-|-=|/=|/|:|;|<<|<<=|<=|<|===|==|=|>>>=|>>=|>=|>>>|>>|>|\\?|\\[|\\{|\\(|\\^|\\^=|\\||\\|=|\\|\\||~";
    var SHEBANG = (opts = {}) => {
      const beginShebang = /^#![ ]*\//;
      if (opts.binary) {
        opts.begin = concat(beginShebang, /.*\b/, opts.binary, /\b.*/);
      }
      return inherit$1({
        scope: "meta",
        begin: beginShebang,
        end: /$/,
        relevance: 0,
        "on:begin": (m2, resp) => {
          if (m2.index !== 0)
            resp.ignoreMatch();
        }
      }, opts);
    };
    var BACKSLASH_ESCAPE = {
      begin: "\\\\[\\s\\S]",
      relevance: 0
    };
    var APOS_STRING_MODE = {
      scope: "string",
      begin: "'",
      end: "'",
      illegal: "\\n",
      contains: [BACKSLASH_ESCAPE]
    };
    var QUOTE_STRING_MODE = {
      scope: "string",
      begin: '"',
      end: '"',
      illegal: "\\n",
      contains: [BACKSLASH_ESCAPE]
    };
    var PHRASAL_WORDS_MODE = {
      begin: /\b(a|an|the|are|I'm|isn't|don't|doesn't|won't|but|just|should|pretty|simply|enough|gonna|going|wtf|so|such|will|you|your|they|like|more)\b/
    };
    var COMMENT = function(begin, end, modeOptions = {}) {
      const mode = inherit$1({
        scope: "comment",
        begin,
        end,
        contains: []
      }, modeOptions);
      mode.contains.push({
        scope: "doctag",
        begin: "[ ]*(?=(TODO|FIXME|NOTE|BUG|OPTIMIZE|HACK|XXX):)",
        end: /(TODO|FIXME|NOTE|BUG|OPTIMIZE|HACK|XXX):/,
        excludeBegin: true,
        relevance: 0
      });
      const ENGLISH_WORD = either("I", "a", "is", "so", "us", "to", "at", "if", "in", "it", "on", /[A-Za-z]+['](d|ve|re|ll|t|s|n)/, /[A-Za-z]+[-][a-z]+/, /[A-Za-z][a-z]{2,}/);
      mode.contains.push({
        begin: concat(/[ ]+/, "(", ENGLISH_WORD, /[.]?[:]?([.][ ]|[ ])/, "){3}")
      });
      return mode;
    };
    var C_LINE_COMMENT_MODE = COMMENT("//", "$");
    var C_BLOCK_COMMENT_MODE = COMMENT("/\\*", "\\*/");
    var HASH_COMMENT_MODE = COMMENT("#", "$");
    var NUMBER_MODE = {
      scope: "number",
      begin: NUMBER_RE,
      relevance: 0
    };
    var C_NUMBER_MODE = {
      scope: "number",
      begin: C_NUMBER_RE,
      relevance: 0
    };
    var BINARY_NUMBER_MODE = {
      scope: "number",
      begin: BINARY_NUMBER_RE,
      relevance: 0
    };
    var REGEXP_MODE = {
      scope: "regexp",
      begin: /\/(?=[^/\n]*\/)/,
      end: /\/[gimuy]*/,
      contains: [
        BACKSLASH_ESCAPE,
        {
          begin: /\[/,
          end: /\]/,
          relevance: 0,
          contains: [BACKSLASH_ESCAPE]
        }
      ]
    };
    var TITLE_MODE = {
      scope: "title",
      begin: IDENT_RE,
      relevance: 0
    };
    var UNDERSCORE_TITLE_MODE = {
      scope: "title",
      begin: UNDERSCORE_IDENT_RE,
      relevance: 0
    };
    var METHOD_GUARD = {
      begin: "\\.\\s*" + UNDERSCORE_IDENT_RE,
      relevance: 0
    };
    var END_SAME_AS_BEGIN = function(mode) {
      return Object.assign(mode, {
        "on:begin": (m2, resp) => {
          resp.data._beginMatch = m2[1];
        },
        "on:end": (m2, resp) => {
          if (resp.data._beginMatch !== m2[1])
            resp.ignoreMatch();
        }
      });
    };
    var MODES = /* @__PURE__ */ Object.freeze({
      __proto__: null,
      APOS_STRING_MODE,
      BACKSLASH_ESCAPE,
      BINARY_NUMBER_MODE,
      BINARY_NUMBER_RE,
      COMMENT,
      C_BLOCK_COMMENT_MODE,
      C_LINE_COMMENT_MODE,
      C_NUMBER_MODE,
      C_NUMBER_RE,
      END_SAME_AS_BEGIN,
      HASH_COMMENT_MODE,
      IDENT_RE,
      MATCH_NOTHING_RE,
      METHOD_GUARD,
      NUMBER_MODE,
      NUMBER_RE,
      PHRASAL_WORDS_MODE,
      QUOTE_STRING_MODE,
      REGEXP_MODE,
      RE_STARTERS_RE,
      SHEBANG,
      TITLE_MODE,
      UNDERSCORE_IDENT_RE,
      UNDERSCORE_TITLE_MODE
    });
    function skipIfHasPrecedingDot(match, response) {
      const before = match.input[match.index - 1];
      if (before === ".") {
        response.ignoreMatch();
      }
    }
    function scopeClassName(mode, _parent) {
      if (mode.className !== undefined) {
        mode.scope = mode.className;
        delete mode.className;
      }
    }
    function beginKeywords(mode, parent) {
      if (!parent)
        return;
      if (!mode.beginKeywords)
        return;
      mode.begin = "\\b(" + mode.beginKeywords.split(" ").join("|") + ")(?!\\.)(?=\\b|\\s)";
      mode.__beforeBegin = skipIfHasPrecedingDot;
      mode.keywords = mode.keywords || mode.beginKeywords;
      delete mode.beginKeywords;
      if (mode.relevance === undefined)
        mode.relevance = 0;
    }
    function compileIllegal(mode, _parent) {
      if (!Array.isArray(mode.illegal))
        return;
      mode.illegal = either(...mode.illegal);
    }
    function compileMatch(mode, _parent) {
      if (!mode.match)
        return;
      if (mode.begin || mode.end)
        throw new Error("begin & end are not supported with match");
      mode.begin = mode.match;
      delete mode.match;
    }
    function compileRelevance(mode, _parent) {
      if (mode.relevance === undefined)
        mode.relevance = 1;
    }
    var beforeMatchExt = (mode, parent) => {
      if (!mode.beforeMatch)
        return;
      if (mode.starts)
        throw new Error("beforeMatch cannot be used with starts");
      const originalMode = Object.assign({}, mode);
      Object.keys(mode).forEach((key) => {
        delete mode[key];
      });
      mode.keywords = originalMode.keywords;
      mode.begin = concat(originalMode.beforeMatch, lookahead(originalMode.begin));
      mode.starts = {
        relevance: 0,
        contains: [
          Object.assign(originalMode, { endsParent: true })
        ]
      };
      mode.relevance = 0;
      delete originalMode.beforeMatch;
    };
    var COMMON_KEYWORDS = [
      "of",
      "and",
      "for",
      "in",
      "not",
      "or",
      "if",
      "then",
      "parent",
      "list",
      "value"
    ];
    var DEFAULT_KEYWORD_SCOPE = "keyword";
    function compileKeywords(rawKeywords, caseInsensitive, scopeName = DEFAULT_KEYWORD_SCOPE) {
      const compiledKeywords = Object.create(null);
      if (typeof rawKeywords === "string") {
        compileList(scopeName, rawKeywords.split(" "));
      } else if (Array.isArray(rawKeywords)) {
        compileList(scopeName, rawKeywords);
      } else {
        Object.keys(rawKeywords).forEach(function(scopeName2) {
          Object.assign(compiledKeywords, compileKeywords(rawKeywords[scopeName2], caseInsensitive, scopeName2));
        });
      }
      return compiledKeywords;
      function compileList(scopeName2, keywordList) {
        if (caseInsensitive) {
          keywordList = keywordList.map((x2) => x2.toLowerCase());
        }
        keywordList.forEach(function(keyword) {
          const pair = keyword.split("|");
          compiledKeywords[pair[0]] = [scopeName2, scoreForKeyword(pair[0], pair[1])];
        });
      }
    }
    function scoreForKeyword(keyword, providedScore) {
      if (providedScore) {
        return Number(providedScore);
      }
      return commonKeyword(keyword) ? 0 : 1;
    }
    function commonKeyword(keyword) {
      return COMMON_KEYWORDS.includes(keyword.toLowerCase());
    }
    var seenDeprecations = {};
    var error = (message) => {
      console.error(message);
    };
    var warn = (message, ...args) => {
      console.log(`WARN: ${message}`, ...args);
    };
    var deprecated = (version2, message) => {
      if (seenDeprecations[`${version2}/${message}`])
        return;
      console.log(`Deprecated as of ${version2}. ${message}`);
      seenDeprecations[`${version2}/${message}`] = true;
    };
    var MultiClassError = new Error;
    function remapScopeNames(mode, regexes, { key }) {
      let offset = 0;
      const scopeNames = mode[key];
      const emit = {};
      const positions = {};
      for (let i = 1;i <= regexes.length; i++) {
        positions[i + offset] = scopeNames[i];
        emit[i + offset] = true;
        offset += countMatchGroups(regexes[i - 1]);
      }
      mode[key] = positions;
      mode[key]._emit = emit;
      mode[key]._multi = true;
    }
    function beginMultiClass(mode) {
      if (!Array.isArray(mode.begin))
        return;
      if (mode.skip || mode.excludeBegin || mode.returnBegin) {
        error("skip, excludeBegin, returnBegin not compatible with beginScope: {}");
        throw MultiClassError;
      }
      if (typeof mode.beginScope !== "object" || mode.beginScope === null) {
        error("beginScope must be object");
        throw MultiClassError;
      }
      remapScopeNames(mode, mode.begin, { key: "beginScope" });
      mode.begin = _rewriteBackreferences(mode.begin, { joinWith: "" });
    }
    function endMultiClass(mode) {
      if (!Array.isArray(mode.end))
        return;
      if (mode.skip || mode.excludeEnd || mode.returnEnd) {
        error("skip, excludeEnd, returnEnd not compatible with endScope: {}");
        throw MultiClassError;
      }
      if (typeof mode.endScope !== "object" || mode.endScope === null) {
        error("endScope must be object");
        throw MultiClassError;
      }
      remapScopeNames(mode, mode.end, { key: "endScope" });
      mode.end = _rewriteBackreferences(mode.end, { joinWith: "" });
    }
    function scopeSugar(mode) {
      if (mode.scope && typeof mode.scope === "object" && mode.scope !== null) {
        mode.beginScope = mode.scope;
        delete mode.scope;
      }
    }
    function MultiClass(mode) {
      scopeSugar(mode);
      if (typeof mode.beginScope === "string") {
        mode.beginScope = { _wrap: mode.beginScope };
      }
      if (typeof mode.endScope === "string") {
        mode.endScope = { _wrap: mode.endScope };
      }
      beginMultiClass(mode);
      endMultiClass(mode);
    }
    function compileLanguage(language) {
      function langRe(value, global) {
        return new RegExp(source(value), "m" + (language.case_insensitive ? "i" : "") + (language.unicodeRegex ? "u" : "") + (global ? "g" : ""));
      }

      class MultiRegex {
        constructor() {
          this.matchIndexes = {};
          this.regexes = [];
          this.matchAt = 1;
          this.position = 0;
        }
        addRule(re2, opts) {
          opts.position = this.position++;
          this.matchIndexes[this.matchAt] = opts;
          this.regexes.push([opts, re2]);
          this.matchAt += countMatchGroups(re2) + 1;
        }
        compile() {
          if (this.regexes.length === 0) {
            this.exec = () => null;
          }
          const terminators = this.regexes.map((el) => el[1]);
          this.matcherRe = langRe(_rewriteBackreferences(terminators, { joinWith: "|" }), true);
          this.lastIndex = 0;
        }
        exec(s) {
          this.matcherRe.lastIndex = this.lastIndex;
          const match = this.matcherRe.exec(s);
          if (!match) {
            return null;
          }
          const i = match.findIndex((el, i2) => i2 > 0 && el !== undefined);
          const matchData = this.matchIndexes[i];
          match.splice(0, i);
          return Object.assign(match, matchData);
        }
      }

      class ResumableMultiRegex {
        constructor() {
          this.rules = [];
          this.multiRegexes = [];
          this.count = 0;
          this.lastIndex = 0;
          this.regexIndex = 0;
        }
        getMatcher(index) {
          if (this.multiRegexes[index])
            return this.multiRegexes[index];
          const matcher = new MultiRegex;
          this.rules.slice(index).forEach(([re2, opts]) => matcher.addRule(re2, opts));
          matcher.compile();
          this.multiRegexes[index] = matcher;
          return matcher;
        }
        resumingScanAtSamePosition() {
          return this.regexIndex !== 0;
        }
        considerAll() {
          this.regexIndex = 0;
        }
        addRule(re2, opts) {
          this.rules.push([re2, opts]);
          if (opts.type === "begin")
            this.count++;
        }
        exec(s) {
          const m2 = this.getMatcher(this.regexIndex);
          m2.lastIndex = this.lastIndex;
          let result = m2.exec(s);
          if (this.resumingScanAtSamePosition()) {
            if (result && result.index === this.lastIndex)
              ;
            else {
              const m22 = this.getMatcher(0);
              m22.lastIndex = this.lastIndex + 1;
              result = m22.exec(s);
            }
          }
          if (result) {
            this.regexIndex += result.position + 1;
            if (this.regexIndex === this.count) {
              this.considerAll();
            }
          }
          return result;
        }
      }
      function buildModeRegex(mode) {
        const mm = new ResumableMultiRegex;
        mode.contains.forEach((term) => mm.addRule(term.begin, { rule: term, type: "begin" }));
        if (mode.terminatorEnd) {
          mm.addRule(mode.terminatorEnd, { type: "end" });
        }
        if (mode.illegal) {
          mm.addRule(mode.illegal, { type: "illegal" });
        }
        return mm;
      }
      function compileMode(mode, parent) {
        const cmode = mode;
        if (mode.isCompiled)
          return cmode;
        [
          scopeClassName,
          compileMatch,
          MultiClass,
          beforeMatchExt
        ].forEach((ext) => ext(mode, parent));
        language.compilerExtensions.forEach((ext) => ext(mode, parent));
        mode.__beforeBegin = null;
        [
          beginKeywords,
          compileIllegal,
          compileRelevance
        ].forEach((ext) => ext(mode, parent));
        mode.isCompiled = true;
        let keywordPattern = null;
        if (typeof mode.keywords === "object" && mode.keywords.$pattern) {
          mode.keywords = Object.assign({}, mode.keywords);
          keywordPattern = mode.keywords.$pattern;
          delete mode.keywords.$pattern;
        }
        keywordPattern = keywordPattern || /\w+/;
        if (mode.keywords) {
          mode.keywords = compileKeywords(mode.keywords, language.case_insensitive);
        }
        cmode.keywordPatternRe = langRe(keywordPattern, true);
        if (parent) {
          if (!mode.begin)
            mode.begin = /\B|\b/;
          cmode.beginRe = langRe(cmode.begin);
          if (!mode.end && !mode.endsWithParent)
            mode.end = /\B|\b/;
          if (mode.end)
            cmode.endRe = langRe(cmode.end);
          cmode.terminatorEnd = source(cmode.end) || "";
          if (mode.endsWithParent && parent.terminatorEnd) {
            cmode.terminatorEnd += (mode.end ? "|" : "") + parent.terminatorEnd;
          }
        }
        if (mode.illegal)
          cmode.illegalRe = langRe(mode.illegal);
        if (!mode.contains)
          mode.contains = [];
        mode.contains = [].concat(...mode.contains.map(function(c) {
          return expandOrCloneMode(c === "self" ? mode : c);
        }));
        mode.contains.forEach(function(c) {
          compileMode(c, cmode);
        });
        if (mode.starts) {
          compileMode(mode.starts, parent);
        }
        cmode.matcher = buildModeRegex(cmode);
        return cmode;
      }
      if (!language.compilerExtensions)
        language.compilerExtensions = [];
      if (language.contains && language.contains.includes("self")) {
        throw new Error("ERR: contains `self` is not supported at the top-level of a language.  See documentation.");
      }
      language.classNameAliases = inherit$1(language.classNameAliases || {});
      return compileMode(language);
    }
    function dependencyOnParent(mode) {
      if (!mode)
        return false;
      return mode.endsWithParent || dependencyOnParent(mode.starts);
    }
    function expandOrCloneMode(mode) {
      if (mode.variants && !mode.cachedVariants) {
        mode.cachedVariants = mode.variants.map(function(variant) {
          return inherit$1(mode, { variants: null }, variant);
        });
      }
      if (mode.cachedVariants) {
        return mode.cachedVariants;
      }
      if (dependencyOnParent(mode)) {
        return inherit$1(mode, { starts: mode.starts ? inherit$1(mode.starts) : null });
      }
      if (Object.isFrozen(mode)) {
        return inherit$1(mode);
      }
      return mode;
    }
    var version = "11.11.1";

    class HTMLInjectionError extends Error {
      constructor(reason, html) {
        super(reason);
        this.name = "HTMLInjectionError";
        this.html = html;
      }
    }
    var escape = escapeHTML;
    var inherit = inherit$1;
    var NO_MATCH = Symbol("nomatch");
    var MAX_KEYWORD_HITS = 7;
    var HLJS = function(hljs) {
      const languages = Object.create(null);
      const aliases = Object.create(null);
      const plugins = [];
      let SAFE_MODE = true;
      const LANGUAGE_NOT_FOUND = "Could not find the language '{}', did you forget to load/include a language module?";
      const PLAINTEXT_LANGUAGE = { disableAutodetect: true, name: "Plain text", contains: [] };
      let options = {
        ignoreUnescapedHTML: false,
        throwUnescapedHTML: false,
        noHighlightRe: /^(no-?highlight)$/i,
        languageDetectRe: /\blang(?:uage)?-([\w-]+)\b/i,
        classPrefix: "hljs-",
        cssSelector: "pre code",
        languages: null,
        __emitter: TokenTreeEmitter
      };
      function shouldNotHighlight(languageName) {
        return options.noHighlightRe.test(languageName);
      }
      function blockLanguage(block) {
        let classes = block.className + " ";
        classes += block.parentNode ? block.parentNode.className : "";
        const match = options.languageDetectRe.exec(classes);
        if (match) {
          const language = getLanguage(match[1]);
          if (!language) {
            warn(LANGUAGE_NOT_FOUND.replace("{}", match[1]));
            warn("Falling back to no-highlight mode for this block.", block);
          }
          return language ? match[1] : "no-highlight";
        }
        return classes.split(/\s+/).find((_class) => shouldNotHighlight(_class) || getLanguage(_class));
      }
      function highlight2(codeOrLanguageName, optionsOrCode, ignoreIllegals) {
        let code = "";
        let languageName = "";
        if (typeof optionsOrCode === "object") {
          code = codeOrLanguageName;
          ignoreIllegals = optionsOrCode.ignoreIllegals;
          languageName = optionsOrCode.language;
        } else {
          deprecated("10.7.0", "highlight(lang, code, ...args) has been deprecated.");
          deprecated("10.7.0", `Please use highlight(code, options) instead.
https://github.com/highlightjs/highlight.js/issues/2277`);
          languageName = codeOrLanguageName;
          code = optionsOrCode;
        }
        if (ignoreIllegals === undefined) {
          ignoreIllegals = true;
        }
        const context = {
          code,
          language: languageName
        };
        fire("before:highlight", context);
        const result = context.result ? context.result : _highlight(context.language, context.code, ignoreIllegals);
        result.code = context.code;
        fire("after:highlight", result);
        return result;
      }
      function _highlight(languageName, codeToHighlight, ignoreIllegals, continuation) {
        const keywordHits = Object.create(null);
        function keywordData(mode, matchText) {
          return mode.keywords[matchText];
        }
        function processKeywords() {
          if (!top.keywords) {
            emitter.addText(modeBuffer);
            return;
          }
          let lastIndex = 0;
          top.keywordPatternRe.lastIndex = 0;
          let match = top.keywordPatternRe.exec(modeBuffer);
          let buf = "";
          while (match) {
            buf += modeBuffer.substring(lastIndex, match.index);
            const word = language.case_insensitive ? match[0].toLowerCase() : match[0];
            const data = keywordData(top, word);
            if (data) {
              const [kind, keywordRelevance] = data;
              emitter.addText(buf);
              buf = "";
              keywordHits[word] = (keywordHits[word] || 0) + 1;
              if (keywordHits[word] <= MAX_KEYWORD_HITS)
                relevance += keywordRelevance;
              if (kind.startsWith("_")) {
                buf += match[0];
              } else {
                const cssClass = language.classNameAliases[kind] || kind;
                emitKeyword(match[0], cssClass);
              }
            } else {
              buf += match[0];
            }
            lastIndex = top.keywordPatternRe.lastIndex;
            match = top.keywordPatternRe.exec(modeBuffer);
          }
          buf += modeBuffer.substring(lastIndex);
          emitter.addText(buf);
        }
        function processSubLanguage() {
          if (modeBuffer === "")
            return;
          let result2 = null;
          if (typeof top.subLanguage === "string") {
            if (!languages[top.subLanguage]) {
              emitter.addText(modeBuffer);
              return;
            }
            result2 = _highlight(top.subLanguage, modeBuffer, true, continuations[top.subLanguage]);
            continuations[top.subLanguage] = result2._top;
          } else {
            result2 = highlightAuto(modeBuffer, top.subLanguage.length ? top.subLanguage : null);
          }
          if (top.relevance > 0) {
            relevance += result2.relevance;
          }
          emitter.__addSublanguage(result2._emitter, result2.language);
        }
        function processBuffer() {
          if (top.subLanguage != null) {
            processSubLanguage();
          } else {
            processKeywords();
          }
          modeBuffer = "";
        }
        function emitKeyword(keyword, scope) {
          if (keyword === "")
            return;
          emitter.startScope(scope);
          emitter.addText(keyword);
          emitter.endScope();
        }
        function emitMultiClass(scope, match) {
          let i = 1;
          const max = match.length - 1;
          while (i <= max) {
            if (!scope._emit[i]) {
              i++;
              continue;
            }
            const klass = language.classNameAliases[scope[i]] || scope[i];
            const text = match[i];
            if (klass) {
              emitKeyword(text, klass);
            } else {
              modeBuffer = text;
              processKeywords();
              modeBuffer = "";
            }
            i++;
          }
        }
        function startNewMode(mode, match) {
          if (mode.scope && typeof mode.scope === "string") {
            emitter.openNode(language.classNameAliases[mode.scope] || mode.scope);
          }
          if (mode.beginScope) {
            if (mode.beginScope._wrap) {
              emitKeyword(modeBuffer, language.classNameAliases[mode.beginScope._wrap] || mode.beginScope._wrap);
              modeBuffer = "";
            } else if (mode.beginScope._multi) {
              emitMultiClass(mode.beginScope, match);
              modeBuffer = "";
            }
          }
          top = Object.create(mode, { parent: { value: top } });
          return top;
        }
        function endOfMode(mode, match, matchPlusRemainder) {
          let matched = startsWith(mode.endRe, matchPlusRemainder);
          if (matched) {
            if (mode["on:end"]) {
              const resp = new Response(mode);
              mode["on:end"](match, resp);
              if (resp.isMatchIgnored)
                matched = false;
            }
            if (matched) {
              while (mode.endsParent && mode.parent) {
                mode = mode.parent;
              }
              return mode;
            }
          }
          if (mode.endsWithParent) {
            return endOfMode(mode.parent, match, matchPlusRemainder);
          }
        }
        function doIgnore(lexeme) {
          if (top.matcher.regexIndex === 0) {
            modeBuffer += lexeme[0];
            return 1;
          } else {
            resumeScanAtSamePosition = true;
            return 0;
          }
        }
        function doBeginMatch(match) {
          const lexeme = match[0];
          const newMode = match.rule;
          const resp = new Response(newMode);
          const beforeCallbacks = [newMode.__beforeBegin, newMode["on:begin"]];
          for (const cb of beforeCallbacks) {
            if (!cb)
              continue;
            cb(match, resp);
            if (resp.isMatchIgnored)
              return doIgnore(lexeme);
          }
          if (newMode.skip) {
            modeBuffer += lexeme;
          } else {
            if (newMode.excludeBegin) {
              modeBuffer += lexeme;
            }
            processBuffer();
            if (!newMode.returnBegin && !newMode.excludeBegin) {
              modeBuffer = lexeme;
            }
          }
          startNewMode(newMode, match);
          return newMode.returnBegin ? 0 : lexeme.length;
        }
        function doEndMatch(match) {
          const lexeme = match[0];
          const matchPlusRemainder = codeToHighlight.substring(match.index);
          const endMode = endOfMode(top, match, matchPlusRemainder);
          if (!endMode) {
            return NO_MATCH;
          }
          const origin = top;
          if (top.endScope && top.endScope._wrap) {
            processBuffer();
            emitKeyword(lexeme, top.endScope._wrap);
          } else if (top.endScope && top.endScope._multi) {
            processBuffer();
            emitMultiClass(top.endScope, match);
          } else if (origin.skip) {
            modeBuffer += lexeme;
          } else {
            if (!(origin.returnEnd || origin.excludeEnd)) {
              modeBuffer += lexeme;
            }
            processBuffer();
            if (origin.excludeEnd) {
              modeBuffer = lexeme;
            }
          }
          do {
            if (top.scope) {
              emitter.closeNode();
            }
            if (!top.skip && !top.subLanguage) {
              relevance += top.relevance;
            }
            top = top.parent;
          } while (top !== endMode.parent);
          if (endMode.starts) {
            startNewMode(endMode.starts, match);
          }
          return origin.returnEnd ? 0 : lexeme.length;
        }
        function processContinuations() {
          const list = [];
          for (let current = top;current !== language; current = current.parent) {
            if (current.scope) {
              list.unshift(current.scope);
            }
          }
          list.forEach((item) => emitter.openNode(item));
        }
        let lastMatch = {};
        function processLexeme(textBeforeMatch, match) {
          const lexeme = match && match[0];
          modeBuffer += textBeforeMatch;
          if (lexeme == null) {
            processBuffer();
            return 0;
          }
          if (lastMatch.type === "begin" && match.type === "end" && lastMatch.index === match.index && lexeme === "") {
            modeBuffer += codeToHighlight.slice(match.index, match.index + 1);
            if (!SAFE_MODE) {
              const err = new Error(`0 width match regex (${languageName})`);
              err.languageName = languageName;
              err.badRule = lastMatch.rule;
              throw err;
            }
            return 1;
          }
          lastMatch = match;
          if (match.type === "begin") {
            return doBeginMatch(match);
          } else if (match.type === "illegal" && !ignoreIllegals) {
            const err = new Error('Illegal lexeme "' + lexeme + '" for mode "' + (top.scope || "<unnamed>") + '"');
            err.mode = top;
            throw err;
          } else if (match.type === "end") {
            const processed = doEndMatch(match);
            if (processed !== NO_MATCH) {
              return processed;
            }
          }
          if (match.type === "illegal" && lexeme === "") {
            modeBuffer += `
`;
            return 1;
          }
          if (iterations > 1e5 && iterations > match.index * 3) {
            const err = new Error("potential infinite loop, way more iterations than matches");
            throw err;
          }
          modeBuffer += lexeme;
          return lexeme.length;
        }
        const language = getLanguage(languageName);
        if (!language) {
          error(LANGUAGE_NOT_FOUND.replace("{}", languageName));
          throw new Error('Unknown language: "' + languageName + '"');
        }
        const md = compileLanguage(language);
        let result = "";
        let top = continuation || md;
        const continuations = {};
        const emitter = new options.__emitter(options);
        processContinuations();
        let modeBuffer = "";
        let relevance = 0;
        let index = 0;
        let iterations = 0;
        let resumeScanAtSamePosition = false;
        try {
          if (!language.__emitTokens) {
            top.matcher.considerAll();
            for (;; ) {
              iterations++;
              if (resumeScanAtSamePosition) {
                resumeScanAtSamePosition = false;
              } else {
                top.matcher.considerAll();
              }
              top.matcher.lastIndex = index;
              const match = top.matcher.exec(codeToHighlight);
              if (!match)
                break;
              const beforeMatch = codeToHighlight.substring(index, match.index);
              const processedCount = processLexeme(beforeMatch, match);
              index = match.index + processedCount;
            }
            processLexeme(codeToHighlight.substring(index));
          } else {
            language.__emitTokens(codeToHighlight, emitter);
          }
          emitter.finalize();
          result = emitter.toHTML();
          return {
            language: languageName,
            value: result,
            relevance,
            illegal: false,
            _emitter: emitter,
            _top: top
          };
        } catch (err) {
          if (err.message && err.message.includes("Illegal")) {
            return {
              language: languageName,
              value: escape(codeToHighlight),
              illegal: true,
              relevance: 0,
              _illegalBy: {
                message: err.message,
                index,
                context: codeToHighlight.slice(index - 100, index + 100),
                mode: err.mode,
                resultSoFar: result
              },
              _emitter: emitter
            };
          } else if (SAFE_MODE) {
            return {
              language: languageName,
              value: escape(codeToHighlight),
              illegal: false,
              relevance: 0,
              errorRaised: err,
              _emitter: emitter,
              _top: top
            };
          } else {
            throw err;
          }
        }
      }
      function justTextHighlightResult(code) {
        const result = {
          value: escape(code),
          illegal: false,
          relevance: 0,
          _top: PLAINTEXT_LANGUAGE,
          _emitter: new options.__emitter(options)
        };
        result._emitter.addText(code);
        return result;
      }
      function highlightAuto(code, languageSubset) {
        languageSubset = languageSubset || options.languages || Object.keys(languages);
        const plaintext = justTextHighlightResult(code);
        const results = languageSubset.filter(getLanguage).filter(autoDetection).map((name) => _highlight(name, code, false));
        results.unshift(plaintext);
        const sorted = results.sort((a, b2) => {
          if (a.relevance !== b2.relevance)
            return b2.relevance - a.relevance;
          if (a.language && b2.language) {
            if (getLanguage(a.language).supersetOf === b2.language) {
              return 1;
            } else if (getLanguage(b2.language).supersetOf === a.language) {
              return -1;
            }
          }
          return 0;
        });
        const [best, secondBest] = sorted;
        const result = best;
        result.secondBest = secondBest;
        return result;
      }
      function updateClassName(element, currentLang, resultLang) {
        const language = currentLang && aliases[currentLang] || resultLang;
        element.classList.add("hljs");
        element.classList.add(`language-${language}`);
      }
      function highlightElement(element) {
        let node = null;
        const language = blockLanguage(element);
        if (shouldNotHighlight(language))
          return;
        fire("before:highlightElement", { el: element, language });
        if (element.dataset.highlighted) {
          console.log("Element previously highlighted. To highlight again, first unset `dataset.highlighted`.", element);
          return;
        }
        if (element.children.length > 0) {
          if (!options.ignoreUnescapedHTML) {
            console.warn("One of your code blocks includes unescaped HTML. This is a potentially serious security risk.");
            console.warn("https://github.com/highlightjs/highlight.js/wiki/security");
            console.warn("The element with unescaped HTML:");
            console.warn(element);
          }
          if (options.throwUnescapedHTML) {
            const err = new HTMLInjectionError("One of your code blocks includes unescaped HTML.", element.innerHTML);
            throw err;
          }
        }
        node = element;
        const text = node.textContent;
        const result = language ? highlight2(text, { language, ignoreIllegals: true }) : highlightAuto(text);
        element.innerHTML = result.value;
        element.dataset.highlighted = "yes";
        updateClassName(element, language, result.language);
        element.result = {
          language: result.language,
          re: result.relevance,
          relevance: result.relevance
        };
        if (result.secondBest) {
          element.secondBest = {
            language: result.secondBest.language,
            relevance: result.secondBest.relevance
          };
        }
        fire("after:highlightElement", { el: element, result, text });
      }
      function configure(userOptions) {
        options = inherit(options, userOptions);
      }
      const initHighlighting = () => {
        highlightAll();
        deprecated("10.6.0", "initHighlighting() deprecated.  Use highlightAll() now.");
      };
      function initHighlightingOnLoad() {
        highlightAll();
        deprecated("10.6.0", "initHighlightingOnLoad() deprecated.  Use highlightAll() now.");
      }
      let wantsHighlight = false;
      function highlightAll() {
        function boot() {
          highlightAll();
        }
        if (document.readyState === "loading") {
          if (!wantsHighlight) {
            window.addEventListener("DOMContentLoaded", boot, false);
          }
          wantsHighlight = true;
          return;
        }
        const blocks = document.querySelectorAll(options.cssSelector);
        blocks.forEach(highlightElement);
      }
      function registerLanguage(languageName, languageDefinition) {
        let lang = null;
        try {
          lang = languageDefinition(hljs);
        } catch (error$1) {
          error("Language definition for '{}' could not be registered.".replace("{}", languageName));
          if (!SAFE_MODE) {
            throw error$1;
          } else {
            error(error$1);
          }
          lang = PLAINTEXT_LANGUAGE;
        }
        if (!lang.name)
          lang.name = languageName;
        languages[languageName] = lang;
        lang.rawDefinition = languageDefinition.bind(null, hljs);
        if (lang.aliases) {
          registerAliases(lang.aliases, { languageName });
        }
      }
      function unregisterLanguage(languageName) {
        delete languages[languageName];
        for (const alias of Object.keys(aliases)) {
          if (aliases[alias] === languageName) {
            delete aliases[alias];
          }
        }
      }
      function listLanguages() {
        return Object.keys(languages);
      }
      function getLanguage(name) {
        name = (name || "").toLowerCase();
        return languages[name] || languages[aliases[name]];
      }
      function registerAliases(aliasList, { languageName }) {
        if (typeof aliasList === "string") {
          aliasList = [aliasList];
        }
        aliasList.forEach((alias) => {
          aliases[alias.toLowerCase()] = languageName;
        });
      }
      function autoDetection(name) {
        const lang = getLanguage(name);
        return lang && !lang.disableAutodetect;
      }
      function upgradePluginAPI(plugin) {
        if (plugin["before:highlightBlock"] && !plugin["before:highlightElement"]) {
          plugin["before:highlightElement"] = (data) => {
            plugin["before:highlightBlock"](Object.assign({ block: data.el }, data));
          };
        }
        if (plugin["after:highlightBlock"] && !plugin["after:highlightElement"]) {
          plugin["after:highlightElement"] = (data) => {
            plugin["after:highlightBlock"](Object.assign({ block: data.el }, data));
          };
        }
      }
      function addPlugin(plugin) {
        upgradePluginAPI(plugin);
        plugins.push(plugin);
      }
      function removePlugin(plugin) {
        const index = plugins.indexOf(plugin);
        if (index !== -1) {
          plugins.splice(index, 1);
        }
      }
      function fire(event, args) {
        const cb = event;
        plugins.forEach(function(plugin) {
          if (plugin[cb]) {
            plugin[cb](args);
          }
        });
      }
      function deprecateHighlightBlock(el) {
        deprecated("10.7.0", "highlightBlock will be removed entirely in v12.0");
        deprecated("10.7.0", "Please use highlightElement now.");
        return highlightElement(el);
      }
      Object.assign(hljs, {
        highlight: highlight2,
        highlightAuto,
        highlightAll,
        highlightElement,
        highlightBlock: deprecateHighlightBlock,
        configure,
        initHighlighting,
        initHighlightingOnLoad,
        registerLanguage,
        unregisterLanguage,
        listLanguages,
        getLanguage,
        registerAliases,
        autoDetection,
        inherit,
        addPlugin,
        removePlugin
      });
      hljs.debugMode = function() {
        SAFE_MODE = false;
      };
      hljs.safeMode = function() {
        SAFE_MODE = true;
      };
      hljs.versionString = version;
      hljs.regex = {
        concat,
        lookahead,
        either,
        optional,
        anyNumberOfTimes
      };
      for (const key in MODES) {
        if (typeof MODES[key] === "object") {
          deepFreeze(MODES[key]);
        }
      }
      Object.assign(hljs, MODES);
      return hljs;
    };
    var highlight = HLJS({});
    highlight.newInstance = () => HLJS({});
    module.exports = highlight;
    highlight.HighlightJS = highlight;
    highlight.default = highlight;
  });

  // node_modules/marked/lib/marked.esm.js
  function M() {
    return { async: false, breaks: false, extensions: null, gfm: true, hooks: null, pedantic: false, renderer: null, silent: false, tokenizer: null, walkTokens: null };
  }
  var T = M();
  function G(u) {
    T = u;
  }
  var _ = { exec: () => null };
  function k(u, e = "") {
    let t = typeof u == "string" ? u : u.source, n = { replace: (r, i) => {
      let s = typeof i == "string" ? i : i.source;
      return s = s.replace(m.caret, "$1"), t = t.replace(r, s), n;
    }, getRegex: () => new RegExp(t, e) };
    return n;
  }
  var Re = (() => {
    try {
      return !!new RegExp("(?<=1)(?<!1)");
    } catch {
      return false;
    }
  })();
  var m = { codeRemoveIndent: /^(?: {1,4}| {0,3}\t)/gm, outputLinkReplace: /\\([\[\]])/g, indentCodeCompensation: /^(\s+)(?:```)/, beginningSpace: /^\s+/, endingHash: /#$/, startingSpaceChar: /^ /, endingSpaceChar: / $/, nonSpaceChar: /[^ ]/, newLineCharGlobal: /\n/g, tabCharGlobal: /\t/g, multipleSpaceGlobal: /\s+/g, blankLine: /^[ \t]*$/, doubleBlankLine: /\n[ \t]*\n[ \t]*$/, blockquoteStart: /^ {0,3}>/, blockquoteSetextReplace: /\n {0,3}((?:=+|-+) *)(?=\n|$)/g, blockquoteSetextReplace2: /^ {0,3}>[ \t]?/gm, listReplaceNesting: /^ {1,4}(?=( {4})*[^ ])/g, listIsTask: /^\[[ xX]\] +\S/, listReplaceTask: /^\[[ xX]\] +/, listTaskCheckbox: /\[[ xX]\]/, anyLine: /\n.*\n/, hrefBrackets: /^<(.*)>$/, tableDelimiter: /[:|]/, tableAlignChars: /^\||\| *$/g, tableRowBlankLine: /\n[ \t]*$/, tableAlignRight: /^ *-+: *$/, tableAlignCenter: /^ *:-+: *$/, tableAlignLeft: /^ *:-+ *$/, startATag: /^<a /i, endATag: /^<\/a>/i, startPreScriptTag: /^<(pre|code|kbd|script)(\s|>)/i, endPreScriptTag: /^<\/(pre|code|kbd|script)(\s|>)/i, startAngleBracket: /^</, endAngleBracket: />$/, pedanticHrefTitle: /^([^'"]*[^\s])\s+(['"])(.*)\2/, unicodeAlphaNumeric: /[\p{L}\p{N}]/u, escapeTest: /[&<>"']/, escapeReplace: /[&<>"']/g, escapeTestNoEncode: /[<>"']|&(?!(#\d{1,7}|#[Xx][a-fA-F0-9]{1,6}|\w+);)/, escapeReplaceNoEncode: /[<>"']|&(?!(#\d{1,7}|#[Xx][a-fA-F0-9]{1,6}|\w+);)/g, caret: /(^|[^\[])\^/g, percentDecode: /%25/g, findPipe: /\|/g, splitPipe: / \|/, slashPipe: /\\\|/g, carriageReturn: /\r\n|\r/g, spaceLine: /^ +$/gm, notSpaceStart: /^\S*/, endingNewline: /\n$/, listItemRegex: (u) => new RegExp(`^( {0,3}${u})((?:[	 ][^\\n]*)?(?:\\n|$))`), nextBulletRegex: (u) => new RegExp(`^ {0,${Math.min(3, u - 1)}}(?:[*+-]|\\d{1,9}[.)])((?:[ 	][^\\n]*)?(?:\\n|$))`), hrRegex: (u) => new RegExp(`^ {0,${Math.min(3, u - 1)}}((?:- *){3,}|(?:_ *){3,}|(?:\\* *){3,})(?:\\n+|$)`), fencesBeginRegex: (u) => new RegExp(`^ {0,${Math.min(3, u - 1)}}(?:\`\`\`|~~~)`), headingBeginRegex: (u) => new RegExp(`^ {0,${Math.min(3, u - 1)}}#`), htmlBeginRegex: (u) => new RegExp(`^ {0,${Math.min(3, u - 1)}}<(?:[a-z].*>|!--)`, "i"), blockquoteBeginRegex: (u) => new RegExp(`^ {0,${Math.min(3, u - 1)}}>`) };
  var Te = /^(?:[ \t]*(?:\n|$))+/;
  var Oe = /^((?: {4}| {0,3}\t)[^\n]+(?:\n(?:[ \t]*(?:\n|$))*)?)+/;
  var we = /^ {0,3}(`{3,}(?=[^`\n]*(?:\n|$))|~{3,})([^\n]*)(?:\n|$)(?:|([\s\S]*?)(?:\n|$))(?: {0,3}\1[~`]* *(?=\n|$)|$)/;
  var A = /^ {0,3}((?:-[\t ]*){3,}|(?:_[ \t]*){3,}|(?:\*[ \t]*){3,})(?:\n+|$)/;
  var ye = /^ {0,3}(#{1,6})(?=\s|$)(.*)(?:\n+|$)/;
  var N = / {0,3}(?:[*+-]|\d{1,9}[.)])/;
  var re = /^(?!bull |blockCode|fences|blockquote|heading|html|table)((?:.|\n(?!\s*?\n|bull |blockCode|fences|blockquote|heading|html|table))+?)\n {0,3}(=+|-+) *(?:\n+|$)/;
  var se = k(re).replace(/bull/g, N).replace(/blockCode/g, /(?: {4}| {0,3}\t)/).replace(/fences/g, / {0,3}(?:`{3,}|~{3,})/).replace(/blockquote/g, / {0,3}>/).replace(/heading/g, / {0,3}#{1,6}/).replace(/html/g, / {0,3}<[^\n>]+>\n/).replace(/\|table/g, "").getRegex();
  var Pe = k(re).replace(/bull/g, N).replace(/blockCode/g, /(?: {4}| {0,3}\t)/).replace(/fences/g, / {0,3}(?:`{3,}|~{3,})/).replace(/blockquote/g, / {0,3}>/).replace(/heading/g, / {0,3}#{1,6}/).replace(/html/g, / {0,3}<[^\n>]+>\n/).replace(/table/g, / {0,3}\|?(?:[:\- ]*\|)+[\:\- ]*\n/).getRegex();
  var Q = /^([^\n]+(?:\n(?!hr|heading|lheading|blockquote|fences|list|html|table| +\n)[^\n]+)*)/;
  var Se = /^[^\n]+/;
  var j = /(?!\s*\])(?:\\[\s\S]|[^\[\]\\])+/;
  var $e = k(/^ {0,3}\[(label)\]: *(?:\n[ \t]*)?([^<\s][^\s]*|<.*?>)(?:(?: +(?:\n[ \t]*)?| *\n[ \t]*)(title))? *(?:\n+|$)/).replace("label", j).replace("title", /(?:"(?:\\"?|[^"\\])*"|'[^'\n]*(?:\n[^'\n]+)*\n?'|\([^()]*\))/).getRegex();
  var _e = k(/^(bull)([ \t][^\n]+?)?(?:\n|$)/).replace(/bull/g, N).getRegex();
  var q = "address|article|aside|base|basefont|blockquote|body|caption|center|col|colgroup|dd|details|dialog|dir|div|dl|dt|fieldset|figcaption|figure|footer|form|frame|frameset|h[1-6]|head|header|hr|html|iframe|legend|li|link|main|menu|menuitem|meta|nav|noframes|ol|optgroup|option|p|param|search|section|summary|table|tbody|td|tfoot|th|thead|title|tr|track|ul";
  var F = /<!--(?:-?>|[\s\S]*?(?:-->|$))/;
  var Le = k("^ {0,3}(?:<(script|pre|style|textarea)[\\s>][\\s\\S]*?(?:</\\1>[^\\n]*\\n+|$)|comment[^\\n]*(\\n+|$)|<\\?[\\s\\S]*?(?:\\?>\\n*|$)|<![A-Z][\\s\\S]*?(?:>\\n*|$)|<!\\[CDATA\\[[\\s\\S]*?(?:\\]\\]>\\n*|$)|</?(tag)(?: +|\\n|/?>)[\\s\\S]*?(?:(?:\\n[ \t]*)+\\n|$)|<(?!script|pre|style|textarea)([a-z][\\w-]*)(?:attribute)*? */?>(?=[ \\t]*(?:\\n|$))[\\s\\S]*?(?:(?:\\n[ \t]*)+\\n|$)|</(?!script|pre|style|textarea)[a-z][\\w-]*\\s*>(?=[ \\t]*(?:\\n|$))[\\s\\S]*?(?:(?:\\n[ \t]*)+\\n|$))", "i").replace("comment", F).replace("tag", q).replace("attribute", / +[a-zA-Z:_][\w.:-]*(?: *= *"[^"\n]*"| *= *'[^'\n]*'| *= *[^\s"'=<>`]+)?/).getRegex();
  var ie = k(Q).replace("hr", A).replace("heading", " {0,3}#{1,6}(?:\\s|$)").replace("|lheading", "").replace("|table", "").replace("blockquote", " {0,3}>").replace("fences", " {0,3}(?:`{3,}(?=[^`\\n]*\\n)|~{3,})[^\\n]*\\n").replace("list", " {0,3}(?:[*+-]|1[.)])[ \\t]").replace("html", "</?(?:tag)(?: +|\\n|/?>)|<(?:script|pre|style|textarea|!--)").replace("tag", q).getRegex();
  var Me = k(/^( {0,3}> ?(paragraph|[^\n]*)(?:\n|$))+/).replace("paragraph", ie).getRegex();
  var U = { blockquote: Me, code: Oe, def: $e, fences: we, heading: ye, hr: A, html: Le, lheading: se, list: _e, newline: Te, paragraph: ie, table: _, text: Se };
  var te = k("^ *([^\\n ].*)\\n {0,3}((?:\\| *)?:?-+:? *(?:\\| *:?-+:? *)*(?:\\| *)?)(?:\\n((?:(?! *\\n|hr|heading|blockquote|code|fences|list|html).*(?:\\n|$))*)\\n*|$)").replace("hr", A).replace("heading", " {0,3}#{1,6}(?:\\s|$)").replace("blockquote", " {0,3}>").replace("code", "(?: {4}| {0,3}\t)[^\\n]").replace("fences", " {0,3}(?:`{3,}(?=[^`\\n]*\\n)|~{3,})[^\\n]*\\n").replace("list", " {0,3}(?:[*+-]|1[.)])[ \\t]").replace("html", "</?(?:tag)(?: +|\\n|/?>)|<(?:script|pre|style|textarea|!--)").replace("tag", q).getRegex();
  var ze = { ...U, lheading: Pe, table: te, paragraph: k(Q).replace("hr", A).replace("heading", " {0,3}#{1,6}(?:\\s|$)").replace("|lheading", "").replace("table", te).replace("blockquote", " {0,3}>").replace("fences", " {0,3}(?:`{3,}(?=[^`\\n]*\\n)|~{3,})[^\\n]*\\n").replace("list", " {0,3}(?:[*+-]|1[.)])[ \\t]").replace("html", "</?(?:tag)(?: +|\\n|/?>)|<(?:script|pre|style|textarea|!--)").replace("tag", q).getRegex() };
  var Ee = { ...U, html: k(`^ *(?:comment *(?:\\n|\\s*$)|<(tag)[\\s\\S]+?</\\1> *(?:\\n{2,}|\\s*$)|<tag(?:"[^"]*"|'[^']*'|\\s[^'"/>\\s]*)*?/?> *(?:\\n{2,}|\\s*$))`).replace("comment", F).replace(/tag/g, "(?!(?:a|em|strong|small|s|cite|q|dfn|abbr|data|time|code|var|samp|kbd|sub|sup|i|b|u|mark|ruby|rt|rp|bdi|bdo|span|br|wbr|ins|del|img)\\b)\\w+(?!:|[^\\w\\s@]*@)\\b").getRegex(), def: /^ *\[([^\]]+)\]: *<?([^\s>]+)>?(?: +(["(][^\n]+[")]))? *(?:\n+|$)/, heading: /^(#{1,6})(.*)(?:\n+|$)/, fences: _, lheading: /^(.+?)\n {0,3}(=+|-+) *(?:\n+|$)/, paragraph: k(Q).replace("hr", A).replace("heading", ` *#{1,6} *[^
]`).replace("lheading", se).replace("|table", "").replace("blockquote", " {0,3}>").replace("|fences", "").replace("|list", "").replace("|html", "").replace("|tag", "").getRegex() };
  var Ie = /^\\([!"#$%&'()*+,\-./:;<=>?@\[\]\\^_`{|}~])/;
  var Ae = /^(`+)([^`]|[^`][\s\S]*?[^`])\1(?!`)/;
  var oe = /^( {2,}|\\)\n(?!\s*$)/;
  var Ce = /^(`+|[^`])(?:(?= {2,}\n)|[\s\S]*?(?:(?=[\\<!\[`*_]|\b_|$)|[^ ](?= {2,}\n)))/;
  var v = /[\p{P}\p{S}]/u;
  var K = /[\s\p{P}\p{S}]/u;
  var ae = /[^\s\p{P}\p{S}]/u;
  var Be = k(/^((?![*_])punctSpace)/, "u").replace(/punctSpace/g, K).getRegex();
  var le = /(?!~)[\p{P}\p{S}]/u;
  var De = /(?!~)[\s\p{P}\p{S}]/u;
  var qe = /(?:[^\s\p{P}\p{S}]|~)/u;
  var ue = /(?![*_])[\p{P}\p{S}]/u;
  var ve = /(?![*_])[\s\p{P}\p{S}]/u;
  var He = /(?:[^\s\p{P}\p{S}]|[*_])/u;
  var Ge = k(/link|precode-code|html/, "g").replace("link", /\[(?:[^\[\]`]|(?<a>`+)[^`]+\k<a>(?!`))*?\]\((?:\\[\s\S]|[^\\\(\)]|\((?:\\[\s\S]|[^\\\(\)])*\))*\)/).replace("precode-", Re ? "(?<!`)()" : "(^^|[^`])").replace("code", /(?<b>`+)[^`]+\k<b>(?!`)/).replace("html", /<(?! )[^<>]*?>/).getRegex();
  var pe = /^(?:\*+(?:((?!\*)punct)|[^\s*]))|^_+(?:((?!_)punct)|([^\s_]))/;
  var Ze = k(pe, "u").replace(/punct/g, v).getRegex();
  var Ne = k(pe, "u").replace(/punct/g, le).getRegex();
  var ce = "^[^_*]*?__[^_*]*?\\*[^_*]*?(?=__)|[^*]+(?=[^*])|(?!\\*)punct(\\*+)(?=[\\s]|$)|notPunctSpace(\\*+)(?!\\*)(?=punctSpace|$)|(?!\\*)punctSpace(\\*+)(?=notPunctSpace)|[\\s](\\*+)(?!\\*)(?=punct)|(?!\\*)punct(\\*+)(?!\\*)(?=punct)|notPunctSpace(\\*+)(?=notPunctSpace)";
  var Qe = k(ce, "gu").replace(/notPunctSpace/g, ae).replace(/punctSpace/g, K).replace(/punct/g, v).getRegex();
  var je = k(ce, "gu").replace(/notPunctSpace/g, qe).replace(/punctSpace/g, De).replace(/punct/g, le).getRegex();
  var Fe = k("^[^_*]*?\\*\\*[^_*]*?_[^_*]*?(?=\\*\\*)|[^_]+(?=[^_])|(?!_)punct(_+)(?=[\\s]|$)|notPunctSpace(_+)(?!_)(?=punctSpace|$)|(?!_)punctSpace(_+)(?=notPunctSpace)|[\\s](_+)(?!_)(?=punct)|(?!_)punct(_+)(?!_)(?=punct)", "gu").replace(/notPunctSpace/g, ae).replace(/punctSpace/g, K).replace(/punct/g, v).getRegex();
  var Ue = k(/^~~?(?:((?!~)punct)|[^\s~])/, "u").replace(/punct/g, ue).getRegex();
  var Ke = "^[^~]+(?=[^~])|(?!~)punct(~~?)(?=[\\s]|$)|notPunctSpace(~~?)(?!~)(?=punctSpace|$)|(?!~)punctSpace(~~?)(?=notPunctSpace)|[\\s](~~?)(?!~)(?=punct)|(?!~)punct(~~?)(?!~)(?=punct)|notPunctSpace(~~?)(?=notPunctSpace)";
  var We = k(Ke, "gu").replace(/notPunctSpace/g, He).replace(/punctSpace/g, ve).replace(/punct/g, ue).getRegex();
  var Xe = k(/\\(punct)/, "gu").replace(/punct/g, v).getRegex();
  var Je = k(/^<(scheme:[^\s\x00-\x1f<>]*|email)>/).replace("scheme", /[a-zA-Z][a-zA-Z0-9+.-]{1,31}/).replace("email", /[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+(@)[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)+(?![-_])/).getRegex();
  var Ve = k(F).replace("(?:-->|$)", "-->").getRegex();
  var Ye = k("^comment|^</[a-zA-Z][\\w:-]*\\s*>|^<[a-zA-Z][\\w-]*(?:attribute)*?\\s*/?>|^<\\?[\\s\\S]*?\\?>|^<![a-zA-Z]+\\s[\\s\\S]*?>|^<!\\[CDATA\\[[\\s\\S]*?\\]\\]>").replace("comment", Ve).replace("attribute", /\s+[a-zA-Z:_][\w.:-]*(?:\s*=\s*"[^"]*"|\s*=\s*'[^']*'|\s*=\s*[^\s"'=<>`]+)?/).getRegex();
  var D = /(?:\[(?:\\[\s\S]|[^\[\]\\])*\]|\\[\s\S]|`+[^`]*?`+(?!`)|[^\[\]\\`])*?/;
  var et = k(/^!?\[(label)\]\(\s*(href)(?:(?:[ \t]+(?:\n[ \t]*)?|\n[ \t]*)(title))?\s*\)/).replace("label", D).replace("href", /<(?:\\.|[^\n<>\\])+>|[^ \t\n\x00-\x1f]*/).replace("title", /"(?:\\"?|[^"\\])*"|'(?:\\'?|[^'\\])*'|\((?:\\\)?|[^)\\])*\)/).getRegex();
  var he = k(/^!?\[(label)\]\[(ref)\]/).replace("label", D).replace("ref", j).getRegex();
  var ke = k(/^!?\[(ref)\](?:\[\])?/).replace("ref", j).getRegex();
  var tt = k("reflink|nolink(?!\\()", "g").replace("reflink", he).replace("nolink", ke).getRegex();
  var ne = /[hH][tT][tT][pP][sS]?|[fF][tT][pP]/;
  var W = { _backpedal: _, anyPunctuation: Xe, autolink: Je, blockSkip: Ge, br: oe, code: Ae, del: _, delLDelim: _, delRDelim: _, emStrongLDelim: Ze, emStrongRDelimAst: Qe, emStrongRDelimUnd: Fe, escape: Ie, link: et, nolink: ke, punctuation: Be, reflink: he, reflinkSearch: tt, tag: Ye, text: Ce, url: _ };
  var nt = { ...W, link: k(/^!?\[(label)\]\((.*?)\)/).replace("label", D).getRegex(), reflink: k(/^!?\[(label)\]\s*\[([^\]]*)\]/).replace("label", D).getRegex() };
  var Z = { ...W, emStrongRDelimAst: je, emStrongLDelim: Ne, delLDelim: Ue, delRDelim: We, url: k(/^((?:protocol):\/\/|www\.)(?:[a-zA-Z0-9\-]+\.?)+[^\s<]*|^email/).replace("protocol", ne).replace("email", /[A-Za-z0-9._+-]+(@)[a-zA-Z0-9-_]+(?:\.[a-zA-Z0-9-_]*[a-zA-Z0-9])+(?![-_])/).getRegex(), _backpedal: /(?:[^?!.,:;*_'"~()&]+|\([^)]*\)|&(?![a-zA-Z0-9]+;$)|[?!.,:;*_'"~)]+(?!$))+/, del: /^(~~?)(?=[^\s~])((?:\\[\s\S]|[^\\])*?(?:\\[\s\S]|[^\s~\\]))\1(?=[^~]|$)/, text: k(/^([`~]+|[^`~])(?:(?= {2,}\n)|(?=[a-zA-Z0-9.!#$%&'*+\/=?_`{\|}~-]+@)|[\s\S]*?(?:(?=[\\<!\[`*~_]|\b_|protocol:\/\/|www\.|$)|[^ ](?= {2,}\n)|[^a-zA-Z0-9.!#$%&'*+\/=?_`{\|}~-](?=[a-zA-Z0-9.!#$%&'*+\/=?_`{\|}~-]+@)))/).replace("protocol", ne).getRegex() };
  var rt = { ...Z, br: k(oe).replace("{2,}", "*").getRegex(), text: k(Z.text).replace("\\b_", "\\b_| {2,}\\n").replace(/\{2,\}/g, "*").getRegex() };
  var C = { normal: U, gfm: ze, pedantic: Ee };
  var z = { normal: W, gfm: Z, breaks: rt, pedantic: nt };
  var st = { "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" };
  var de = (u) => st[u];
  function O(u, e) {
    if (e) {
      if (m.escapeTest.test(u))
        return u.replace(m.escapeReplace, de);
    } else if (m.escapeTestNoEncode.test(u))
      return u.replace(m.escapeReplaceNoEncode, de);
    return u;
  }
  function X(u) {
    try {
      u = encodeURI(u).replace(m.percentDecode, "%");
    } catch {
      return null;
    }
    return u;
  }
  function J(u, e) {
    let t = u.replace(m.findPipe, (i, s, a) => {
      let o = false, l = s;
      for (;--l >= 0 && a[l] === "\\"; )
        o = !o;
      return o ? "|" : " |";
    }), n = t.split(m.splitPipe), r = 0;
    if (n[0].trim() || n.shift(), n.length > 0 && !n.at(-1)?.trim() && n.pop(), e)
      if (n.length > e)
        n.splice(e);
      else
        for (;n.length < e; )
          n.push("");
    for (;r < n.length; r++)
      n[r] = n[r].trim().replace(m.slashPipe, "|");
    return n;
  }
  function E(u, e, t) {
    let n = u.length;
    if (n === 0)
      return "";
    let r = 0;
    for (;r < n; ) {
      let i = u.charAt(n - r - 1);
      if (i === e && !t)
        r++;
      else if (i !== e && t)
        r++;
      else
        break;
    }
    return u.slice(0, n - r);
  }
  function ge(u, e) {
    if (u.indexOf(e[1]) === -1)
      return -1;
    let t = 0;
    for (let n = 0;n < u.length; n++)
      if (u[n] === "\\")
        n++;
      else if (u[n] === e[0])
        t++;
      else if (u[n] === e[1] && (t--, t < 0))
        return n;
    return t > 0 ? -2 : -1;
  }
  function fe(u, e = 0) {
    let t = e, n = "";
    for (let r of u)
      if (r === "\t") {
        let i = 4 - t % 4;
        n += " ".repeat(i), t += i;
      } else
        n += r, t++;
    return n;
  }
  function me(u, e, t, n, r) {
    let i = e.href, s = e.title || null, a = u[1].replace(r.other.outputLinkReplace, "$1");
    n.state.inLink = true;
    let o = { type: u[0].charAt(0) === "!" ? "image" : "link", raw: t, href: i, title: s, text: a, tokens: n.inlineTokens(a) };
    return n.state.inLink = false, o;
  }
  function it(u, e, t) {
    let n = u.match(t.other.indentCodeCompensation);
    if (n === null)
      return e;
    let r = n[1];
    return e.split(`
`).map((i) => {
      let s = i.match(t.other.beginningSpace);
      if (s === null)
        return i;
      let [a] = s;
      return a.length >= r.length ? i.slice(r.length) : i;
    }).join(`
`);
  }
  var w = class {
    options;
    rules;
    lexer;
    constructor(e) {
      this.options = e || T;
    }
    space(e) {
      let t = this.rules.block.newline.exec(e);
      if (t && t[0].length > 0)
        return { type: "space", raw: t[0] };
    }
    code(e) {
      let t = this.rules.block.code.exec(e);
      if (t) {
        let n = t[0].replace(this.rules.other.codeRemoveIndent, "");
        return { type: "code", raw: t[0], codeBlockStyle: "indented", text: this.options.pedantic ? n : E(n, `
`) };
      }
    }
    fences(e) {
      let t = this.rules.block.fences.exec(e);
      if (t) {
        let n = t[0], r = it(n, t[3] || "", this.rules);
        return { type: "code", raw: n, lang: t[2] ? t[2].trim().replace(this.rules.inline.anyPunctuation, "$1") : t[2], text: r };
      }
    }
    heading(e) {
      let t = this.rules.block.heading.exec(e);
      if (t) {
        let n = t[2].trim();
        if (this.rules.other.endingHash.test(n)) {
          let r = E(n, "#");
          (this.options.pedantic || !r || this.rules.other.endingSpaceChar.test(r)) && (n = r.trim());
        }
        return { type: "heading", raw: t[0], depth: t[1].length, text: n, tokens: this.lexer.inline(n) };
      }
    }
    hr(e) {
      let t = this.rules.block.hr.exec(e);
      if (t)
        return { type: "hr", raw: E(t[0], `
`) };
    }
    blockquote(e) {
      let t = this.rules.block.blockquote.exec(e);
      if (t) {
        let n = E(t[0], `
`).split(`
`), r = "", i = "", s = [];
        for (;n.length > 0; ) {
          let a = false, o = [], l;
          for (l = 0;l < n.length; l++)
            if (this.rules.other.blockquoteStart.test(n[l]))
              o.push(n[l]), a = true;
            else if (!a)
              o.push(n[l]);
            else
              break;
          n = n.slice(l);
          let p = o.join(`
`), c = p.replace(this.rules.other.blockquoteSetextReplace, `
    $1`).replace(this.rules.other.blockquoteSetextReplace2, "");
          r = r ? `${r}
${p}` : p, i = i ? `${i}
${c}` : c;
          let d = this.lexer.state.top;
          if (this.lexer.state.top = true, this.lexer.blockTokens(c, s, true), this.lexer.state.top = d, n.length === 0)
            break;
          let h = s.at(-1);
          if (h?.type === "code")
            break;
          if (h?.type === "blockquote") {
            let R = h, f = R.raw + `
` + n.join(`
`), S = this.blockquote(f);
            s[s.length - 1] = S, r = r.substring(0, r.length - R.raw.length) + S.raw, i = i.substring(0, i.length - R.text.length) + S.text;
            break;
          } else if (h?.type === "list") {
            let R = h, f = R.raw + `
` + n.join(`
`), S = this.list(f);
            s[s.length - 1] = S, r = r.substring(0, r.length - h.raw.length) + S.raw, i = i.substring(0, i.length - R.raw.length) + S.raw, n = f.substring(s.at(-1).raw.length).split(`
`);
            continue;
          }
        }
        return { type: "blockquote", raw: r, tokens: s, text: i };
      }
    }
    list(e) {
      let t = this.rules.block.list.exec(e);
      if (t) {
        let n = t[1].trim(), r = n.length > 1, i = { type: "list", raw: "", ordered: r, start: r ? +n.slice(0, -1) : "", loose: false, items: [] };
        n = r ? `\\d{1,9}\\${n.slice(-1)}` : `\\${n}`, this.options.pedantic && (n = r ? n : "[*+-]");
        let s = this.rules.other.listItemRegex(n), a = false;
        for (;e; ) {
          let l = false, p = "", c = "";
          if (!(t = s.exec(e)) || this.rules.block.hr.test(e))
            break;
          p = t[0], e = e.substring(p.length);
          let d = fe(t[2].split(`
`, 1)[0], t[1].length), h = e.split(`
`, 1)[0], R = !d.trim(), f = 0;
          if (this.options.pedantic ? (f = 2, c = d.trimStart()) : R ? f = t[1].length + 1 : (f = d.search(this.rules.other.nonSpaceChar), f = f > 4 ? 1 : f, c = d.slice(f), f += t[1].length), R && this.rules.other.blankLine.test(h) && (p += h + `
`, e = e.substring(h.length + 1), l = true), !l) {
            let S = this.rules.other.nextBulletRegex(f), V = this.rules.other.hrRegex(f), Y = this.rules.other.fencesBeginRegex(f), ee = this.rules.other.headingBeginRegex(f), xe = this.rules.other.htmlBeginRegex(f), be = this.rules.other.blockquoteBeginRegex(f);
            for (;e; ) {
              let H = e.split(`
`, 1)[0], I;
              if (h = H, this.options.pedantic ? (h = h.replace(this.rules.other.listReplaceNesting, "  "), I = h) : I = h.replace(this.rules.other.tabCharGlobal, "    "), Y.test(h) || ee.test(h) || xe.test(h) || be.test(h) || S.test(h) || V.test(h))
                break;
              if (I.search(this.rules.other.nonSpaceChar) >= f || !h.trim())
                c += `
` + I.slice(f);
              else {
                if (R || d.replace(this.rules.other.tabCharGlobal, "    ").search(this.rules.other.nonSpaceChar) >= 4 || Y.test(d) || ee.test(d) || V.test(d))
                  break;
                c += `
` + h;
              }
              R = !h.trim(), p += H + `
`, e = e.substring(H.length + 1), d = I.slice(f);
            }
          }
          i.loose || (a ? i.loose = true : this.rules.other.doubleBlankLine.test(p) && (a = true)), i.items.push({ type: "list_item", raw: p, task: !!this.options.gfm && this.rules.other.listIsTask.test(c), loose: false, text: c, tokens: [] }), i.raw += p;
        }
        let o = i.items.at(-1);
        if (o)
          o.raw = o.raw.trimEnd(), o.text = o.text.trimEnd();
        else
          return;
        i.raw = i.raw.trimEnd();
        for (let l of i.items) {
          if (this.lexer.state.top = false, l.tokens = this.lexer.blockTokens(l.text, []), l.task) {
            if (l.text = l.text.replace(this.rules.other.listReplaceTask, ""), l.tokens[0]?.type === "text" || l.tokens[0]?.type === "paragraph") {
              l.tokens[0].raw = l.tokens[0].raw.replace(this.rules.other.listReplaceTask, ""), l.tokens[0].text = l.tokens[0].text.replace(this.rules.other.listReplaceTask, "");
              for (let c = this.lexer.inlineQueue.length - 1;c >= 0; c--)
                if (this.rules.other.listIsTask.test(this.lexer.inlineQueue[c].src)) {
                  this.lexer.inlineQueue[c].src = this.lexer.inlineQueue[c].src.replace(this.rules.other.listReplaceTask, "");
                  break;
                }
            }
            let p = this.rules.other.listTaskCheckbox.exec(l.raw);
            if (p) {
              let c = { type: "checkbox", raw: p[0] + " ", checked: p[0] !== "[ ]" };
              l.checked = c.checked, i.loose ? l.tokens[0] && ["paragraph", "text"].includes(l.tokens[0].type) && "tokens" in l.tokens[0] && l.tokens[0].tokens ? (l.tokens[0].raw = c.raw + l.tokens[0].raw, l.tokens[0].text = c.raw + l.tokens[0].text, l.tokens[0].tokens.unshift(c)) : l.tokens.unshift({ type: "paragraph", raw: c.raw, text: c.raw, tokens: [c] }) : l.tokens.unshift(c);
            }
          }
          if (!i.loose) {
            let p = l.tokens.filter((d) => d.type === "space"), c = p.length > 0 && p.some((d) => this.rules.other.anyLine.test(d.raw));
            i.loose = c;
          }
        }
        if (i.loose)
          for (let l of i.items) {
            l.loose = true;
            for (let p of l.tokens)
              p.type === "text" && (p.type = "paragraph");
          }
        return i;
      }
    }
    html(e) {
      let t = this.rules.block.html.exec(e);
      if (t)
        return { type: "html", block: true, raw: t[0], pre: t[1] === "pre" || t[1] === "script" || t[1] === "style", text: t[0] };
    }
    def(e) {
      let t = this.rules.block.def.exec(e);
      if (t) {
        let n = t[1].toLowerCase().replace(this.rules.other.multipleSpaceGlobal, " "), r = t[2] ? t[2].replace(this.rules.other.hrefBrackets, "$1").replace(this.rules.inline.anyPunctuation, "$1") : "", i = t[3] ? t[3].substring(1, t[3].length - 1).replace(this.rules.inline.anyPunctuation, "$1") : t[3];
        return { type: "def", tag: n, raw: t[0], href: r, title: i };
      }
    }
    table(e) {
      let t = this.rules.block.table.exec(e);
      if (!t || !this.rules.other.tableDelimiter.test(t[2]))
        return;
      let n = J(t[1]), r = t[2].replace(this.rules.other.tableAlignChars, "").split("|"), i = t[3]?.trim() ? t[3].replace(this.rules.other.tableRowBlankLine, "").split(`
`) : [], s = { type: "table", raw: t[0], header: [], align: [], rows: [] };
      if (n.length === r.length) {
        for (let a of r)
          this.rules.other.tableAlignRight.test(a) ? s.align.push("right") : this.rules.other.tableAlignCenter.test(a) ? s.align.push("center") : this.rules.other.tableAlignLeft.test(a) ? s.align.push("left") : s.align.push(null);
        for (let a = 0;a < n.length; a++)
          s.header.push({ text: n[a], tokens: this.lexer.inline(n[a]), header: true, align: s.align[a] });
        for (let a of i)
          s.rows.push(J(a, s.header.length).map((o, l) => ({ text: o, tokens: this.lexer.inline(o), header: false, align: s.align[l] })));
        return s;
      }
    }
    lheading(e) {
      let t = this.rules.block.lheading.exec(e);
      if (t)
        return { type: "heading", raw: t[0], depth: t[2].charAt(0) === "=" ? 1 : 2, text: t[1], tokens: this.lexer.inline(t[1]) };
    }
    paragraph(e) {
      let t = this.rules.block.paragraph.exec(e);
      if (t) {
        let n = t[1].charAt(t[1].length - 1) === `
` ? t[1].slice(0, -1) : t[1];
        return { type: "paragraph", raw: t[0], text: n, tokens: this.lexer.inline(n) };
      }
    }
    text(e) {
      let t = this.rules.block.text.exec(e);
      if (t)
        return { type: "text", raw: t[0], text: t[0], tokens: this.lexer.inline(t[0]) };
    }
    escape(e) {
      let t = this.rules.inline.escape.exec(e);
      if (t)
        return { type: "escape", raw: t[0], text: t[1] };
    }
    tag(e) {
      let t = this.rules.inline.tag.exec(e);
      if (t)
        return !this.lexer.state.inLink && this.rules.other.startATag.test(t[0]) ? this.lexer.state.inLink = true : this.lexer.state.inLink && this.rules.other.endATag.test(t[0]) && (this.lexer.state.inLink = false), !this.lexer.state.inRawBlock && this.rules.other.startPreScriptTag.test(t[0]) ? this.lexer.state.inRawBlock = true : this.lexer.state.inRawBlock && this.rules.other.endPreScriptTag.test(t[0]) && (this.lexer.state.inRawBlock = false), { type: "html", raw: t[0], inLink: this.lexer.state.inLink, inRawBlock: this.lexer.state.inRawBlock, block: false, text: t[0] };
    }
    link(e) {
      let t = this.rules.inline.link.exec(e);
      if (t) {
        let n = t[2].trim();
        if (!this.options.pedantic && this.rules.other.startAngleBracket.test(n)) {
          if (!this.rules.other.endAngleBracket.test(n))
            return;
          let s = E(n.slice(0, -1), "\\");
          if ((n.length - s.length) % 2 === 0)
            return;
        } else {
          let s = ge(t[2], "()");
          if (s === -2)
            return;
          if (s > -1) {
            let o = (t[0].indexOf("!") === 0 ? 5 : 4) + t[1].length + s;
            t[2] = t[2].substring(0, s), t[0] = t[0].substring(0, o).trim(), t[3] = "";
          }
        }
        let r = t[2], i = "";
        if (this.options.pedantic) {
          let s = this.rules.other.pedanticHrefTitle.exec(r);
          s && (r = s[1], i = s[3]);
        } else
          i = t[3] ? t[3].slice(1, -1) : "";
        return r = r.trim(), this.rules.other.startAngleBracket.test(r) && (this.options.pedantic && !this.rules.other.endAngleBracket.test(n) ? r = r.slice(1) : r = r.slice(1, -1)), me(t, { href: r && r.replace(this.rules.inline.anyPunctuation, "$1"), title: i && i.replace(this.rules.inline.anyPunctuation, "$1") }, t[0], this.lexer, this.rules);
      }
    }
    reflink(e, t) {
      let n;
      if ((n = this.rules.inline.reflink.exec(e)) || (n = this.rules.inline.nolink.exec(e))) {
        let r = (n[2] || n[1]).replace(this.rules.other.multipleSpaceGlobal, " "), i = t[r.toLowerCase()];
        if (!i) {
          let s = n[0].charAt(0);
          return { type: "text", raw: s, text: s };
        }
        return me(n, i, n[0], this.lexer, this.rules);
      }
    }
    emStrong(e, t, n = "") {
      let r = this.rules.inline.emStrongLDelim.exec(e);
      if (!r || r[3] && n.match(this.rules.other.unicodeAlphaNumeric))
        return;
      if (!(r[1] || r[2] || "") || !n || this.rules.inline.punctuation.exec(n)) {
        let s = [...r[0]].length - 1, a, o, l = s, p = 0, c = r[0][0] === "*" ? this.rules.inline.emStrongRDelimAst : this.rules.inline.emStrongRDelimUnd;
        for (c.lastIndex = 0, t = t.slice(-1 * e.length + s);(r = c.exec(t)) != null; ) {
          if (a = r[1] || r[2] || r[3] || r[4] || r[5] || r[6], !a)
            continue;
          if (o = [...a].length, r[3] || r[4]) {
            l += o;
            continue;
          } else if ((r[5] || r[6]) && s % 3 && !((s + o) % 3)) {
            p += o;
            continue;
          }
          if (l -= o, l > 0)
            continue;
          o = Math.min(o, o + l + p);
          let d = [...r[0]][0].length, h = e.slice(0, s + r.index + d + o);
          if (Math.min(s, o) % 2) {
            let f = h.slice(1, -1);
            return { type: "em", raw: h, text: f, tokens: this.lexer.inlineTokens(f) };
          }
          let R = h.slice(2, -2);
          return { type: "strong", raw: h, text: R, tokens: this.lexer.inlineTokens(R) };
        }
      }
    }
    codespan(e) {
      let t = this.rules.inline.code.exec(e);
      if (t) {
        let n = t[2].replace(this.rules.other.newLineCharGlobal, " "), r = this.rules.other.nonSpaceChar.test(n), i = this.rules.other.startingSpaceChar.test(n) && this.rules.other.endingSpaceChar.test(n);
        return r && i && (n = n.substring(1, n.length - 1)), { type: "codespan", raw: t[0], text: n };
      }
    }
    br(e) {
      let t = this.rules.inline.br.exec(e);
      if (t)
        return { type: "br", raw: t[0] };
    }
    del(e, t, n = "") {
      let r = this.rules.inline.delLDelim.exec(e);
      if (!r)
        return;
      if (!(r[1] || "") || !n || this.rules.inline.punctuation.exec(n)) {
        let s = [...r[0]].length - 1, a, o, l = s, p = this.rules.inline.delRDelim;
        for (p.lastIndex = 0, t = t.slice(-1 * e.length + s);(r = p.exec(t)) != null; ) {
          if (a = r[1] || r[2] || r[3] || r[4] || r[5] || r[6], !a || (o = [...a].length, o !== s))
            continue;
          if (r[3] || r[4]) {
            l += o;
            continue;
          }
          if (l -= o, l > 0)
            continue;
          o = Math.min(o, o + l);
          let c = [...r[0]][0].length, d = e.slice(0, s + r.index + c + o), h = d.slice(s, -s);
          return { type: "del", raw: d, text: h, tokens: this.lexer.inlineTokens(h) };
        }
      }
    }
    autolink(e) {
      let t = this.rules.inline.autolink.exec(e);
      if (t) {
        let n, r;
        return t[2] === "@" ? (n = t[1], r = "mailto:" + n) : (n = t[1], r = n), { type: "link", raw: t[0], text: n, href: r, tokens: [{ type: "text", raw: n, text: n }] };
      }
    }
    url(e) {
      let t;
      if (t = this.rules.inline.url.exec(e)) {
        let n, r;
        if (t[2] === "@")
          n = t[0], r = "mailto:" + n;
        else {
          let i;
          do
            i = t[0], t[0] = this.rules.inline._backpedal.exec(t[0])?.[0] ?? "";
          while (i !== t[0]);
          n = t[0], t[1] === "www." ? r = "http://" + t[0] : r = t[0];
        }
        return { type: "link", raw: t[0], text: n, href: r, tokens: [{ type: "text", raw: n, text: n }] };
      }
    }
    inlineText(e) {
      let t = this.rules.inline.text.exec(e);
      if (t) {
        let n = this.lexer.state.inRawBlock;
        return { type: "text", raw: t[0], text: t[0], escaped: n };
      }
    }
  };
  var x = class u {
    tokens;
    options;
    state;
    inlineQueue;
    tokenizer;
    constructor(e) {
      this.tokens = [], this.tokens.links = Object.create(null), this.options = e || T, this.options.tokenizer = this.options.tokenizer || new w, this.tokenizer = this.options.tokenizer, this.tokenizer.options = this.options, this.tokenizer.lexer = this, this.inlineQueue = [], this.state = { inLink: false, inRawBlock: false, top: true };
      let t = { other: m, block: C.normal, inline: z.normal };
      this.options.pedantic ? (t.block = C.pedantic, t.inline = z.pedantic) : this.options.gfm && (t.block = C.gfm, this.options.breaks ? t.inline = z.breaks : t.inline = z.gfm), this.tokenizer.rules = t;
    }
    static get rules() {
      return { block: C, inline: z };
    }
    static lex(e, t) {
      return new u(t).lex(e);
    }
    static lexInline(e, t) {
      return new u(t).inlineTokens(e);
    }
    lex(e) {
      e = e.replace(m.carriageReturn, `
`), this.blockTokens(e, this.tokens);
      for (let t = 0;t < this.inlineQueue.length; t++) {
        let n = this.inlineQueue[t];
        this.inlineTokens(n.src, n.tokens);
      }
      return this.inlineQueue = [], this.tokens;
    }
    blockTokens(e, t = [], n = false) {
      for (this.options.pedantic && (e = e.replace(m.tabCharGlobal, "    ").replace(m.spaceLine, ""));e; ) {
        let r;
        if (this.options.extensions?.block?.some((s) => (r = s.call({ lexer: this }, e, t)) ? (e = e.substring(r.raw.length), t.push(r), true) : false))
          continue;
        if (r = this.tokenizer.space(e)) {
          e = e.substring(r.raw.length);
          let s = t.at(-1);
          r.raw.length === 1 && s !== undefined ? s.raw += `
` : t.push(r);
          continue;
        }
        if (r = this.tokenizer.code(e)) {
          e = e.substring(r.raw.length);
          let s = t.at(-1);
          s?.type === "paragraph" || s?.type === "text" ? (s.raw += (s.raw.endsWith(`
`) ? "" : `
`) + r.raw, s.text += `
` + r.text, this.inlineQueue.at(-1).src = s.text) : t.push(r);
          continue;
        }
        if (r = this.tokenizer.fences(e)) {
          e = e.substring(r.raw.length), t.push(r);
          continue;
        }
        if (r = this.tokenizer.heading(e)) {
          e = e.substring(r.raw.length), t.push(r);
          continue;
        }
        if (r = this.tokenizer.hr(e)) {
          e = e.substring(r.raw.length), t.push(r);
          continue;
        }
        if (r = this.tokenizer.blockquote(e)) {
          e = e.substring(r.raw.length), t.push(r);
          continue;
        }
        if (r = this.tokenizer.list(e)) {
          e = e.substring(r.raw.length), t.push(r);
          continue;
        }
        if (r = this.tokenizer.html(e)) {
          e = e.substring(r.raw.length), t.push(r);
          continue;
        }
        if (r = this.tokenizer.def(e)) {
          e = e.substring(r.raw.length);
          let s = t.at(-1);
          s?.type === "paragraph" || s?.type === "text" ? (s.raw += (s.raw.endsWith(`
`) ? "" : `
`) + r.raw, s.text += `
` + r.raw, this.inlineQueue.at(-1).src = s.text) : this.tokens.links[r.tag] || (this.tokens.links[r.tag] = { href: r.href, title: r.title }, t.push(r));
          continue;
        }
        if (r = this.tokenizer.table(e)) {
          e = e.substring(r.raw.length), t.push(r);
          continue;
        }
        if (r = this.tokenizer.lheading(e)) {
          e = e.substring(r.raw.length), t.push(r);
          continue;
        }
        let i = e;
        if (this.options.extensions?.startBlock) {
          let s = 1 / 0, a = e.slice(1), o;
          this.options.extensions.startBlock.forEach((l) => {
            o = l.call({ lexer: this }, a), typeof o == "number" && o >= 0 && (s = Math.min(s, o));
          }), s < 1 / 0 && s >= 0 && (i = e.substring(0, s + 1));
        }
        if (this.state.top && (r = this.tokenizer.paragraph(i))) {
          let s = t.at(-1);
          n && s?.type === "paragraph" ? (s.raw += (s.raw.endsWith(`
`) ? "" : `
`) + r.raw, s.text += `
` + r.text, this.inlineQueue.pop(), this.inlineQueue.at(-1).src = s.text) : t.push(r), n = i.length !== e.length, e = e.substring(r.raw.length);
          continue;
        }
        if (r = this.tokenizer.text(e)) {
          e = e.substring(r.raw.length);
          let s = t.at(-1);
          s?.type === "text" ? (s.raw += (s.raw.endsWith(`
`) ? "" : `
`) + r.raw, s.text += `
` + r.text, this.inlineQueue.pop(), this.inlineQueue.at(-1).src = s.text) : t.push(r);
          continue;
        }
        if (e) {
          let s = "Infinite loop on byte: " + e.charCodeAt(0);
          if (this.options.silent) {
            console.error(s);
            break;
          } else
            throw new Error(s);
        }
      }
      return this.state.top = true, t;
    }
    inline(e, t = []) {
      return this.inlineQueue.push({ src: e, tokens: t }), t;
    }
    inlineTokens(e, t = []) {
      let n = e, r = null;
      if (this.tokens.links) {
        let o = Object.keys(this.tokens.links);
        if (o.length > 0)
          for (;(r = this.tokenizer.rules.inline.reflinkSearch.exec(n)) != null; )
            o.includes(r[0].slice(r[0].lastIndexOf("[") + 1, -1)) && (n = n.slice(0, r.index) + "[" + "a".repeat(r[0].length - 2) + "]" + n.slice(this.tokenizer.rules.inline.reflinkSearch.lastIndex));
      }
      for (;(r = this.tokenizer.rules.inline.anyPunctuation.exec(n)) != null; )
        n = n.slice(0, r.index) + "++" + n.slice(this.tokenizer.rules.inline.anyPunctuation.lastIndex);
      let i;
      for (;(r = this.tokenizer.rules.inline.blockSkip.exec(n)) != null; )
        i = r[2] ? r[2].length : 0, n = n.slice(0, r.index + i) + "[" + "a".repeat(r[0].length - i - 2) + "]" + n.slice(this.tokenizer.rules.inline.blockSkip.lastIndex);
      n = this.options.hooks?.emStrongMask?.call({ lexer: this }, n) ?? n;
      let s = false, a = "";
      for (;e; ) {
        s || (a = ""), s = false;
        let o;
        if (this.options.extensions?.inline?.some((p) => (o = p.call({ lexer: this }, e, t)) ? (e = e.substring(o.raw.length), t.push(o), true) : false))
          continue;
        if (o = this.tokenizer.escape(e)) {
          e = e.substring(o.raw.length), t.push(o);
          continue;
        }
        if (o = this.tokenizer.tag(e)) {
          e = e.substring(o.raw.length), t.push(o);
          continue;
        }
        if (o = this.tokenizer.link(e)) {
          e = e.substring(o.raw.length), t.push(o);
          continue;
        }
        if (o = this.tokenizer.reflink(e, this.tokens.links)) {
          e = e.substring(o.raw.length);
          let p = t.at(-1);
          o.type === "text" && p?.type === "text" ? (p.raw += o.raw, p.text += o.text) : t.push(o);
          continue;
        }
        if (o = this.tokenizer.emStrong(e, n, a)) {
          e = e.substring(o.raw.length), t.push(o);
          continue;
        }
        if (o = this.tokenizer.codespan(e)) {
          e = e.substring(o.raw.length), t.push(o);
          continue;
        }
        if (o = this.tokenizer.br(e)) {
          e = e.substring(o.raw.length), t.push(o);
          continue;
        }
        if (o = this.tokenizer.del(e, n, a)) {
          e = e.substring(o.raw.length), t.push(o);
          continue;
        }
        if (o = this.tokenizer.autolink(e)) {
          e = e.substring(o.raw.length), t.push(o);
          continue;
        }
        if (!this.state.inLink && (o = this.tokenizer.url(e))) {
          e = e.substring(o.raw.length), t.push(o);
          continue;
        }
        let l = e;
        if (this.options.extensions?.startInline) {
          let p = 1 / 0, c = e.slice(1), d;
          this.options.extensions.startInline.forEach((h) => {
            d = h.call({ lexer: this }, c), typeof d == "number" && d >= 0 && (p = Math.min(p, d));
          }), p < 1 / 0 && p >= 0 && (l = e.substring(0, p + 1));
        }
        if (o = this.tokenizer.inlineText(l)) {
          e = e.substring(o.raw.length), o.raw.slice(-1) !== "_" && (a = o.raw.slice(-1)), s = true;
          let p = t.at(-1);
          p?.type === "text" ? (p.raw += o.raw, p.text += o.text) : t.push(o);
          continue;
        }
        if (e) {
          let p = "Infinite loop on byte: " + e.charCodeAt(0);
          if (this.options.silent) {
            console.error(p);
            break;
          } else
            throw new Error(p);
        }
      }
      return t;
    }
  };
  var y = class {
    options;
    parser;
    constructor(e) {
      this.options = e || T;
    }
    space(e) {
      return "";
    }
    code({ text: e, lang: t, escaped: n }) {
      let r = (t || "").match(m.notSpaceStart)?.[0], i = e.replace(m.endingNewline, "") + `
`;
      return r ? '<pre><code class="language-' + O(r) + '">' + (n ? i : O(i, true)) + `</code></pre>
` : "<pre><code>" + (n ? i : O(i, true)) + `</code></pre>
`;
    }
    blockquote({ tokens: e }) {
      return `<blockquote>
${this.parser.parse(e)}</blockquote>
`;
    }
    html({ text: e }) {
      return e;
    }
    def(e) {
      return "";
    }
    heading({ tokens: e, depth: t }) {
      return `<h${t}>${this.parser.parseInline(e)}</h${t}>
`;
    }
    hr(e) {
      return `<hr>
`;
    }
    list(e) {
      let { ordered: t, start: n } = e, r = "";
      for (let a = 0;a < e.items.length; a++) {
        let o = e.items[a];
        r += this.listitem(o);
      }
      let i = t ? "ol" : "ul", s = t && n !== 1 ? ' start="' + n + '"' : "";
      return "<" + i + s + `>
` + r + "</" + i + `>
`;
    }
    listitem(e) {
      return `<li>${this.parser.parse(e.tokens)}</li>
`;
    }
    checkbox({ checked: e }) {
      return "<input " + (e ? 'checked="" ' : "") + 'disabled="" type="checkbox"> ';
    }
    paragraph({ tokens: e }) {
      return `<p>${this.parser.parseInline(e)}</p>
`;
    }
    table(e) {
      let t = "", n = "";
      for (let i = 0;i < e.header.length; i++)
        n += this.tablecell(e.header[i]);
      t += this.tablerow({ text: n });
      let r = "";
      for (let i = 0;i < e.rows.length; i++) {
        let s = e.rows[i];
        n = "";
        for (let a = 0;a < s.length; a++)
          n += this.tablecell(s[a]);
        r += this.tablerow({ text: n });
      }
      return r && (r = `<tbody>${r}</tbody>`), `<table>
<thead>
` + t + `</thead>
` + r + `</table>
`;
    }
    tablerow({ text: e }) {
      return `<tr>
${e}</tr>
`;
    }
    tablecell(e) {
      let t = this.parser.parseInline(e.tokens), n = e.header ? "th" : "td";
      return (e.align ? `<${n} align="${e.align}">` : `<${n}>`) + t + `</${n}>
`;
    }
    strong({ tokens: e }) {
      return `<strong>${this.parser.parseInline(e)}</strong>`;
    }
    em({ tokens: e }) {
      return `<em>${this.parser.parseInline(e)}</em>`;
    }
    codespan({ text: e }) {
      return `<code>${O(e, true)}</code>`;
    }
    br(e) {
      return "<br>";
    }
    del({ tokens: e }) {
      return `<del>${this.parser.parseInline(e)}</del>`;
    }
    link({ href: e, title: t, tokens: n }) {
      let r = this.parser.parseInline(n), i = X(e);
      if (i === null)
        return r;
      e = i;
      let s = '<a href="' + e + '"';
      return t && (s += ' title="' + O(t) + '"'), s += ">" + r + "</a>", s;
    }
    image({ href: e, title: t, text: n, tokens: r }) {
      r && (n = this.parser.parseInline(r, this.parser.textRenderer));
      let i = X(e);
      if (i === null)
        return O(n);
      e = i;
      let s = `<img src="${e}" alt="${O(n)}"`;
      return t && (s += ` title="${O(t)}"`), s += ">", s;
    }
    text(e) {
      return "tokens" in e && e.tokens ? this.parser.parseInline(e.tokens) : ("escaped" in e) && e.escaped ? e.text : O(e.text);
    }
  };
  var $ = class {
    strong({ text: e }) {
      return e;
    }
    em({ text: e }) {
      return e;
    }
    codespan({ text: e }) {
      return e;
    }
    del({ text: e }) {
      return e;
    }
    html({ text: e }) {
      return e;
    }
    text({ text: e }) {
      return e;
    }
    link({ text: e }) {
      return "" + e;
    }
    image({ text: e }) {
      return "" + e;
    }
    br() {
      return "";
    }
    checkbox({ raw: e }) {
      return e;
    }
  };
  var b = class u2 {
    options;
    renderer;
    textRenderer;
    constructor(e) {
      this.options = e || T, this.options.renderer = this.options.renderer || new y, this.renderer = this.options.renderer, this.renderer.options = this.options, this.renderer.parser = this, this.textRenderer = new $;
    }
    static parse(e, t) {
      return new u2(t).parse(e);
    }
    static parseInline(e, t) {
      return new u2(t).parseInline(e);
    }
    parse(e) {
      let t = "";
      for (let n = 0;n < e.length; n++) {
        let r = e[n];
        if (this.options.extensions?.renderers?.[r.type]) {
          let s = r, a = this.options.extensions.renderers[s.type].call({ parser: this }, s);
          if (a !== false || !["space", "hr", "heading", "code", "table", "blockquote", "list", "html", "def", "paragraph", "text"].includes(s.type)) {
            t += a || "";
            continue;
          }
        }
        let i = r;
        switch (i.type) {
          case "space": {
            t += this.renderer.space(i);
            break;
          }
          case "hr": {
            t += this.renderer.hr(i);
            break;
          }
          case "heading": {
            t += this.renderer.heading(i);
            break;
          }
          case "code": {
            t += this.renderer.code(i);
            break;
          }
          case "table": {
            t += this.renderer.table(i);
            break;
          }
          case "blockquote": {
            t += this.renderer.blockquote(i);
            break;
          }
          case "list": {
            t += this.renderer.list(i);
            break;
          }
          case "checkbox": {
            t += this.renderer.checkbox(i);
            break;
          }
          case "html": {
            t += this.renderer.html(i);
            break;
          }
          case "def": {
            t += this.renderer.def(i);
            break;
          }
          case "paragraph": {
            t += this.renderer.paragraph(i);
            break;
          }
          case "text": {
            t += this.renderer.text(i);
            break;
          }
          default: {
            let s = 'Token with "' + i.type + '" type was not found.';
            if (this.options.silent)
              return console.error(s), "";
            throw new Error(s);
          }
        }
      }
      return t;
    }
    parseInline(e, t = this.renderer) {
      let n = "";
      for (let r = 0;r < e.length; r++) {
        let i = e[r];
        if (this.options.extensions?.renderers?.[i.type]) {
          let a = this.options.extensions.renderers[i.type].call({ parser: this }, i);
          if (a !== false || !["escape", "html", "link", "image", "strong", "em", "codespan", "br", "del", "text"].includes(i.type)) {
            n += a || "";
            continue;
          }
        }
        let s = i;
        switch (s.type) {
          case "escape": {
            n += t.text(s);
            break;
          }
          case "html": {
            n += t.html(s);
            break;
          }
          case "link": {
            n += t.link(s);
            break;
          }
          case "image": {
            n += t.image(s);
            break;
          }
          case "checkbox": {
            n += t.checkbox(s);
            break;
          }
          case "strong": {
            n += t.strong(s);
            break;
          }
          case "em": {
            n += t.em(s);
            break;
          }
          case "codespan": {
            n += t.codespan(s);
            break;
          }
          case "br": {
            n += t.br(s);
            break;
          }
          case "del": {
            n += t.del(s);
            break;
          }
          case "text": {
            n += t.text(s);
            break;
          }
          default: {
            let a = 'Token with "' + s.type + '" type was not found.';
            if (this.options.silent)
              return console.error(a), "";
            throw new Error(a);
          }
        }
      }
      return n;
    }
  };
  var P = class {
    options;
    block;
    constructor(e) {
      this.options = e || T;
    }
    static passThroughHooks = new Set(["preprocess", "postprocess", "processAllTokens", "emStrongMask"]);
    static passThroughHooksRespectAsync = new Set(["preprocess", "postprocess", "processAllTokens"]);
    preprocess(e) {
      return e;
    }
    postprocess(e) {
      return e;
    }
    processAllTokens(e) {
      return e;
    }
    emStrongMask(e) {
      return e;
    }
    provideLexer() {
      return this.block ? x.lex : x.lexInline;
    }
    provideParser() {
      return this.block ? b.parse : b.parseInline;
    }
  };
  var B = class {
    defaults = M();
    options = this.setOptions;
    parse = this.parseMarkdown(true);
    parseInline = this.parseMarkdown(false);
    Parser = b;
    Renderer = y;
    TextRenderer = $;
    Lexer = x;
    Tokenizer = w;
    Hooks = P;
    constructor(...e) {
      this.use(...e);
    }
    walkTokens(e, t) {
      let n = [];
      for (let r of e)
        switch (n = n.concat(t.call(this, r)), r.type) {
          case "table": {
            let i = r;
            for (let s of i.header)
              n = n.concat(this.walkTokens(s.tokens, t));
            for (let s of i.rows)
              for (let a of s)
                n = n.concat(this.walkTokens(a.tokens, t));
            break;
          }
          case "list": {
            let i = r;
            n = n.concat(this.walkTokens(i.items, t));
            break;
          }
          default: {
            let i = r;
            this.defaults.extensions?.childTokens?.[i.type] ? this.defaults.extensions.childTokens[i.type].forEach((s) => {
              let a = i[s].flat(1 / 0);
              n = n.concat(this.walkTokens(a, t));
            }) : i.tokens && (n = n.concat(this.walkTokens(i.tokens, t)));
          }
        }
      return n;
    }
    use(...e) {
      let t = this.defaults.extensions || { renderers: {}, childTokens: {} };
      return e.forEach((n) => {
        let r = { ...n };
        if (r.async = this.defaults.async || r.async || false, n.extensions && (n.extensions.forEach((i) => {
          if (!i.name)
            throw new Error("extension name required");
          if ("renderer" in i) {
            let s = t.renderers[i.name];
            s ? t.renderers[i.name] = function(...a) {
              let o = i.renderer.apply(this, a);
              return o === false && (o = s.apply(this, a)), o;
            } : t.renderers[i.name] = i.renderer;
          }
          if ("tokenizer" in i) {
            if (!i.level || i.level !== "block" && i.level !== "inline")
              throw new Error("extension level must be 'block' or 'inline'");
            let s = t[i.level];
            s ? s.unshift(i.tokenizer) : t[i.level] = [i.tokenizer], i.start && (i.level === "block" ? t.startBlock ? t.startBlock.push(i.start) : t.startBlock = [i.start] : i.level === "inline" && (t.startInline ? t.startInline.push(i.start) : t.startInline = [i.start]));
          }
          "childTokens" in i && i.childTokens && (t.childTokens[i.name] = i.childTokens);
        }), r.extensions = t), n.renderer) {
          let i = this.defaults.renderer || new y(this.defaults);
          for (let s in n.renderer) {
            if (!(s in i))
              throw new Error(`renderer '${s}' does not exist`);
            if (["options", "parser"].includes(s))
              continue;
            let a = s, o = n.renderer[a], l = i[a];
            i[a] = (...p) => {
              let c = o.apply(i, p);
              return c === false && (c = l.apply(i, p)), c || "";
            };
          }
          r.renderer = i;
        }
        if (n.tokenizer) {
          let i = this.defaults.tokenizer || new w(this.defaults);
          for (let s in n.tokenizer) {
            if (!(s in i))
              throw new Error(`tokenizer '${s}' does not exist`);
            if (["options", "rules", "lexer"].includes(s))
              continue;
            let a = s, o = n.tokenizer[a], l = i[a];
            i[a] = (...p) => {
              let c = o.apply(i, p);
              return c === false && (c = l.apply(i, p)), c;
            };
          }
          r.tokenizer = i;
        }
        if (n.hooks) {
          let i = this.defaults.hooks || new P;
          for (let s in n.hooks) {
            if (!(s in i))
              throw new Error(`hook '${s}' does not exist`);
            if (["options", "block"].includes(s))
              continue;
            let a = s, o = n.hooks[a], l = i[a];
            P.passThroughHooks.has(s) ? i[a] = (p) => {
              if (this.defaults.async && P.passThroughHooksRespectAsync.has(s))
                return (async () => {
                  let d = await o.call(i, p);
                  return l.call(i, d);
                })();
              let c = o.call(i, p);
              return l.call(i, c);
            } : i[a] = (...p) => {
              if (this.defaults.async)
                return (async () => {
                  let d = await o.apply(i, p);
                  return d === false && (d = await l.apply(i, p)), d;
                })();
              let c = o.apply(i, p);
              return c === false && (c = l.apply(i, p)), c;
            };
          }
          r.hooks = i;
        }
        if (n.walkTokens) {
          let i = this.defaults.walkTokens, s = n.walkTokens;
          r.walkTokens = function(a) {
            let o = [];
            return o.push(s.call(this, a)), i && (o = o.concat(i.call(this, a))), o;
          };
        }
        this.defaults = { ...this.defaults, ...r };
      }), this;
    }
    setOptions(e) {
      return this.defaults = { ...this.defaults, ...e }, this;
    }
    lexer(e, t) {
      return x.lex(e, t ?? this.defaults);
    }
    parser(e, t) {
      return b.parse(e, t ?? this.defaults);
    }
    parseMarkdown(e) {
      return (n, r) => {
        let i = { ...r }, s = { ...this.defaults, ...i }, a = this.onError(!!s.silent, !!s.async);
        if (this.defaults.async === true && i.async === false)
          return a(new Error("marked(): The async option was set to true by an extension. Remove async: false from the parse options object to return a Promise."));
        if (typeof n > "u" || n === null)
          return a(new Error("marked(): input parameter is undefined or null"));
        if (typeof n != "string")
          return a(new Error("marked(): input parameter is of type " + Object.prototype.toString.call(n) + ", string expected"));
        if (s.hooks && (s.hooks.options = s, s.hooks.block = e), s.async)
          return (async () => {
            let o = s.hooks ? await s.hooks.preprocess(n) : n, p = await (s.hooks ? await s.hooks.provideLexer() : e ? x.lex : x.lexInline)(o, s), c = s.hooks ? await s.hooks.processAllTokens(p) : p;
            s.walkTokens && await Promise.all(this.walkTokens(c, s.walkTokens));
            let h = await (s.hooks ? await s.hooks.provideParser() : e ? b.parse : b.parseInline)(c, s);
            return s.hooks ? await s.hooks.postprocess(h) : h;
          })().catch(a);
        try {
          s.hooks && (n = s.hooks.preprocess(n));
          let l = (s.hooks ? s.hooks.provideLexer() : e ? x.lex : x.lexInline)(n, s);
          s.hooks && (l = s.hooks.processAllTokens(l)), s.walkTokens && this.walkTokens(l, s.walkTokens);
          let c = (s.hooks ? s.hooks.provideParser() : e ? b.parse : b.parseInline)(l, s);
          return s.hooks && (c = s.hooks.postprocess(c)), c;
        } catch (o) {
          return a(o);
        }
      };
    }
    onError(e, t) {
      return (n) => {
        if (n.message += `
Please report this to https://github.com/markedjs/marked.`, e) {
          let r = "<p>An error occurred:</p><pre>" + O(n.message + "", true) + "</pre>";
          return t ? Promise.resolve(r) : r;
        }
        if (t)
          return Promise.reject(n);
        throw n;
      };
    }
  };
  var L = new B;
  function g(u3, e) {
    return L.parse(u3, e);
  }
  g.options = g.setOptions = function(u3) {
    return L.setOptions(u3), g.defaults = L.defaults, G(g.defaults), g;
  };
  g.getDefaults = M;
  g.defaults = T;
  g.use = function(...u3) {
    return L.use(...u3), g.defaults = L.defaults, G(g.defaults), g;
  };
  g.walkTokens = function(u3, e) {
    return L.walkTokens(u3, e);
  };
  g.parseInline = L.parseInline;
  g.Parser = b;
  g.parser = b.parse;
  g.Renderer = y;
  g.TextRenderer = $;
  g.Lexer = x;
  g.lexer = x.lex;
  g.Tokenizer = w;
  g.Hooks = P;
  g.parse = g;
  var Ut = g.options;
  var Kt = g.setOptions;
  var Wt = g.use;
  var Xt = g.walkTokens;
  var Jt = g.parseInline;
  var Yt = b.parse;
  var en = x.lex;

  // node_modules/highlight.js/es/core.js
  var import_core = __toESM(require_core(), 1);
  var core_default = import_core.default;

  // node_modules/highlight.js/es/languages/javascript.js
  var IDENT_RE = "[A-Za-z$_][0-9A-Za-z$_]*";
  var KEYWORDS = [
    "as",
    "in",
    "of",
    "if",
    "for",
    "while",
    "finally",
    "var",
    "new",
    "function",
    "do",
    "return",
    "void",
    "else",
    "break",
    "catch",
    "instanceof",
    "with",
    "throw",
    "case",
    "default",
    "try",
    "switch",
    "continue",
    "typeof",
    "delete",
    "let",
    "yield",
    "const",
    "class",
    "debugger",
    "async",
    "await",
    "static",
    "import",
    "from",
    "export",
    "extends",
    "using"
  ];
  var LITERALS = [
    "true",
    "false",
    "null",
    "undefined",
    "NaN",
    "Infinity"
  ];
  var TYPES = [
    "Object",
    "Function",
    "Boolean",
    "Symbol",
    "Math",
    "Date",
    "Number",
    "BigInt",
    "String",
    "RegExp",
    "Array",
    "Float32Array",
    "Float64Array",
    "Int8Array",
    "Uint8Array",
    "Uint8ClampedArray",
    "Int16Array",
    "Int32Array",
    "Uint16Array",
    "Uint32Array",
    "BigInt64Array",
    "BigUint64Array",
    "Set",
    "Map",
    "WeakSet",
    "WeakMap",
    "ArrayBuffer",
    "SharedArrayBuffer",
    "Atomics",
    "DataView",
    "JSON",
    "Promise",
    "Generator",
    "GeneratorFunction",
    "AsyncFunction",
    "Reflect",
    "Proxy",
    "Intl",
    "WebAssembly"
  ];
  var ERROR_TYPES = [
    "Error",
    "EvalError",
    "InternalError",
    "RangeError",
    "ReferenceError",
    "SyntaxError",
    "TypeError",
    "URIError"
  ];
  var BUILT_IN_GLOBALS = [
    "setInterval",
    "setTimeout",
    "clearInterval",
    "clearTimeout",
    "require",
    "exports",
    "eval",
    "isFinite",
    "isNaN",
    "parseFloat",
    "parseInt",
    "decodeURI",
    "decodeURIComponent",
    "encodeURI",
    "encodeURIComponent",
    "escape",
    "unescape"
  ];
  var BUILT_IN_VARIABLES = [
    "arguments",
    "this",
    "super",
    "console",
    "window",
    "document",
    "localStorage",
    "sessionStorage",
    "module",
    "global"
  ];
  var BUILT_INS = [].concat(BUILT_IN_GLOBALS, TYPES, ERROR_TYPES);
  function javascript(hljs) {
    const regex = hljs.regex;
    const hasClosingTag = (match, { after }) => {
      const tag = "</" + match[0].slice(1);
      const pos = match.input.indexOf(tag, after);
      return pos !== -1;
    };
    const IDENT_RE$1 = IDENT_RE;
    const FRAGMENT = {
      begin: "<>",
      end: "</>"
    };
    const XML_SELF_CLOSING = /<[A-Za-z0-9\\._:-]+\s*\/>/;
    const XML_TAG = {
      begin: /<[A-Za-z0-9\\._:-]+/,
      end: /\/[A-Za-z0-9\\._:-]+>|\/>/,
      isTrulyOpeningTag: (match, response) => {
        const afterMatchIndex = match[0].length + match.index;
        const nextChar = match.input[afterMatchIndex];
        if (nextChar === "<" || nextChar === ",") {
          response.ignoreMatch();
          return;
        }
        if (nextChar === ">") {
          if (!hasClosingTag(match, { after: afterMatchIndex })) {
            response.ignoreMatch();
          }
        }
        let m2;
        const afterMatch = match.input.substring(afterMatchIndex);
        if (m2 = afterMatch.match(/^\s*=/)) {
          response.ignoreMatch();
          return;
        }
        if (m2 = afterMatch.match(/^\s+extends\s+/)) {
          if (m2.index === 0) {
            response.ignoreMatch();
            return;
          }
        }
      }
    };
    const KEYWORDS$1 = {
      $pattern: IDENT_RE,
      keyword: KEYWORDS,
      literal: LITERALS,
      built_in: BUILT_INS,
      "variable.language": BUILT_IN_VARIABLES
    };
    const decimalDigits = "[0-9](_?[0-9])*";
    const frac = `\\.(${decimalDigits})`;
    const decimalInteger = `0|[1-9](_?[0-9])*|0[0-7]*[89][0-9]*`;
    const NUMBER = {
      className: "number",
      variants: [
        { begin: `(\\b(${decimalInteger})((${frac})|\\.)?|(${frac}))` + `[eE][+-]?(${decimalDigits})\\b` },
        { begin: `\\b(${decimalInteger})\\b((${frac})\\b|\\.)?|(${frac})\\b` },
        { begin: `\\b(0|[1-9](_?[0-9])*)n\\b` },
        { begin: "\\b0[xX][0-9a-fA-F](_?[0-9a-fA-F])*n?\\b" },
        { begin: "\\b0[bB][0-1](_?[0-1])*n?\\b" },
        { begin: "\\b0[oO][0-7](_?[0-7])*n?\\b" },
        { begin: "\\b0[0-7]+n?\\b" }
      ],
      relevance: 0
    };
    const SUBST = {
      className: "subst",
      begin: "\\$\\{",
      end: "\\}",
      keywords: KEYWORDS$1,
      contains: []
    };
    const HTML_TEMPLATE = {
      begin: ".?html`",
      end: "",
      starts: {
        end: "`",
        returnEnd: false,
        contains: [
          hljs.BACKSLASH_ESCAPE,
          SUBST
        ],
        subLanguage: "xml"
      }
    };
    const CSS_TEMPLATE = {
      begin: ".?css`",
      end: "",
      starts: {
        end: "`",
        returnEnd: false,
        contains: [
          hljs.BACKSLASH_ESCAPE,
          SUBST
        ],
        subLanguage: "css"
      }
    };
    const GRAPHQL_TEMPLATE = {
      begin: ".?gql`",
      end: "",
      starts: {
        end: "`",
        returnEnd: false,
        contains: [
          hljs.BACKSLASH_ESCAPE,
          SUBST
        ],
        subLanguage: "graphql"
      }
    };
    const TEMPLATE_STRING = {
      className: "string",
      begin: "`",
      end: "`",
      contains: [
        hljs.BACKSLASH_ESCAPE,
        SUBST
      ]
    };
    const JSDOC_COMMENT = hljs.COMMENT(/\/\*\*(?!\/)/, "\\*/", {
      relevance: 0,
      contains: [
        {
          begin: "(?=@[A-Za-z]+)",
          relevance: 0,
          contains: [
            {
              className: "doctag",
              begin: "@[A-Za-z]+"
            },
            {
              className: "type",
              begin: "\\{",
              end: "\\}",
              excludeEnd: true,
              excludeBegin: true,
              relevance: 0
            },
            {
              className: "variable",
              begin: IDENT_RE$1 + "(?=\\s*(-)|$)",
              endsParent: true,
              relevance: 0
            },
            {
              begin: /(?=[^\n])\s/,
              relevance: 0
            }
          ]
        }
      ]
    });
    const COMMENT = {
      className: "comment",
      variants: [
        JSDOC_COMMENT,
        hljs.C_BLOCK_COMMENT_MODE,
        hljs.C_LINE_COMMENT_MODE
      ]
    };
    const SUBST_INTERNALS = [
      hljs.APOS_STRING_MODE,
      hljs.QUOTE_STRING_MODE,
      HTML_TEMPLATE,
      CSS_TEMPLATE,
      GRAPHQL_TEMPLATE,
      TEMPLATE_STRING,
      { match: /\$\d+/ },
      NUMBER
    ];
    SUBST.contains = SUBST_INTERNALS.concat({
      begin: /\{/,
      end: /\}/,
      keywords: KEYWORDS$1,
      contains: [
        "self"
      ].concat(SUBST_INTERNALS)
    });
    const SUBST_AND_COMMENTS = [].concat(COMMENT, SUBST.contains);
    const PARAMS_CONTAINS = SUBST_AND_COMMENTS.concat([
      {
        begin: /(\s*)\(/,
        end: /\)/,
        keywords: KEYWORDS$1,
        contains: ["self"].concat(SUBST_AND_COMMENTS)
      }
    ]);
    const PARAMS = {
      className: "params",
      begin: /(\s*)\(/,
      end: /\)/,
      excludeBegin: true,
      excludeEnd: true,
      keywords: KEYWORDS$1,
      contains: PARAMS_CONTAINS
    };
    const CLASS_OR_EXTENDS = {
      variants: [
        {
          match: [
            /class/,
            /\s+/,
            IDENT_RE$1,
            /\s+/,
            /extends/,
            /\s+/,
            regex.concat(IDENT_RE$1, "(", regex.concat(/\./, IDENT_RE$1), ")*")
          ],
          scope: {
            1: "keyword",
            3: "title.class",
            5: "keyword",
            7: "title.class.inherited"
          }
        },
        {
          match: [
            /class/,
            /\s+/,
            IDENT_RE$1
          ],
          scope: {
            1: "keyword",
            3: "title.class"
          }
        }
      ]
    };
    const CLASS_REFERENCE = {
      relevance: 0,
      match: regex.either(/\bJSON/, /\b[A-Z][a-z]+([A-Z][a-z]*|\d)*/, /\b[A-Z]{2,}([A-Z][a-z]+|\d)+([A-Z][a-z]*)*/, /\b[A-Z]{2,}[a-z]+([A-Z][a-z]+|\d)*([A-Z][a-z]*)*/),
      className: "title.class",
      keywords: {
        _: [
          ...TYPES,
          ...ERROR_TYPES
        ]
      }
    };
    const USE_STRICT = {
      label: "use_strict",
      className: "meta",
      relevance: 10,
      begin: /^\s*['"]use (strict|asm)['"]/
    };
    const FUNCTION_DEFINITION = {
      variants: [
        {
          match: [
            /function/,
            /\s+/,
            IDENT_RE$1,
            /(?=\s*\()/
          ]
        },
        {
          match: [
            /function/,
            /\s*(?=\()/
          ]
        }
      ],
      className: {
        1: "keyword",
        3: "title.function"
      },
      label: "func.def",
      contains: [PARAMS],
      illegal: /%/
    };
    const UPPER_CASE_CONSTANT = {
      relevance: 0,
      match: /\b[A-Z][A-Z_0-9]+\b/,
      className: "variable.constant"
    };
    function noneOf(list) {
      return regex.concat("(?!", list.join("|"), ")");
    }
    const FUNCTION_CALL = {
      match: regex.concat(/\b/, noneOf([
        ...BUILT_IN_GLOBALS,
        "super",
        "import"
      ].map((x2) => `${x2}\\s*\\(`)), IDENT_RE$1, regex.lookahead(/\s*\(/)),
      className: "title.function",
      relevance: 0
    };
    const PROPERTY_ACCESS = {
      begin: regex.concat(/\./, regex.lookahead(regex.concat(IDENT_RE$1, /(?![0-9A-Za-z$_(])/))),
      end: IDENT_RE$1,
      excludeBegin: true,
      keywords: "prototype",
      className: "property",
      relevance: 0
    };
    const GETTER_OR_SETTER = {
      match: [
        /get|set/,
        /\s+/,
        IDENT_RE$1,
        /(?=\()/
      ],
      className: {
        1: "keyword",
        3: "title.function"
      },
      contains: [
        {
          begin: /\(\)/
        },
        PARAMS
      ]
    };
    const FUNC_LEAD_IN_RE = "(\\(" + "[^()]*(\\(" + "[^()]*(\\(" + "[^()]*" + "\\)[^()]*)*" + "\\)[^()]*)*" + "\\)|" + hljs.UNDERSCORE_IDENT_RE + ")\\s*=>";
    const FUNCTION_VARIABLE = {
      match: [
        /const|var|let/,
        /\s+/,
        IDENT_RE$1,
        /\s*/,
        /=\s*/,
        /(async\s*)?/,
        regex.lookahead(FUNC_LEAD_IN_RE)
      ],
      keywords: "async",
      className: {
        1: "keyword",
        3: "title.function"
      },
      contains: [
        PARAMS
      ]
    };
    return {
      name: "JavaScript",
      aliases: ["js", "jsx", "mjs", "cjs"],
      keywords: KEYWORDS$1,
      exports: { PARAMS_CONTAINS, CLASS_REFERENCE },
      illegal: /#(?![$_A-z])/,
      contains: [
        hljs.SHEBANG({
          label: "shebang",
          binary: "node",
          relevance: 5
        }),
        USE_STRICT,
        hljs.APOS_STRING_MODE,
        hljs.QUOTE_STRING_MODE,
        HTML_TEMPLATE,
        CSS_TEMPLATE,
        GRAPHQL_TEMPLATE,
        TEMPLATE_STRING,
        COMMENT,
        { match: /\$\d+/ },
        NUMBER,
        CLASS_REFERENCE,
        {
          scope: "attr",
          match: IDENT_RE$1 + regex.lookahead(":"),
          relevance: 0
        },
        FUNCTION_VARIABLE,
        {
          begin: "(" + hljs.RE_STARTERS_RE + "|\\b(case|return|throw)\\b)\\s*",
          keywords: "return throw case",
          relevance: 0,
          contains: [
            COMMENT,
            hljs.REGEXP_MODE,
            {
              className: "function",
              begin: FUNC_LEAD_IN_RE,
              returnBegin: true,
              end: "\\s*=>",
              contains: [
                {
                  className: "params",
                  variants: [
                    {
                      begin: hljs.UNDERSCORE_IDENT_RE,
                      relevance: 0
                    },
                    {
                      className: null,
                      begin: /\(\s*\)/,
                      skip: true
                    },
                    {
                      begin: /(\s*)\(/,
                      end: /\)/,
                      excludeBegin: true,
                      excludeEnd: true,
                      keywords: KEYWORDS$1,
                      contains: PARAMS_CONTAINS
                    }
                  ]
                }
              ]
            },
            {
              begin: /,/,
              relevance: 0
            },
            {
              match: /\s+/,
              relevance: 0
            },
            {
              variants: [
                { begin: FRAGMENT.begin, end: FRAGMENT.end },
                { match: XML_SELF_CLOSING },
                {
                  begin: XML_TAG.begin,
                  "on:begin": XML_TAG.isTrulyOpeningTag,
                  end: XML_TAG.end
                }
              ],
              subLanguage: "xml",
              contains: [
                {
                  begin: XML_TAG.begin,
                  end: XML_TAG.end,
                  skip: true,
                  contains: ["self"]
                }
              ]
            }
          ]
        },
        FUNCTION_DEFINITION,
        {
          beginKeywords: "while if switch catch for"
        },
        {
          begin: "\\b(?!function)" + hljs.UNDERSCORE_IDENT_RE + "\\(" + "[^()]*(\\(" + "[^()]*(\\(" + "[^()]*" + "\\)[^()]*)*" + "\\)[^()]*)*" + "\\)\\s*\\{",
          returnBegin: true,
          label: "func.def",
          contains: [
            PARAMS,
            hljs.inherit(hljs.TITLE_MODE, { begin: IDENT_RE$1, className: "title.function" })
          ]
        },
        {
          match: /\.\.\./,
          relevance: 0
        },
        PROPERTY_ACCESS,
        {
          match: "\\$" + IDENT_RE$1,
          relevance: 0
        },
        {
          match: [/\bconstructor(?=\s*\()/],
          className: { 1: "title.function" },
          contains: [PARAMS]
        },
        FUNCTION_CALL,
        UPPER_CASE_CONSTANT,
        CLASS_OR_EXTENDS,
        GETTER_OR_SETTER,
        {
          match: /\$[(.]/
        }
      ]
    };
  }

  // node_modules/highlight.js/es/languages/python.js
  function python(hljs) {
    const regex = hljs.regex;
    const IDENT_RE2 = /[\p{XID_Start}_]\p{XID_Continue}*/u;
    const RESERVED_WORDS = [
      "and",
      "as",
      "assert",
      "async",
      "await",
      "break",
      "case",
      "class",
      "continue",
      "def",
      "del",
      "elif",
      "else",
      "except",
      "finally",
      "for",
      "from",
      "global",
      "if",
      "import",
      "in",
      "is",
      "lambda",
      "match",
      "nonlocal|10",
      "not",
      "or",
      "pass",
      "raise",
      "return",
      "try",
      "while",
      "with",
      "yield"
    ];
    const BUILT_INS2 = [
      "__import__",
      "abs",
      "all",
      "any",
      "ascii",
      "bin",
      "bool",
      "breakpoint",
      "bytearray",
      "bytes",
      "callable",
      "chr",
      "classmethod",
      "compile",
      "complex",
      "delattr",
      "dict",
      "dir",
      "divmod",
      "enumerate",
      "eval",
      "exec",
      "filter",
      "float",
      "format",
      "frozenset",
      "getattr",
      "globals",
      "hasattr",
      "hash",
      "help",
      "hex",
      "id",
      "input",
      "int",
      "isinstance",
      "issubclass",
      "iter",
      "len",
      "list",
      "locals",
      "map",
      "max",
      "memoryview",
      "min",
      "next",
      "object",
      "oct",
      "open",
      "ord",
      "pow",
      "print",
      "property",
      "range",
      "repr",
      "reversed",
      "round",
      "set",
      "setattr",
      "slice",
      "sorted",
      "staticmethod",
      "str",
      "sum",
      "super",
      "tuple",
      "type",
      "vars",
      "zip"
    ];
    const LITERALS2 = [
      "__debug__",
      "Ellipsis",
      "False",
      "None",
      "NotImplemented",
      "True"
    ];
    const TYPES2 = [
      "Any",
      "Callable",
      "Coroutine",
      "Dict",
      "List",
      "Literal",
      "Generic",
      "Optional",
      "Sequence",
      "Set",
      "Tuple",
      "Type",
      "Union"
    ];
    const KEYWORDS2 = {
      $pattern: /[A-Za-z]\w+|__\w+__/,
      keyword: RESERVED_WORDS,
      built_in: BUILT_INS2,
      literal: LITERALS2,
      type: TYPES2
    };
    const PROMPT = {
      className: "meta",
      begin: /^(>>>|\.\.\.) /
    };
    const SUBST = {
      className: "subst",
      begin: /\{/,
      end: /\}/,
      keywords: KEYWORDS2,
      illegal: /#/
    };
    const LITERAL_BRACKET = {
      begin: /\{\{/,
      relevance: 0
    };
    const STRING = {
      className: "string",
      contains: [hljs.BACKSLASH_ESCAPE],
      variants: [
        {
          begin: /([uU]|[bB]|[rR]|[bB][rR]|[rR][bB])?'''/,
          end: /'''/,
          contains: [
            hljs.BACKSLASH_ESCAPE,
            PROMPT
          ],
          relevance: 10
        },
        {
          begin: /([uU]|[bB]|[rR]|[bB][rR]|[rR][bB])?"""/,
          end: /"""/,
          contains: [
            hljs.BACKSLASH_ESCAPE,
            PROMPT
          ],
          relevance: 10
        },
        {
          begin: /([fF][rR]|[rR][fF]|[fF])'''/,
          end: /'''/,
          contains: [
            hljs.BACKSLASH_ESCAPE,
            PROMPT,
            LITERAL_BRACKET,
            SUBST
          ]
        },
        {
          begin: /([fF][rR]|[rR][fF]|[fF])"""/,
          end: /"""/,
          contains: [
            hljs.BACKSLASH_ESCAPE,
            PROMPT,
            LITERAL_BRACKET,
            SUBST
          ]
        },
        {
          begin: /([uU]|[rR])'/,
          end: /'/,
          relevance: 10
        },
        {
          begin: /([uU]|[rR])"/,
          end: /"/,
          relevance: 10
        },
        {
          begin: /([bB]|[bB][rR]|[rR][bB])'/,
          end: /'/
        },
        {
          begin: /([bB]|[bB][rR]|[rR][bB])"/,
          end: /"/
        },
        {
          begin: /([fF][rR]|[rR][fF]|[fF])'/,
          end: /'/,
          contains: [
            hljs.BACKSLASH_ESCAPE,
            LITERAL_BRACKET,
            SUBST
          ]
        },
        {
          begin: /([fF][rR]|[rR][fF]|[fF])"/,
          end: /"/,
          contains: [
            hljs.BACKSLASH_ESCAPE,
            LITERAL_BRACKET,
            SUBST
          ]
        },
        hljs.APOS_STRING_MODE,
        hljs.QUOTE_STRING_MODE
      ]
    };
    const digitpart = "[0-9](_?[0-9])*";
    const pointfloat = `(\\b(${digitpart}))?\\.(${digitpart})|\\b(${digitpart})\\.`;
    const lookahead = `\\b|${RESERVED_WORDS.join("|")}`;
    const NUMBER = {
      className: "number",
      relevance: 0,
      variants: [
        {
          begin: `(\\b(${digitpart})|(${pointfloat}))[eE][+-]?(${digitpart})[jJ]?(?=${lookahead})`
        },
        {
          begin: `(${pointfloat})[jJ]?`
        },
        {
          begin: `\\b([1-9](_?[0-9])*|0+(_?0)*)[lLjJ]?(?=${lookahead})`
        },
        {
          begin: `\\b0[bB](_?[01])+[lL]?(?=${lookahead})`
        },
        {
          begin: `\\b0[oO](_?[0-7])+[lL]?(?=${lookahead})`
        },
        {
          begin: `\\b0[xX](_?[0-9a-fA-F])+[lL]?(?=${lookahead})`
        },
        {
          begin: `\\b(${digitpart})[jJ](?=${lookahead})`
        }
      ]
    };
    const COMMENT_TYPE = {
      className: "comment",
      begin: regex.lookahead(/# type:/),
      end: /$/,
      keywords: KEYWORDS2,
      contains: [
        {
          begin: /# type:/
        },
        {
          begin: /#/,
          end: /\b\B/,
          endsWithParent: true
        }
      ]
    };
    const PARAMS = {
      className: "params",
      variants: [
        {
          className: "",
          begin: /\(\s*\)/,
          skip: true
        },
        {
          begin: /\(/,
          end: /\)/,
          excludeBegin: true,
          excludeEnd: true,
          keywords: KEYWORDS2,
          contains: [
            "self",
            PROMPT,
            NUMBER,
            STRING,
            hljs.HASH_COMMENT_MODE
          ]
        }
      ]
    };
    SUBST.contains = [
      STRING,
      NUMBER,
      PROMPT
    ];
    return {
      name: "Python",
      aliases: [
        "py",
        "gyp",
        "ipython"
      ],
      unicodeRegex: true,
      keywords: KEYWORDS2,
      illegal: /(<\/|\?)|=>/,
      contains: [
        PROMPT,
        NUMBER,
        {
          scope: "variable.language",
          match: /\bself\b/
        },
        {
          beginKeywords: "if",
          relevance: 0
        },
        { match: /\bor\b/, scope: "keyword" },
        STRING,
        COMMENT_TYPE,
        hljs.HASH_COMMENT_MODE,
        {
          match: [
            /\bdef/,
            /\s+/,
            IDENT_RE2
          ],
          scope: {
            1: "keyword",
            3: "title.function"
          },
          contains: [PARAMS]
        },
        {
          variants: [
            {
              match: [
                /\bclass/,
                /\s+/,
                IDENT_RE2,
                /\s*/,
                /\(\s*/,
                IDENT_RE2,
                /\s*\)/
              ]
            },
            {
              match: [
                /\bclass/,
                /\s+/,
                IDENT_RE2
              ]
            }
          ],
          scope: {
            1: "keyword",
            3: "title.class",
            6: "title.class.inherited"
          }
        },
        {
          className: "meta",
          begin: /^[\t ]*@/,
          end: /(?=#)|$/,
          contains: [
            NUMBER,
            PARAMS,
            STRING
          ]
        }
      ]
    };
  }

  // node_modules/highlight.js/es/languages/go.js
  function go(hljs) {
    const LITERALS2 = [
      "true",
      "false",
      "iota",
      "nil"
    ];
    const BUILT_INS2 = [
      "append",
      "cap",
      "close",
      "complex",
      "copy",
      "imag",
      "len",
      "make",
      "new",
      "panic",
      "print",
      "println",
      "real",
      "recover",
      "delete"
    ];
    const TYPES2 = [
      "bool",
      "byte",
      "complex64",
      "complex128",
      "error",
      "float32",
      "float64",
      "int8",
      "int16",
      "int32",
      "int64",
      "string",
      "uint8",
      "uint16",
      "uint32",
      "uint64",
      "int",
      "uint",
      "uintptr",
      "rune"
    ];
    const KWS = [
      "break",
      "case",
      "chan",
      "const",
      "continue",
      "default",
      "defer",
      "else",
      "fallthrough",
      "for",
      "func",
      "go",
      "goto",
      "if",
      "import",
      "interface",
      "map",
      "package",
      "range",
      "return",
      "select",
      "struct",
      "switch",
      "type",
      "var"
    ];
    const KEYWORDS2 = {
      keyword: KWS,
      type: TYPES2,
      literal: LITERALS2,
      built_in: BUILT_INS2
    };
    return {
      name: "Go",
      aliases: ["golang"],
      keywords: KEYWORDS2,
      illegal: "</",
      contains: [
        hljs.C_LINE_COMMENT_MODE,
        hljs.C_BLOCK_COMMENT_MODE,
        {
          className: "string",
          variants: [
            hljs.QUOTE_STRING_MODE,
            hljs.APOS_STRING_MODE,
            {
              begin: "`",
              end: "`"
            }
          ]
        },
        {
          className: "number",
          variants: [
            {
              match: /-?\b0[xX]\.[a-fA-F0-9](_?[a-fA-F0-9])*[pP][+-]?\d(_?\d)*i?/,
              relevance: 0
            },
            {
              match: /-?\b0[xX](_?[a-fA-F0-9])+((\.([a-fA-F0-9](_?[a-fA-F0-9])*)?)?[pP][+-]?\d(_?\d)*)?i?/,
              relevance: 0
            },
            {
              match: /-?\b0[oO](_?[0-7])*i?/,
              relevance: 0
            },
            {
              match: /-?\.\d(_?\d)*([eE][+-]?\d(_?\d)*)?i?/,
              relevance: 0
            },
            {
              match: /-?\b\d(_?\d)*(\.(\d(_?\d)*)?)?([eE][+-]?\d(_?\d)*)?i?/,
              relevance: 0
            }
          ]
        },
        {
          begin: /:=/
        },
        {
          className: "function",
          beginKeywords: "func",
          end: "\\s*(\\{|$)",
          excludeEnd: true,
          contains: [
            hljs.TITLE_MODE,
            {
              className: "params",
              begin: /\(/,
              end: /\)/,
              endsParent: true,
              keywords: KEYWORDS2,
              illegal: /["']/
            }
          ]
        }
      ]
    };
  }

  // node_modules/highlight.js/es/languages/bash.js
  function bash(hljs) {
    const regex = hljs.regex;
    const VAR = {};
    const BRACED_VAR = {
      begin: /\$\{/,
      end: /\}/,
      contains: [
        "self",
        {
          begin: /:-/,
          contains: [VAR]
        }
      ]
    };
    Object.assign(VAR, {
      className: "variable",
      variants: [
        { begin: regex.concat(/\$[\w\d#@][\w\d_]*/, `(?![\\w\\d])(?![$])`) },
        BRACED_VAR
      ]
    });
    const SUBST = {
      className: "subst",
      begin: /\$\(/,
      end: /\)/,
      contains: [hljs.BACKSLASH_ESCAPE]
    };
    const COMMENT = hljs.inherit(hljs.COMMENT(), {
      match: [
        /(^|\s)/,
        /#.*$/
      ],
      scope: {
        2: "comment"
      }
    });
    const HERE_DOC = {
      begin: /<<-?\s*(?=\w+)/,
      starts: { contains: [
        hljs.END_SAME_AS_BEGIN({
          begin: /(\w+)/,
          end: /(\w+)/,
          className: "string"
        })
      ] }
    };
    const QUOTE_STRING = {
      className: "string",
      begin: /"/,
      end: /"/,
      contains: [
        hljs.BACKSLASH_ESCAPE,
        VAR,
        SUBST
      ]
    };
    SUBST.contains.push(QUOTE_STRING);
    const ESCAPED_QUOTE = {
      match: /\\"/
    };
    const APOS_STRING = {
      className: "string",
      begin: /'/,
      end: /'/
    };
    const ESCAPED_APOS = {
      match: /\\'/
    };
    const ARITHMETIC = {
      begin: /\$?\(\(/,
      end: /\)\)/,
      contains: [
        {
          begin: /\d+#[0-9a-f]+/,
          className: "number"
        },
        hljs.NUMBER_MODE,
        VAR
      ]
    };
    const SH_LIKE_SHELLS = [
      "fish",
      "bash",
      "zsh",
      "sh",
      "csh",
      "ksh",
      "tcsh",
      "dash",
      "scsh"
    ];
    const KNOWN_SHEBANG = hljs.SHEBANG({
      binary: `(${SH_LIKE_SHELLS.join("|")})`,
      relevance: 10
    });
    const FUNCTION = {
      className: "function",
      begin: /\w[\w\d_]*\s*\(\s*\)\s*\{/,
      returnBegin: true,
      contains: [hljs.inherit(hljs.TITLE_MODE, { begin: /\w[\w\d_]*/ })],
      relevance: 0
    };
    const KEYWORDS2 = [
      "if",
      "then",
      "else",
      "elif",
      "fi",
      "time",
      "for",
      "while",
      "until",
      "in",
      "do",
      "done",
      "case",
      "esac",
      "coproc",
      "function",
      "select"
    ];
    const LITERALS2 = [
      "true",
      "false"
    ];
    const PATH_MODE = { match: /(\/[a-z._-]+)+/ };
    const SHELL_BUILT_INS = [
      "break",
      "cd",
      "continue",
      "eval",
      "exec",
      "exit",
      "export",
      "getopts",
      "hash",
      "pwd",
      "readonly",
      "return",
      "shift",
      "test",
      "times",
      "trap",
      "umask",
      "unset"
    ];
    const BASH_BUILT_INS = [
      "alias",
      "bind",
      "builtin",
      "caller",
      "command",
      "declare",
      "echo",
      "enable",
      "help",
      "let",
      "local",
      "logout",
      "mapfile",
      "printf",
      "read",
      "readarray",
      "source",
      "sudo",
      "type",
      "typeset",
      "ulimit",
      "unalias"
    ];
    const ZSH_BUILT_INS = [
      "autoload",
      "bg",
      "bindkey",
      "bye",
      "cap",
      "chdir",
      "clone",
      "comparguments",
      "compcall",
      "compctl",
      "compdescribe",
      "compfiles",
      "compgroups",
      "compquote",
      "comptags",
      "comptry",
      "compvalues",
      "dirs",
      "disable",
      "disown",
      "echotc",
      "echoti",
      "emulate",
      "fc",
      "fg",
      "float",
      "functions",
      "getcap",
      "getln",
      "history",
      "integer",
      "jobs",
      "kill",
      "limit",
      "log",
      "noglob",
      "popd",
      "print",
      "pushd",
      "pushln",
      "rehash",
      "sched",
      "setcap",
      "setopt",
      "stat",
      "suspend",
      "ttyctl",
      "unfunction",
      "unhash",
      "unlimit",
      "unsetopt",
      "vared",
      "wait",
      "whence",
      "where",
      "which",
      "zcompile",
      "zformat",
      "zftp",
      "zle",
      "zmodload",
      "zparseopts",
      "zprof",
      "zpty",
      "zregexparse",
      "zsocket",
      "zstyle",
      "ztcp"
    ];
    const GNU_CORE_UTILS = [
      "chcon",
      "chgrp",
      "chown",
      "chmod",
      "cp",
      "dd",
      "df",
      "dir",
      "dircolors",
      "ln",
      "ls",
      "mkdir",
      "mkfifo",
      "mknod",
      "mktemp",
      "mv",
      "realpath",
      "rm",
      "rmdir",
      "shred",
      "sync",
      "touch",
      "truncate",
      "vdir",
      "b2sum",
      "base32",
      "base64",
      "cat",
      "cksum",
      "comm",
      "csplit",
      "cut",
      "expand",
      "fmt",
      "fold",
      "head",
      "join",
      "md5sum",
      "nl",
      "numfmt",
      "od",
      "paste",
      "ptx",
      "pr",
      "sha1sum",
      "sha224sum",
      "sha256sum",
      "sha384sum",
      "sha512sum",
      "shuf",
      "sort",
      "split",
      "sum",
      "tac",
      "tail",
      "tr",
      "tsort",
      "unexpand",
      "uniq",
      "wc",
      "arch",
      "basename",
      "chroot",
      "date",
      "dirname",
      "du",
      "echo",
      "env",
      "expr",
      "factor",
      "groups",
      "hostid",
      "id",
      "link",
      "logname",
      "nice",
      "nohup",
      "nproc",
      "pathchk",
      "pinky",
      "printenv",
      "printf",
      "pwd",
      "readlink",
      "runcon",
      "seq",
      "sleep",
      "stat",
      "stdbuf",
      "stty",
      "tee",
      "test",
      "timeout",
      "tty",
      "uname",
      "unlink",
      "uptime",
      "users",
      "who",
      "whoami",
      "yes"
    ];
    return {
      name: "Bash",
      aliases: [
        "sh",
        "zsh"
      ],
      keywords: {
        $pattern: /\b[a-z][a-z0-9._-]+\b/,
        keyword: KEYWORDS2,
        literal: LITERALS2,
        built_in: [
          ...SHELL_BUILT_INS,
          ...BASH_BUILT_INS,
          "set",
          "shopt",
          ...ZSH_BUILT_INS,
          ...GNU_CORE_UTILS
        ]
      },
      contains: [
        KNOWN_SHEBANG,
        hljs.SHEBANG(),
        FUNCTION,
        ARITHMETIC,
        COMMENT,
        HERE_DOC,
        PATH_MODE,
        QUOTE_STRING,
        ESCAPED_QUOTE,
        APOS_STRING,
        ESCAPED_APOS,
        VAR
      ]
    };
  }

  // node_modules/highlight.js/es/languages/json.js
  function json(hljs) {
    const ATTRIBUTE = {
      className: "attr",
      begin: /"(\\.|[^\\"\r\n])*"(?=\s*:)/,
      relevance: 1.01
    };
    const PUNCTUATION = {
      match: /[{}[\],:]/,
      className: "punctuation",
      relevance: 0
    };
    const LITERALS2 = [
      "true",
      "false",
      "null"
    ];
    const LITERALS_MODE = {
      scope: "literal",
      beginKeywords: LITERALS2.join(" ")
    };
    return {
      name: "JSON",
      aliases: ["jsonc"],
      keywords: {
        literal: LITERALS2
      },
      contains: [
        ATTRIBUTE,
        PUNCTUATION,
        hljs.QUOTE_STRING_MODE,
        LITERALS_MODE,
        hljs.C_NUMBER_MODE,
        hljs.C_LINE_COMMENT_MODE,
        hljs.C_BLOCK_COMMENT_MODE
      ],
      illegal: "\\S"
    };
  }

  // node_modules/highlight.js/es/languages/yaml.js
  function yaml(hljs) {
    const LITERALS2 = "true false yes no null";
    const URI_CHARACTERS = "[\\w#;/?:@&=+$,.~*'()[\\]]+";
    const KEY = {
      className: "attr",
      variants: [
        { begin: /[\w*@][\w*@ :()\./-]*:(?=[ \t]|$)/ },
        {
          begin: /"[\w*@][\w*@ :()\./-]*":(?=[ \t]|$)/
        },
        {
          begin: /'[\w*@][\w*@ :()\./-]*':(?=[ \t]|$)/
        }
      ]
    };
    const TEMPLATE_VARIABLES = {
      className: "template-variable",
      variants: [
        {
          begin: /\{\{/,
          end: /\}\}/
        },
        {
          begin: /%\{/,
          end: /\}/
        }
      ]
    };
    const SINGLE_QUOTE_STRING = {
      className: "string",
      relevance: 0,
      begin: /'/,
      end: /'/,
      contains: [
        {
          match: /''/,
          scope: "char.escape",
          relevance: 0
        }
      ]
    };
    const STRING = {
      className: "string",
      relevance: 0,
      variants: [
        {
          begin: /"/,
          end: /"/
        },
        { begin: /\S+/ }
      ],
      contains: [
        hljs.BACKSLASH_ESCAPE,
        TEMPLATE_VARIABLES
      ]
    };
    const CONTAINER_STRING = hljs.inherit(STRING, { variants: [
      {
        begin: /'/,
        end: /'/,
        contains: [
          {
            begin: /''/,
            relevance: 0
          }
        ]
      },
      {
        begin: /"/,
        end: /"/
      },
      { begin: /[^\s,{}[\]]+/ }
    ] });
    const DATE_RE = "[0-9]{4}(-[0-9][0-9]){0,2}";
    const TIME_RE = "([Tt \\t][0-9][0-9]?(:[0-9][0-9]){2})?";
    const FRACTION_RE = "(\\.[0-9]*)?";
    const ZONE_RE = "([ \\t])*(Z|[-+][0-9][0-9]?(:[0-9][0-9])?)?";
    const TIMESTAMP = {
      className: "number",
      begin: "\\b" + DATE_RE + TIME_RE + FRACTION_RE + ZONE_RE + "\\b"
    };
    const VALUE_CONTAINER = {
      end: ",",
      endsWithParent: true,
      excludeEnd: true,
      keywords: LITERALS2,
      relevance: 0
    };
    const OBJECT = {
      begin: /\{/,
      end: /\}/,
      contains: [VALUE_CONTAINER],
      illegal: "\\n",
      relevance: 0
    };
    const ARRAY = {
      begin: "\\[",
      end: "\\]",
      contains: [VALUE_CONTAINER],
      illegal: "\\n",
      relevance: 0
    };
    const MODES = [
      KEY,
      {
        className: "meta",
        begin: "^---\\s*$",
        relevance: 10
      },
      {
        className: "string",
        begin: "[\\|>]([1-9]?[+-])?[ ]*\\n( +)[^ ][^\\n]*\\n(\\2[^\\n]+\\n?)*"
      },
      {
        begin: "<%[%=-]?",
        end: "[%-]?%>",
        subLanguage: "ruby",
        excludeBegin: true,
        excludeEnd: true,
        relevance: 0
      },
      {
        className: "type",
        begin: "!\\w+!" + URI_CHARACTERS
      },
      {
        className: "type",
        begin: "!<" + URI_CHARACTERS + ">"
      },
      {
        className: "type",
        begin: "!" + URI_CHARACTERS
      },
      {
        className: "type",
        begin: "!!" + URI_CHARACTERS
      },
      {
        className: "meta",
        begin: "&" + hljs.UNDERSCORE_IDENT_RE + "$"
      },
      {
        className: "meta",
        begin: "\\*" + hljs.UNDERSCORE_IDENT_RE + "$"
      },
      {
        className: "bullet",
        begin: "-(?=[ ]|$)",
        relevance: 0
      },
      hljs.HASH_COMMENT_MODE,
      {
        beginKeywords: LITERALS2,
        keywords: { literal: LITERALS2 }
      },
      TIMESTAMP,
      {
        className: "number",
        begin: hljs.C_NUMBER_RE + "\\b",
        relevance: 0
      },
      OBJECT,
      ARRAY,
      SINGLE_QUOTE_STRING,
      STRING
    ];
    const VALUE_MODES = [...MODES];
    VALUE_MODES.pop();
    VALUE_MODES.push(CONTAINER_STRING);
    VALUE_CONTAINER.contains = VALUE_MODES;
    return {
      name: "YAML",
      case_insensitive: true,
      aliases: ["yml"],
      contains: MODES
    };
  }

  // node_modules/highlight.js/es/languages/typescript.js
  var IDENT_RE2 = "[A-Za-z$_][0-9A-Za-z$_]*";
  var KEYWORDS2 = [
    "as",
    "in",
    "of",
    "if",
    "for",
    "while",
    "finally",
    "var",
    "new",
    "function",
    "do",
    "return",
    "void",
    "else",
    "break",
    "catch",
    "instanceof",
    "with",
    "throw",
    "case",
    "default",
    "try",
    "switch",
    "continue",
    "typeof",
    "delete",
    "let",
    "yield",
    "const",
    "class",
    "debugger",
    "async",
    "await",
    "static",
    "import",
    "from",
    "export",
    "extends",
    "using"
  ];
  var LITERALS2 = [
    "true",
    "false",
    "null",
    "undefined",
    "NaN",
    "Infinity"
  ];
  var TYPES2 = [
    "Object",
    "Function",
    "Boolean",
    "Symbol",
    "Math",
    "Date",
    "Number",
    "BigInt",
    "String",
    "RegExp",
    "Array",
    "Float32Array",
    "Float64Array",
    "Int8Array",
    "Uint8Array",
    "Uint8ClampedArray",
    "Int16Array",
    "Int32Array",
    "Uint16Array",
    "Uint32Array",
    "BigInt64Array",
    "BigUint64Array",
    "Set",
    "Map",
    "WeakSet",
    "WeakMap",
    "ArrayBuffer",
    "SharedArrayBuffer",
    "Atomics",
    "DataView",
    "JSON",
    "Promise",
    "Generator",
    "GeneratorFunction",
    "AsyncFunction",
    "Reflect",
    "Proxy",
    "Intl",
    "WebAssembly"
  ];
  var ERROR_TYPES2 = [
    "Error",
    "EvalError",
    "InternalError",
    "RangeError",
    "ReferenceError",
    "SyntaxError",
    "TypeError",
    "URIError"
  ];
  var BUILT_IN_GLOBALS2 = [
    "setInterval",
    "setTimeout",
    "clearInterval",
    "clearTimeout",
    "require",
    "exports",
    "eval",
    "isFinite",
    "isNaN",
    "parseFloat",
    "parseInt",
    "decodeURI",
    "decodeURIComponent",
    "encodeURI",
    "encodeURIComponent",
    "escape",
    "unescape"
  ];
  var BUILT_IN_VARIABLES2 = [
    "arguments",
    "this",
    "super",
    "console",
    "window",
    "document",
    "localStorage",
    "sessionStorage",
    "module",
    "global"
  ];
  var BUILT_INS2 = [].concat(BUILT_IN_GLOBALS2, TYPES2, ERROR_TYPES2);
  function javascript2(hljs) {
    const regex = hljs.regex;
    const hasClosingTag = (match, { after }) => {
      const tag = "</" + match[0].slice(1);
      const pos = match.input.indexOf(tag, after);
      return pos !== -1;
    };
    const IDENT_RE$1 = IDENT_RE2;
    const FRAGMENT = {
      begin: "<>",
      end: "</>"
    };
    const XML_SELF_CLOSING = /<[A-Za-z0-9\\._:-]+\s*\/>/;
    const XML_TAG = {
      begin: /<[A-Za-z0-9\\._:-]+/,
      end: /\/[A-Za-z0-9\\._:-]+>|\/>/,
      isTrulyOpeningTag: (match, response) => {
        const afterMatchIndex = match[0].length + match.index;
        const nextChar = match.input[afterMatchIndex];
        if (nextChar === "<" || nextChar === ",") {
          response.ignoreMatch();
          return;
        }
        if (nextChar === ">") {
          if (!hasClosingTag(match, { after: afterMatchIndex })) {
            response.ignoreMatch();
          }
        }
        let m2;
        const afterMatch = match.input.substring(afterMatchIndex);
        if (m2 = afterMatch.match(/^\s*=/)) {
          response.ignoreMatch();
          return;
        }
        if (m2 = afterMatch.match(/^\s+extends\s+/)) {
          if (m2.index === 0) {
            response.ignoreMatch();
            return;
          }
        }
      }
    };
    const KEYWORDS$1 = {
      $pattern: IDENT_RE2,
      keyword: KEYWORDS2,
      literal: LITERALS2,
      built_in: BUILT_INS2,
      "variable.language": BUILT_IN_VARIABLES2
    };
    const decimalDigits = "[0-9](_?[0-9])*";
    const frac = `\\.(${decimalDigits})`;
    const decimalInteger = `0|[1-9](_?[0-9])*|0[0-7]*[89][0-9]*`;
    const NUMBER = {
      className: "number",
      variants: [
        { begin: `(\\b(${decimalInteger})((${frac})|\\.)?|(${frac}))` + `[eE][+-]?(${decimalDigits})\\b` },
        { begin: `\\b(${decimalInteger})\\b((${frac})\\b|\\.)?|(${frac})\\b` },
        { begin: `\\b(0|[1-9](_?[0-9])*)n\\b` },
        { begin: "\\b0[xX][0-9a-fA-F](_?[0-9a-fA-F])*n?\\b" },
        { begin: "\\b0[bB][0-1](_?[0-1])*n?\\b" },
        { begin: "\\b0[oO][0-7](_?[0-7])*n?\\b" },
        { begin: "\\b0[0-7]+n?\\b" }
      ],
      relevance: 0
    };
    const SUBST = {
      className: "subst",
      begin: "\\$\\{",
      end: "\\}",
      keywords: KEYWORDS$1,
      contains: []
    };
    const HTML_TEMPLATE = {
      begin: ".?html`",
      end: "",
      starts: {
        end: "`",
        returnEnd: false,
        contains: [
          hljs.BACKSLASH_ESCAPE,
          SUBST
        ],
        subLanguage: "xml"
      }
    };
    const CSS_TEMPLATE = {
      begin: ".?css`",
      end: "",
      starts: {
        end: "`",
        returnEnd: false,
        contains: [
          hljs.BACKSLASH_ESCAPE,
          SUBST
        ],
        subLanguage: "css"
      }
    };
    const GRAPHQL_TEMPLATE = {
      begin: ".?gql`",
      end: "",
      starts: {
        end: "`",
        returnEnd: false,
        contains: [
          hljs.BACKSLASH_ESCAPE,
          SUBST
        ],
        subLanguage: "graphql"
      }
    };
    const TEMPLATE_STRING = {
      className: "string",
      begin: "`",
      end: "`",
      contains: [
        hljs.BACKSLASH_ESCAPE,
        SUBST
      ]
    };
    const JSDOC_COMMENT = hljs.COMMENT(/\/\*\*(?!\/)/, "\\*/", {
      relevance: 0,
      contains: [
        {
          begin: "(?=@[A-Za-z]+)",
          relevance: 0,
          contains: [
            {
              className: "doctag",
              begin: "@[A-Za-z]+"
            },
            {
              className: "type",
              begin: "\\{",
              end: "\\}",
              excludeEnd: true,
              excludeBegin: true,
              relevance: 0
            },
            {
              className: "variable",
              begin: IDENT_RE$1 + "(?=\\s*(-)|$)",
              endsParent: true,
              relevance: 0
            },
            {
              begin: /(?=[^\n])\s/,
              relevance: 0
            }
          ]
        }
      ]
    });
    const COMMENT = {
      className: "comment",
      variants: [
        JSDOC_COMMENT,
        hljs.C_BLOCK_COMMENT_MODE,
        hljs.C_LINE_COMMENT_MODE
      ]
    };
    const SUBST_INTERNALS = [
      hljs.APOS_STRING_MODE,
      hljs.QUOTE_STRING_MODE,
      HTML_TEMPLATE,
      CSS_TEMPLATE,
      GRAPHQL_TEMPLATE,
      TEMPLATE_STRING,
      { match: /\$\d+/ },
      NUMBER
    ];
    SUBST.contains = SUBST_INTERNALS.concat({
      begin: /\{/,
      end: /\}/,
      keywords: KEYWORDS$1,
      contains: [
        "self"
      ].concat(SUBST_INTERNALS)
    });
    const SUBST_AND_COMMENTS = [].concat(COMMENT, SUBST.contains);
    const PARAMS_CONTAINS = SUBST_AND_COMMENTS.concat([
      {
        begin: /(\s*)\(/,
        end: /\)/,
        keywords: KEYWORDS$1,
        contains: ["self"].concat(SUBST_AND_COMMENTS)
      }
    ]);
    const PARAMS = {
      className: "params",
      begin: /(\s*)\(/,
      end: /\)/,
      excludeBegin: true,
      excludeEnd: true,
      keywords: KEYWORDS$1,
      contains: PARAMS_CONTAINS
    };
    const CLASS_OR_EXTENDS = {
      variants: [
        {
          match: [
            /class/,
            /\s+/,
            IDENT_RE$1,
            /\s+/,
            /extends/,
            /\s+/,
            regex.concat(IDENT_RE$1, "(", regex.concat(/\./, IDENT_RE$1), ")*")
          ],
          scope: {
            1: "keyword",
            3: "title.class",
            5: "keyword",
            7: "title.class.inherited"
          }
        },
        {
          match: [
            /class/,
            /\s+/,
            IDENT_RE$1
          ],
          scope: {
            1: "keyword",
            3: "title.class"
          }
        }
      ]
    };
    const CLASS_REFERENCE = {
      relevance: 0,
      match: regex.either(/\bJSON/, /\b[A-Z][a-z]+([A-Z][a-z]*|\d)*/, /\b[A-Z]{2,}([A-Z][a-z]+|\d)+([A-Z][a-z]*)*/, /\b[A-Z]{2,}[a-z]+([A-Z][a-z]+|\d)*([A-Z][a-z]*)*/),
      className: "title.class",
      keywords: {
        _: [
          ...TYPES2,
          ...ERROR_TYPES2
        ]
      }
    };
    const USE_STRICT = {
      label: "use_strict",
      className: "meta",
      relevance: 10,
      begin: /^\s*['"]use (strict|asm)['"]/
    };
    const FUNCTION_DEFINITION = {
      variants: [
        {
          match: [
            /function/,
            /\s+/,
            IDENT_RE$1,
            /(?=\s*\()/
          ]
        },
        {
          match: [
            /function/,
            /\s*(?=\()/
          ]
        }
      ],
      className: {
        1: "keyword",
        3: "title.function"
      },
      label: "func.def",
      contains: [PARAMS],
      illegal: /%/
    };
    const UPPER_CASE_CONSTANT = {
      relevance: 0,
      match: /\b[A-Z][A-Z_0-9]+\b/,
      className: "variable.constant"
    };
    function noneOf(list) {
      return regex.concat("(?!", list.join("|"), ")");
    }
    const FUNCTION_CALL = {
      match: regex.concat(/\b/, noneOf([
        ...BUILT_IN_GLOBALS2,
        "super",
        "import"
      ].map((x2) => `${x2}\\s*\\(`)), IDENT_RE$1, regex.lookahead(/\s*\(/)),
      className: "title.function",
      relevance: 0
    };
    const PROPERTY_ACCESS = {
      begin: regex.concat(/\./, regex.lookahead(regex.concat(IDENT_RE$1, /(?![0-9A-Za-z$_(])/))),
      end: IDENT_RE$1,
      excludeBegin: true,
      keywords: "prototype",
      className: "property",
      relevance: 0
    };
    const GETTER_OR_SETTER = {
      match: [
        /get|set/,
        /\s+/,
        IDENT_RE$1,
        /(?=\()/
      ],
      className: {
        1: "keyword",
        3: "title.function"
      },
      contains: [
        {
          begin: /\(\)/
        },
        PARAMS
      ]
    };
    const FUNC_LEAD_IN_RE = "(\\(" + "[^()]*(\\(" + "[^()]*(\\(" + "[^()]*" + "\\)[^()]*)*" + "\\)[^()]*)*" + "\\)|" + hljs.UNDERSCORE_IDENT_RE + ")\\s*=>";
    const FUNCTION_VARIABLE = {
      match: [
        /const|var|let/,
        /\s+/,
        IDENT_RE$1,
        /\s*/,
        /=\s*/,
        /(async\s*)?/,
        regex.lookahead(FUNC_LEAD_IN_RE)
      ],
      keywords: "async",
      className: {
        1: "keyword",
        3: "title.function"
      },
      contains: [
        PARAMS
      ]
    };
    return {
      name: "JavaScript",
      aliases: ["js", "jsx", "mjs", "cjs"],
      keywords: KEYWORDS$1,
      exports: { PARAMS_CONTAINS, CLASS_REFERENCE },
      illegal: /#(?![$_A-z])/,
      contains: [
        hljs.SHEBANG({
          label: "shebang",
          binary: "node",
          relevance: 5
        }),
        USE_STRICT,
        hljs.APOS_STRING_MODE,
        hljs.QUOTE_STRING_MODE,
        HTML_TEMPLATE,
        CSS_TEMPLATE,
        GRAPHQL_TEMPLATE,
        TEMPLATE_STRING,
        COMMENT,
        { match: /\$\d+/ },
        NUMBER,
        CLASS_REFERENCE,
        {
          scope: "attr",
          match: IDENT_RE$1 + regex.lookahead(":"),
          relevance: 0
        },
        FUNCTION_VARIABLE,
        {
          begin: "(" + hljs.RE_STARTERS_RE + "|\\b(case|return|throw)\\b)\\s*",
          keywords: "return throw case",
          relevance: 0,
          contains: [
            COMMENT,
            hljs.REGEXP_MODE,
            {
              className: "function",
              begin: FUNC_LEAD_IN_RE,
              returnBegin: true,
              end: "\\s*=>",
              contains: [
                {
                  className: "params",
                  variants: [
                    {
                      begin: hljs.UNDERSCORE_IDENT_RE,
                      relevance: 0
                    },
                    {
                      className: null,
                      begin: /\(\s*\)/,
                      skip: true
                    },
                    {
                      begin: /(\s*)\(/,
                      end: /\)/,
                      excludeBegin: true,
                      excludeEnd: true,
                      keywords: KEYWORDS$1,
                      contains: PARAMS_CONTAINS
                    }
                  ]
                }
              ]
            },
            {
              begin: /,/,
              relevance: 0
            },
            {
              match: /\s+/,
              relevance: 0
            },
            {
              variants: [
                { begin: FRAGMENT.begin, end: FRAGMENT.end },
                { match: XML_SELF_CLOSING },
                {
                  begin: XML_TAG.begin,
                  "on:begin": XML_TAG.isTrulyOpeningTag,
                  end: XML_TAG.end
                }
              ],
              subLanguage: "xml",
              contains: [
                {
                  begin: XML_TAG.begin,
                  end: XML_TAG.end,
                  skip: true,
                  contains: ["self"]
                }
              ]
            }
          ]
        },
        FUNCTION_DEFINITION,
        {
          beginKeywords: "while if switch catch for"
        },
        {
          begin: "\\b(?!function)" + hljs.UNDERSCORE_IDENT_RE + "\\(" + "[^()]*(\\(" + "[^()]*(\\(" + "[^()]*" + "\\)[^()]*)*" + "\\)[^()]*)*" + "\\)\\s*\\{",
          returnBegin: true,
          label: "func.def",
          contains: [
            PARAMS,
            hljs.inherit(hljs.TITLE_MODE, { begin: IDENT_RE$1, className: "title.function" })
          ]
        },
        {
          match: /\.\.\./,
          relevance: 0
        },
        PROPERTY_ACCESS,
        {
          match: "\\$" + IDENT_RE$1,
          relevance: 0
        },
        {
          match: [/\bconstructor(?=\s*\()/],
          className: { 1: "title.function" },
          contains: [PARAMS]
        },
        FUNCTION_CALL,
        UPPER_CASE_CONSTANT,
        CLASS_OR_EXTENDS,
        GETTER_OR_SETTER,
        {
          match: /\$[(.]/
        }
      ]
    };
  }
  function typescript(hljs) {
    const regex = hljs.regex;
    const tsLanguage = javascript2(hljs);
    const IDENT_RE$1 = IDENT_RE2;
    const TYPES3 = [
      "any",
      "void",
      "number",
      "boolean",
      "string",
      "object",
      "never",
      "symbol",
      "bigint",
      "unknown"
    ];
    const NAMESPACE = {
      begin: [
        /namespace/,
        /\s+/,
        hljs.IDENT_RE
      ],
      beginScope: {
        1: "keyword",
        3: "title.class"
      }
    };
    const INTERFACE = {
      beginKeywords: "interface",
      end: /\{/,
      excludeEnd: true,
      keywords: {
        keyword: "interface extends",
        built_in: TYPES3
      },
      contains: [tsLanguage.exports.CLASS_REFERENCE]
    };
    const USE_STRICT = {
      className: "meta",
      relevance: 10,
      begin: /^\s*['"]use strict['"]/
    };
    const TS_SPECIFIC_KEYWORDS = [
      "type",
      "interface",
      "public",
      "private",
      "protected",
      "implements",
      "declare",
      "abstract",
      "readonly",
      "enum",
      "override",
      "satisfies"
    ];
    const KEYWORDS$1 = {
      $pattern: IDENT_RE2,
      keyword: KEYWORDS2.concat(TS_SPECIFIC_KEYWORDS),
      literal: LITERALS2,
      built_in: BUILT_INS2.concat(TYPES3),
      "variable.language": BUILT_IN_VARIABLES2
    };
    const DECORATOR = {
      className: "meta",
      begin: "@" + IDENT_RE$1
    };
    const swapMode = (mode, label, replacement) => {
      const indx = mode.contains.findIndex((m2) => m2.label === label);
      if (indx === -1) {
        throw new Error("can not find mode to replace");
      }
      mode.contains.splice(indx, 1, replacement);
    };
    Object.assign(tsLanguage.keywords, KEYWORDS$1);
    tsLanguage.exports.PARAMS_CONTAINS.push(DECORATOR);
    const ATTRIBUTE_HIGHLIGHT = tsLanguage.contains.find((c) => c.scope === "attr");
    const OPTIONAL_KEY_OR_ARGUMENT = Object.assign({}, ATTRIBUTE_HIGHLIGHT, { match: regex.concat(IDENT_RE$1, regex.lookahead(/\s*\?:/)) });
    tsLanguage.exports.PARAMS_CONTAINS.push([
      tsLanguage.exports.CLASS_REFERENCE,
      ATTRIBUTE_HIGHLIGHT,
      OPTIONAL_KEY_OR_ARGUMENT
    ]);
    tsLanguage.contains = tsLanguage.contains.concat([
      DECORATOR,
      NAMESPACE,
      INTERFACE,
      OPTIONAL_KEY_OR_ARGUMENT
    ]);
    swapMode(tsLanguage, "shebang", hljs.SHEBANG());
    swapMode(tsLanguage, "use_strict", USE_STRICT);
    const functionDeclaration = tsLanguage.contains.find((m2) => m2.label === "func.def");
    functionDeclaration.relevance = 0;
    Object.assign(tsLanguage, {
      name: "TypeScript",
      aliases: [
        "ts",
        "tsx",
        "mts",
        "cts"
      ]
    });
    return tsLanguage;
  }

  // node_modules/highlight.js/es/languages/sql.js
  function sql(hljs) {
    const regex = hljs.regex;
    const COMMENT_MODE = hljs.COMMENT("--", "$");
    const STRING = {
      scope: "string",
      variants: [
        {
          begin: /'/,
          end: /'/,
          contains: [{ match: /''/ }]
        }
      ]
    };
    const QUOTED_IDENTIFIER = {
      begin: /"/,
      end: /"/,
      contains: [{ match: /""/ }]
    };
    const LITERALS3 = [
      "true",
      "false",
      "unknown"
    ];
    const MULTI_WORD_TYPES = [
      "double precision",
      "large object",
      "with timezone",
      "without timezone"
    ];
    const TYPES3 = [
      "bigint",
      "binary",
      "blob",
      "boolean",
      "char",
      "character",
      "clob",
      "date",
      "dec",
      "decfloat",
      "decimal",
      "float",
      "int",
      "integer",
      "interval",
      "nchar",
      "nclob",
      "national",
      "numeric",
      "real",
      "row",
      "smallint",
      "time",
      "timestamp",
      "varchar",
      "varying",
      "varbinary"
    ];
    const NON_RESERVED_WORDS = [
      "add",
      "asc",
      "collation",
      "desc",
      "final",
      "first",
      "last",
      "view"
    ];
    const RESERVED_WORDS = [
      "abs",
      "acos",
      "all",
      "allocate",
      "alter",
      "and",
      "any",
      "are",
      "array",
      "array_agg",
      "array_max_cardinality",
      "as",
      "asensitive",
      "asin",
      "asymmetric",
      "at",
      "atan",
      "atomic",
      "authorization",
      "avg",
      "begin",
      "begin_frame",
      "begin_partition",
      "between",
      "bigint",
      "binary",
      "blob",
      "boolean",
      "both",
      "by",
      "call",
      "called",
      "cardinality",
      "cascaded",
      "case",
      "cast",
      "ceil",
      "ceiling",
      "char",
      "char_length",
      "character",
      "character_length",
      "check",
      "classifier",
      "clob",
      "close",
      "coalesce",
      "collate",
      "collect",
      "column",
      "commit",
      "condition",
      "connect",
      "constraint",
      "contains",
      "convert",
      "copy",
      "corr",
      "corresponding",
      "cos",
      "cosh",
      "count",
      "covar_pop",
      "covar_samp",
      "create",
      "cross",
      "cube",
      "cume_dist",
      "current",
      "current_catalog",
      "current_date",
      "current_default_transform_group",
      "current_path",
      "current_role",
      "current_row",
      "current_schema",
      "current_time",
      "current_timestamp",
      "current_path",
      "current_role",
      "current_transform_group_for_type",
      "current_user",
      "cursor",
      "cycle",
      "date",
      "day",
      "deallocate",
      "dec",
      "decimal",
      "decfloat",
      "declare",
      "default",
      "define",
      "delete",
      "dense_rank",
      "deref",
      "describe",
      "deterministic",
      "disconnect",
      "distinct",
      "double",
      "drop",
      "dynamic",
      "each",
      "element",
      "else",
      "empty",
      "end",
      "end_frame",
      "end_partition",
      "end-exec",
      "equals",
      "escape",
      "every",
      "except",
      "exec",
      "execute",
      "exists",
      "exp",
      "external",
      "extract",
      "false",
      "fetch",
      "filter",
      "first_value",
      "float",
      "floor",
      "for",
      "foreign",
      "frame_row",
      "free",
      "from",
      "full",
      "function",
      "fusion",
      "get",
      "global",
      "grant",
      "group",
      "grouping",
      "groups",
      "having",
      "hold",
      "hour",
      "identity",
      "in",
      "indicator",
      "initial",
      "inner",
      "inout",
      "insensitive",
      "insert",
      "int",
      "integer",
      "intersect",
      "intersection",
      "interval",
      "into",
      "is",
      "join",
      "json_array",
      "json_arrayagg",
      "json_exists",
      "json_object",
      "json_objectagg",
      "json_query",
      "json_table",
      "json_table_primitive",
      "json_value",
      "lag",
      "language",
      "large",
      "last_value",
      "lateral",
      "lead",
      "leading",
      "left",
      "like",
      "like_regex",
      "listagg",
      "ln",
      "local",
      "localtime",
      "localtimestamp",
      "log",
      "log10",
      "lower",
      "match",
      "match_number",
      "match_recognize",
      "matches",
      "max",
      "member",
      "merge",
      "method",
      "min",
      "minute",
      "mod",
      "modifies",
      "module",
      "month",
      "multiset",
      "national",
      "natural",
      "nchar",
      "nclob",
      "new",
      "no",
      "none",
      "normalize",
      "not",
      "nth_value",
      "ntile",
      "null",
      "nullif",
      "numeric",
      "octet_length",
      "occurrences_regex",
      "of",
      "offset",
      "old",
      "omit",
      "on",
      "one",
      "only",
      "open",
      "or",
      "order",
      "out",
      "outer",
      "over",
      "overlaps",
      "overlay",
      "parameter",
      "partition",
      "pattern",
      "per",
      "percent",
      "percent_rank",
      "percentile_cont",
      "percentile_disc",
      "period",
      "portion",
      "position",
      "position_regex",
      "power",
      "precedes",
      "precision",
      "prepare",
      "primary",
      "procedure",
      "ptf",
      "range",
      "rank",
      "reads",
      "real",
      "recursive",
      "ref",
      "references",
      "referencing",
      "regr_avgx",
      "regr_avgy",
      "regr_count",
      "regr_intercept",
      "regr_r2",
      "regr_slope",
      "regr_sxx",
      "regr_sxy",
      "regr_syy",
      "release",
      "result",
      "return",
      "returns",
      "revoke",
      "right",
      "rollback",
      "rollup",
      "row",
      "row_number",
      "rows",
      "running",
      "savepoint",
      "scope",
      "scroll",
      "search",
      "second",
      "seek",
      "select",
      "sensitive",
      "session_user",
      "set",
      "show",
      "similar",
      "sin",
      "sinh",
      "skip",
      "smallint",
      "some",
      "specific",
      "specifictype",
      "sql",
      "sqlexception",
      "sqlstate",
      "sqlwarning",
      "sqrt",
      "start",
      "static",
      "stddev_pop",
      "stddev_samp",
      "submultiset",
      "subset",
      "substring",
      "substring_regex",
      "succeeds",
      "sum",
      "symmetric",
      "system",
      "system_time",
      "system_user",
      "table",
      "tablesample",
      "tan",
      "tanh",
      "then",
      "time",
      "timestamp",
      "timezone_hour",
      "timezone_minute",
      "to",
      "trailing",
      "translate",
      "translate_regex",
      "translation",
      "treat",
      "trigger",
      "trim",
      "trim_array",
      "true",
      "truncate",
      "uescape",
      "union",
      "unique",
      "unknown",
      "unnest",
      "update",
      "upper",
      "user",
      "using",
      "value",
      "values",
      "value_of",
      "var_pop",
      "var_samp",
      "varbinary",
      "varchar",
      "varying",
      "versioning",
      "when",
      "whenever",
      "where",
      "width_bucket",
      "window",
      "with",
      "within",
      "without",
      "year"
    ];
    const RESERVED_FUNCTIONS = [
      "abs",
      "acos",
      "array_agg",
      "asin",
      "atan",
      "avg",
      "cast",
      "ceil",
      "ceiling",
      "coalesce",
      "corr",
      "cos",
      "cosh",
      "count",
      "covar_pop",
      "covar_samp",
      "cume_dist",
      "dense_rank",
      "deref",
      "element",
      "exp",
      "extract",
      "first_value",
      "floor",
      "json_array",
      "json_arrayagg",
      "json_exists",
      "json_object",
      "json_objectagg",
      "json_query",
      "json_table",
      "json_table_primitive",
      "json_value",
      "lag",
      "last_value",
      "lead",
      "listagg",
      "ln",
      "log",
      "log10",
      "lower",
      "max",
      "min",
      "mod",
      "nth_value",
      "ntile",
      "nullif",
      "percent_rank",
      "percentile_cont",
      "percentile_disc",
      "position",
      "position_regex",
      "power",
      "rank",
      "regr_avgx",
      "regr_avgy",
      "regr_count",
      "regr_intercept",
      "regr_r2",
      "regr_slope",
      "regr_sxx",
      "regr_sxy",
      "regr_syy",
      "row_number",
      "sin",
      "sinh",
      "sqrt",
      "stddev_pop",
      "stddev_samp",
      "substring",
      "substring_regex",
      "sum",
      "tan",
      "tanh",
      "translate",
      "translate_regex",
      "treat",
      "trim",
      "trim_array",
      "unnest",
      "upper",
      "value_of",
      "var_pop",
      "var_samp",
      "width_bucket"
    ];
    const POSSIBLE_WITHOUT_PARENS = [
      "current_catalog",
      "current_date",
      "current_default_transform_group",
      "current_path",
      "current_role",
      "current_schema",
      "current_transform_group_for_type",
      "current_user",
      "session_user",
      "system_time",
      "system_user",
      "current_time",
      "localtime",
      "current_timestamp",
      "localtimestamp"
    ];
    const COMBOS = [
      "create table",
      "insert into",
      "primary key",
      "foreign key",
      "not null",
      "alter table",
      "add constraint",
      "grouping sets",
      "on overflow",
      "character set",
      "respect nulls",
      "ignore nulls",
      "nulls first",
      "nulls last",
      "depth first",
      "breadth first"
    ];
    const FUNCTIONS = RESERVED_FUNCTIONS;
    const KEYWORDS3 = [
      ...RESERVED_WORDS,
      ...NON_RESERVED_WORDS
    ].filter((keyword) => {
      return !RESERVED_FUNCTIONS.includes(keyword);
    });
    const VARIABLE = {
      scope: "variable",
      match: /@[a-z0-9][a-z0-9_]*/
    };
    const OPERATOR = {
      scope: "operator",
      match: /[-+*/=%^~]|&&?|\|\|?|!=?|<(?:=>?|<|>)?|>[>=]?/,
      relevance: 0
    };
    const FUNCTION_CALL = {
      match: regex.concat(/\b/, regex.either(...FUNCTIONS), /\s*\(/),
      relevance: 0,
      keywords: { built_in: FUNCTIONS }
    };
    function kws_to_regex(list) {
      return regex.concat(/\b/, regex.either(...list.map((kw) => {
        return kw.replace(/\s+/, "\\s+");
      })), /\b/);
    }
    const MULTI_WORD_KEYWORDS = {
      scope: "keyword",
      match: kws_to_regex(COMBOS),
      relevance: 0
    };
    function reduceRelevancy(list, {
      exceptions,
      when
    } = {}) {
      const qualifyFn = when;
      exceptions = exceptions || [];
      return list.map((item) => {
        if (item.match(/\|\d+$/) || exceptions.includes(item)) {
          return item;
        } else if (qualifyFn(item)) {
          return `${item}|0`;
        } else {
          return item;
        }
      });
    }
    return {
      name: "SQL",
      case_insensitive: true,
      illegal: /[{}]|<\//,
      keywords: {
        $pattern: /\b[\w\.]+/,
        keyword: reduceRelevancy(KEYWORDS3, { when: (x2) => x2.length < 3 }),
        literal: LITERALS3,
        type: TYPES3,
        built_in: POSSIBLE_WITHOUT_PARENS
      },
      contains: [
        {
          scope: "type",
          match: kws_to_regex(MULTI_WORD_TYPES)
        },
        MULTI_WORD_KEYWORDS,
        FUNCTION_CALL,
        VARIABLE,
        STRING,
        QUOTED_IDENTIFIER,
        hljs.C_NUMBER_MODE,
        hljs.C_BLOCK_COMMENT_MODE,
        COMMENT_MODE,
        OPERATOR
      ]
    };
  }

  // node_modules/highlight.js/es/languages/xml.js
  function xml(hljs) {
    const regex = hljs.regex;
    const TAG_NAME_RE = regex.concat(/[\p{L}_]/u, regex.optional(/[\p{L}0-9_.-]*:/u), /[\p{L}0-9_.-]*/u);
    const XML_IDENT_RE = /[\p{L}0-9._:-]+/u;
    const XML_ENTITIES = {
      className: "symbol",
      begin: /&[a-z]+;|&#[0-9]+;|&#x[a-f0-9]+;/
    };
    const XML_META_KEYWORDS = {
      begin: /\s/,
      contains: [
        {
          className: "keyword",
          begin: /#?[a-z_][a-z1-9_-]+/,
          illegal: /\n/
        }
      ]
    };
    const XML_META_PAR_KEYWORDS = hljs.inherit(XML_META_KEYWORDS, {
      begin: /\(/,
      end: /\)/
    });
    const APOS_META_STRING_MODE = hljs.inherit(hljs.APOS_STRING_MODE, { className: "string" });
    const QUOTE_META_STRING_MODE = hljs.inherit(hljs.QUOTE_STRING_MODE, { className: "string" });
    const TAG_INTERNALS = {
      endsWithParent: true,
      illegal: /</,
      relevance: 0,
      contains: [
        {
          className: "attr",
          begin: XML_IDENT_RE,
          relevance: 0
        },
        {
          begin: /=\s*/,
          relevance: 0,
          contains: [
            {
              className: "string",
              endsParent: true,
              variants: [
                {
                  begin: /"/,
                  end: /"/,
                  contains: [XML_ENTITIES]
                },
                {
                  begin: /'/,
                  end: /'/,
                  contains: [XML_ENTITIES]
                },
                { begin: /[^\s"'=<>`]+/ }
              ]
            }
          ]
        }
      ]
    };
    return {
      name: "HTML, XML",
      aliases: [
        "html",
        "xhtml",
        "rss",
        "atom",
        "xjb",
        "xsd",
        "xsl",
        "plist",
        "wsf",
        "svg"
      ],
      case_insensitive: true,
      unicodeRegex: true,
      contains: [
        {
          className: "meta",
          begin: /<![a-z]/,
          end: />/,
          relevance: 10,
          contains: [
            XML_META_KEYWORDS,
            QUOTE_META_STRING_MODE,
            APOS_META_STRING_MODE,
            XML_META_PAR_KEYWORDS,
            {
              begin: /\[/,
              end: /\]/,
              contains: [
                {
                  className: "meta",
                  begin: /<![a-z]/,
                  end: />/,
                  contains: [
                    XML_META_KEYWORDS,
                    XML_META_PAR_KEYWORDS,
                    QUOTE_META_STRING_MODE,
                    APOS_META_STRING_MODE
                  ]
                }
              ]
            }
          ]
        },
        hljs.COMMENT(/<!--/, /-->/, { relevance: 10 }),
        {
          begin: /<!\[CDATA\[/,
          end: /\]\]>/,
          relevance: 10
        },
        XML_ENTITIES,
        {
          className: "meta",
          end: /\?>/,
          variants: [
            {
              begin: /<\?xml/,
              relevance: 10,
              contains: [
                QUOTE_META_STRING_MODE
              ]
            },
            {
              begin: /<\?[a-z][a-z0-9]+/
            }
          ]
        },
        {
          className: "tag",
          begin: /<style(?=\s|>)/,
          end: />/,
          keywords: { name: "style" },
          contains: [TAG_INTERNALS],
          starts: {
            end: /<\/style>/,
            returnEnd: true,
            subLanguage: [
              "css",
              "xml"
            ]
          }
        },
        {
          className: "tag",
          begin: /<script(?=\s|>)/,
          end: />/,
          keywords: { name: "script" },
          contains: [TAG_INTERNALS],
          starts: {
            end: /<\/script>/,
            returnEnd: true,
            subLanguage: [
              "javascript",
              "handlebars",
              "xml"
            ]
          }
        },
        {
          className: "tag",
          begin: /<>|<\/>/
        },
        {
          className: "tag",
          begin: regex.concat(/</, regex.lookahead(regex.concat(TAG_NAME_RE, regex.either(/\/>/, />/, /\s/)))),
          end: /\/?>/,
          contains: [
            {
              className: "name",
              begin: TAG_NAME_RE,
              relevance: 0,
              starts: TAG_INTERNALS
            }
          ]
        },
        {
          className: "tag",
          begin: regex.concat(/<\//, regex.lookahead(regex.concat(TAG_NAME_RE, />/))),
          contains: [
            {
              className: "name",
              begin: TAG_NAME_RE,
              relevance: 0
            },
            {
              begin: />/,
              relevance: 0,
              endsParent: true
            }
          ]
        }
      ]
    };
  }

  // node_modules/highlight.js/es/languages/css.js
  var MODES = (hljs) => {
    return {
      IMPORTANT: {
        scope: "meta",
        begin: "!important"
      },
      BLOCK_COMMENT: hljs.C_BLOCK_COMMENT_MODE,
      HEXCOLOR: {
        scope: "number",
        begin: /#(([0-9a-fA-F]{3,4})|(([0-9a-fA-F]{2}){3,4}))\b/
      },
      FUNCTION_DISPATCH: {
        className: "built_in",
        begin: /[\w-]+(?=\()/
      },
      ATTRIBUTE_SELECTOR_MODE: {
        scope: "selector-attr",
        begin: /\[/,
        end: /\]/,
        illegal: "$",
        contains: [
          hljs.APOS_STRING_MODE,
          hljs.QUOTE_STRING_MODE
        ]
      },
      CSS_NUMBER_MODE: {
        scope: "number",
        begin: hljs.NUMBER_RE + "(" + "%|em|ex|ch|rem" + "|vw|vh|vmin|vmax" + "|cm|mm|in|pt|pc|px" + "|deg|grad|rad|turn" + "|s|ms" + "|Hz|kHz" + "|dpi|dpcm|dppx" + ")?",
        relevance: 0
      },
      CSS_VARIABLE: {
        className: "attr",
        begin: /--[A-Za-z_][A-Za-z0-9_-]*/
      }
    };
  };
  var HTML_TAGS = [
    "a",
    "abbr",
    "address",
    "article",
    "aside",
    "audio",
    "b",
    "blockquote",
    "body",
    "button",
    "canvas",
    "caption",
    "cite",
    "code",
    "dd",
    "del",
    "details",
    "dfn",
    "div",
    "dl",
    "dt",
    "em",
    "fieldset",
    "figcaption",
    "figure",
    "footer",
    "form",
    "h1",
    "h2",
    "h3",
    "h4",
    "h5",
    "h6",
    "header",
    "hgroup",
    "html",
    "i",
    "iframe",
    "img",
    "input",
    "ins",
    "kbd",
    "label",
    "legend",
    "li",
    "main",
    "mark",
    "menu",
    "nav",
    "object",
    "ol",
    "optgroup",
    "option",
    "p",
    "picture",
    "q",
    "quote",
    "samp",
    "section",
    "select",
    "source",
    "span",
    "strong",
    "summary",
    "sup",
    "table",
    "tbody",
    "td",
    "textarea",
    "tfoot",
    "th",
    "thead",
    "time",
    "tr",
    "ul",
    "var",
    "video"
  ];
  var SVG_TAGS = [
    "defs",
    "g",
    "marker",
    "mask",
    "pattern",
    "svg",
    "switch",
    "symbol",
    "feBlend",
    "feColorMatrix",
    "feComponentTransfer",
    "feComposite",
    "feConvolveMatrix",
    "feDiffuseLighting",
    "feDisplacementMap",
    "feFlood",
    "feGaussianBlur",
    "feImage",
    "feMerge",
    "feMorphology",
    "feOffset",
    "feSpecularLighting",
    "feTile",
    "feTurbulence",
    "linearGradient",
    "radialGradient",
    "stop",
    "circle",
    "ellipse",
    "image",
    "line",
    "path",
    "polygon",
    "polyline",
    "rect",
    "text",
    "use",
    "textPath",
    "tspan",
    "foreignObject",
    "clipPath"
  ];
  var TAGS = [
    ...HTML_TAGS,
    ...SVG_TAGS
  ];
  var MEDIA_FEATURES = [
    "any-hover",
    "any-pointer",
    "aspect-ratio",
    "color",
    "color-gamut",
    "color-index",
    "device-aspect-ratio",
    "device-height",
    "device-width",
    "display-mode",
    "forced-colors",
    "grid",
    "height",
    "hover",
    "inverted-colors",
    "monochrome",
    "orientation",
    "overflow-block",
    "overflow-inline",
    "pointer",
    "prefers-color-scheme",
    "prefers-contrast",
    "prefers-reduced-motion",
    "prefers-reduced-transparency",
    "resolution",
    "scan",
    "scripting",
    "update",
    "width",
    "min-width",
    "max-width",
    "min-height",
    "max-height"
  ].sort().reverse();
  var PSEUDO_CLASSES = [
    "active",
    "any-link",
    "blank",
    "checked",
    "current",
    "default",
    "defined",
    "dir",
    "disabled",
    "drop",
    "empty",
    "enabled",
    "first",
    "first-child",
    "first-of-type",
    "fullscreen",
    "future",
    "focus",
    "focus-visible",
    "focus-within",
    "has",
    "host",
    "host-context",
    "hover",
    "indeterminate",
    "in-range",
    "invalid",
    "is",
    "lang",
    "last-child",
    "last-of-type",
    "left",
    "link",
    "local-link",
    "not",
    "nth-child",
    "nth-col",
    "nth-last-child",
    "nth-last-col",
    "nth-last-of-type",
    "nth-of-type",
    "only-child",
    "only-of-type",
    "optional",
    "out-of-range",
    "past",
    "placeholder-shown",
    "read-only",
    "read-write",
    "required",
    "right",
    "root",
    "scope",
    "target",
    "target-within",
    "user-invalid",
    "valid",
    "visited",
    "where"
  ].sort().reverse();
  var PSEUDO_ELEMENTS = [
    "after",
    "backdrop",
    "before",
    "cue",
    "cue-region",
    "first-letter",
    "first-line",
    "grammar-error",
    "marker",
    "part",
    "placeholder",
    "selection",
    "slotted",
    "spelling-error"
  ].sort().reverse();
  var ATTRIBUTES = [
    "accent-color",
    "align-content",
    "align-items",
    "align-self",
    "alignment-baseline",
    "all",
    "anchor-name",
    "animation",
    "animation-composition",
    "animation-delay",
    "animation-direction",
    "animation-duration",
    "animation-fill-mode",
    "animation-iteration-count",
    "animation-name",
    "animation-play-state",
    "animation-range",
    "animation-range-end",
    "animation-range-start",
    "animation-timeline",
    "animation-timing-function",
    "appearance",
    "aspect-ratio",
    "backdrop-filter",
    "backface-visibility",
    "background",
    "background-attachment",
    "background-blend-mode",
    "background-clip",
    "background-color",
    "background-image",
    "background-origin",
    "background-position",
    "background-position-x",
    "background-position-y",
    "background-repeat",
    "background-size",
    "baseline-shift",
    "block-size",
    "border",
    "border-block",
    "border-block-color",
    "border-block-end",
    "border-block-end-color",
    "border-block-end-style",
    "border-block-end-width",
    "border-block-start",
    "border-block-start-color",
    "border-block-start-style",
    "border-block-start-width",
    "border-block-style",
    "border-block-width",
    "border-bottom",
    "border-bottom-color",
    "border-bottom-left-radius",
    "border-bottom-right-radius",
    "border-bottom-style",
    "border-bottom-width",
    "border-collapse",
    "border-color",
    "border-end-end-radius",
    "border-end-start-radius",
    "border-image",
    "border-image-outset",
    "border-image-repeat",
    "border-image-slice",
    "border-image-source",
    "border-image-width",
    "border-inline",
    "border-inline-color",
    "border-inline-end",
    "border-inline-end-color",
    "border-inline-end-style",
    "border-inline-end-width",
    "border-inline-start",
    "border-inline-start-color",
    "border-inline-start-style",
    "border-inline-start-width",
    "border-inline-style",
    "border-inline-width",
    "border-left",
    "border-left-color",
    "border-left-style",
    "border-left-width",
    "border-radius",
    "border-right",
    "border-right-color",
    "border-right-style",
    "border-right-width",
    "border-spacing",
    "border-start-end-radius",
    "border-start-start-radius",
    "border-style",
    "border-top",
    "border-top-color",
    "border-top-left-radius",
    "border-top-right-radius",
    "border-top-style",
    "border-top-width",
    "border-width",
    "bottom",
    "box-align",
    "box-decoration-break",
    "box-direction",
    "box-flex",
    "box-flex-group",
    "box-lines",
    "box-ordinal-group",
    "box-orient",
    "box-pack",
    "box-shadow",
    "box-sizing",
    "break-after",
    "break-before",
    "break-inside",
    "caption-side",
    "caret-color",
    "clear",
    "clip",
    "clip-path",
    "clip-rule",
    "color",
    "color-interpolation",
    "color-interpolation-filters",
    "color-profile",
    "color-rendering",
    "color-scheme",
    "column-count",
    "column-fill",
    "column-gap",
    "column-rule",
    "column-rule-color",
    "column-rule-style",
    "column-rule-width",
    "column-span",
    "column-width",
    "columns",
    "contain",
    "contain-intrinsic-block-size",
    "contain-intrinsic-height",
    "contain-intrinsic-inline-size",
    "contain-intrinsic-size",
    "contain-intrinsic-width",
    "container",
    "container-name",
    "container-type",
    "content",
    "content-visibility",
    "counter-increment",
    "counter-reset",
    "counter-set",
    "cue",
    "cue-after",
    "cue-before",
    "cursor",
    "cx",
    "cy",
    "direction",
    "display",
    "dominant-baseline",
    "empty-cells",
    "enable-background",
    "field-sizing",
    "fill",
    "fill-opacity",
    "fill-rule",
    "filter",
    "flex",
    "flex-basis",
    "flex-direction",
    "flex-flow",
    "flex-grow",
    "flex-shrink",
    "flex-wrap",
    "float",
    "flood-color",
    "flood-opacity",
    "flow",
    "font",
    "font-display",
    "font-family",
    "font-feature-settings",
    "font-kerning",
    "font-language-override",
    "font-optical-sizing",
    "font-palette",
    "font-size",
    "font-size-adjust",
    "font-smooth",
    "font-smoothing",
    "font-stretch",
    "font-style",
    "font-synthesis",
    "font-synthesis-position",
    "font-synthesis-small-caps",
    "font-synthesis-style",
    "font-synthesis-weight",
    "font-variant",
    "font-variant-alternates",
    "font-variant-caps",
    "font-variant-east-asian",
    "font-variant-emoji",
    "font-variant-ligatures",
    "font-variant-numeric",
    "font-variant-position",
    "font-variation-settings",
    "font-weight",
    "forced-color-adjust",
    "gap",
    "glyph-orientation-horizontal",
    "glyph-orientation-vertical",
    "grid",
    "grid-area",
    "grid-auto-columns",
    "grid-auto-flow",
    "grid-auto-rows",
    "grid-column",
    "grid-column-end",
    "grid-column-start",
    "grid-gap",
    "grid-row",
    "grid-row-end",
    "grid-row-start",
    "grid-template",
    "grid-template-areas",
    "grid-template-columns",
    "grid-template-rows",
    "hanging-punctuation",
    "height",
    "hyphenate-character",
    "hyphenate-limit-chars",
    "hyphens",
    "icon",
    "image-orientation",
    "image-rendering",
    "image-resolution",
    "ime-mode",
    "initial-letter",
    "initial-letter-align",
    "inline-size",
    "inset",
    "inset-area",
    "inset-block",
    "inset-block-end",
    "inset-block-start",
    "inset-inline",
    "inset-inline-end",
    "inset-inline-start",
    "isolation",
    "justify-content",
    "justify-items",
    "justify-self",
    "kerning",
    "left",
    "letter-spacing",
    "lighting-color",
    "line-break",
    "line-height",
    "line-height-step",
    "list-style",
    "list-style-image",
    "list-style-position",
    "list-style-type",
    "margin",
    "margin-block",
    "margin-block-end",
    "margin-block-start",
    "margin-bottom",
    "margin-inline",
    "margin-inline-end",
    "margin-inline-start",
    "margin-left",
    "margin-right",
    "margin-top",
    "margin-trim",
    "marker",
    "marker-end",
    "marker-mid",
    "marker-start",
    "marks",
    "mask",
    "mask-border",
    "mask-border-mode",
    "mask-border-outset",
    "mask-border-repeat",
    "mask-border-slice",
    "mask-border-source",
    "mask-border-width",
    "mask-clip",
    "mask-composite",
    "mask-image",
    "mask-mode",
    "mask-origin",
    "mask-position",
    "mask-repeat",
    "mask-size",
    "mask-type",
    "masonry-auto-flow",
    "math-depth",
    "math-shift",
    "math-style",
    "max-block-size",
    "max-height",
    "max-inline-size",
    "max-width",
    "min-block-size",
    "min-height",
    "min-inline-size",
    "min-width",
    "mix-blend-mode",
    "nav-down",
    "nav-index",
    "nav-left",
    "nav-right",
    "nav-up",
    "none",
    "normal",
    "object-fit",
    "object-position",
    "offset",
    "offset-anchor",
    "offset-distance",
    "offset-path",
    "offset-position",
    "offset-rotate",
    "opacity",
    "order",
    "orphans",
    "outline",
    "outline-color",
    "outline-offset",
    "outline-style",
    "outline-width",
    "overflow",
    "overflow-anchor",
    "overflow-block",
    "overflow-clip-margin",
    "overflow-inline",
    "overflow-wrap",
    "overflow-x",
    "overflow-y",
    "overlay",
    "overscroll-behavior",
    "overscroll-behavior-block",
    "overscroll-behavior-inline",
    "overscroll-behavior-x",
    "overscroll-behavior-y",
    "padding",
    "padding-block",
    "padding-block-end",
    "padding-block-start",
    "padding-bottom",
    "padding-inline",
    "padding-inline-end",
    "padding-inline-start",
    "padding-left",
    "padding-right",
    "padding-top",
    "page",
    "page-break-after",
    "page-break-before",
    "page-break-inside",
    "paint-order",
    "pause",
    "pause-after",
    "pause-before",
    "perspective",
    "perspective-origin",
    "place-content",
    "place-items",
    "place-self",
    "pointer-events",
    "position",
    "position-anchor",
    "position-visibility",
    "print-color-adjust",
    "quotes",
    "r",
    "resize",
    "rest",
    "rest-after",
    "rest-before",
    "right",
    "rotate",
    "row-gap",
    "ruby-align",
    "ruby-position",
    "scale",
    "scroll-behavior",
    "scroll-margin",
    "scroll-margin-block",
    "scroll-margin-block-end",
    "scroll-margin-block-start",
    "scroll-margin-bottom",
    "scroll-margin-inline",
    "scroll-margin-inline-end",
    "scroll-margin-inline-start",
    "scroll-margin-left",
    "scroll-margin-right",
    "scroll-margin-top",
    "scroll-padding",
    "scroll-padding-block",
    "scroll-padding-block-end",
    "scroll-padding-block-start",
    "scroll-padding-bottom",
    "scroll-padding-inline",
    "scroll-padding-inline-end",
    "scroll-padding-inline-start",
    "scroll-padding-left",
    "scroll-padding-right",
    "scroll-padding-top",
    "scroll-snap-align",
    "scroll-snap-stop",
    "scroll-snap-type",
    "scroll-timeline",
    "scroll-timeline-axis",
    "scroll-timeline-name",
    "scrollbar-color",
    "scrollbar-gutter",
    "scrollbar-width",
    "shape-image-threshold",
    "shape-margin",
    "shape-outside",
    "shape-rendering",
    "speak",
    "speak-as",
    "src",
    "stop-color",
    "stop-opacity",
    "stroke",
    "stroke-dasharray",
    "stroke-dashoffset",
    "stroke-linecap",
    "stroke-linejoin",
    "stroke-miterlimit",
    "stroke-opacity",
    "stroke-width",
    "tab-size",
    "table-layout",
    "text-align",
    "text-align-all",
    "text-align-last",
    "text-anchor",
    "text-combine-upright",
    "text-decoration",
    "text-decoration-color",
    "text-decoration-line",
    "text-decoration-skip",
    "text-decoration-skip-ink",
    "text-decoration-style",
    "text-decoration-thickness",
    "text-emphasis",
    "text-emphasis-color",
    "text-emphasis-position",
    "text-emphasis-style",
    "text-indent",
    "text-justify",
    "text-orientation",
    "text-overflow",
    "text-rendering",
    "text-shadow",
    "text-size-adjust",
    "text-transform",
    "text-underline-offset",
    "text-underline-position",
    "text-wrap",
    "text-wrap-mode",
    "text-wrap-style",
    "timeline-scope",
    "top",
    "touch-action",
    "transform",
    "transform-box",
    "transform-origin",
    "transform-style",
    "transition",
    "transition-behavior",
    "transition-delay",
    "transition-duration",
    "transition-property",
    "transition-timing-function",
    "translate",
    "unicode-bidi",
    "user-modify",
    "user-select",
    "vector-effect",
    "vertical-align",
    "view-timeline",
    "view-timeline-axis",
    "view-timeline-inset",
    "view-timeline-name",
    "view-transition-name",
    "visibility",
    "voice-balance",
    "voice-duration",
    "voice-family",
    "voice-pitch",
    "voice-range",
    "voice-rate",
    "voice-stress",
    "voice-volume",
    "white-space",
    "white-space-collapse",
    "widows",
    "width",
    "will-change",
    "word-break",
    "word-spacing",
    "word-wrap",
    "writing-mode",
    "x",
    "y",
    "z-index",
    "zoom"
  ].sort().reverse();
  function css(hljs) {
    const regex = hljs.regex;
    const modes = MODES(hljs);
    const VENDOR_PREFIX = { begin: /-(webkit|moz|ms|o)-(?=[a-z])/ };
    const AT_MODIFIERS = "and or not only";
    const AT_PROPERTY_RE = /@-?\w[\w]*(-\w+)*/;
    const IDENT_RE3 = "[a-zA-Z-][a-zA-Z0-9_-]*";
    const STRINGS = [
      hljs.APOS_STRING_MODE,
      hljs.QUOTE_STRING_MODE
    ];
    return {
      name: "CSS",
      case_insensitive: true,
      illegal: /[=|'\$]/,
      keywords: { keyframePosition: "from to" },
      classNameAliases: {
        keyframePosition: "selector-tag"
      },
      contains: [
        modes.BLOCK_COMMENT,
        VENDOR_PREFIX,
        modes.CSS_NUMBER_MODE,
        {
          className: "selector-id",
          begin: /#[A-Za-z0-9_-]+/,
          relevance: 0
        },
        {
          className: "selector-class",
          begin: "\\." + IDENT_RE3,
          relevance: 0
        },
        modes.ATTRIBUTE_SELECTOR_MODE,
        {
          className: "selector-pseudo",
          variants: [
            { begin: ":(" + PSEUDO_CLASSES.join("|") + ")" },
            { begin: ":(:)?(" + PSEUDO_ELEMENTS.join("|") + ")" }
          ]
        },
        modes.CSS_VARIABLE,
        {
          className: "attribute",
          begin: "\\b(" + ATTRIBUTES.join("|") + ")\\b"
        },
        {
          begin: /:/,
          end: /[;}{]/,
          contains: [
            modes.BLOCK_COMMENT,
            modes.HEXCOLOR,
            modes.IMPORTANT,
            modes.CSS_NUMBER_MODE,
            ...STRINGS,
            {
              begin: /(url|data-uri)\(/,
              end: /\)/,
              relevance: 0,
              keywords: { built_in: "url data-uri" },
              contains: [
                ...STRINGS,
                {
                  className: "string",
                  begin: /[^)]/,
                  endsWithParent: true,
                  excludeEnd: true
                }
              ]
            },
            modes.FUNCTION_DISPATCH
          ]
        },
        {
          begin: regex.lookahead(/@/),
          end: "[{;]",
          relevance: 0,
          illegal: /:/,
          contains: [
            {
              className: "keyword",
              begin: AT_PROPERTY_RE
            },
            {
              begin: /\s/,
              endsWithParent: true,
              excludeEnd: true,
              relevance: 0,
              keywords: {
                $pattern: /[a-z-]+/,
                keyword: AT_MODIFIERS,
                attribute: MEDIA_FEATURES.join(" ")
              },
              contains: [
                {
                  begin: /[a-z-]+(?=:)/,
                  className: "attribute"
                },
                ...STRINGS,
                modes.CSS_NUMBER_MODE
              ]
            }
          ]
        },
        {
          className: "selector-tag",
          begin: "\\b(" + TAGS.join("|") + ")\\b"
        }
      ]
    };
  }

  // node_modules/highlight.js/es/languages/markdown.js
  function markdown(hljs) {
    const regex = hljs.regex;
    const INLINE_HTML = {
      begin: /<\/?[A-Za-z_]/,
      end: ">",
      subLanguage: "xml",
      relevance: 0
    };
    const HORIZONTAL_RULE = {
      begin: "^[-\\*]{3,}",
      end: "$"
    };
    const CODE = {
      className: "code",
      variants: [
        { begin: "(`{3,})[^`](.|\\n)*?\\1`*[ ]*" },
        { begin: "(~{3,})[^~](.|\\n)*?\\1~*[ ]*" },
        {
          begin: "```",
          end: "```+[ ]*$"
        },
        {
          begin: "~~~",
          end: "~~~+[ ]*$"
        },
        { begin: "`.+?`" },
        {
          begin: "(?=^( {4}|\\t))",
          contains: [
            {
              begin: "^( {4}|\\t)",
              end: "(\\n)$"
            }
          ],
          relevance: 0
        }
      ]
    };
    const LIST = {
      className: "bullet",
      begin: "^[ \t]*([*+-]|(\\d+\\.))(?=\\s+)",
      end: "\\s+",
      excludeEnd: true
    };
    const LINK_REFERENCE = {
      begin: /^\[[^\n]+\]:/,
      returnBegin: true,
      contains: [
        {
          className: "symbol",
          begin: /\[/,
          end: /\]/,
          excludeBegin: true,
          excludeEnd: true
        },
        {
          className: "link",
          begin: /:\s*/,
          end: /$/,
          excludeBegin: true
        }
      ]
    };
    const URL_SCHEME = /[A-Za-z][A-Za-z0-9+.-]*/;
    const LINK = {
      variants: [
        {
          begin: /\[.+?\]\[.*?\]/,
          relevance: 0
        },
        {
          begin: /\[.+?\]\(((data|javascript|mailto):|(?:http|ftp)s?:\/\/).*?\)/,
          relevance: 2
        },
        {
          begin: regex.concat(/\[.+?\]\(/, URL_SCHEME, /:\/\/.*?\)/),
          relevance: 2
        },
        {
          begin: /\[.+?\]\([./?&#].*?\)/,
          relevance: 1
        },
        {
          begin: /\[.*?\]\(.*?\)/,
          relevance: 0
        }
      ],
      returnBegin: true,
      contains: [
        {
          match: /\[(?=\])/
        },
        {
          className: "string",
          relevance: 0,
          begin: "\\[",
          end: "\\]",
          excludeBegin: true,
          returnEnd: true
        },
        {
          className: "link",
          relevance: 0,
          begin: "\\]\\(",
          end: "\\)",
          excludeBegin: true,
          excludeEnd: true
        },
        {
          className: "symbol",
          relevance: 0,
          begin: "\\]\\[",
          end: "\\]",
          excludeBegin: true,
          excludeEnd: true
        }
      ]
    };
    const BOLD = {
      className: "strong",
      contains: [],
      variants: [
        {
          begin: /_{2}(?!\s)/,
          end: /_{2}/
        },
        {
          begin: /\*{2}(?!\s)/,
          end: /\*{2}/
        }
      ]
    };
    const ITALIC = {
      className: "emphasis",
      contains: [],
      variants: [
        {
          begin: /\*(?![*\s])/,
          end: /\*/
        },
        {
          begin: /_(?![_\s])/,
          end: /_/,
          relevance: 0
        }
      ]
    };
    const BOLD_WITHOUT_ITALIC = hljs.inherit(BOLD, { contains: [] });
    const ITALIC_WITHOUT_BOLD = hljs.inherit(ITALIC, { contains: [] });
    BOLD.contains.push(ITALIC_WITHOUT_BOLD);
    ITALIC.contains.push(BOLD_WITHOUT_ITALIC);
    let CONTAINABLE = [
      INLINE_HTML,
      LINK
    ];
    [
      BOLD,
      ITALIC,
      BOLD_WITHOUT_ITALIC,
      ITALIC_WITHOUT_BOLD
    ].forEach((m2) => {
      m2.contains = m2.contains.concat(CONTAINABLE);
    });
    CONTAINABLE = CONTAINABLE.concat(BOLD, ITALIC);
    const HEADER = {
      className: "section",
      variants: [
        {
          begin: "^#{1,6}",
          end: "$",
          contains: CONTAINABLE
        },
        {
          begin: "(?=^.+?\\n[=-]{2,}$)",
          contains: [
            { begin: "^[=-]*$" },
            {
              begin: "^",
              end: "\\n",
              contains: CONTAINABLE
            }
          ]
        }
      ]
    };
    const BLOCKQUOTE = {
      className: "quote",
      begin: "^>\\s+",
      contains: CONTAINABLE,
      end: "$"
    };
    const ENTITY = {
      scope: "literal",
      match: /&([a-zA-Z0-9]+|#[0-9]{1,7}|#[Xx][0-9a-fA-F]{1,6});/
    };
    return {
      name: "Markdown",
      aliases: [
        "md",
        "mkdown",
        "mkd"
      ],
      contains: [
        HEADER,
        INLINE_HTML,
        LIST,
        BOLD,
        ITALIC,
        BLOCKQUOTE,
        CODE,
        HORIZONTAL_RULE,
        LINK,
        LINK_REFERENCE,
        ENTITY
      ]
    };
  }

  // src/logs_view.js
  var SAFE_LEVELS = new Set(["debug", "info", "warn", "error"]);
  function stringifyFieldValue(value) {
    if (value === null || value === undefined) {
      return "";
    }
    if (typeof value === "string") {
      return value;
    }
    if (typeof value === "number" || typeof value === "boolean") {
      return String(value);
    }
    try {
      return JSON.stringify(value);
    } catch {
      return String(value);
    }
  }
  function escapeHtml(value) {
    return String(value == null ? "" : value).replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
  }
  function renderFields(fields) {
    if (!fields || typeof fields !== "object" || Array.isArray(fields)) {
      return "";
    }
    const keys = Object.keys(fields);
    if (keys.length === 0) {
      return "";
    }
    const parts = keys.map((key) => `${key}=${stringifyFieldValue(fields[key])}`);
    return ` <span class="log-fields">{${escapeHtml(parts.join(", "))}}</span>`;
  }
  function filterLogs(entries, component = "") {
    if (!Array.isArray(entries) || entries.length === 0) {
      return [];
    }
    if (!component) {
      return entries.slice();
    }
    return entries.filter((entry) => (entry?.component || "") === component);
  }
  function paginateLogs(entries, page = 1, pageSize = 100) {
    const list = Array.isArray(entries) ? entries : [];
    const size = Math.max(1, Number(pageSize) || 100);
    const totalPages = Math.max(1, Math.ceil(list.length / size));
    const currentPage = Math.min(Math.max(1, Number(page) || 1), totalPages);
    const end = list.length - (currentPage - 1) * size;
    const start = Math.max(0, end - size);
    return {
      items: list.slice(start, Math.max(start, end)),
      currentPage,
      totalPages,
      pageSize: size
    };
  }
  function renderLogs(entries, options = {}) {
    const component = options.component || "";
    const filtered = filterLogs(entries, component);
    const paged = paginateLogs(filtered, options.page, options.pageSize);
    let html = "";
    for (const entry of paged.items) {
      const levelRaw = String(entry?.level || "info").toLowerCase();
      const level = SAFE_LEVELS.has(levelRaw) ? levelRaw : "info";
      const ts = entry?.timestamp ? String(entry.timestamp).substring(11, 19) : "";
      const componentHTML = entry?.component ? `<span class="log-comp">${escapeHtml(entry.component)}</span>` : "";
      const fieldsHTML = renderFields(entry?.fields);
      const message = escapeHtml(entry?.message || "");
      html += '<div class="log-entry">' + `<span class="log-ts">${ts}</span>` + `<span class="log-badge ${level}">${level}</span>` + componentHTML + `<span class="log-msg">${message}${fieldsHTML}</span>` + "</div>";
    }
    return {
      html,
      totalItems: filtered.length,
      currentPage: paged.currentPage,
      totalPages: paged.totalPages,
      pageSize: paged.pageSize
    };
  }

  // src/app.js
  core_default.registerLanguage("javascript", javascript);
  core_default.registerLanguage("js", javascript);
  core_default.registerLanguage("python", python);
  core_default.registerLanguage("py", python);
  core_default.registerLanguage("go", go);
  core_default.registerLanguage("bash", bash);
  core_default.registerLanguage("sh", bash);
  core_default.registerLanguage("json", json);
  core_default.registerLanguage("yaml", yaml);
  core_default.registerLanguage("yml", yaml);
  core_default.registerLanguage("typescript", typescript);
  core_default.registerLanguage("ts", typescript);
  core_default.registerLanguage("sql", sql);
  core_default.registerLanguage("xml", xml);
  core_default.registerLanguage("html", xml);
  core_default.registerLanguage("css", css);
  core_default.registerLanguage("markdown", markdown);
  core_default.registerLanguage("md", markdown);
  g.setOptions({
    highlight: function(code, lang) {
      if (lang && core_default.getLanguage(lang)) {
        return core_default.highlight(code, { language: lang }).value;
      }
      return core_default.highlightAuto(code).value;
    },
    breaks: true,
    gfm: true
  });
  var tg = window.Telegram.WebApp;
  tg.ready();
  var API_BASE = location.origin;
  var initData = tg.initData || "";
  var selectedSkill = null;
  var lastSSE = { plan: 0, skills: 0, session: 0, dev: 0 };
  if (!window.ORCH_ENABLED) {
    orchTabBtn = document.querySelector('.tab[data-panel="orch"]');
    orchPanel = document.getElementById("orch");
    if (orchTabBtn)
      orchTabBtn.style.display = "none";
    if (orchPanel)
      orchPanel.style.display = "none";
    document.documentElement.style.setProperty("--tab-count", "7");
  }
  var orchTabBtn;
  var orchPanel;
  var tabs = document.querySelectorAll('.tab:not([style*="display: none"])');
  var tabIndicator = document.querySelector(".tab-indicator");
  function moveIndicator(index) {
    tabIndicator.style.transform = "translateX(" + index * 100 + "%)";
  }
  tabs.forEach((tab, index) => {
    tab.addEventListener("click", () => {
      tabs.forEach((t) => t.classList.remove("active"));
      document.querySelectorAll(".panel").forEach((p2) => p2.classList.remove("active"));
      tab.classList.add("active");
      document.getElementById(tab.dataset.panel).classList.add("active");
      moveIndicator(index);
      document.getElementById("send-bar").classList.toggle("hidden", !(tab.dataset.panel === "skills" && selectedSkill));
      var p = tab.dataset.panel;
      var fresh = lastSSE[p] && Date.now() - lastSSE[p] < 5000;
      if (p === "plan" && !fresh)
        loadPlan();
      if (p === "skills" && !fresh)
        loadSkills();
      if (p === "session" && !fresh)
        loadSession();
      if (p === "research")
        loadResearch();
      if (p === "git")
        loadGit();
      if (p === "dev" && !fresh)
        loadDev();
      if (p === "config")
        connectLogsWs();
      else
        disconnectLogsWs();
      if (p === "orch")
        connectOrchWs();
      else
        disconnectOrchWs();
    });
  });
  document.querySelectorAll(".cmd-tile").forEach((tile) => {
    tile.addEventListener("click", async () => {
      const ok = await sendCommand(tile.dataset.cmd);
      if (ok)
        flashSent(tile);
    });
  });
  async function sendCommand(cmd) {
    if (!cmd.startsWith("/"))
      return false;
    try {
      const res = await fetch(API_BASE + "/miniapp/api/command?initData=" + encodeURIComponent(initData), {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ command: cmd })
      });
      if (!res.ok)
        throw new Error("API error: " + res.status);
      return true;
    } catch (e) {
      return false;
    }
  }
  async function sendCustomCmd() {
    const input = document.getElementById("custom-cmd");
    const btn = input.nextElementSibling;
    const cmd = input.value.trim();
    if (!cmd)
      return;
    if (!cmd.startsWith("/"))
      return;
    const ok = await sendCommand(cmd);
    if (ok) {
      input.value = "";
      flashSent(btn);
    }
  }
  async function sendSkillCommand() {
    if (!selectedSkill)
      return;
    const msg = document.getElementById("skill-msg").value.trim();
    const cmd = msg ? "/skill " + selectedSkill + " " + msg : "/skill " + selectedSkill;
    const btn = document.getElementById("send-skill-btn");
    const ok = await sendCommand(cmd);
    if (ok)
      flashSent(btn);
  }
  async function startPlan() {
    const input = document.getElementById("plan-task");
    const btn = input.nextElementSibling;
    const task = input.value.trim();
    if (!task)
      return;
    const ok = await sendCommand("/plan " + task);
    if (ok) {
      input.value = "";
      flashSent(btn);
    }
  }
  function flashSent(el) {
    el.classList.add("sent");
    setTimeout(() => el.classList.remove("sent"), 600);
  }
  async function apiFetch(path) {
    const sep = path.includes("?") ? "&" : "?";
    const res = await fetch(API_BASE + path + sep + "initData=" + encodeURIComponent(initData));
    if (!res.ok)
      throw new Error("API error: " + res.status);
    return res.json();
  }
  function renderPlanFromData(data) {
    var loading = document.getElementById("plan-loading");
    var el = document.getElementById("plan-content");
    loading.classList.add("hidden");
    el.classList.remove("hidden");
    if (!data.has_plan) {
      el.innerHTML = `<div class="empty-state">No active plan.</div>
      <div class="card glass" style="margin-top:16px">
        <div class="card-title">Start a Plan</div>
        <div style="display:flex;gap:8px;margin-top:8px">
          <input id="plan-task" class="send-input glass glass-interactive" placeholder="Describe your task...">
          <button class="send-btn" onclick="startPlan()">Start</button>
        </div>
      </div>`;
      return;
    }
    var html = `<div class="card glass">
    <div class="card-title">Status</div>
    <div class="card-value">${escapeHtml2(data.status)}</div>
    <div style="color:var(--hint);margin-top:4px">Phase ${data.current_phase} / ${data.total_phases}</div>
  </div>`;
    if (data.status === "interviewing" || data.status === "review") {
      if (data.memory) {
        html += `<div class="memory-view glass">${renderSimpleMarkdown(data.memory)}</div>`;
      }
      if (data.status === "review") {
        html += `<div class="slide-approve-wrap">
        <div class="slide-approve-track glass glass-interactive" data-cmd="/plan start">
          <div class="slide-approve-thumb"><svg viewBox="0 0 24 24"><path d="M5 12h14m-6-6 6 6-6 6" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" fill="none"/></svg></div>
          <div class="slide-approve-label">Slide to Approve</div>
        </div>
      </div>
      <div class="slide-approve-wrap">
        <div class="slide-approve-track glass glass-interactive" data-cmd="/plan start clear" style="border-color:var(--warn,#ff9800)">
          <div class="slide-approve-thumb" style="background:var(--warn,#ff9800)"><svg viewBox="0 0 24 24"><path d="M5 12h14m-6-6 6 6-6 6" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" fill="none"/></svg></div>
          <div class="slide-approve-label">Approve &amp; Clear History</div>
        </div>
      </div>`;
      }
    } else {
      if (data.phases && data.phases.length > 0) {
        html += renderPhases(data.phases, data.current_phase);
      }
    }
    el.innerHTML = html;
    if (data.status === "review")
      setupSlideApprove();
  }
  var slideApproveAC = null;
  function setupSlideApprove() {
    if (slideApproveAC)
      slideApproveAC.abort();
    slideApproveAC = new AbortController;
    var signal = slideApproveAC.signal;
    var tracks = document.querySelectorAll(".slide-approve-track");
    if (!tracks.length)
      return;
    tracks.forEach(function(track) {
      var thumb = track.querySelector(".slide-approve-thumb");
      var label = track.querySelector(".slide-approve-label");
      var cmd = track.getAttribute("data-cmd") || "/plan start";
      var dragging = false;
      var startX = 0;
      var thumbStartLeft = 0;
      function getMaxLeft() {
        return track.offsetWidth - thumb.offsetWidth - 6;
      }
      function markAllApproved() {
        tracks.forEach(function(t) {
          t.classList.add("approved");
          t.querySelector(".slide-approve-label").textContent = "Approved!";
          t.querySelector(".slide-approve-thumb").classList.add("hidden");
        });
      }
      function onStart(e) {
        if (track.classList.contains("approved"))
          return;
        dragging = true;
        thumb.classList.add("dragging");
        var clientX = e.touches ? e.touches[0].clientX : e.clientX;
        startX = clientX;
        thumbStartLeft = thumb.offsetLeft - 3;
        e.preventDefault();
      }
      function onMove(e) {
        if (!dragging)
          return;
        var clientX = e.touches ? e.touches[0].clientX : e.clientX;
        var dx = clientX - startX;
        var newLeft = Math.max(0, Math.min(thumbStartLeft + dx, getMaxLeft()));
        thumb.style.left = newLeft + 3 + "px";
        e.preventDefault();
      }
      function onEnd(e) {
        if (!dragging)
          return;
        dragging = false;
        thumb.classList.remove("dragging");
        var currentLeft = thumb.offsetLeft - 3;
        var maxLeft = getMaxLeft();
        if (currentLeft >= maxLeft * 0.8) {
          markAllApproved();
          sendCommand(cmd);
        } else {
          thumb.style.left = "3px";
        }
      }
      thumb.addEventListener("touchstart", onStart, { passive: false, signal });
      thumb.addEventListener("mousedown", onStart, { signal });
      document.addEventListener("touchmove", onMove, { passive: false, signal });
      document.addEventListener("mousemove", onMove, { signal });
      document.addEventListener("touchend", onEnd, { signal });
      document.addEventListener("mouseup", onEnd, { signal });
    });
  }
  function renderSimpleMarkdown(text) {
    return g.parse(text);
  }
  async function loadTab(loadingId, contentId, label, fetchFn, renderFn) {
    var loading = document.getElementById(loadingId);
    var el = document.getElementById(contentId);
    loading.classList.remove("hidden");
    loading.textContent = "Loading " + label + "...";
    el.classList.add("hidden");
    try {
      renderFn(await fetchFn());
    } catch (e) {
      loading.textContent = "Failed to load " + label + ".";
    }
  }
  function loadPlan() {
    return loadTab("plan-loading", "plan-content", "plan", function() {
      return apiFetch("/miniapp/api/plan");
    }, renderPlanFromData);
  }
  function renderPhases(phases, currentPhase) {
    return phases.map((phase) => {
      const doneCount = phase.steps.filter((s) => s.done).length;
      const total = phase.steps.length;
      let indicatorClass, indicator;
      if (phase.number < currentPhase || total > 0 && doneCount === total) {
        indicatorClass = "done";
        indicator = "✓";
      } else if (phase.number === currentPhase) {
        indicatorClass = "current";
        indicator = String(phase.number);
      } else {
        indicatorClass = "pending";
        indicator = String(phase.number);
      }
      const progressHtml = total > 0 ? `<span class="phase-progress">${doneCount}/${total}</span>` : "";
      const stepsHtml = phase.steps.map((step) => {
        const doneClass = step.done ? "done" : "";
        const stepClass = step.done ? "step step-done" : "step";
        return `<div class="${stepClass}" data-phase="${phase.number}" data-step="${step.index}" data-done="${step.done}">
        <div class="step-check ${doneClass}"></div>
        <div class="step-text ${doneClass}">${escapeHtml2(step.description)}</div>
      </div>`;
      }).join("");
      return `<div class="phase">
      <div class="phase-header">
        <div class="phase-indicator ${indicatorClass}">${indicator}</div>
        <span class="phase-title">${escapeHtml2(phase.title || "Phase " + phase.number)}</span>
        ${progressHtml}
      </div>
      ${stepsHtml}
    </div>`;
    }).join("");
  }
  document.getElementById("plan-content").addEventListener("click", function(e) {
    const step = e.target.closest(".step");
    if (!step)
      return;
    if (step.dataset.done === "true")
      return;
    const phase = step.dataset.phase;
    const stepIdx = step.dataset.step;
    sendCommand("/plan done " + stepIdx);
  });
  function renderSkillsFromData(data) {
    var loading = document.getElementById("skills-loading");
    var el = document.getElementById("skills-list");
    loading.classList.add("hidden");
    el.classList.remove("hidden");
    if (!data || data.length === 0) {
      el.innerHTML = '<div class="empty-state">No skills installed.</div>';
      return;
    }
    el.innerHTML = data.map((s) => `<div class="skill-item glass glass-interactive" data-skill="${escapeAttr(s.name)}">
    <div class="skill-body">
      <div class="skill-name">${escapeHtml2(s.name)}</div>
      <div class="skill-desc">${escapeHtml2(s.description || "No description")}</div>
      <span class="skill-source">${escapeHtml2(s.source)}</span>
    </div>
    <span class="skill-arrow">›</span>
  </div>`).join("");
    if (selectedSkill) {
      var prev = el.querySelector('[data-skill="' + CSS.escape(selectedSkill) + '"]');
      if (prev)
        prev.classList.add("selected");
    }
  }
  document.getElementById("skills-list").addEventListener("click", function(e) {
    var item = e.target.closest(".skill-item");
    if (!item)
      return;
    var el = document.getElementById("skills-list");
    if (selectedSkill === item.dataset.skill) {
      item.classList.remove("selected");
      selectedSkill = null;
      document.getElementById("send-bar").classList.add("hidden");
      return;
    }
    el.querySelectorAll(".skill-item").forEach(function(i) {
      i.classList.remove("selected");
    });
    item.classList.add("selected");
    selectedSkill = item.dataset.skill;
    document.getElementById("send-bar").classList.remove("hidden");
    document.getElementById("skill-msg").placeholder = "Message for /" + selectedSkill + "...";
    document.getElementById("skill-msg").focus();
  });
  function loadSkills() {
    return loadTab("skills-loading", "skills-list", "skills", function() {
      return apiFetch("/miniapp/api/skills");
    }, renderSkillsFromData);
  }
  function formatAge(sec) {
    if (sec < 60)
      return sec + "s ago";
    if (sec < 3600)
      return Math.floor(sec / 60) + "m ago";
    return Math.floor(sec / 3600) + "h ago";
  }
  function shortSessionKey(key) {
    var parts = key.split(":");
    if (parts.length > 2)
      return parts.slice(2).join(":");
    return key;
  }
  function renderActiveSessions(sessions) {
    if (!sessions || sessions.length === 0) {
      return `<div class="card glass">
      <div class="card-title">Active Sessions</div>
      <div style="color:var(--hint);font-size:13px">No active sessions</div>
    </div>`;
    }
    return `<div class="card glass"><div class="card-title">Active Sessions</div>
    ${sessions.map((s) => {
      var touchDir = s.touch_dir || "—";
      return `<div style="padding:6px 0;border-bottom:1px solid var(--secondary-bg)">
        <div style="display:flex;align-items:center;gap:6px">
          <span style="color:var(--done);font-size:10px">●</span>
          <span style="font-weight:600;font-size:13px">${escapeHtml2(shortSessionKey(s.session_key))}</span>
          <span style="margin-left:auto;color:var(--hint);font-size:12px">${formatAge(s.age_sec)}</span>
        </div>
        <div style="color:var(--hint);font-size:12px;padding-left:16px">touch: ${escapeHtml2(touchDir)}</div>
      </div>`;
    }).join("")}
  </div>`;
  }
  function renderSessionFromData(sessions, stats) {
    var loading = document.getElementById("session-loading");
    var el = document.getElementById("session-content");
    loading.classList.add("hidden");
    el.classList.remove("hidden");
    var html = renderActiveSessions(sessions);
    if (!stats || stats.status === "stats not enabled") {
      html += '<div class="empty-state">Stats tracking not enabled.<br>Start gateway with --stats flag.</div>';
      el.innerHTML = html;
      return;
    }
    var since = stats.since ? new Date(stats.since).toLocaleDateString() : "N/A";
    var today = stats.today || {};
    html += `<div class="card glass">
    <div class="card-title">Today</div>
    <div class="stat-row"><span class="stat-label">Prompts</span><span class="stat-value">${today.prompts || 0}</span></div>
    <div class="stat-row"><span class="stat-label">Requests</span><span class="stat-value">${today.requests || 0}</span></div>
    <div class="stat-row"><span class="stat-label">Tokens</span><span class="stat-value">${formatTokens(today.total_tokens || 0)}</span></div>
  </div>
  <div class="card glass">
    <div class="card-title">All Time (since ${escapeHtml2(since)})</div>
    <div class="stat-row"><span class="stat-label">Prompts</span><span class="stat-value">${stats.total_prompts || 0}</span></div>
    <div class="stat-row"><span class="stat-label">Requests</span><span class="stat-value">${stats.total_requests || 0}</span></div>
    <div class="stat-row"><span class="stat-label">Total Tokens</span><span class="stat-value">${formatTokens(stats.total_tokens || 0)}</span></div>
    <div class="stat-row"><span class="stat-label">Prompt Tokens</span><span class="stat-value">${formatTokens(stats.total_prompt_tokens || 0)}</span></div>
    <div class="stat-row"><span class="stat-label">Completion Tokens</span><span class="stat-value">${formatTokens(stats.total_completion_tokens || 0)}</span></div>
  </div>`;
    el.innerHTML = html;
  }
  var cachedContextInfo = null;
  function renderContextCard(ctx) {
    if (!ctx)
      return "";
    cachedContextInfo = ctx;
    var wd = ctx.work_dir || "—";
    var pwd = ctx.plan_work_dir || "—";
    var ws = ctx.workspace || "—";
    var filesHtml = "";
    if (ctx.bootstrap && ctx.bootstrap.length) {
      filesHtml = ctx.bootstrap.map(function(b2) {
        var path = b2.path ? escapeHtml2(b2.path) : "—";
        var scope = b2.scope === "global" ? "global" : "project";
        var found = b2.path ? "var(--text)" : "var(--hint)";
        return `<div style="display:flex;gap:8px;padding:2px 0;font-size:12px">
        <span style="min-width:90px;font-weight:600;color:${found}">${escapeHtml2(b2.name)}</span>
        <span style="color:var(--hint);flex:1;overflow:hidden;text-overflow:ellipsis;white-space:nowrap" title="${path}">${path}</span>
        <span style="color:var(--hint);font-size:11px">${scope}</span>
      </div>`;
      }).join("");
    }
    return `<div class="card glass">
    <div class="card-title">Context</div>
    <div style="font-size:12px">
      <div class="stat-row"><span class="stat-label">workDir</span><span class="stat-value" style="font-size:12px;overflow:hidden;text-overflow:ellipsis" title="${escapeHtml2(wd)}">${escapeHtml2(wd)}</span></div>
      <div class="stat-row"><span class="stat-label">planWorkDir</span><span class="stat-value" style="font-size:12px;overflow:hidden;text-overflow:ellipsis" title="${escapeHtml2(pwd)}">${escapeHtml2(pwd)}</span></div>
      <div class="stat-row"><span class="stat-label">workspace</span><span class="stat-value" style="font-size:12px;overflow:hidden;text-overflow:ellipsis" title="${escapeHtml2(ws)}">${escapeHtml2(ws)}</span></div>
    </div>
    <div style="margin-top:8px">${filesHtml}</div>
    <div style="margin-top:8px;text-align:center">
      <button onclick="toggleSystemPrompt()" style="background:var(--secondary-bg);color:var(--text);border:none;padding:6px 12px;border-radius:8px;font-size:12px;cursor:pointer" id="prompt-toggle-btn">Show System Prompt</button>
    </div>
    <pre id="system-prompt-view" style="display:none;margin-top:8px;font-size:11px;max-height:400px;overflow:auto;background:var(--secondary-bg);padding:8px;border-radius:6px;white-space:pre-wrap;word-break:break-word"></pre>
  </div>`;
  }
  function toggleSystemPrompt() {
    var view = document.getElementById("system-prompt-view");
    var btn = document.getElementById("prompt-toggle-btn");
    if (!view || !btn)
      return;
    if (view.style.display === "none") {
      btn.textContent = "Loading...";
      apiFetch("/miniapp/api/prompt").then(function(data) {
        view.textContent = data.prompt || "(empty)";
        view.style.display = "block";
        btn.textContent = "Hide System Prompt";
      }).catch(function() {
        btn.textContent = "Show System Prompt";
      });
    } else {
      view.style.display = "none";
      btn.textContent = "Show System Prompt";
    }
  }
  function renderContextFromData(ctx) {
    var el = document.getElementById("context-content");
    if (el)
      el.innerHTML = renderContextCard(ctx);
  }
  function loadSession() {
    return loadTab("session-loading", "session-content", "session", function() {
      return Promise.all([
        apiFetch("/miniapp/api/session"),
        apiFetch("/miniapp/api/sessions").catch(function() {
          return [];
        }),
        apiFetch("/miniapp/api/context").catch(function() {
          return null;
        }),
        apiFetch("/miniapp/api/sessions/graph").catch(function() {
          return null;
        })
      ]);
    }, function(results) {
      renderSessionFromData(results[1], results[0]);
      renderContextFromData(results[2]);
      renderSessionGraph(results[3]);
    });
  }
  function renderSessionGraph(graph) {
    var el = document.getElementById("session-graph");
    if (!el)
      return;
    if (!graph || !graph.nodes || graph.nodes.length === 0) {
      el.classList.add("hidden");
      return;
    }
    el.classList.remove("hidden");
    var childrenMap = {};
    var roots = [];
    graph.nodes.forEach(function(n) {
      childrenMap[n.key] = [];
    });
    graph.edges.forEach(function(e) {
      if (childrenMap[e.from])
        childrenMap[e.from].push(e.to);
    });
    var nodeMap = {};
    graph.nodes.forEach(function(n) {
      nodeMap[n.key] = n;
      var isChild = graph.edges.some(function(e) {
        return e.to === n.key;
      });
      if (!isChild)
        roots.push(n.key);
    });
    function renderTreeNode(key) {
      var n = nodeMap[key];
      if (!n)
        return "";
      var icon = n.status === "completed" ? "✓" : "●";
      var iconClass = n.status === "completed" ? "completed" : "active";
      var label = n.label || n.short_key || n.key;
      var kids = childrenMap[key] || [];
      var childHtml = "";
      if (kids.length > 0) {
        childHtml = '<ul class="session-tree-children">' + kids.map(renderTreeNode).join("") + "</ul>";
      }
      return '<li class="session-tree-node">' + '<span class="session-tree-icon ' + iconClass + '">' + icon + "</span>" + '<span class="session-tree-label">' + escapeHtml2(label) + "</span>" + '<span class="session-tree-meta">turns=' + n.turn_count + "</span>" + childHtml + "</li>";
    }
    var html = '<div class="card glass"><div class="card-title">Session Graph</div>' + '<ul class="session-tree">' + roots.map(renderTreeNode).join("") + "</ul></div>";
    el.innerHTML = html;
  }
  function formatTokens(n) {
    if (n >= 1e6)
      return (n / 1e6).toFixed(1) + "M";
    if (n >= 1000)
      return (n / 1000).toFixed(1) + "K";
    return String(n);
  }
  function escapeHtml2(s) {
    if (!s)
      return "";
    return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;");
  }
  function escapeAttr(s) {
    if (!s)
      return "";
    return s.replace(/&/g, "&amp;").replace(/"/g, "&quot;").replace(/'/g, "&#39;");
  }
  var gitSelectedRepo = null;
  function loadGit() {
    gitSelectedRepo = null;
    return loadTab("git-loading", "git-content", "git", function() {
      return Promise.all([
        apiFetch("/miniapp/api/git"),
        apiFetch("/miniapp/api/worktrees").catch(function() {
          return [];
        })
      ]);
    }, function(results) {
      renderGitRepos(results[0], results[1]);
    });
  }
  function renderWorktrees(worktrees) {
    var items = Array.isArray(worktrees) ? worktrees : [];
    var html = '<div class="card glass"><div class="card-title">Worktrees</div>';
    if (items.length === 0) {
      html += '<div class="empty-state" style="padding:12px 0 4px">No active worktrees.</div>';
      html += "</div>";
      return html;
    }
    html += '<div class="worktree-list">';
    items.forEach(function(wt) {
      var dirtyClass = wt.has_uncommitted ? " dirty" : "";
      var dirtyBadge = wt.has_uncommitted ? '<span class="worktree-dirty">DIRTY</span>' : '<span class="worktree-clean">CLEAN</span>';
      var last = "(no commits)";
      if (wt.last_commit_hash) {
        last = wt.last_commit_hash + " " + (wt.last_commit_subject || "");
        if (wt.last_commit_age)
          last += " (" + wt.last_commit_age + ")";
      }
      html += '<div class="worktree-item' + dirtyClass + '">' + '<div class="worktree-main">' + '<div class="worktree-name-row">' + '<span class="worktree-name">' + escapeHtml2(wt.name) + "</span>" + dirtyBadge + "</div>" + '<div class="worktree-branch">' + escapeHtml2(wt.branch || "?") + "</div>" + '<div class="worktree-last">' + escapeHtml2(last) + "</div>" + "</div>" + '<div class="worktree-actions">' + '<button class="worktree-btn merge" data-wt-action="merge" data-wt-name="' + escapeAttr(wt.name) + '">Merge</button>' + '<button class="worktree-btn dispose" data-wt-action="dispose" data-wt-name="' + escapeAttr(wt.name) + '" data-wt-dirty="' + (wt.has_uncommitted ? "1" : "0") + '">Dispose</button>' + "</div>" + "</div>";
    });
    html += "</div></div>";
    return html;
  }
  function renderGitRepos(repos, worktrees) {
    var loading = document.getElementById("git-loading");
    var el = document.getElementById("git-content");
    loading.classList.add("hidden");
    el.classList.remove("hidden");
    var html = renderWorktrees(worktrees);
    if (!repos || repos.length === 0) {
      html += '<div class="empty-state" style="margin-top:12px">No git repositories found.</div>';
      el.innerHTML = html;
      return;
    }
    html += '<div style="padding:10px 4px 8px;font-size:12px;color:var(--hint)">Repositories</div>';
    html += repos.map(function(r) {
      return '<div class="git-repo-item glass glass-interactive" data-repo="' + escapeAttr(r.name) + '">' + '<div class="git-repo-body">' + '<div class="git-repo-name">' + escapeHtml2(r.name) + "</div>" + '<div class="git-repo-branch">' + escapeHtml2(r.branch || "?") + "</div>" + "</div>" + '<span class="git-repo-arrow">›</span>' + "</div>";
    }).join("");
    el.innerHTML = html;
  }
  async function postWorktreeAction(action, name, force) {
    var res = await fetch(API_BASE + "/miniapp/api/worktrees?initData=" + encodeURIComponent(initData), {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ action, name, force: !!force })
    });
    var data = {};
    try {
      data = await res.json();
    } catch (e) {}
    if (!res.ok) {
      throw new Error(data.error || "API error: " + res.status);
    }
    return data;
  }
  document.getElementById("git-content").addEventListener("click", async function(e) {
    var wtBtn = e.target.closest("[data-wt-action]");
    if (wtBtn) {
      var action = wtBtn.dataset.wtAction;
      var name = wtBtn.dataset.wtName;
      var isDirty = wtBtn.dataset.wtDirty === "1";
      var force = false;
      if (action === "merge") {
        if (!confirm('Merge "' + name + '" into base branch?'))
          return;
      } else if (action === "dispose") {
        if (isDirty) {
          if (!confirm('"' + name + '" has uncommitted changes. Force dispose and auto-commit before removal?'))
            return;
          force = true;
        } else if (!confirm('Dispose worktree "' + name + '"?')) {
          return;
        }
      }
      var originalText = wtBtn.textContent;
      wtBtn.disabled = true;
      wtBtn.textContent = action === "merge" ? "Merging..." : "Disposing...";
      try {
        await postWorktreeAction(action, name, force);
        await loadGit();
      } catch (err) {
        alert(err.message || "Action failed");
        wtBtn.disabled = false;
        wtBtn.textContent = originalText;
      }
      return;
    }
    var item = e.target.closest(".git-repo-item");
    if (!item)
      return;
    loadGitDetail(item.dataset.repo);
  });
  function loadGitDetail(name) {
    gitSelectedRepo = name;
    return loadTab("git-loading", "git-content", name, function() {
      return apiFetch("/miniapp/api/git?repo=" + encodeURIComponent(name));
    }, renderGitDetail);
  }
  function renderGitDetail(repo) {
    var loading = document.getElementById("git-loading");
    var el = document.getElementById("git-content");
    loading.classList.add("hidden");
    el.classList.remove("hidden");
    var html = '<button class="git-back-btn" onclick="loadGit()">← ' + escapeHtml2(repo.name || gitSelectedRepo) + "</button>";
    html += '<div class="card glass"><div class="card-title">' + escapeHtml2(repo.name) + " &mdash; " + escapeHtml2(repo.branch || "?") + "</div>";
    if (repo.modified && repo.modified.length > 0) {
      html += '<div style="padding:4px 12px 8px;font-size:12px;color:var(--hint)">Changes (' + repo.modified.length + ")</div>";
      repo.modified.forEach(function(f) {
        html += '<div class="git-commit">' + '<span class="git-status git-status-' + (f.status === "??" ? "u" : f.status.toLowerCase()) + '">' + escapeHtml2(f.status) + "</span>" + '<span class="git-subject">' + escapeHtml2(f.path) + "</span>" + "</div>";
      });
    }
    if (repo.commits && repo.commits.length > 0) {
      html += '<div style="padding:4px 12px 8px;font-size:12px;color:var(--hint)">Commits</div>';
      repo.commits.forEach(function(c) {
        html += '<div class="git-commit">' + '<span class="git-hash">' + escapeHtml2(c.hash) + "</span>" + '<span class="git-subject">' + escapeHtml2(c.subject) + "</span>" + '<span class="git-meta">' + escapeHtml2(c.date) + "</span>" + "</div>";
      });
    } else {
      html += '<div style="padding:12px;color:var(--hint)">No commits found.</div>';
    }
    html += "</div>";
    el.innerHTML = html;
  }
  var devActiveId = "";
  function renderDevFromData(data) {
    var dot = document.getElementById("dev-dot");
    var headerTarget = document.getElementById("dev-header-target");
    var targetsList = document.getElementById("dev-targets-list");
    var iframeWrap = document.getElementById("dev-iframe-wrap");
    var iframe = document.getElementById("dev-iframe");
    var targets = data.targets || [];
    devActiveId = data.active_id || "";
    if (data.active) {
      dot.classList.add("on");
      headerTarget.textContent = data.target ? data.target.replace(/^https?:\/\//, "") : "";
      iframeWrap.classList.remove("hidden");
      var iframeSrc = location.origin + "/miniapp/dev/";
      if (iframe.src !== iframeSrc)
        iframe.src = iframeSrc;
    } else {
      dot.classList.remove("on");
      headerTarget.textContent = "";
      iframeWrap.classList.add("hidden");
      iframe.src = "";
    }
    if (targets.length === 0) {
      targetsList.innerHTML = '<div class="empty-state">No targets registered.<br>Ask the agent to start a dev server.</div>';
      return;
    }
    targetsList.innerHTML = targets.map(function(t) {
      var isActive = t.id === devActiveId;
      var activeClass = isActive ? " active" : "";
      var dotClass = isActive ? " on" : "";
      var displayUrl = t.target.replace(/^https?:\/\//, "");
      return '<div class="dev-target-item glass glass-interactive' + activeClass + '" data-dev-id="' + escapeAttr(t.id) + '">' + '<span class="dev-target-dot' + dotClass + '"></span>' + '<span class="dev-target-name">' + escapeHtml2(t.name) + "</span>" + '<span class="dev-target-url">' + escapeHtml2(displayUrl) + "</span>" + '<span class="dev-target-delete" data-del-id="' + escapeAttr(t.id) + '" data-del-name="' + escapeAttr(t.name) + '">&times;</span>' + "</div>";
    }).join("");
  }
  document.getElementById("dev-targets-list").addEventListener("click", function(e) {
    var delBtn = e.target.closest(".dev-target-delete");
    if (delBtn) {
      e.stopPropagation();
      var id = delBtn.dataset.delId;
      var name = delBtn.dataset.delName;
      if (confirm('Remove "' + name + '"?')) {
        postDevUnregister(id);
      }
      return;
    }
    var card = e.target.closest("[data-dev-id]");
    if (!card)
      return;
    postDevAction(card.dataset.devId);
  });
  async function postDevAction(id) {
    var action = id === devActiveId ? "deactivate" : "activate";
    var body = action === "activate" ? { action: "activate", id } : { action: "deactivate" };
    try {
      var res = await fetch(API_BASE + "/miniapp/api/dev?initData=" + encodeURIComponent(initData), {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body)
      });
      var data = await res.json();
      if (!data.error)
        renderDevFromData(data);
    } catch (e) {}
  }
  async function postDevUnregister(id) {
    try {
      var res = await fetch(API_BASE + "/miniapp/api/dev?initData=" + encodeURIComponent(initData), {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ action: "unregister", id })
      });
      var data = await res.json();
      if (!data.error)
        renderDevFromData(data);
    } catch (e) {}
  }
  function loadDev() {
    apiFetch("/miniapp/api/dev").then(renderDevFromData).catch(function() {});
  }
  var eventSource = null;
  function connectSSE() {
    if (eventSource)
      eventSource.close();
    eventSource = new EventSource(API_BASE + "/miniapp/api/events?initData=" + encodeURIComponent(initData));
    eventSource.addEventListener("plan", function(e) {
      try {
        lastSSE.plan = Date.now();
        renderPlanFromData(JSON.parse(e.data));
      } catch (err) {}
    });
    eventSource.addEventListener("session", function(e) {
      try {
        lastSSE.session = Date.now();
        var d = JSON.parse(e.data);
        renderSessionFromData(d.sessions, d.stats);
        if (d.graph)
          renderSessionGraph(d.graph);
      } catch (err) {}
    });
    eventSource.addEventListener("skills", function(e) {
      try {
        lastSSE.skills = Date.now();
        renderSkillsFromData(JSON.parse(e.data));
      } catch (err) {}
    });
    eventSource.addEventListener("dev", function(e) {
      try {
        lastSSE.dev = Date.now();
        renderDevFromData(JSON.parse(e.data));
      } catch (err) {}
    });
    eventSource.addEventListener("context", function(e) {
      try {
        renderContextFromData(JSON.parse(e.data));
      } catch (err) {}
    });
    eventSource.addEventListener("prompt", function(e) {
      try {
        var d = JSON.parse(e.data);
        var view = document.getElementById("system-prompt-view");
        if (view && view.style.display !== "none") {
          view.textContent = d.prompt || "(empty)";
        }
      } catch (err) {}
    });
    eventSource.onerror = function() {};
  }
  connectSSE();
  loadPlan();
  var logsWs = null;
  var logsComponent = "";
  var logsEntries = [];
  var logsReconnectTimer = null;
  var logsPage = 1;
  var LOGS_PAGE_SIZE = 60;
  function connectLogsWs() {
    if (logsWs && logsWs.readyState <= 1)
      return;
    var wsProto = location.protocol === "https:" ? "wss:" : "ws:";
    var wsUrl = wsProto + "//" + location.host + "/miniapp/api/logs/ws?initData=" + encodeURIComponent(initData);
    if (logsComponent)
      wsUrl += "&component=" + encodeURIComponent(logsComponent);
    logsWs = new WebSocket(wsUrl);
    var statusDot = document.getElementById("logs-status");
    logsWs.onopen = function() {
      statusDot.classList.add("on");
    };
    logsWs.onmessage = function(e) {
      var msg = JSON.parse(e.data);
      if (msg.type === "init") {
        logsEntries = msg.entries || [];
        logsPage = 1;
      } else if (msg.type === "entry") {
        logsEntries.push(msg.entry);
        if (logsEntries.length > 200)
          logsEntries.shift();
      }
      renderLogs2();
    };
    logsWs.onclose = function() {
      statusDot.classList.remove("on");
      logsWs = null;
      var activeTab = document.querySelector(".tab.active");
      if (activeTab && activeTab.dataset.panel === "config") {
        logsReconnectTimer = setTimeout(connectLogsWs, 3000);
      }
    };
    logsWs.onerror = function() {};
  }
  function disconnectLogsWs() {
    if (logsReconnectTimer) {
      clearTimeout(logsReconnectTimer);
      logsReconnectTimer = null;
    }
    if (logsWs) {
      logsWs.close();
      logsWs = null;
    }
    document.getElementById("logs-status").classList.remove("on");
  }
  function renderLogs2() {
    var container = document.getElementById("logs-content");
    if (!container)
      return;
    var wasScrolledToBottom = container.scrollHeight - container.scrollTop - container.clientHeight < 30;
    var view = renderLogs(logsEntries, {
      component: logsComponent,
      page: logsPage,
      pageSize: LOGS_PAGE_SIZE
    });
    if (logsPage > view.totalPages) {
      logsPage = view.totalPages;
      view = renderLogs(logsEntries, {
        component: logsComponent,
        page: logsPage,
        pageSize: LOGS_PAGE_SIZE
      });
    }
    if (!view.html) {
      container.innerHTML = '<div class="empty-state">No logs.</div>';
    } else {
      container.innerHTML = view.html;
    }
    updateLogsPager(view);
    if (wasScrolledToBottom)
      container.scrollTop = container.scrollHeight;
  }
  function updateLogsPager(view) {
    var info = document.getElementById("logs-page-info");
    var prev = document.getElementById("logs-page-prev");
    var next = document.getElementById("logs-page-next");
    if (!info || !prev || !next)
      return;
    info.textContent = view.currentPage + "/" + view.totalPages + " (" + view.totalItems + ")";
    prev.disabled = view.currentPage <= 1;
    next.disabled = view.currentPage >= view.totalPages;
  }
  document.querySelector(".log-filter-chips").addEventListener("click", function(e) {
    var chip = e.target.closest(".log-filter-chip");
    if (!chip)
      return;
    document.querySelectorAll(".log-filter-chip").forEach(function(c) {
      c.classList.remove("active");
    });
    chip.classList.add("active");
    logsComponent = chip.dataset.component || "";
    logsPage = 1;
    logsEntries = [];
    renderLogs2();
    disconnectLogsWs();
    connectLogsWs();
  });
  var logsPrevButton = document.getElementById("logs-page-prev");
  if (logsPrevButton) {
    logsPrevButton.addEventListener("click", function() {
      if (logsPage <= 1)
        return;
      logsPage--;
      renderLogs2();
    });
  }
  var logsNextButton = document.getElementById("logs-page-next");
  if (logsNextButton) {
    logsNextButton.addEventListener("click", function() {
      logsPage++;
      renderLogs2();
    });
  }
  var orchCanvas = null;
  var orchCtx = null;
  var orchInited = false;
  var orchWs = null;
  var orchReconnectTimer = null;
  var _orchLastTs = null;
  var _orchBOB = [0, -1, -2, -1];
  var _orchFRAME_MS = { idle: 450, waiting: 650, toolcall: 90, talking: 280, entering: 220, exiting: 220 };
  var _orchWALK = 55;
  var _orchConductor;
  var _orchSecretary;
  var _orchHeartbeat;
  var _orchSubagents;
  var _orchSlots;
  var _orchFreeSlots;
  function _orchMakeChar(id, emoji, home) {
    return {
      id,
      emoji,
      x: home.x,
      y: home.y,
      home,
      target: null,
      state: "idle",
      frame: 0,
      frameTimer: 0,
      bubble: null,
      alive: false,
      _onArrive: null
    };
  }
  function _orchInitChars() {
    _orchConductor = _orchMakeChar("conductor", "\uD83D\uDC51", MAP_POSITIONS.conductor);
    _orchSecretary = _orchMakeChar("secretary", "\uD83D\uDC69‍\uD83D\uDCBC", MAP_POSITIONS.secretary);
    _orchHeartbeat = _orchMakeChar("heartbeat", "\uD83D\uDD4A️", MAP_POSITIONS.heartbeat || { x: 230, y: 58 });
    _orchConductor.alive = true;
    _orchSecretary.alive = false;
    _orchHeartbeat.alive = true;
    _orchConductor.statusText = null;
    _orchHeartbeat.facing = 1;
    _orchHeartbeat.flipTimer = 0;
    var ps = [
      { id: "s0", emoji: "\uD83D\uDD0D" },
      { id: "s1", emoji: "\uD83D\uDCCA" },
      { id: "s2", emoji: "\uD83D\uDCBB" },
      { id: "s3", emoji: "\uD83D\uDD27" },
      { id: "s4", emoji: "\uD83C\uDFAF" }
    ];
    _orchSubagents = ps.map(function(p, i) {
      var c = _orchMakeChar(p.id, p.emoji, MAP_POSITIONS.stations[i]);
      c.x = MAP_POSITIONS.door.x;
      c.y = MAP_POSITIONS.door.y;
      return c;
    });
    _orchSlots = {};
    _orchFreeSlots = _orchSubagents.slice();
  }
  function _orchAllChars() {
    return [_orchConductor, _orchSecretary, _orchHeartbeat].concat(_orchSubagents);
  }
  function _orchSyncBadge(id, state, alive) {
    var el = document.getElementById("orch-badge-" + id);
    if (!el)
      return;
    el.className = "orch-badge" + (alive ? " alive" : "") + (state === "talking" ? " talking" : "") + (state === "toolcall" ? " toolcall" : "") + (state === "waiting" ? " waiting" : "");
  }
  function _orchSetState(c, state, tool) {
    c.state = state;
    _orchSyncBadge(c.id, state, c.alive);
    if (c === _orchConductor) {
      if (state === "waiting")
        c.statusText = "\uD83E\uDD14";
      else if (state === "toolcall")
        c.statusText = "⌨";
      else if (state === "user_waiting")
        c.statusText = "⏳";
      else if (state === "plan_interviewing")
        c.statusText = "\uD83D\uDCCB";
      else if (state === "plan_review")
        c.statusText = "\uD83D\uDD0D";
      else if (state === "plan_executing")
        c.statusText = "▶️";
      else if (state === "plan_completed")
        c.statusText = "✅";
      else
        c.statusText = null;
      var inPlan = state.indexOf("plan_") === 0;
      if (_orchSecretary.alive !== inPlan) {
        _orchSecretary.alive = inPlan;
        _orchSyncBadge("secretary", _orchSecretary.state, _orchSecretary.alive);
      }
    }
  }
  function _orchMoveTo(c, pos, cb) {
    c.target = pos;
    c._onArrive = cb || null;
  }
  function _orchSay(c, text, ttl) {
    c.bubble = { text, ttl: ttl || 2200 };
  }
  function _orchCharForId(id) {
    if (id === "heartbeat")
      return _orchHeartbeat;
    if (_orchSlots[id])
      return _orchSlots[id];
    return _orchConductor;
  }
  function _orchSpawn(id) {
    if (/^subagent-/.test(id)) {
      var c = _orchFreeSlots.shift();
      if (!c)
        return;
      _orchSlots[id] = c;
      c.alive = true;
      c.x = MAP_POSITIONS.door.x;
      c.y = MAP_POSITIONS.door.y;
      _orchSetState(c, "entering");
      _orchMoveTo(c, c.home, function() {
        _orchSetState(c, "idle");
      });
    } else {
      var ch = _orchCharForId(id);
      ch.alive = true;
      _orchSetState(ch, "waiting");
    }
  }
  function _orchGC(id) {
    if (/^subagent-/.test(id)) {
      var c = _orchSlots[id];
      if (!c)
        return;
      delete _orchSlots[id];
      _orchFreeSlots.push(c);
      _orchSetState(c, "exiting");
      _orchMoveTo(c, MAP_POSITIONS.door, function() {
        c.alive = false;
        _orchSetState(c, "idle");
      });
    } else {
      var ch = _orchCharForId(id);
      if (ch === _orchHeartbeat) {
        _orchSetState(ch, "idle");
      } else if (ch === _orchConductor) {
        _orchSetState(ch, "user_waiting");
      } else {
        ch.alive = false;
        _orchSetState(ch, "idle");
      }
    }
  }
  function _orchConverse(fromId, toId, text) {
    var from = _orchCharForId(fromId), to = _orchCharForId(toId);
    if (!from || !to || from === to)
      return;
    var label = (text || "").slice(0, 18);
    var mid = { x: (from.x + to.x) / 2, y: (from.y + to.y) / 2 };
    _orchSetState(from, "talking");
    _orchSetState(to, "talking");
    _orchMoveTo(from, { x: mid.x - 18, y: mid.y }, function() {
      _orchSay(from, label, 2400);
    });
    _orchMoveTo(to, { x: mid.x + 18, y: mid.y }, function() {
      setTimeout(function() {
        _orchMoveTo(from, from.home, function() {
          _orchSetState(from, "idle");
        });
        _orchMoveTo(to, to.home, function() {
          _orchSetState(to, "idle");
        });
      }, 2600);
    });
  }
  function _orchUpdate(dt) {
    _orchAllChars().forEach(function(c) {
      if (!c.alive && c.state !== "entering")
        return;
      if (c === _orchHeartbeat) {
        if (c.state === "idle") {
          c.frame = 0;
        } else {
          c.frameTimer += dt;
          var pDur = c.state === "toolcall" ? 130 : 380;
          if (c.frameTimer >= pDur) {
            c.frame = (c.frame + 1) % 4;
            c.frameTimer -= pDur;
          }
        }
      } else {
        c.frameTimer += dt;
        var dur = _orchFRAME_MS[c.state] || 450;
        if (c.frameTimer >= dur) {
          c.frame = (c.frame + 1) % 4;
          c.frameTimer -= dur;
        }
      }
      if (c.target) {
        var dx = c.target.x - c.x, dy = c.target.y - c.y, dist = Math.sqrt(dx * dx + dy * dy);
        if (dist > 1.5) {
          var spd = _orchWALK * dt / 1000;
          c.x += dx / dist * spd;
          c.y += dy / dist * spd;
        } else {
          c.x = c.target.x;
          c.y = c.target.y;
          c.target = null;
          if (c._onArrive) {
            c._onArrive();
            c._onArrive = null;
          }
        }
      }
      if (c.bubble) {
        c.bubble.ttl -= dt;
        if (c.bubble.ttl <= 0)
          c.bubble = null;
      }
      if (c === _orchHeartbeat) {
        if (c.target) {
          var pdx = c.target.x - c.x;
          if (Math.abs(pdx) > 1)
            c.facing = pdx > 0 ? 1 : -1;
        } else {
          var flipRate = c.state === "toolcall" ? 280 : c.state === "waiting" ? 600 : 2800;
          c.flipTimer += dt;
          if (c.flipTimer >= flipRate) {
            c.flipTimer -= flipRate;
            c.facing = -c.facing;
          }
        }
      }
    });
  }
  function _orchDrawStatus(c) {
    if (!c.statusText)
      return;
    var yOff = _orchBOB[c.frame], cx = Math.floor(c.x), cy = Math.floor(c.y + yOff) - 20;
    orchCtx.font = "11px serif";
    orchCtx.textAlign = "center";
    orchCtx.textBaseline = "middle";
    orchCtx.fillText(c.statusText, cx, cy);
  }
  function _orchDrawBubble(c) {
    if (!c.bubble)
      return;
    var yOff = _orchBOB[c.frame], bx = c.x, by = c.y + yOff - 18;
    orchCtx.font = "7px Silkscreen,monospace";
    var tw = orchCtx.measureText(c.bubble.text).width, pw = tw + 8, ph = 12;
    var lx = Math.max(4, Math.min(316 - pw, bx - pw / 2));
    orchCtx.fillStyle = "#facc15";
    orchCtx.fillRect(Math.floor(lx), Math.floor(by - ph), Math.ceil(pw), Math.ceil(ph));
    orchCtx.fillRect(Math.floor(bx) - 1, Math.floor(by), 3, 3);
    orchCtx.fillStyle = "#0a0a00";
    orchCtx.textAlign = "left";
    orchCtx.textBaseline = "middle";
    orchCtx.fillText(c.bubble.text, Math.floor(lx + 4), Math.floor(by - ph / 2));
  }
  function _orchDrawChar(c) {
    if (!c.alive && c.state !== "entering" && c.state !== "exiting")
      return;
    var yOff = _orchBOB[c.frame], cx = Math.floor(c.x), cy = Math.floor(c.y + yOff);
    if (c.state === "toolcall") {
      orchCtx.fillStyle = "rgba(251,146,60,0.35)";
      orchCtx.beginPath();
      orchCtx.arc(cx, cy, 13, 0, Math.PI * 2);
      orchCtx.fill();
    } else if (c.state === "waiting") {
      orchCtx.fillStyle = "rgba(96,165,250,0.25)";
      orchCtx.beginPath();
      orchCtx.arc(cx, cy, 11, 0, Math.PI * 2);
      orchCtx.fill();
    } else if (c.state === "user_waiting" || c.state === "plan_review") {
      orchCtx.fillStyle = "rgba(167,139,250,0.18)";
      orchCtx.beginPath();
      orchCtx.arc(cx, cy, 10, 0, Math.PI * 2);
      orchCtx.fill();
    } else if (c.state === "plan_executing") {
      orchCtx.fillStyle = "rgba(74,222,128,0.18)";
      orchCtx.beginPath();
      orchCtx.arc(cx, cy, 10, 0, Math.PI * 2);
      orchCtx.fill();
    }
    orchCtx.font = "18px serif";
    orchCtx.textAlign = "center";
    orchCtx.textBaseline = "middle";
    if (c.facing === -1) {
      orchCtx.save();
      orchCtx.translate(cx, cy);
      orchCtx.scale(-1, 1);
      orchCtx.fillText(c.emoji, 0, 0);
      orchCtx.restore();
    } else {
      orchCtx.fillText(c.emoji, cx, cy);
    }
    orchCtx.font = "6px Silkscreen,monospace";
    orchCtx.textAlign = "center";
    orchCtx.textBaseline = "top";
    orchCtx.fillStyle = c.state === "talking" ? "#facc15" : "#3a4a7a";
    orchCtx.fillText(c.id.toUpperCase(), cx, cy + 11);
    _orchDrawStatus(c);
    _orchDrawBubble(c);
  }
  function _orchRender(ts) {
    if (_orchLastTs === null)
      _orchLastTs = ts;
    var dt = Math.min(ts - _orchLastTs, 80);
    _orchLastTs = ts;
    _orchUpdate(dt);
    orchCtx.imageSmoothingEnabled = false;
    drawMap(orchCtx);
    _orchAllChars().forEach(_orchDrawChar);
    requestAnimationFrame(_orchRender);
  }
  function orchInit() {
    if (orchInited)
      return;
    orchInited = true;
    orchCanvas = document.getElementById("orch-canvas");
    orchCtx = orchCanvas.getContext("2d");
    orchCtx.imageSmoothingEnabled = false;
    _orchInitChars();
    loadMapAsset(function() {
      _orchLastTs = null;
      requestAnimationFrame(_orchRender);
    });
  }
  function connectOrchWs() {
    orchInit();
    if (orchWs && orchWs.readyState <= 1)
      return;
    var proto = location.protocol === "https:" ? "wss:" : "ws:";
    var url = proto + "//" + location.host + "/miniapp/api/orchestration/ws?initData=" + encodeURIComponent(initData);
    orchWs = new WebSocket(url);
    orchWs.onopen = function() {
      document.getElementById("orch-status-dot").classList.add("on");
      document.getElementById("orch-status-text").textContent = "Live";
    };
    orchWs.onmessage = function(e) {
      var msg;
      try {
        msg = JSON.parse(e.data);
      } catch (_2) {
        return;
      }
      if (msg.type === "init") {
        (msg.agents || []).forEach(function(info) {
          _orchSpawn(info.id);
          if (info.state && info.state !== "idle") {
            var c2 = _orchCharForId(info.id);
            if (c2)
              _orchSetState(c2, info.state);
          }
        });
      } else if (msg.type === "event") {
        var ev = msg.event || {};
        if (ev.type === "agent_spawn")
          _orchSpawn(ev.id);
        if (ev.type === "agent_state") {
          var c = _orchCharForId(ev.id);
          if (c)
            _orchSetState(c, ev.state, ev.tool);
        }
        if (ev.type === "agent_gc")
          _orchGC(ev.id);
        if (ev.type === "conversation")
          _orchConverse(ev.from, ev.to, ev.text);
      }
    };
    orchWs.onclose = function() {
      document.getElementById("orch-status-dot").classList.remove("on");
      document.getElementById("orch-status-text").textContent = "Disconnected";
      orchWs = null;
      var at = document.querySelector(".tab.active");
      if (at && at.dataset.panel === "orch")
        orchReconnectTimer = setTimeout(connectOrchWs, 3000);
    };
    orchWs.onerror = function() {};
  }
  function disconnectOrchWs() {
    if (orchReconnectTimer) {
      clearTimeout(orchReconnectTimer);
      orchReconnectTimer = null;
    }
    if (orchWs) {
      orchWs.close();
      orchWs = null;
    }
    var dot = document.getElementById("orch-status-dot");
    var txt = document.getElementById("orch-status-text");
    if (dot)
      dot.classList.remove("on");
    if (txt)
      txt.textContent = "Offline";
  }
  async function saveLogSnapshot() {
    try {
      var res = await fetch(API_BASE + "/miniapp/api/logs/snapshot?initData=" + encodeURIComponent(initData), {
        method: "POST"
      });
      if (!res.ok)
        throw new Error("API error: " + res.status);
      var data = await res.json();
      if (data.download_url) {
        var a = document.createElement("a");
        a.href = API_BASE + data.download_url + "?initData=" + encodeURIComponent(initData);
        a.download = "";
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
      }
    } catch (e) {}
  }
  window.sendCustomCmd = sendCustomCmd;
  window.sendSkillCommand = sendSkillCommand;
  window.startPlan = startPlan;
  window.toggleSystemPrompt = toggleSystemPrompt;
  window.loadGit = loadGit;
  window.saveLogSnapshot = saveLogSnapshot;
  var researchCurrentTaskId = null;
  var STATUS_COLORS = {
    pending: { bg: "rgba(234,179,8,0.15)", text: "#ca8a04" },
    active: { bg: "rgba(59,130,246,0.15)", text: "#2563eb" },
    completed: { bg: "rgba(34,197,94,0.15)", text: "#16a34a" },
    failed: { bg: "rgba(239,68,68,0.15)", text: "#dc2626" },
    canceled: { bg: "rgba(107,114,128,0.15)", text: "#6b7280" }
  };
  async function loadResearch() {
    var el = document.getElementById("research-tasks");
    var loading = document.getElementById("research-loading");
    loading.classList.remove("hidden");
    el.innerHTML = "";
    try {
      var resp = await fetch(API_BASE + "/miniapp/api/research?initData=" + encodeURIComponent(initData));
      var tasks = await resp.json();
      loading.classList.add("hidden");
      if (!tasks || tasks.length === 0) {
        el.innerHTML = '<div class="empty-state">No research tasks yet.</div>';
        return;
      }
      el.innerHTML = tasks.map(function(t) {
        var sc = STATUS_COLORS[t.status] || STATUS_COLORS.pending;
        var focusBadge = t.focused ? '<span style="font-size:10px;font-weight:600;padding:2px 6px;border-radius:8px;background:rgba(168,85,247,0.2);color:#a855f7">focused</span>' : "";
        return '<div class="card glass glass-interactive" style="cursor:pointer;padding:14px' + (t.focused ? ";border-left:3px solid #a855f7" : "") + `" onclick="openResearchTask('` + t.id + `')">` + '<div style="display:flex;align-items:center;justify-content:space-between;gap:8px">' + '<span style="font-weight:600;font-size:15px">' + esc(t.title) + "</span>" + '<div style="display:flex;gap:4px;align-items:center">' + focusBadge + '<span style="font-size:11px;font-weight:600;padding:2px 8px;border-radius:10px;background:' + sc.bg + ";color:" + sc.text + '">' + t.status + "</span>" + "</div>" + "</div>" + (t.description ? '<div style="color:var(--hint);font-size:13px;margin-top:4px;line-height:1.4">' + esc(t.description).substring(0, 120) + "</div>" : "") + '<div style="color:var(--hint);font-size:11px;margin-top:6px">' + t.document_count + " docs · " + new Date(t.created_at).toLocaleDateString() + "</div>" + "</div>";
      }).join("");
    } catch (e) {
      loading.classList.add("hidden");
      el.innerHTML = '<div class="empty-state">Failed to load tasks.</div>';
    }
  }
  function esc(s) {
    var d = document.createElement("div");
    d.textContent = s;
    return d.innerHTML;
  }
  async function openResearchTask(id) {
    researchCurrentTaskId = id;
    document.getElementById("research-list-view").classList.add("hidden");
    document.getElementById("research-detail-view").classList.remove("hidden");
    var el = document.getElementById("research-detail-content");
    el.innerHTML = '<div class="loading">Loading...</div>';
    try {
      var resp = await fetch(API_BASE + "/miniapp/api/research/" + id + "?initData=" + encodeURIComponent(initData));
      var task = await resp.json();
      var sc = STATUS_COLORS[task.status] || STATUS_COLORS.pending;
      var canCancel = task.status === "pending" || task.status === "active";
      var canReopen = task.status === "completed" || task.status === "failed";
      var html = '<div class="card glass">' + '<div style="display:flex;align-items:center;justify-content:space-between;gap:8px;margin-bottom:8px">' + '<span style="font-weight:700;font-size:17px">' + esc(task.title) + "</span>" + '<span style="font-size:11px;font-weight:600;padding:2px 8px;border-radius:10px;background:' + sc.bg + ";color:" + sc.text + '">' + task.status + "</span>" + "</div>";
      if (task.description) {
        html += '<div style="color:var(--hint);font-size:13px;line-height:1.5;margin-bottom:8px;white-space:pre-wrap">' + esc(task.description) + "</div>";
      }
      html += '<div style="color:var(--hint);font-size:11px">' + "Created: " + new Date(task.created_at).toLocaleString() + (task.completed_at ? " · Completed: " + new Date(task.completed_at).toLocaleString() : "") + "</div>";
      html += '<div style="margin-top:10px;display:flex;gap:8px">';
      if (task.focused) {
        html += `<button class="worktree-btn dispose" onclick="researchSetFocus('` + id + `',false)" style="background:rgba(168,85,247,0.15);color:#a855f7;border-color:#a855f7">Forget</button>`;
      } else {
        html += `<button class="worktree-btn merge" onclick="researchSetFocus('` + id + `',true)" style="background:rgba(168,85,247,0.15);color:#a855f7;border-color:#a855f7">Recall</button>`;
      }
      if (canCancel)
        html += `<button class="worktree-btn dispose" onclick="researchAction('` + id + `','cancel')">Cancel</button>`;
      if (canReopen)
        html += `<button class="worktree-btn merge" onclick="researchAction('` + id + `','reopen')">Reopen</button>`;
      html += "</div>";
      html += "</div>";
      html += '<div class="card-title" style="margin-top:12px">Documents (' + task.documents.length + ")</div>";
      if (task.documents.length === 0) {
        html += '<div class="empty-state" style="padding:24px">No documents yet.</div>';
      } else {
        html += task.documents.map(function(d) {
          return `<div class="card glass" style="padding:12px;cursor:pointer" onclick="toggleResearchDoc(this, '` + id + "', '" + d.id + `')">` + '<div style="display:flex;align-items:center;gap:8px">' + '<span style="color:var(--hint);font-family:monospace;font-size:12px">#' + d.seq + "</span>" + '<span style="font-weight:600;font-size:14px;flex:1">' + esc(d.title) + "</span>" + '<span style="font-size:10px;padding:2px 6px;border-radius:8px;background:var(--tab-track-bg);color:var(--hint)">' + d.doc_type + "</span>" + "</div>" + (d.summary ? '<div style="color:var(--hint);font-size:12px;margin-top:4px">' + esc(d.summary) + "</div>" : "") + '<div class="research-doc-body hidden" style="margin-top:8px;border-top:1px solid var(--glass-divider);padding-top:8px">' + '<div class="loading" style="padding:12px">Loading...</div>' + "</div>" + "</div>";
        }).join("");
      }
      el.innerHTML = html;
    } catch (e) {
      el.innerHTML = '<div class="empty-state">Failed to load task.</div>';
    }
  }
  async function toggleResearchDoc(card, taskId, docId) {
    var body = card.querySelector(".research-doc-body");
    if (!body)
      return;
    if (!body.classList.contains("hidden")) {
      body.classList.add("hidden");
      return;
    }
    body.classList.remove("hidden");
    if (body.dataset.loaded)
      return;
    try {
      var resp = await fetch(API_BASE + "/miniapp/api/research/" + taskId + "/doc/" + docId + "?initData=" + encodeURIComponent(initData));
      var data = await resp.json();
      body.dataset.loaded = "1";
      body.innerHTML = '<div class="md-rendered" style="max-height:50vh;overflow:auto;padding:8px 0">' + g.parse(data.content) + "</div>";
    } catch (e) {
      body.innerHTML = '<div style="color:var(--hint);font-size:12px">Failed to load document.</div>';
    }
  }
  async function researchAction(taskId, action) {
    try {
      await fetch(API_BASE + "/miniapp/api/research/" + taskId + "?initData=" + encodeURIComponent(initData), { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ action }) });
      openResearchTask(taskId);
    } catch (e) {}
  }
  function showResearchList() {
    document.getElementById("research-detail-view").classList.add("hidden");
    document.getElementById("research-list-view").classList.remove("hidden");
    researchCurrentTaskId = null;
    loadResearch();
  }
  function showNewTaskForm() {
    document.getElementById("research-new-form").classList.remove("hidden");
    document.getElementById("research-title").focus();
  }
  function hideNewTaskForm() {
    document.getElementById("research-new-form").classList.add("hidden");
    document.getElementById("research-title").value = "";
    document.getElementById("research-desc").value = "";
  }
  async function createResearchTask() {
    var title = document.getElementById("research-title").value.trim();
    var desc = document.getElementById("research-desc").value.trim();
    if (!title)
      return;
    try {
      await fetch(API_BASE + "/miniapp/api/research?initData=" + encodeURIComponent(initData), { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ title, description: desc }) });
      hideNewTaskForm();
      loadResearch();
    } catch (e) {}
  }
  async function researchSetFocus(taskId, recall) {
    try {
      await fetch(API_BASE + "/miniapp/api/research/focus?initData=" + encodeURIComponent(initData), {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ action: recall ? "recall" : "forget", task_id: taskId })
      });
      openResearchTask(taskId);
    } catch (e) {}
  }
  window.showNewTaskForm = showNewTaskForm;
  window.hideNewTaskForm = hideNewTaskForm;
  window.createResearchTask = createResearchTask;
  window.openResearchTask = openResearchTask;
  window.toggleResearchDoc = toggleResearchDoc;
  window.researchAction = researchAction;
  window.researchSetFocus = researchSetFocus;
  window.showResearchList = showResearchList;
})();
