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
          ...pieces.map((x3, i3) => `${x3}${"_".repeat(i3 + 1)}`)
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
      const joined = args.map((x3) => source(x3)).join("");
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
      const joined = "(" + (opts.capture ? "" : "?:") + args.map((x3) => source(x3)).join("|") + ")";
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
        "on:begin": (m4, resp) => {
          if (m4.index !== 0)
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
        "on:begin": (m4, resp) => {
          resp.data._beginMatch = m4[1];
        },
        "on:end": (m4, resp) => {
          if (resp.data._beginMatch !== m4[1])
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
          keywordList = keywordList.map((x3) => x3.toLowerCase());
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
      for (let i3 = 1;i3 <= regexes.length; i3++) {
        positions[i3 + offset] = scopeNames[i3];
        emit[i3 + offset] = true;
        offset += countMatchGroups(regexes[i3 - 1]);
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
        exec(s3) {
          this.matcherRe.lastIndex = this.lastIndex;
          const match = this.matcherRe.exec(s3);
          if (!match) {
            return null;
          }
          const i3 = match.findIndex((el, i4) => i4 > 0 && el !== undefined);
          const matchData = this.matchIndexes[i3];
          match.splice(0, i3);
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
        exec(s3) {
          const m4 = this.getMatcher(this.regexIndex);
          m4.lastIndex = this.lastIndex;
          let result = m4.exec(s3);
          if (this.resumingScanAtSamePosition()) {
            if (result && result.index === this.lastIndex)
              ;
            else {
              const m22 = this.getMatcher(0);
              m22.lastIndex = this.lastIndex + 1;
              result = m22.exec(s3);
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
        mode.contains = [].concat(...mode.contains.map(function(c3) {
          return expandOrCloneMode(c3 === "self" ? mode : c3);
        }));
        mode.contains.forEach(function(c3) {
          compileMode(c3, cmode);
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
          let i3 = 1;
          const max = match.length - 1;
          while (i3 <= max) {
            if (!scope._emit[i3]) {
              i3++;
              continue;
            }
            const klass = language.classNameAliases[scope[i3]] || scope[i3];
            const text = match[i3];
            if (klass) {
              emitKeyword(text, klass);
            } else {
              modeBuffer = text;
              processKeywords();
              modeBuffer = "";
            }
            i3++;
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
        const sorted = results.sort((a3, b2) => {
          if (a3.relevance !== b2.relevance)
            return b2.relevance - a3.relevance;
          if (a3.language && b2.language) {
            if (getLanguage(a3.language).supersetOf === b2.language) {
              return 1;
            } else if (getLanguage(b2.language).supersetOf === a3.language) {
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

  // node_modules/preact/dist/preact.module.js
  var n;
  var l;
  var u;
  var t;
  var i;
  var r;
  var o;
  var e;
  var f;
  var c;
  var s;
  var a;
  var h;
  var p = {};
  var v = [];
  var y = /acit|ex(?:s|g|n|p|$)|rph|grid|ows|mnc|ntw|ine[ch]|zoo|^ord|itera/i;
  var d = Array.isArray;
  function w(n2, l2) {
    for (var u2 in l2)
      n2[u2] = l2[u2];
    return n2;
  }
  function g(n2) {
    n2 && n2.parentNode && n2.parentNode.removeChild(n2);
  }
  function _(l2, u2, t2) {
    var i2, r2, o2, e2 = {};
    for (o2 in u2)
      o2 == "key" ? i2 = u2[o2] : o2 == "ref" ? r2 = u2[o2] : e2[o2] = u2[o2];
    if (arguments.length > 2 && (e2.children = arguments.length > 3 ? n.call(arguments, 2) : t2), typeof l2 == "function" && l2.defaultProps != null)
      for (o2 in l2.defaultProps)
        e2[o2] === undefined && (e2[o2] = l2.defaultProps[o2]);
    return m(l2, e2, i2, r2, null);
  }
  function m(n2, t2, i2, r2, o2) {
    var e2 = { type: n2, props: t2, key: i2, ref: r2, __k: null, __: null, __b: 0, __e: null, __c: null, constructor: undefined, __v: o2 == null ? ++u : o2, __i: -1, __u: 0 };
    return o2 == null && l.vnode != null && l.vnode(e2), e2;
  }
  function k(n2) {
    return n2.children;
  }
  function x(n2, l2) {
    this.props = n2, this.context = l2;
  }
  function S(n2, l2) {
    if (l2 == null)
      return n2.__ ? S(n2.__, n2.__i + 1) : null;
    for (var u2;l2 < n2.__k.length; l2++)
      if ((u2 = n2.__k[l2]) != null && u2.__e != null)
        return u2.__e;
    return typeof n2.type == "function" ? S(n2) : null;
  }
  function C(n2) {
    if (n2.__P && n2.__d) {
      var u2 = n2.__v, t2 = u2.__e, i2 = [], r2 = [], o2 = w({}, u2);
      o2.__v = u2.__v + 1, l.vnode && l.vnode(o2), z(n2.__P, o2, u2, n2.__n, n2.__P.namespaceURI, 32 & u2.__u ? [t2] : null, i2, t2 == null ? S(u2) : t2, !!(32 & u2.__u), r2), o2.__v = u2.__v, o2.__.__k[o2.__i] = o2, V(i2, o2, r2), u2.__e = u2.__ = null, o2.__e != t2 && M(o2);
    }
  }
  function M(n2) {
    if ((n2 = n2.__) != null && n2.__c != null)
      return n2.__e = n2.__c.base = null, n2.__k.some(function(l2) {
        if (l2 != null && l2.__e != null)
          return n2.__e = n2.__c.base = l2.__e;
      }), M(n2);
  }
  function $(n2) {
    (!n2.__d && (n2.__d = true) && i.push(n2) && !I.__r++ || r != l.debounceRendering) && ((r = l.debounceRendering) || o)(I);
  }
  function I() {
    try {
      for (var n2, l2 = 1;i.length; )
        i.length > l2 && i.sort(e), n2 = i.shift(), l2 = i.length, C(n2);
    } finally {
      i.length = I.__r = 0;
    }
  }
  function P(n2, l2, u2, t2, i2, r2, o2, e2, f2, c2, s2) {
    var a2, h2, y2, d2, w2, g2, _2, m2 = t2 && t2.__k || v, b = l2.length;
    for (f2 = A(u2, l2, m2, f2, b), a2 = 0;a2 < b; a2++)
      (y2 = u2.__k[a2]) != null && (h2 = y2.__i != -1 && m2[y2.__i] || p, y2.__i = a2, g2 = z(n2, y2, h2, i2, r2, o2, e2, f2, c2, s2), d2 = y2.__e, y2.ref && h2.ref != y2.ref && (h2.ref && D(h2.ref, null, y2), s2.push(y2.ref, y2.__c || d2, y2)), w2 == null && d2 != null && (w2 = d2), (_2 = !!(4 & y2.__u)) || h2.__k === y2.__k ? f2 = H(y2, f2, n2, _2) : typeof y2.type == "function" && g2 !== undefined ? f2 = g2 : d2 && (f2 = d2.nextSibling), y2.__u &= -7);
    return u2.__e = w2, f2;
  }
  function A(n2, l2, u2, t2, i2) {
    var r2, o2, e2, f2, c2, s2 = u2.length, a2 = s2, h2 = 0;
    for (n2.__k = new Array(i2), r2 = 0;r2 < i2; r2++)
      (o2 = l2[r2]) != null && typeof o2 != "boolean" && typeof o2 != "function" ? (typeof o2 == "string" || typeof o2 == "number" || typeof o2 == "bigint" || o2.constructor == String ? o2 = n2.__k[r2] = m(null, o2, null, null, null) : d(o2) ? o2 = n2.__k[r2] = m(k, { children: o2 }, null, null, null) : o2.constructor === undefined && o2.__b > 0 ? o2 = n2.__k[r2] = m(o2.type, o2.props, o2.key, o2.ref ? o2.ref : null, o2.__v) : n2.__k[r2] = o2, f2 = r2 + h2, o2.__ = n2, o2.__b = n2.__b + 1, e2 = null, (c2 = o2.__i = T(o2, u2, f2, a2)) != -1 && (a2--, (e2 = u2[c2]) && (e2.__u |= 2)), e2 == null || e2.__v == null ? (c2 == -1 && (i2 > s2 ? h2-- : i2 < s2 && h2++), typeof o2.type != "function" && (o2.__u |= 4)) : c2 != f2 && (c2 == f2 - 1 ? h2-- : c2 == f2 + 1 ? h2++ : (c2 > f2 ? h2-- : h2++, o2.__u |= 4))) : n2.__k[r2] = null;
    if (a2)
      for (r2 = 0;r2 < s2; r2++)
        (e2 = u2[r2]) != null && (2 & e2.__u) == 0 && (e2.__e == t2 && (t2 = S(e2)), E(e2, e2));
    return t2;
  }
  function H(n2, l2, u2, t2) {
    var i2, r2;
    if (typeof n2.type == "function") {
      for (i2 = n2.__k, r2 = 0;i2 && r2 < i2.length; r2++)
        i2[r2] && (i2[r2].__ = n2, l2 = H(i2[r2], l2, u2, t2));
      return l2;
    }
    n2.__e != l2 && (t2 && (l2 && n2.type && !l2.parentNode && (l2 = S(n2)), u2.insertBefore(n2.__e, l2 || null)), l2 = n2.__e);
    do {
      l2 = l2 && l2.nextSibling;
    } while (l2 != null && l2.nodeType == 8);
    return l2;
  }
  function T(n2, l2, u2, t2) {
    var i2, r2, o2, e2 = n2.key, f2 = n2.type, c2 = l2[u2], s2 = c2 != null && (2 & c2.__u) == 0;
    if (c2 === null && e2 == null || s2 && e2 == c2.key && f2 == c2.type)
      return u2;
    if (t2 > (s2 ? 1 : 0)) {
      for (i2 = u2 - 1, r2 = u2 + 1;i2 >= 0 || r2 < l2.length; )
        if ((c2 = l2[o2 = i2 >= 0 ? i2-- : r2++]) != null && (2 & c2.__u) == 0 && e2 == c2.key && f2 == c2.type)
          return o2;
    }
    return -1;
  }
  function j(n2, l2, u2) {
    l2[0] == "-" ? n2.setProperty(l2, u2 == null ? "" : u2) : n2[l2] = u2 == null ? "" : typeof u2 != "number" || y.test(l2) ? u2 : u2 + "px";
  }
  function F(n2, l2, u2, t2, i2) {
    var r2, o2;
    n:
      if (l2 == "style")
        if (typeof u2 == "string")
          n2.style.cssText = u2;
        else {
          if (typeof t2 == "string" && (n2.style.cssText = t2 = ""), t2)
            for (l2 in t2)
              u2 && l2 in u2 || j(n2.style, l2, "");
          if (u2)
            for (l2 in u2)
              t2 && u2[l2] == t2[l2] || j(n2.style, l2, u2[l2]);
        }
      else if (l2[0] == "o" && l2[1] == "n")
        r2 = l2 != (l2 = l2.replace(f, "$1")), o2 = l2.toLowerCase(), l2 = o2 in n2 || l2 == "onFocusOut" || l2 == "onFocusIn" ? o2.slice(2) : l2.slice(2), n2.l || (n2.l = {}), n2.l[l2 + r2] = u2, u2 ? t2 ? u2.u = t2.u : (u2.u = c, n2.addEventListener(l2, r2 ? a : s, r2)) : n2.removeEventListener(l2, r2 ? a : s, r2);
      else {
        if (i2 == "http://www.w3.org/2000/svg")
          l2 = l2.replace(/xlink(H|:h)/, "h").replace(/sName$/, "s");
        else if (l2 != "width" && l2 != "height" && l2 != "href" && l2 != "list" && l2 != "form" && l2 != "tabIndex" && l2 != "download" && l2 != "rowSpan" && l2 != "colSpan" && l2 != "role" && l2 != "popover" && l2 in n2)
          try {
            n2[l2] = u2 == null ? "" : u2;
            break n;
          } catch (n3) {}
        typeof u2 == "function" || (u2 == null || u2 === false && l2[4] != "-" ? n2.removeAttribute(l2) : n2.setAttribute(l2, l2 == "popover" && u2 == 1 ? "" : u2));
      }
  }
  function O(n2) {
    return function(u2) {
      if (this.l) {
        var t2 = this.l[u2.type + n2];
        if (u2.t == null)
          u2.t = c++;
        else if (u2.t < t2.u)
          return;
        return t2(l.event ? l.event(u2) : u2);
      }
    };
  }
  function z(n2, u2, t2, i2, r2, o2, e2, f2, c2, s2) {
    var a2, h2, p2, y2, _2, m2, b, S2, C2, M2, $2, I2, A2, H2, L, T2 = u2.type;
    if (u2.constructor !== undefined)
      return null;
    128 & t2.__u && (c2 = !!(32 & t2.__u), o2 = [f2 = u2.__e = t2.__e]), (a2 = l.__b) && a2(u2);
    n:
      if (typeof T2 == "function")
        try {
          if (S2 = u2.props, C2 = T2.prototype && T2.prototype.render, M2 = (a2 = T2.contextType) && i2[a2.__c], $2 = a2 ? M2 ? M2.props.value : a2.__ : i2, t2.__c ? b = (h2 = u2.__c = t2.__c).__ = h2.__E : (C2 ? u2.__c = h2 = new T2(S2, $2) : (u2.__c = h2 = new x(S2, $2), h2.constructor = T2, h2.render = G), M2 && M2.sub(h2), h2.state || (h2.state = {}), h2.__n = i2, p2 = h2.__d = true, h2.__h = [], h2._sb = []), C2 && h2.__s == null && (h2.__s = h2.state), C2 && T2.getDerivedStateFromProps != null && (h2.__s == h2.state && (h2.__s = w({}, h2.__s)), w(h2.__s, T2.getDerivedStateFromProps(S2, h2.__s))), y2 = h2.props, _2 = h2.state, h2.__v = u2, p2)
            C2 && T2.getDerivedStateFromProps == null && h2.componentWillMount != null && h2.componentWillMount(), C2 && h2.componentDidMount != null && h2.__h.push(h2.componentDidMount);
          else {
            if (C2 && T2.getDerivedStateFromProps == null && S2 !== y2 && h2.componentWillReceiveProps != null && h2.componentWillReceiveProps(S2, $2), u2.__v == t2.__v || !h2.__e && h2.shouldComponentUpdate != null && h2.shouldComponentUpdate(S2, h2.__s, $2) === false) {
              u2.__v != t2.__v && (h2.props = S2, h2.state = h2.__s, h2.__d = false), u2.__e = t2.__e, u2.__k = t2.__k, u2.__k.some(function(n3) {
                n3 && (n3.__ = u2);
              }), v.push.apply(h2.__h, h2._sb), h2._sb = [], h2.__h.length && e2.push(h2);
              break n;
            }
            h2.componentWillUpdate != null && h2.componentWillUpdate(S2, h2.__s, $2), C2 && h2.componentDidUpdate != null && h2.__h.push(function() {
              h2.componentDidUpdate(y2, _2, m2);
            });
          }
          if (h2.context = $2, h2.props = S2, h2.__P = n2, h2.__e = false, I2 = l.__r, A2 = 0, C2)
            h2.state = h2.__s, h2.__d = false, I2 && I2(u2), a2 = h2.render(h2.props, h2.state, h2.context), v.push.apply(h2.__h, h2._sb), h2._sb = [];
          else
            do {
              h2.__d = false, I2 && I2(u2), a2 = h2.render(h2.props, h2.state, h2.context), h2.state = h2.__s;
            } while (h2.__d && ++A2 < 25);
          h2.state = h2.__s, h2.getChildContext != null && (i2 = w(w({}, i2), h2.getChildContext())), C2 && !p2 && h2.getSnapshotBeforeUpdate != null && (m2 = h2.getSnapshotBeforeUpdate(y2, _2)), H2 = a2 != null && a2.type === k && a2.key == null ? q(a2.props.children) : a2, f2 = P(n2, d(H2) ? H2 : [H2], u2, t2, i2, r2, o2, e2, f2, c2, s2), h2.base = u2.__e, u2.__u &= -161, h2.__h.length && e2.push(h2), b && (h2.__E = h2.__ = null);
        } catch (n3) {
          if (u2.__v = null, c2 || o2 != null)
            if (n3.then) {
              for (u2.__u |= c2 ? 160 : 128;f2 && f2.nodeType == 8 && f2.nextSibling; )
                f2 = f2.nextSibling;
              o2[o2.indexOf(f2)] = null, u2.__e = f2;
            } else {
              for (L = o2.length;L--; )
                g(o2[L]);
              N(u2);
            }
          else
            u2.__e = t2.__e, u2.__k = t2.__k, n3.then || N(u2);
          l.__e(n3, u2, t2);
        }
      else
        o2 == null && u2.__v == t2.__v ? (u2.__k = t2.__k, u2.__e = t2.__e) : f2 = u2.__e = B(t2.__e, u2, t2, i2, r2, o2, e2, c2, s2);
    return (a2 = l.diffed) && a2(u2), 128 & u2.__u ? undefined : f2;
  }
  function N(n2) {
    n2 && (n2.__c && (n2.__c.__e = true), n2.__k && n2.__k.some(N));
  }
  function V(n2, u2, t2) {
    for (var i2 = 0;i2 < t2.length; i2++)
      D(t2[i2], t2[++i2], t2[++i2]);
    l.__c && l.__c(u2, n2), n2.some(function(u3) {
      try {
        n2 = u3.__h, u3.__h = [], n2.some(function(n3) {
          n3.call(u3);
        });
      } catch (n3) {
        l.__e(n3, u3.__v);
      }
    });
  }
  function q(n2) {
    return typeof n2 != "object" || n2 == null || n2.__b > 0 ? n2 : d(n2) ? n2.map(q) : w({}, n2);
  }
  function B(u2, t2, i2, r2, o2, e2, f2, c2, s2) {
    var a2, h2, v2, y2, w2, _2, m2, b = i2.props || p, k2 = t2.props, x2 = t2.type;
    if (x2 == "svg" ? o2 = "http://www.w3.org/2000/svg" : x2 == "math" ? o2 = "http://www.w3.org/1998/Math/MathML" : o2 || (o2 = "http://www.w3.org/1999/xhtml"), e2 != null) {
      for (a2 = 0;a2 < e2.length; a2++)
        if ((w2 = e2[a2]) && "setAttribute" in w2 == !!x2 && (x2 ? w2.localName == x2 : w2.nodeType == 3)) {
          u2 = w2, e2[a2] = null;
          break;
        }
    }
    if (u2 == null) {
      if (x2 == null)
        return document.createTextNode(k2);
      u2 = document.createElementNS(o2, x2, k2.is && k2), c2 && (l.__m && l.__m(t2, e2), c2 = false), e2 = null;
    }
    if (x2 == null)
      b === k2 || c2 && u2.data == k2 || (u2.data = k2);
    else {
      if (e2 = e2 && n.call(u2.childNodes), !c2 && e2 != null)
        for (b = {}, a2 = 0;a2 < u2.attributes.length; a2++)
          b[(w2 = u2.attributes[a2]).name] = w2.value;
      for (a2 in b)
        w2 = b[a2], a2 == "dangerouslySetInnerHTML" ? v2 = w2 : a2 == "children" || (a2 in k2) || a2 == "value" && ("defaultValue" in k2) || a2 == "checked" && ("defaultChecked" in k2) || F(u2, a2, null, w2, o2);
      for (a2 in k2)
        w2 = k2[a2], a2 == "children" ? y2 = w2 : a2 == "dangerouslySetInnerHTML" ? h2 = w2 : a2 == "value" ? _2 = w2 : a2 == "checked" ? m2 = w2 : c2 && typeof w2 != "function" || b[a2] === w2 || F(u2, a2, w2, b[a2], o2);
      if (h2)
        c2 || v2 && (h2.__html == v2.__html || h2.__html == u2.innerHTML) || (u2.innerHTML = h2.__html), t2.__k = [];
      else if (v2 && (u2.innerHTML = ""), P(t2.type == "template" ? u2.content : u2, d(y2) ? y2 : [y2], t2, i2, r2, x2 == "foreignObject" ? "http://www.w3.org/1999/xhtml" : o2, e2, f2, e2 ? e2[0] : i2.__k && S(i2, 0), c2, s2), e2 != null)
        for (a2 = e2.length;a2--; )
          g(e2[a2]);
      c2 || (a2 = "value", x2 == "progress" && _2 == null ? u2.removeAttribute("value") : _2 != null && (_2 !== u2[a2] || x2 == "progress" && !_2 || x2 == "option" && _2 != b[a2]) && F(u2, a2, _2, b[a2], o2), a2 = "checked", m2 != null && m2 != u2[a2] && F(u2, a2, m2, b[a2], o2));
    }
    return u2;
  }
  function D(n2, u2, t2) {
    try {
      if (typeof n2 == "function") {
        var i2 = typeof n2.__u == "function";
        i2 && n2.__u(), i2 && u2 == null || (n2.__u = n2(u2));
      } else
        n2.current = u2;
    } catch (n3) {
      l.__e(n3, t2);
    }
  }
  function E(n2, u2, t2) {
    var i2, r2;
    if (l.unmount && l.unmount(n2), (i2 = n2.ref) && (i2.current && i2.current != n2.__e || D(i2, null, u2)), (i2 = n2.__c) != null) {
      if (i2.componentWillUnmount)
        try {
          i2.componentWillUnmount();
        } catch (n3) {
          l.__e(n3, u2);
        }
      i2.base = i2.__P = null;
    }
    if (i2 = n2.__k)
      for (r2 = 0;r2 < i2.length; r2++)
        i2[r2] && E(i2[r2], u2, t2 || typeof n2.type != "function");
    t2 || g(n2.__e), n2.__c = n2.__ = n2.__e = undefined;
  }
  function G(n2, l2, u2) {
    return this.constructor(n2, u2);
  }
  function J(u2, t2, i2) {
    var r2, o2, e2, f2;
    t2 == document && (t2 = document.documentElement), l.__ && l.__(u2, t2), o2 = (r2 = typeof i2 == "function") ? null : i2 && i2.__k || t2.__k, e2 = [], f2 = [], z(t2, u2 = (!r2 && i2 || t2).__k = _(k, null, [u2]), o2 || p, p, t2.namespaceURI, !r2 && i2 ? [i2] : o2 ? null : t2.firstChild ? n.call(t2.childNodes) : null, e2, !r2 && i2 ? i2 : o2 ? o2.__e : t2.firstChild, r2, f2), V(e2, u2, f2);
  }
  n = v.slice, l = { __e: function(n2, l2, u2, t2) {
    for (var i2, r2, o2;l2 = l2.__; )
      if ((i2 = l2.__c) && !i2.__)
        try {
          if ((r2 = i2.constructor) && r2.getDerivedStateFromError != null && (i2.setState(r2.getDerivedStateFromError(n2)), o2 = i2.__d), i2.componentDidCatch != null && (i2.componentDidCatch(n2, t2 || {}), o2 = i2.__d), o2)
            return i2.__E = i2;
        } catch (l3) {
          n2 = l3;
        }
    throw n2;
  } }, u = 0, t = function(n2) {
    return n2 != null && n2.constructor === undefined;
  }, x.prototype.setState = function(n2, l2) {
    var u2;
    u2 = this.__s != null && this.__s != this.state ? this.__s : this.__s = w({}, this.state), typeof n2 == "function" && (n2 = n2(w({}, u2), this.props)), n2 && w(u2, n2), n2 != null && this.__v && (l2 && this._sb.push(l2), $(this));
  }, x.prototype.forceUpdate = function(n2) {
    this.__v && (this.__e = true, n2 && this.__h.push(n2), $(this));
  }, x.prototype.render = k, i = [], o = typeof Promise == "function" ? Promise.prototype.then.bind(Promise.resolve()) : setTimeout, e = function(n2, l2) {
    return n2.__v.__b - l2.__v.__b;
  }, I.__r = 0, f = /(PointerCapture)$|Capture$/i, c = 0, s = O(false), a = O(true), h = 0;

  // node_modules/preact/hooks/dist/hooks.module.js
  var t2;
  var r2;
  var u2;
  var i2;
  var o2 = 0;
  var f2 = [];
  var c2 = l;
  var e2 = c2.__b;
  var a2 = c2.__r;
  var v2 = c2.diffed;
  var l2 = c2.__c;
  var m2 = c2.unmount;
  var s2 = c2.__;
  function p2(n2, t3) {
    c2.__h && c2.__h(r2, n2, o2 || t3), o2 = 0;
    var u3 = r2.__H || (r2.__H = { __: [], __h: [] });
    return n2 >= u3.__.length && u3.__.push({}), u3.__[n2];
  }
  function d2(n2) {
    return o2 = 1, h2(D2, n2);
  }
  function h2(n2, u3, i3) {
    var o3 = p2(t2++, 2);
    if (o3.t = n2, !o3.__c && (o3.__ = [i3 ? i3(u3) : D2(undefined, u3), function(n3) {
      var t3 = o3.__N ? o3.__N[0] : o3.__[0], r3 = o3.t(t3, n3);
      t3 !== r3 && (o3.__N = [r3, o3.__[1]], o3.__c.setState({}));
    }], o3.__c = r2, !r2.__f)) {
      var f3 = function(n3, t3, r3) {
        if (!o3.__c.__H)
          return true;
        var u4 = o3.__c.__H.__.filter(function(n4) {
          return n4.__c;
        });
        if (u4.every(function(n4) {
          return !n4.__N;
        }))
          return !c3 || c3.call(this, n3, t3, r3);
        var i4 = o3.__c.props !== n3;
        return u4.some(function(n4) {
          if (n4.__N) {
            var t4 = n4.__[0];
            n4.__ = n4.__N, n4.__N = undefined, t4 !== n4.__[0] && (i4 = true);
          }
        }), c3 && c3.call(this, n3, t3, r3) || i4;
      };
      r2.__f = true;
      var { shouldComponentUpdate: c3, componentWillUpdate: e3 } = r2;
      r2.componentWillUpdate = function(n3, t3, r3) {
        if (this.__e) {
          var u4 = c3;
          c3 = undefined, f3(n3, t3, r3), c3 = u4;
        }
        e3 && e3.call(this, n3, t3, r3);
      }, r2.shouldComponentUpdate = f3;
    }
    return o3.__N || o3.__;
  }
  function y2(n2, u3) {
    var i3 = p2(t2++, 3);
    !c2.__s && C2(i3.__H, u3) && (i3.__ = n2, i3.u = u3, r2.__H.__h.push(i3));
  }
  function A2(n2) {
    return o2 = 5, T2(function() {
      return { current: n2 };
    }, []);
  }
  function T2(n2, r3) {
    var u3 = p2(t2++, 7);
    return C2(u3.__H, r3) && (u3.__ = n2(), u3.__H = r3, u3.__h = n2), u3.__;
  }
  function q2(n2, t3) {
    return o2 = 8, T2(function() {
      return n2;
    }, t3);
  }
  function j2() {
    for (var n2;n2 = f2.shift(); ) {
      var t3 = n2.__H;
      if (n2.__P && t3)
        try {
          t3.__h.some(z2), t3.__h.some(B2), t3.__h = [];
        } catch (r3) {
          t3.__h = [], c2.__e(r3, n2.__v);
        }
    }
  }
  c2.__b = function(n2) {
    r2 = null, e2 && e2(n2);
  }, c2.__ = function(n2, t3) {
    n2 && t3.__k && t3.__k.__m && (n2.__m = t3.__k.__m), s2 && s2(n2, t3);
  }, c2.__r = function(n2) {
    a2 && a2(n2), t2 = 0;
    var i3 = (r2 = n2.__c).__H;
    i3 && (u2 === r2 ? (i3.__h = [], r2.__h = [], i3.__.some(function(n3) {
      n3.__N && (n3.__ = n3.__N), n3.u = n3.__N = undefined;
    })) : (i3.__h.some(z2), i3.__h.some(B2), i3.__h = [], t2 = 0)), u2 = r2;
  }, c2.diffed = function(n2) {
    v2 && v2(n2);
    var t3 = n2.__c;
    t3 && t3.__H && (t3.__H.__h.length && (f2.push(t3) !== 1 && i2 === c2.requestAnimationFrame || ((i2 = c2.requestAnimationFrame) || w2)(j2)), t3.__H.__.some(function(n3) {
      n3.u && (n3.__H = n3.u), n3.u = undefined;
    })), u2 = r2 = null;
  }, c2.__c = function(n2, t3) {
    t3.some(function(n3) {
      try {
        n3.__h.some(z2), n3.__h = n3.__h.filter(function(n4) {
          return !n4.__ || B2(n4);
        });
      } catch (r3) {
        t3.some(function(n4) {
          n4.__h && (n4.__h = []);
        }), t3 = [], c2.__e(r3, n3.__v);
      }
    }), l2 && l2(n2, t3);
  }, c2.unmount = function(n2) {
    m2 && m2(n2);
    var t3, r3 = n2.__c;
    r3 && r3.__H && (r3.__H.__.some(function(n3) {
      try {
        z2(n3);
      } catch (n4) {
        t3 = n4;
      }
    }), r3.__H = undefined, t3 && c2.__e(t3, r3.__v));
  };
  var k2 = typeof requestAnimationFrame == "function";
  function w2(n2) {
    var t3, r3 = function() {
      clearTimeout(u3), k2 && cancelAnimationFrame(t3), setTimeout(n2);
    }, u3 = setTimeout(r3, 35);
    k2 && (t3 = requestAnimationFrame(r3));
  }
  function z2(n2) {
    var t3 = r2, u3 = n2.__c;
    typeof u3 == "function" && (n2.__c = undefined, u3()), r2 = t3;
  }
  function B2(n2) {
    var t3 = r2;
    n2.__c = n2.__(), r2 = t3;
  }
  function C2(n2, t3) {
    return !n2 || n2.length !== t3.length || t3.some(function(t4, r3) {
      return t4 !== n2[r3];
    });
  }
  function D2(n2, t3) {
    return typeof t3 == "function" ? t3(n2) : t3;
  }

  // src/hooks/use-sse.ts
  function useSSE() {
    const [data, setData] = d2({
      plan: null,
      session: null,
      skills: null,
      dev: null,
      context: null,
      prompt: null
    });
    const lastUpdate = A2({});
    y2(() => {
      const initData = window.Telegram?.WebApp?.initData || "";
      const url = location.origin + "/miniapp/api/events?initData=" + encodeURIComponent(initData);
      const es = new EventSource(url);
      es.addEventListener("plan", (e3) => {
        try {
          const d3 = JSON.parse(e3.data);
          lastUpdate.current.plan = Date.now();
          setData((prev) => ({ ...prev, plan: d3 }));
        } catch {}
      });
      es.addEventListener("session", (e3) => {
        try {
          const d3 = JSON.parse(e3.data);
          lastUpdate.current.session = Date.now();
          setData((prev) => ({ ...prev, session: d3 }));
        } catch {}
      });
      es.addEventListener("skills", (e3) => {
        try {
          const d3 = JSON.parse(e3.data);
          lastUpdate.current.skills = Date.now();
          setData((prev) => ({ ...prev, skills: d3 }));
        } catch {}
      });
      es.addEventListener("dev", (e3) => {
        try {
          const d3 = JSON.parse(e3.data);
          lastUpdate.current.dev = Date.now();
          setData((prev) => ({ ...prev, dev: d3 }));
        } catch {}
      });
      es.addEventListener("context", (e3) => {
        try {
          const d3 = JSON.parse(e3.data);
          setData((prev) => ({ ...prev, context: d3 }));
        } catch {}
      });
      es.addEventListener("prompt", (e3) => {
        try {
          const d3 = JSON.parse(e3.data);
          setData((prev) => ({ ...prev, prompt: d3.prompt || null }));
        } catch {}
      });
      return () => es.close();
    }, []);
    return { ...data, lastUpdate: lastUpdate.current };
  }

  // src/hooks/use-api.ts
  var API_BASE = location.origin;
  function getInitData() {
    return window.Telegram?.WebApp?.initData || "";
  }
  async function apiFetch(path) {
    const sep = path.includes("?") ? "&" : "?";
    const res = await fetch(API_BASE + path + sep + "initData=" + encodeURIComponent(getInitData()));
    if (!res.ok)
      throw new Error("API error: " + res.status);
    return res.json();
  }
  async function apiPost(path, body) {
    const sep = path.includes("?") ? "&" : "?";
    const res = await fetch(API_BASE + path + sep + "initData=" + encodeURIComponent(getInitData()), {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body)
    });
    if (!res.ok) {
      let errMsg = "API error: " + res.status;
      try {
        const data = await res.json();
        if (data.error)
          errMsg = data.error;
      } catch {}
      throw new Error(errMsg);
    }
    return res.json();
  }
  async function sendCommand(cmd) {
    if (!cmd.startsWith("/"))
      return false;
    try {
      await apiPost("/miniapp/api/command", { command: cmd });
      return true;
    } catch {
      return false;
    }
  }

  // src/utils.ts
  function formatAge(sec) {
    if (sec < 60)
      return sec + "s ago";
    if (sec < 3600)
      return Math.floor(sec / 60) + "m ago";
    return Math.floor(sec / 3600) + "h ago";
  }
  function formatTokens(n2) {
    if (n2 >= 1e6)
      return (n2 / 1e6).toFixed(1) + "M";
    if (n2 >= 1000)
      return (n2 / 1000).toFixed(1) + "K";
    return String(n2);
  }
  function flashSent(el) {
    el.classList.add("sent");
    setTimeout(() => el.classList.remove("sent"), 600);
  }
  function formatSessionLabel(key) {
    if (key.startsWith("heartbeat:"))
      return "Heartbeat";
    const parts = key.split(":");
    if (parts.length >= 4) {
      const channel = parts[2];
      const scope = parts[3];
      return capitalize(channel) + " " + capitalize(scope);
    }
    if (parts.length > 2)
      return parts.slice(2).join(":");
    return key;
  }
  function capitalize(s3) {
    return s3.charAt(0).toUpperCase() + s3.slice(1);
  }
  function isFresh(lastUpdate, key, ms = 5000) {
    return !!lastUpdate[key] && Date.now() - lastUpdate[key] < ms;
  }

  // node_modules/marked/lib/marked.esm.js
  function M2() {
    return { async: false, breaks: false, extensions: null, gfm: true, hooks: null, pedantic: false, renderer: null, silent: false, tokenizer: null, walkTokens: null };
  }
  var T3 = M2();
  function G2(u3) {
    T3 = u3;
  }
  var _2 = { exec: () => null };
  function k3(u3, e3 = "") {
    let t3 = typeof u3 == "string" ? u3 : u3.source, n2 = { replace: (r3, i3) => {
      let s3 = typeof i3 == "string" ? i3 : i3.source;
      return s3 = s3.replace(m3.caret, "$1"), t3 = t3.replace(r3, s3), n2;
    }, getRegex: () => new RegExp(t3, e3) };
    return n2;
  }
  var Re = (() => {
    try {
      return !!new RegExp("(?<=1)(?<!1)");
    } catch {
      return false;
    }
  })();
  var m3 = { codeRemoveIndent: /^(?: {1,4}| {0,3}\t)/gm, outputLinkReplace: /\\([\[\]])/g, indentCodeCompensation: /^(\s+)(?:```)/, beginningSpace: /^\s+/, endingHash: /#$/, startingSpaceChar: /^ /, endingSpaceChar: / $/, nonSpaceChar: /[^ ]/, newLineCharGlobal: /\n/g, tabCharGlobal: /\t/g, multipleSpaceGlobal: /\s+/g, blankLine: /^[ \t]*$/, doubleBlankLine: /\n[ \t]*\n[ \t]*$/, blockquoteStart: /^ {0,3}>/, blockquoteSetextReplace: /\n {0,3}((?:=+|-+) *)(?=\n|$)/g, blockquoteSetextReplace2: /^ {0,3}>[ \t]?/gm, listReplaceNesting: /^ {1,4}(?=( {4})*[^ ])/g, listIsTask: /^\[[ xX]\] +\S/, listReplaceTask: /^\[[ xX]\] +/, listTaskCheckbox: /\[[ xX]\]/, anyLine: /\n.*\n/, hrefBrackets: /^<(.*)>$/, tableDelimiter: /[:|]/, tableAlignChars: /^\||\| *$/g, tableRowBlankLine: /\n[ \t]*$/, tableAlignRight: /^ *-+: *$/, tableAlignCenter: /^ *:-+: *$/, tableAlignLeft: /^ *:-+ *$/, startATag: /^<a /i, endATag: /^<\/a>/i, startPreScriptTag: /^<(pre|code|kbd|script)(\s|>)/i, endPreScriptTag: /^<\/(pre|code|kbd|script)(\s|>)/i, startAngleBracket: /^</, endAngleBracket: />$/, pedanticHrefTitle: /^([^'"]*[^\s])\s+(['"])(.*)\2/, unicodeAlphaNumeric: /[\p{L}\p{N}]/u, escapeTest: /[&<>"']/, escapeReplace: /[&<>"']/g, escapeTestNoEncode: /[<>"']|&(?!(#\d{1,7}|#[Xx][a-fA-F0-9]{1,6}|\w+);)/, escapeReplaceNoEncode: /[<>"']|&(?!(#\d{1,7}|#[Xx][a-fA-F0-9]{1,6}|\w+);)/g, caret: /(^|[^\[])\^/g, percentDecode: /%25/g, findPipe: /\|/g, splitPipe: / \|/, slashPipe: /\\\|/g, carriageReturn: /\r\n|\r/g, spaceLine: /^ +$/gm, notSpaceStart: /^\S*/, endingNewline: /\n$/, listItemRegex: (u3) => new RegExp(`^( {0,3}${u3})((?:[	 ][^\\n]*)?(?:\\n|$))`), nextBulletRegex: (u3) => new RegExp(`^ {0,${Math.min(3, u3 - 1)}}(?:[*+-]|\\d{1,9}[.)])((?:[ 	][^\\n]*)?(?:\\n|$))`), hrRegex: (u3) => new RegExp(`^ {0,${Math.min(3, u3 - 1)}}((?:- *){3,}|(?:_ *){3,}|(?:\\* *){3,})(?:\\n+|$)`), fencesBeginRegex: (u3) => new RegExp(`^ {0,${Math.min(3, u3 - 1)}}(?:\`\`\`|~~~)`), headingBeginRegex: (u3) => new RegExp(`^ {0,${Math.min(3, u3 - 1)}}#`), htmlBeginRegex: (u3) => new RegExp(`^ {0,${Math.min(3, u3 - 1)}}<(?:[a-z].*>|!--)`, "i"), blockquoteBeginRegex: (u3) => new RegExp(`^ {0,${Math.min(3, u3 - 1)}}>`) };
  var Te = /^(?:[ \t]*(?:\n|$))+/;
  var Oe = /^((?: {4}| {0,3}\t)[^\n]+(?:\n(?:[ \t]*(?:\n|$))*)?)+/;
  var we = /^ {0,3}(`{3,}(?=[^`\n]*(?:\n|$))|~{3,})([^\n]*)(?:\n|$)(?:|([\s\S]*?)(?:\n|$))(?: {0,3}\1[~`]* *(?=\n|$)|$)/;
  var A3 = /^ {0,3}((?:-[\t ]*){3,}|(?:_[ \t]*){3,}|(?:\*[ \t]*){3,})(?:\n+|$)/;
  var ye = /^ {0,3}(#{1,6})(?=\s|$)(.*)(?:\n+|$)/;
  var N2 = / {0,3}(?:[*+-]|\d{1,9}[.)])/;
  var re = /^(?!bull |blockCode|fences|blockquote|heading|html|table)((?:.|\n(?!\s*?\n|bull |blockCode|fences|blockquote|heading|html|table))+?)\n {0,3}(=+|-+) *(?:\n+|$)/;
  var se = k3(re).replace(/bull/g, N2).replace(/blockCode/g, /(?: {4}| {0,3}\t)/).replace(/fences/g, / {0,3}(?:`{3,}|~{3,})/).replace(/blockquote/g, / {0,3}>/).replace(/heading/g, / {0,3}#{1,6}/).replace(/html/g, / {0,3}<[^\n>]+>\n/).replace(/\|table/g, "").getRegex();
  var Pe = k3(re).replace(/bull/g, N2).replace(/blockCode/g, /(?: {4}| {0,3}\t)/).replace(/fences/g, / {0,3}(?:`{3,}|~{3,})/).replace(/blockquote/g, / {0,3}>/).replace(/heading/g, / {0,3}#{1,6}/).replace(/html/g, / {0,3}<[^\n>]+>\n/).replace(/table/g, / {0,3}\|?(?:[:\- ]*\|)+[\:\- ]*\n/).getRegex();
  var Q = /^([^\n]+(?:\n(?!hr|heading|lheading|blockquote|fences|list|html|table| +\n)[^\n]+)*)/;
  var Se = /^[^\n]+/;
  var j3 = /(?!\s*\])(?:\\[\s\S]|[^\[\]\\])+/;
  var $e = k3(/^ {0,3}\[(label)\]: *(?:\n[ \t]*)?([^<\s][^\s]*|<.*?>)(?:(?: +(?:\n[ \t]*)?| *\n[ \t]*)(title))? *(?:\n+|$)/).replace("label", j3).replace("title", /(?:"(?:\\"?|[^"\\])*"|'[^'\n]*(?:\n[^'\n]+)*\n?'|\([^()]*\))/).getRegex();
  var _e = k3(/^(bull)([ \t][^\n]+?)?(?:\n|$)/).replace(/bull/g, N2).getRegex();
  var q3 = "address|article|aside|base|basefont|blockquote|body|caption|center|col|colgroup|dd|details|dialog|dir|div|dl|dt|fieldset|figcaption|figure|footer|form|frame|frameset|h[1-6]|head|header|hr|html|iframe|legend|li|link|main|menu|menuitem|meta|nav|noframes|ol|optgroup|option|p|param|search|section|summary|table|tbody|td|tfoot|th|thead|title|tr|track|ul";
  var F2 = /<!--(?:-?>|[\s\S]*?(?:-->|$))/;
  var Le = k3("^ {0,3}(?:<(script|pre|style|textarea)[\\s>][\\s\\S]*?(?:</\\1>[^\\n]*\\n+|$)|comment[^\\n]*(\\n+|$)|<\\?[\\s\\S]*?(?:\\?>\\n*|$)|<![A-Z][\\s\\S]*?(?:>\\n*|$)|<!\\[CDATA\\[[\\s\\S]*?(?:\\]\\]>\\n*|$)|</?(tag)(?: +|\\n|/?>)[\\s\\S]*?(?:(?:\\n[ \t]*)+\\n|$)|<(?!script|pre|style|textarea)([a-z][\\w-]*)(?:attribute)*? */?>(?=[ \\t]*(?:\\n|$))[\\s\\S]*?(?:(?:\\n[ \t]*)+\\n|$)|</(?!script|pre|style|textarea)[a-z][\\w-]*\\s*>(?=[ \\t]*(?:\\n|$))[\\s\\S]*?(?:(?:\\n[ \t]*)+\\n|$))", "i").replace("comment", F2).replace("tag", q3).replace("attribute", / +[a-zA-Z:_][\w.:-]*(?: *= *"[^"\n]*"| *= *'[^'\n]*'| *= *[^\s"'=<>`]+)?/).getRegex();
  var ie = k3(Q).replace("hr", A3).replace("heading", " {0,3}#{1,6}(?:\\s|$)").replace("|lheading", "").replace("|table", "").replace("blockquote", " {0,3}>").replace("fences", " {0,3}(?:`{3,}(?=[^`\\n]*\\n)|~{3,})[^\\n]*\\n").replace("list", " {0,3}(?:[*+-]|1[.)])[ \\t]").replace("html", "</?(?:tag)(?: +|\\n|/?>)|<(?:script|pre|style|textarea|!--)").replace("tag", q3).getRegex();
  var Me = k3(/^( {0,3}> ?(paragraph|[^\n]*)(?:\n|$))+/).replace("paragraph", ie).getRegex();
  var U = { blockquote: Me, code: Oe, def: $e, fences: we, heading: ye, hr: A3, html: Le, lheading: se, list: _e, newline: Te, paragraph: ie, table: _2, text: Se };
  var te = k3("^ *([^\\n ].*)\\n {0,3}((?:\\| *)?:?-+:? *(?:\\| *:?-+:? *)*(?:\\| *)?)(?:\\n((?:(?! *\\n|hr|heading|blockquote|code|fences|list|html).*(?:\\n|$))*)\\n*|$)").replace("hr", A3).replace("heading", " {0,3}#{1,6}(?:\\s|$)").replace("blockquote", " {0,3}>").replace("code", "(?: {4}| {0,3}\t)[^\\n]").replace("fences", " {0,3}(?:`{3,}(?=[^`\\n]*\\n)|~{3,})[^\\n]*\\n").replace("list", " {0,3}(?:[*+-]|1[.)])[ \\t]").replace("html", "</?(?:tag)(?: +|\\n|/?>)|<(?:script|pre|style|textarea|!--)").replace("tag", q3).getRegex();
  var ze = { ...U, lheading: Pe, table: te, paragraph: k3(Q).replace("hr", A3).replace("heading", " {0,3}#{1,6}(?:\\s|$)").replace("|lheading", "").replace("table", te).replace("blockquote", " {0,3}>").replace("fences", " {0,3}(?:`{3,}(?=[^`\\n]*\\n)|~{3,})[^\\n]*\\n").replace("list", " {0,3}(?:[*+-]|1[.)])[ \\t]").replace("html", "</?(?:tag)(?: +|\\n|/?>)|<(?:script|pre|style|textarea|!--)").replace("tag", q3).getRegex() };
  var Ee = { ...U, html: k3(`^ *(?:comment *(?:\\n|\\s*$)|<(tag)[\\s\\S]+?</\\1> *(?:\\n{2,}|\\s*$)|<tag(?:"[^"]*"|'[^']*'|\\s[^'"/>\\s]*)*?/?> *(?:\\n{2,}|\\s*$))`).replace("comment", F2).replace(/tag/g, "(?!(?:a|em|strong|small|s|cite|q|dfn|abbr|data|time|code|var|samp|kbd|sub|sup|i|b|u|mark|ruby|rt|rp|bdi|bdo|span|br|wbr|ins|del|img)\\b)\\w+(?!:|[^\\w\\s@]*@)\\b").getRegex(), def: /^ *\[([^\]]+)\]: *<?([^\s>]+)>?(?: +(["(][^\n]+[")]))? *(?:\n+|$)/, heading: /^(#{1,6})(.*)(?:\n+|$)/, fences: _2, lheading: /^(.+?)\n {0,3}(=+|-+) *(?:\n+|$)/, paragraph: k3(Q).replace("hr", A3).replace("heading", ` *#{1,6} *[^
]`).replace("lheading", se).replace("|table", "").replace("blockquote", " {0,3}>").replace("|fences", "").replace("|list", "").replace("|html", "").replace("|tag", "").getRegex() };
  var Ie = /^\\([!"#$%&'()*+,\-./:;<=>?@\[\]\\^_`{|}~])/;
  var Ae = /^(`+)([^`]|[^`][\s\S]*?[^`])\1(?!`)/;
  var oe = /^( {2,}|\\)\n(?!\s*$)/;
  var Ce = /^(`+|[^`])(?:(?= {2,}\n)|[\s\S]*?(?:(?=[\\<!\[`*_]|\b_|$)|[^ ](?= {2,}\n)))/;
  var v3 = /[\p{P}\p{S}]/u;
  var K = /[\s\p{P}\p{S}]/u;
  var ae = /[^\s\p{P}\p{S}]/u;
  var Be = k3(/^((?![*_])punctSpace)/, "u").replace(/punctSpace/g, K).getRegex();
  var le = /(?!~)[\p{P}\p{S}]/u;
  var De = /(?!~)[\s\p{P}\p{S}]/u;
  var qe = /(?:[^\s\p{P}\p{S}]|~)/u;
  var ue = /(?![*_])[\p{P}\p{S}]/u;
  var ve = /(?![*_])[\s\p{P}\p{S}]/u;
  var He = /(?:[^\s\p{P}\p{S}]|[*_])/u;
  var Ge = k3(/link|precode-code|html/, "g").replace("link", /\[(?:[^\[\]`]|(?<a>`+)[^`]+\k<a>(?!`))*?\]\((?:\\[\s\S]|[^\\\(\)]|\((?:\\[\s\S]|[^\\\(\)])*\))*\)/).replace("precode-", Re ? "(?<!`)()" : "(^^|[^`])").replace("code", /(?<b>`+)[^`]+\k<b>(?!`)/).replace("html", /<(?! )[^<>]*?>/).getRegex();
  var pe = /^(?:\*+(?:((?!\*)punct)|[^\s*]))|^_+(?:((?!_)punct)|([^\s_]))/;
  var Ze = k3(pe, "u").replace(/punct/g, v3).getRegex();
  var Ne = k3(pe, "u").replace(/punct/g, le).getRegex();
  var ce = "^[^_*]*?__[^_*]*?\\*[^_*]*?(?=__)|[^*]+(?=[^*])|(?!\\*)punct(\\*+)(?=[\\s]|$)|notPunctSpace(\\*+)(?!\\*)(?=punctSpace|$)|(?!\\*)punctSpace(\\*+)(?=notPunctSpace)|[\\s](\\*+)(?!\\*)(?=punct)|(?!\\*)punct(\\*+)(?!\\*)(?=punct)|notPunctSpace(\\*+)(?=notPunctSpace)";
  var Qe = k3(ce, "gu").replace(/notPunctSpace/g, ae).replace(/punctSpace/g, K).replace(/punct/g, v3).getRegex();
  var je = k3(ce, "gu").replace(/notPunctSpace/g, qe).replace(/punctSpace/g, De).replace(/punct/g, le).getRegex();
  var Fe = k3("^[^_*]*?\\*\\*[^_*]*?_[^_*]*?(?=\\*\\*)|[^_]+(?=[^_])|(?!_)punct(_+)(?=[\\s]|$)|notPunctSpace(_+)(?!_)(?=punctSpace|$)|(?!_)punctSpace(_+)(?=notPunctSpace)|[\\s](_+)(?!_)(?=punct)|(?!_)punct(_+)(?!_)(?=punct)", "gu").replace(/notPunctSpace/g, ae).replace(/punctSpace/g, K).replace(/punct/g, v3).getRegex();
  var Ue = k3(/^~~?(?:((?!~)punct)|[^\s~])/, "u").replace(/punct/g, ue).getRegex();
  var Ke = "^[^~]+(?=[^~])|(?!~)punct(~~?)(?=[\\s]|$)|notPunctSpace(~~?)(?!~)(?=punctSpace|$)|(?!~)punctSpace(~~?)(?=notPunctSpace)|[\\s](~~?)(?!~)(?=punct)|(?!~)punct(~~?)(?!~)(?=punct)|notPunctSpace(~~?)(?=notPunctSpace)";
  var We = k3(Ke, "gu").replace(/notPunctSpace/g, He).replace(/punctSpace/g, ve).replace(/punct/g, ue).getRegex();
  var Xe = k3(/\\(punct)/, "gu").replace(/punct/g, v3).getRegex();
  var Je = k3(/^<(scheme:[^\s\x00-\x1f<>]*|email)>/).replace("scheme", /[a-zA-Z][a-zA-Z0-9+.-]{1,31}/).replace("email", /[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+(@)[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)+(?![-_])/).getRegex();
  var Ve = k3(F2).replace("(?:-->|$)", "-->").getRegex();
  var Ye = k3("^comment|^</[a-zA-Z][\\w:-]*\\s*>|^<[a-zA-Z][\\w-]*(?:attribute)*?\\s*/?>|^<\\?[\\s\\S]*?\\?>|^<![a-zA-Z]+\\s[\\s\\S]*?>|^<!\\[CDATA\\[[\\s\\S]*?\\]\\]>").replace("comment", Ve).replace("attribute", /\s+[a-zA-Z:_][\w.:-]*(?:\s*=\s*"[^"]*"|\s*=\s*'[^']*'|\s*=\s*[^\s"'=<>`]+)?/).getRegex();
  var D3 = /(?:\[(?:\\[\s\S]|[^\[\]\\])*\]|\\[\s\S]|`+[^`]*?`+(?!`)|[^\[\]\\`])*?/;
  var et = k3(/^!?\[(label)\]\(\s*(href)(?:(?:[ \t]+(?:\n[ \t]*)?|\n[ \t]*)(title))?\s*\)/).replace("label", D3).replace("href", /<(?:\\.|[^\n<>\\])+>|[^ \t\n\x00-\x1f]*/).replace("title", /"(?:\\"?|[^"\\])*"|'(?:\\'?|[^'\\])*'|\((?:\\\)?|[^)\\])*\)/).getRegex();
  var he = k3(/^!?\[(label)\]\[(ref)\]/).replace("label", D3).replace("ref", j3).getRegex();
  var ke = k3(/^!?\[(ref)\](?:\[\])?/).replace("ref", j3).getRegex();
  var tt = k3("reflink|nolink(?!\\()", "g").replace("reflink", he).replace("nolink", ke).getRegex();
  var ne = /[hH][tT][tT][pP][sS]?|[fF][tT][pP]/;
  var W = { _backpedal: _2, anyPunctuation: Xe, autolink: Je, blockSkip: Ge, br: oe, code: Ae, del: _2, delLDelim: _2, delRDelim: _2, emStrongLDelim: Ze, emStrongRDelimAst: Qe, emStrongRDelimUnd: Fe, escape: Ie, link: et, nolink: ke, punctuation: Be, reflink: he, reflinkSearch: tt, tag: Ye, text: Ce, url: _2 };
  var nt = { ...W, link: k3(/^!?\[(label)\]\((.*?)\)/).replace("label", D3).getRegex(), reflink: k3(/^!?\[(label)\]\s*\[([^\]]*)\]/).replace("label", D3).getRegex() };
  var Z = { ...W, emStrongRDelimAst: je, emStrongLDelim: Ne, delLDelim: Ue, delRDelim: We, url: k3(/^((?:protocol):\/\/|www\.)(?:[a-zA-Z0-9\-]+\.?)+[^\s<]*|^email/).replace("protocol", ne).replace("email", /[A-Za-z0-9._+-]+(@)[a-zA-Z0-9-_]+(?:\.[a-zA-Z0-9-_]*[a-zA-Z0-9])+(?![-_])/).getRegex(), _backpedal: /(?:[^?!.,:;*_'"~()&]+|\([^)]*\)|&(?![a-zA-Z0-9]+;$)|[?!.,:;*_'"~)]+(?!$))+/, del: /^(~~?)(?=[^\s~])((?:\\[\s\S]|[^\\])*?(?:\\[\s\S]|[^\s~\\]))\1(?=[^~]|$)/, text: k3(/^([`~]+|[^`~])(?:(?= {2,}\n)|(?=[a-zA-Z0-9.!#$%&'*+\/=?_`{\|}~-]+@)|[\s\S]*?(?:(?=[\\<!\[`*~_]|\b_|protocol:\/\/|www\.|$)|[^ ](?= {2,}\n)|[^a-zA-Z0-9.!#$%&'*+\/=?_`{\|}~-](?=[a-zA-Z0-9.!#$%&'*+\/=?_`{\|}~-]+@)))/).replace("protocol", ne).getRegex() };
  var rt = { ...Z, br: k3(oe).replace("{2,}", "*").getRegex(), text: k3(Z.text).replace("\\b_", "\\b_| {2,}\\n").replace(/\{2,\}/g, "*").getRegex() };
  var C3 = { normal: U, gfm: ze, pedantic: Ee };
  var z3 = { normal: W, gfm: Z, breaks: rt, pedantic: nt };
  var st = { "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" };
  var de = (u3) => st[u3];
  function O2(u3, e3) {
    if (e3) {
      if (m3.escapeTest.test(u3))
        return u3.replace(m3.escapeReplace, de);
    } else if (m3.escapeTestNoEncode.test(u3))
      return u3.replace(m3.escapeReplaceNoEncode, de);
    return u3;
  }
  function X(u3) {
    try {
      u3 = encodeURI(u3).replace(m3.percentDecode, "%");
    } catch {
      return null;
    }
    return u3;
  }
  function J2(u3, e3) {
    let t3 = u3.replace(m3.findPipe, (i3, s3, a3) => {
      let o3 = false, l3 = s3;
      for (;--l3 >= 0 && a3[l3] === "\\"; )
        o3 = !o3;
      return o3 ? "|" : " |";
    }), n2 = t3.split(m3.splitPipe), r3 = 0;
    if (n2[0].trim() || n2.shift(), n2.length > 0 && !n2.at(-1)?.trim() && n2.pop(), e3)
      if (n2.length > e3)
        n2.splice(e3);
      else
        for (;n2.length < e3; )
          n2.push("");
    for (;r3 < n2.length; r3++)
      n2[r3] = n2[r3].trim().replace(m3.slashPipe, "|");
    return n2;
  }
  function E2(u3, e3, t3) {
    let n2 = u3.length;
    if (n2 === 0)
      return "";
    let r3 = 0;
    for (;r3 < n2; ) {
      let i3 = u3.charAt(n2 - r3 - 1);
      if (i3 === e3 && !t3)
        r3++;
      else if (i3 !== e3 && t3)
        r3++;
      else
        break;
    }
    return u3.slice(0, n2 - r3);
  }
  function ge(u3, e3) {
    if (u3.indexOf(e3[1]) === -1)
      return -1;
    let t3 = 0;
    for (let n2 = 0;n2 < u3.length; n2++)
      if (u3[n2] === "\\")
        n2++;
      else if (u3[n2] === e3[0])
        t3++;
      else if (u3[n2] === e3[1] && (t3--, t3 < 0))
        return n2;
    return t3 > 0 ? -2 : -1;
  }
  function fe(u3, e3 = 0) {
    let t3 = e3, n2 = "";
    for (let r3 of u3)
      if (r3 === "\t") {
        let i3 = 4 - t3 % 4;
        n2 += " ".repeat(i3), t3 += i3;
      } else
        n2 += r3, t3++;
    return n2;
  }
  function me(u3, e3, t3, n2, r3) {
    let i3 = e3.href, s3 = e3.title || null, a3 = u3[1].replace(r3.other.outputLinkReplace, "$1");
    n2.state.inLink = true;
    let o3 = { type: u3[0].charAt(0) === "!" ? "image" : "link", raw: t3, href: i3, title: s3, text: a3, tokens: n2.inlineTokens(a3) };
    return n2.state.inLink = false, o3;
  }
  function it(u3, e3, t3) {
    let n2 = u3.match(t3.other.indentCodeCompensation);
    if (n2 === null)
      return e3;
    let r3 = n2[1];
    return e3.split(`
`).map((i3) => {
      let s3 = i3.match(t3.other.beginningSpace);
      if (s3 === null)
        return i3;
      let [a3] = s3;
      return a3.length >= r3.length ? i3.slice(r3.length) : i3;
    }).join(`
`);
  }
  var w3 = class {
    options;
    rules;
    lexer;
    constructor(e3) {
      this.options = e3 || T3;
    }
    space(e3) {
      let t3 = this.rules.block.newline.exec(e3);
      if (t3 && t3[0].length > 0)
        return { type: "space", raw: t3[0] };
    }
    code(e3) {
      let t3 = this.rules.block.code.exec(e3);
      if (t3) {
        let n2 = t3[0].replace(this.rules.other.codeRemoveIndent, "");
        return { type: "code", raw: t3[0], codeBlockStyle: "indented", text: this.options.pedantic ? n2 : E2(n2, `
`) };
      }
    }
    fences(e3) {
      let t3 = this.rules.block.fences.exec(e3);
      if (t3) {
        let n2 = t3[0], r3 = it(n2, t3[3] || "", this.rules);
        return { type: "code", raw: n2, lang: t3[2] ? t3[2].trim().replace(this.rules.inline.anyPunctuation, "$1") : t3[2], text: r3 };
      }
    }
    heading(e3) {
      let t3 = this.rules.block.heading.exec(e3);
      if (t3) {
        let n2 = t3[2].trim();
        if (this.rules.other.endingHash.test(n2)) {
          let r3 = E2(n2, "#");
          (this.options.pedantic || !r3 || this.rules.other.endingSpaceChar.test(r3)) && (n2 = r3.trim());
        }
        return { type: "heading", raw: t3[0], depth: t3[1].length, text: n2, tokens: this.lexer.inline(n2) };
      }
    }
    hr(e3) {
      let t3 = this.rules.block.hr.exec(e3);
      if (t3)
        return { type: "hr", raw: E2(t3[0], `
`) };
    }
    blockquote(e3) {
      let t3 = this.rules.block.blockquote.exec(e3);
      if (t3) {
        let n2 = E2(t3[0], `
`).split(`
`), r3 = "", i3 = "", s3 = [];
        for (;n2.length > 0; ) {
          let a3 = false, o3 = [], l3;
          for (l3 = 0;l3 < n2.length; l3++)
            if (this.rules.other.blockquoteStart.test(n2[l3]))
              o3.push(n2[l3]), a3 = true;
            else if (!a3)
              o3.push(n2[l3]);
            else
              break;
          n2 = n2.slice(l3);
          let p3 = o3.join(`
`), c3 = p3.replace(this.rules.other.blockquoteSetextReplace, `
    $1`).replace(this.rules.other.blockquoteSetextReplace2, "");
          r3 = r3 ? `${r3}
${p3}` : p3, i3 = i3 ? `${i3}
${c3}` : c3;
          let d3 = this.lexer.state.top;
          if (this.lexer.state.top = true, this.lexer.blockTokens(c3, s3, true), this.lexer.state.top = d3, n2.length === 0)
            break;
          let h3 = s3.at(-1);
          if (h3?.type === "code")
            break;
          if (h3?.type === "blockquote") {
            let R = h3, f3 = R.raw + `
` + n2.join(`
`), S2 = this.blockquote(f3);
            s3[s3.length - 1] = S2, r3 = r3.substring(0, r3.length - R.raw.length) + S2.raw, i3 = i3.substring(0, i3.length - R.text.length) + S2.text;
            break;
          } else if (h3?.type === "list") {
            let R = h3, f3 = R.raw + `
` + n2.join(`
`), S2 = this.list(f3);
            s3[s3.length - 1] = S2, r3 = r3.substring(0, r3.length - h3.raw.length) + S2.raw, i3 = i3.substring(0, i3.length - R.raw.length) + S2.raw, n2 = f3.substring(s3.at(-1).raw.length).split(`
`);
            continue;
          }
        }
        return { type: "blockquote", raw: r3, tokens: s3, text: i3 };
      }
    }
    list(e3) {
      let t3 = this.rules.block.list.exec(e3);
      if (t3) {
        let n2 = t3[1].trim(), r3 = n2.length > 1, i3 = { type: "list", raw: "", ordered: r3, start: r3 ? +n2.slice(0, -1) : "", loose: false, items: [] };
        n2 = r3 ? `\\d{1,9}\\${n2.slice(-1)}` : `\\${n2}`, this.options.pedantic && (n2 = r3 ? n2 : "[*+-]");
        let s3 = this.rules.other.listItemRegex(n2), a3 = false;
        for (;e3; ) {
          let l3 = false, p3 = "", c3 = "";
          if (!(t3 = s3.exec(e3)) || this.rules.block.hr.test(e3))
            break;
          p3 = t3[0], e3 = e3.substring(p3.length);
          let d3 = fe(t3[2].split(`
`, 1)[0], t3[1].length), h3 = e3.split(`
`, 1)[0], R = !d3.trim(), f3 = 0;
          if (this.options.pedantic ? (f3 = 2, c3 = d3.trimStart()) : R ? f3 = t3[1].length + 1 : (f3 = d3.search(this.rules.other.nonSpaceChar), f3 = f3 > 4 ? 1 : f3, c3 = d3.slice(f3), f3 += t3[1].length), R && this.rules.other.blankLine.test(h3) && (p3 += h3 + `
`, e3 = e3.substring(h3.length + 1), l3 = true), !l3) {
            let S2 = this.rules.other.nextBulletRegex(f3), V2 = this.rules.other.hrRegex(f3), Y = this.rules.other.fencesBeginRegex(f3), ee = this.rules.other.headingBeginRegex(f3), xe = this.rules.other.htmlBeginRegex(f3), be = this.rules.other.blockquoteBeginRegex(f3);
            for (;e3; ) {
              let H2 = e3.split(`
`, 1)[0], I2;
              if (h3 = H2, this.options.pedantic ? (h3 = h3.replace(this.rules.other.listReplaceNesting, "  "), I2 = h3) : I2 = h3.replace(this.rules.other.tabCharGlobal, "    "), Y.test(h3) || ee.test(h3) || xe.test(h3) || be.test(h3) || S2.test(h3) || V2.test(h3))
                break;
              if (I2.search(this.rules.other.nonSpaceChar) >= f3 || !h3.trim())
                c3 += `
` + I2.slice(f3);
              else {
                if (R || d3.replace(this.rules.other.tabCharGlobal, "    ").search(this.rules.other.nonSpaceChar) >= 4 || Y.test(d3) || ee.test(d3) || V2.test(d3))
                  break;
                c3 += `
` + h3;
              }
              R = !h3.trim(), p3 += H2 + `
`, e3 = e3.substring(H2.length + 1), d3 = I2.slice(f3);
            }
          }
          i3.loose || (a3 ? i3.loose = true : this.rules.other.doubleBlankLine.test(p3) && (a3 = true)), i3.items.push({ type: "list_item", raw: p3, task: !!this.options.gfm && this.rules.other.listIsTask.test(c3), loose: false, text: c3, tokens: [] }), i3.raw += p3;
        }
        let o3 = i3.items.at(-1);
        if (o3)
          o3.raw = o3.raw.trimEnd(), o3.text = o3.text.trimEnd();
        else
          return;
        i3.raw = i3.raw.trimEnd();
        for (let l3 of i3.items) {
          if (this.lexer.state.top = false, l3.tokens = this.lexer.blockTokens(l3.text, []), l3.task) {
            if (l3.text = l3.text.replace(this.rules.other.listReplaceTask, ""), l3.tokens[0]?.type === "text" || l3.tokens[0]?.type === "paragraph") {
              l3.tokens[0].raw = l3.tokens[0].raw.replace(this.rules.other.listReplaceTask, ""), l3.tokens[0].text = l3.tokens[0].text.replace(this.rules.other.listReplaceTask, "");
              for (let c3 = this.lexer.inlineQueue.length - 1;c3 >= 0; c3--)
                if (this.rules.other.listIsTask.test(this.lexer.inlineQueue[c3].src)) {
                  this.lexer.inlineQueue[c3].src = this.lexer.inlineQueue[c3].src.replace(this.rules.other.listReplaceTask, "");
                  break;
                }
            }
            let p3 = this.rules.other.listTaskCheckbox.exec(l3.raw);
            if (p3) {
              let c3 = { type: "checkbox", raw: p3[0] + " ", checked: p3[0] !== "[ ]" };
              l3.checked = c3.checked, i3.loose ? l3.tokens[0] && ["paragraph", "text"].includes(l3.tokens[0].type) && "tokens" in l3.tokens[0] && l3.tokens[0].tokens ? (l3.tokens[0].raw = c3.raw + l3.tokens[0].raw, l3.tokens[0].text = c3.raw + l3.tokens[0].text, l3.tokens[0].tokens.unshift(c3)) : l3.tokens.unshift({ type: "paragraph", raw: c3.raw, text: c3.raw, tokens: [c3] }) : l3.tokens.unshift(c3);
            }
          }
          if (!i3.loose) {
            let p3 = l3.tokens.filter((d3) => d3.type === "space"), c3 = p3.length > 0 && p3.some((d3) => this.rules.other.anyLine.test(d3.raw));
            i3.loose = c3;
          }
        }
        if (i3.loose)
          for (let l3 of i3.items) {
            l3.loose = true;
            for (let p3 of l3.tokens)
              p3.type === "text" && (p3.type = "paragraph");
          }
        return i3;
      }
    }
    html(e3) {
      let t3 = this.rules.block.html.exec(e3);
      if (t3)
        return { type: "html", block: true, raw: t3[0], pre: t3[1] === "pre" || t3[1] === "script" || t3[1] === "style", text: t3[0] };
    }
    def(e3) {
      let t3 = this.rules.block.def.exec(e3);
      if (t3) {
        let n2 = t3[1].toLowerCase().replace(this.rules.other.multipleSpaceGlobal, " "), r3 = t3[2] ? t3[2].replace(this.rules.other.hrefBrackets, "$1").replace(this.rules.inline.anyPunctuation, "$1") : "", i3 = t3[3] ? t3[3].substring(1, t3[3].length - 1).replace(this.rules.inline.anyPunctuation, "$1") : t3[3];
        return { type: "def", tag: n2, raw: t3[0], href: r3, title: i3 };
      }
    }
    table(e3) {
      let t3 = this.rules.block.table.exec(e3);
      if (!t3 || !this.rules.other.tableDelimiter.test(t3[2]))
        return;
      let n2 = J2(t3[1]), r3 = t3[2].replace(this.rules.other.tableAlignChars, "").split("|"), i3 = t3[3]?.trim() ? t3[3].replace(this.rules.other.tableRowBlankLine, "").split(`
`) : [], s3 = { type: "table", raw: t3[0], header: [], align: [], rows: [] };
      if (n2.length === r3.length) {
        for (let a3 of r3)
          this.rules.other.tableAlignRight.test(a3) ? s3.align.push("right") : this.rules.other.tableAlignCenter.test(a3) ? s3.align.push("center") : this.rules.other.tableAlignLeft.test(a3) ? s3.align.push("left") : s3.align.push(null);
        for (let a3 = 0;a3 < n2.length; a3++)
          s3.header.push({ text: n2[a3], tokens: this.lexer.inline(n2[a3]), header: true, align: s3.align[a3] });
        for (let a3 of i3)
          s3.rows.push(J2(a3, s3.header.length).map((o3, l3) => ({ text: o3, tokens: this.lexer.inline(o3), header: false, align: s3.align[l3] })));
        return s3;
      }
    }
    lheading(e3) {
      let t3 = this.rules.block.lheading.exec(e3);
      if (t3)
        return { type: "heading", raw: t3[0], depth: t3[2].charAt(0) === "=" ? 1 : 2, text: t3[1], tokens: this.lexer.inline(t3[1]) };
    }
    paragraph(e3) {
      let t3 = this.rules.block.paragraph.exec(e3);
      if (t3) {
        let n2 = t3[1].charAt(t3[1].length - 1) === `
` ? t3[1].slice(0, -1) : t3[1];
        return { type: "paragraph", raw: t3[0], text: n2, tokens: this.lexer.inline(n2) };
      }
    }
    text(e3) {
      let t3 = this.rules.block.text.exec(e3);
      if (t3)
        return { type: "text", raw: t3[0], text: t3[0], tokens: this.lexer.inline(t3[0]) };
    }
    escape(e3) {
      let t3 = this.rules.inline.escape.exec(e3);
      if (t3)
        return { type: "escape", raw: t3[0], text: t3[1] };
    }
    tag(e3) {
      let t3 = this.rules.inline.tag.exec(e3);
      if (t3)
        return !this.lexer.state.inLink && this.rules.other.startATag.test(t3[0]) ? this.lexer.state.inLink = true : this.lexer.state.inLink && this.rules.other.endATag.test(t3[0]) && (this.lexer.state.inLink = false), !this.lexer.state.inRawBlock && this.rules.other.startPreScriptTag.test(t3[0]) ? this.lexer.state.inRawBlock = true : this.lexer.state.inRawBlock && this.rules.other.endPreScriptTag.test(t3[0]) && (this.lexer.state.inRawBlock = false), { type: "html", raw: t3[0], inLink: this.lexer.state.inLink, inRawBlock: this.lexer.state.inRawBlock, block: false, text: t3[0] };
    }
    link(e3) {
      let t3 = this.rules.inline.link.exec(e3);
      if (t3) {
        let n2 = t3[2].trim();
        if (!this.options.pedantic && this.rules.other.startAngleBracket.test(n2)) {
          if (!this.rules.other.endAngleBracket.test(n2))
            return;
          let s3 = E2(n2.slice(0, -1), "\\");
          if ((n2.length - s3.length) % 2 === 0)
            return;
        } else {
          let s3 = ge(t3[2], "()");
          if (s3 === -2)
            return;
          if (s3 > -1) {
            let o3 = (t3[0].indexOf("!") === 0 ? 5 : 4) + t3[1].length + s3;
            t3[2] = t3[2].substring(0, s3), t3[0] = t3[0].substring(0, o3).trim(), t3[3] = "";
          }
        }
        let r3 = t3[2], i3 = "";
        if (this.options.pedantic) {
          let s3 = this.rules.other.pedanticHrefTitle.exec(r3);
          s3 && (r3 = s3[1], i3 = s3[3]);
        } else
          i3 = t3[3] ? t3[3].slice(1, -1) : "";
        return r3 = r3.trim(), this.rules.other.startAngleBracket.test(r3) && (this.options.pedantic && !this.rules.other.endAngleBracket.test(n2) ? r3 = r3.slice(1) : r3 = r3.slice(1, -1)), me(t3, { href: r3 && r3.replace(this.rules.inline.anyPunctuation, "$1"), title: i3 && i3.replace(this.rules.inline.anyPunctuation, "$1") }, t3[0], this.lexer, this.rules);
      }
    }
    reflink(e3, t3) {
      let n2;
      if ((n2 = this.rules.inline.reflink.exec(e3)) || (n2 = this.rules.inline.nolink.exec(e3))) {
        let r3 = (n2[2] || n2[1]).replace(this.rules.other.multipleSpaceGlobal, " "), i3 = t3[r3.toLowerCase()];
        if (!i3) {
          let s3 = n2[0].charAt(0);
          return { type: "text", raw: s3, text: s3 };
        }
        return me(n2, i3, n2[0], this.lexer, this.rules);
      }
    }
    emStrong(e3, t3, n2 = "") {
      let r3 = this.rules.inline.emStrongLDelim.exec(e3);
      if (!r3 || r3[3] && n2.match(this.rules.other.unicodeAlphaNumeric))
        return;
      if (!(r3[1] || r3[2] || "") || !n2 || this.rules.inline.punctuation.exec(n2)) {
        let s3 = [...r3[0]].length - 1, a3, o3, l3 = s3, p3 = 0, c3 = r3[0][0] === "*" ? this.rules.inline.emStrongRDelimAst : this.rules.inline.emStrongRDelimUnd;
        for (c3.lastIndex = 0, t3 = t3.slice(-1 * e3.length + s3);(r3 = c3.exec(t3)) != null; ) {
          if (a3 = r3[1] || r3[2] || r3[3] || r3[4] || r3[5] || r3[6], !a3)
            continue;
          if (o3 = [...a3].length, r3[3] || r3[4]) {
            l3 += o3;
            continue;
          } else if ((r3[5] || r3[6]) && s3 % 3 && !((s3 + o3) % 3)) {
            p3 += o3;
            continue;
          }
          if (l3 -= o3, l3 > 0)
            continue;
          o3 = Math.min(o3, o3 + l3 + p3);
          let d3 = [...r3[0]][0].length, h3 = e3.slice(0, s3 + r3.index + d3 + o3);
          if (Math.min(s3, o3) % 2) {
            let f3 = h3.slice(1, -1);
            return { type: "em", raw: h3, text: f3, tokens: this.lexer.inlineTokens(f3) };
          }
          let R = h3.slice(2, -2);
          return { type: "strong", raw: h3, text: R, tokens: this.lexer.inlineTokens(R) };
        }
      }
    }
    codespan(e3) {
      let t3 = this.rules.inline.code.exec(e3);
      if (t3) {
        let n2 = t3[2].replace(this.rules.other.newLineCharGlobal, " "), r3 = this.rules.other.nonSpaceChar.test(n2), i3 = this.rules.other.startingSpaceChar.test(n2) && this.rules.other.endingSpaceChar.test(n2);
        return r3 && i3 && (n2 = n2.substring(1, n2.length - 1)), { type: "codespan", raw: t3[0], text: n2 };
      }
    }
    br(e3) {
      let t3 = this.rules.inline.br.exec(e3);
      if (t3)
        return { type: "br", raw: t3[0] };
    }
    del(e3, t3, n2 = "") {
      let r3 = this.rules.inline.delLDelim.exec(e3);
      if (!r3)
        return;
      if (!(r3[1] || "") || !n2 || this.rules.inline.punctuation.exec(n2)) {
        let s3 = [...r3[0]].length - 1, a3, o3, l3 = s3, p3 = this.rules.inline.delRDelim;
        for (p3.lastIndex = 0, t3 = t3.slice(-1 * e3.length + s3);(r3 = p3.exec(t3)) != null; ) {
          if (a3 = r3[1] || r3[2] || r3[3] || r3[4] || r3[5] || r3[6], !a3 || (o3 = [...a3].length, o3 !== s3))
            continue;
          if (r3[3] || r3[4]) {
            l3 += o3;
            continue;
          }
          if (l3 -= o3, l3 > 0)
            continue;
          o3 = Math.min(o3, o3 + l3);
          let c3 = [...r3[0]][0].length, d3 = e3.slice(0, s3 + r3.index + c3 + o3), h3 = d3.slice(s3, -s3);
          return { type: "del", raw: d3, text: h3, tokens: this.lexer.inlineTokens(h3) };
        }
      }
    }
    autolink(e3) {
      let t3 = this.rules.inline.autolink.exec(e3);
      if (t3) {
        let n2, r3;
        return t3[2] === "@" ? (n2 = t3[1], r3 = "mailto:" + n2) : (n2 = t3[1], r3 = n2), { type: "link", raw: t3[0], text: n2, href: r3, tokens: [{ type: "text", raw: n2, text: n2 }] };
      }
    }
    url(e3) {
      let t3;
      if (t3 = this.rules.inline.url.exec(e3)) {
        let n2, r3;
        if (t3[2] === "@")
          n2 = t3[0], r3 = "mailto:" + n2;
        else {
          let i3;
          do
            i3 = t3[0], t3[0] = this.rules.inline._backpedal.exec(t3[0])?.[0] ?? "";
          while (i3 !== t3[0]);
          n2 = t3[0], t3[1] === "www." ? r3 = "http://" + t3[0] : r3 = t3[0];
        }
        return { type: "link", raw: t3[0], text: n2, href: r3, tokens: [{ type: "text", raw: n2, text: n2 }] };
      }
    }
    inlineText(e3) {
      let t3 = this.rules.inline.text.exec(e3);
      if (t3) {
        let n2 = this.lexer.state.inRawBlock;
        return { type: "text", raw: t3[0], text: t3[0], escaped: n2 };
      }
    }
  };
  var x2 = class u3 {
    tokens;
    options;
    state;
    inlineQueue;
    tokenizer;
    constructor(e3) {
      this.tokens = [], this.tokens.links = Object.create(null), this.options = e3 || T3, this.options.tokenizer = this.options.tokenizer || new w3, this.tokenizer = this.options.tokenizer, this.tokenizer.options = this.options, this.tokenizer.lexer = this, this.inlineQueue = [], this.state = { inLink: false, inRawBlock: false, top: true };
      let t3 = { other: m3, block: C3.normal, inline: z3.normal };
      this.options.pedantic ? (t3.block = C3.pedantic, t3.inline = z3.pedantic) : this.options.gfm && (t3.block = C3.gfm, this.options.breaks ? t3.inline = z3.breaks : t3.inline = z3.gfm), this.tokenizer.rules = t3;
    }
    static get rules() {
      return { block: C3, inline: z3 };
    }
    static lex(e3, t3) {
      return new u3(t3).lex(e3);
    }
    static lexInline(e3, t3) {
      return new u3(t3).inlineTokens(e3);
    }
    lex(e3) {
      e3 = e3.replace(m3.carriageReturn, `
`), this.blockTokens(e3, this.tokens);
      for (let t3 = 0;t3 < this.inlineQueue.length; t3++) {
        let n2 = this.inlineQueue[t3];
        this.inlineTokens(n2.src, n2.tokens);
      }
      return this.inlineQueue = [], this.tokens;
    }
    blockTokens(e3, t3 = [], n2 = false) {
      for (this.options.pedantic && (e3 = e3.replace(m3.tabCharGlobal, "    ").replace(m3.spaceLine, ""));e3; ) {
        let r3;
        if (this.options.extensions?.block?.some((s3) => (r3 = s3.call({ lexer: this }, e3, t3)) ? (e3 = e3.substring(r3.raw.length), t3.push(r3), true) : false))
          continue;
        if (r3 = this.tokenizer.space(e3)) {
          e3 = e3.substring(r3.raw.length);
          let s3 = t3.at(-1);
          r3.raw.length === 1 && s3 !== undefined ? s3.raw += `
` : t3.push(r3);
          continue;
        }
        if (r3 = this.tokenizer.code(e3)) {
          e3 = e3.substring(r3.raw.length);
          let s3 = t3.at(-1);
          s3?.type === "paragraph" || s3?.type === "text" ? (s3.raw += (s3.raw.endsWith(`
`) ? "" : `
`) + r3.raw, s3.text += `
` + r3.text, this.inlineQueue.at(-1).src = s3.text) : t3.push(r3);
          continue;
        }
        if (r3 = this.tokenizer.fences(e3)) {
          e3 = e3.substring(r3.raw.length), t3.push(r3);
          continue;
        }
        if (r3 = this.tokenizer.heading(e3)) {
          e3 = e3.substring(r3.raw.length), t3.push(r3);
          continue;
        }
        if (r3 = this.tokenizer.hr(e3)) {
          e3 = e3.substring(r3.raw.length), t3.push(r3);
          continue;
        }
        if (r3 = this.tokenizer.blockquote(e3)) {
          e3 = e3.substring(r3.raw.length), t3.push(r3);
          continue;
        }
        if (r3 = this.tokenizer.list(e3)) {
          e3 = e3.substring(r3.raw.length), t3.push(r3);
          continue;
        }
        if (r3 = this.tokenizer.html(e3)) {
          e3 = e3.substring(r3.raw.length), t3.push(r3);
          continue;
        }
        if (r3 = this.tokenizer.def(e3)) {
          e3 = e3.substring(r3.raw.length);
          let s3 = t3.at(-1);
          s3?.type === "paragraph" || s3?.type === "text" ? (s3.raw += (s3.raw.endsWith(`
`) ? "" : `
`) + r3.raw, s3.text += `
` + r3.raw, this.inlineQueue.at(-1).src = s3.text) : this.tokens.links[r3.tag] || (this.tokens.links[r3.tag] = { href: r3.href, title: r3.title }, t3.push(r3));
          continue;
        }
        if (r3 = this.tokenizer.table(e3)) {
          e3 = e3.substring(r3.raw.length), t3.push(r3);
          continue;
        }
        if (r3 = this.tokenizer.lheading(e3)) {
          e3 = e3.substring(r3.raw.length), t3.push(r3);
          continue;
        }
        let i3 = e3;
        if (this.options.extensions?.startBlock) {
          let s3 = 1 / 0, a3 = e3.slice(1), o3;
          this.options.extensions.startBlock.forEach((l3) => {
            o3 = l3.call({ lexer: this }, a3), typeof o3 == "number" && o3 >= 0 && (s3 = Math.min(s3, o3));
          }), s3 < 1 / 0 && s3 >= 0 && (i3 = e3.substring(0, s3 + 1));
        }
        if (this.state.top && (r3 = this.tokenizer.paragraph(i3))) {
          let s3 = t3.at(-1);
          n2 && s3?.type === "paragraph" ? (s3.raw += (s3.raw.endsWith(`
`) ? "" : `
`) + r3.raw, s3.text += `
` + r3.text, this.inlineQueue.pop(), this.inlineQueue.at(-1).src = s3.text) : t3.push(r3), n2 = i3.length !== e3.length, e3 = e3.substring(r3.raw.length);
          continue;
        }
        if (r3 = this.tokenizer.text(e3)) {
          e3 = e3.substring(r3.raw.length);
          let s3 = t3.at(-1);
          s3?.type === "text" ? (s3.raw += (s3.raw.endsWith(`
`) ? "" : `
`) + r3.raw, s3.text += `
` + r3.text, this.inlineQueue.pop(), this.inlineQueue.at(-1).src = s3.text) : t3.push(r3);
          continue;
        }
        if (e3) {
          let s3 = "Infinite loop on byte: " + e3.charCodeAt(0);
          if (this.options.silent) {
            console.error(s3);
            break;
          } else
            throw new Error(s3);
        }
      }
      return this.state.top = true, t3;
    }
    inline(e3, t3 = []) {
      return this.inlineQueue.push({ src: e3, tokens: t3 }), t3;
    }
    inlineTokens(e3, t3 = []) {
      let n2 = e3, r3 = null;
      if (this.tokens.links) {
        let o3 = Object.keys(this.tokens.links);
        if (o3.length > 0)
          for (;(r3 = this.tokenizer.rules.inline.reflinkSearch.exec(n2)) != null; )
            o3.includes(r3[0].slice(r3[0].lastIndexOf("[") + 1, -1)) && (n2 = n2.slice(0, r3.index) + "[" + "a".repeat(r3[0].length - 2) + "]" + n2.slice(this.tokenizer.rules.inline.reflinkSearch.lastIndex));
      }
      for (;(r3 = this.tokenizer.rules.inline.anyPunctuation.exec(n2)) != null; )
        n2 = n2.slice(0, r3.index) + "++" + n2.slice(this.tokenizer.rules.inline.anyPunctuation.lastIndex);
      let i3;
      for (;(r3 = this.tokenizer.rules.inline.blockSkip.exec(n2)) != null; )
        i3 = r3[2] ? r3[2].length : 0, n2 = n2.slice(0, r3.index + i3) + "[" + "a".repeat(r3[0].length - i3 - 2) + "]" + n2.slice(this.tokenizer.rules.inline.blockSkip.lastIndex);
      n2 = this.options.hooks?.emStrongMask?.call({ lexer: this }, n2) ?? n2;
      let s3 = false, a3 = "";
      for (;e3; ) {
        s3 || (a3 = ""), s3 = false;
        let o3;
        if (this.options.extensions?.inline?.some((p3) => (o3 = p3.call({ lexer: this }, e3, t3)) ? (e3 = e3.substring(o3.raw.length), t3.push(o3), true) : false))
          continue;
        if (o3 = this.tokenizer.escape(e3)) {
          e3 = e3.substring(o3.raw.length), t3.push(o3);
          continue;
        }
        if (o3 = this.tokenizer.tag(e3)) {
          e3 = e3.substring(o3.raw.length), t3.push(o3);
          continue;
        }
        if (o3 = this.tokenizer.link(e3)) {
          e3 = e3.substring(o3.raw.length), t3.push(o3);
          continue;
        }
        if (o3 = this.tokenizer.reflink(e3, this.tokens.links)) {
          e3 = e3.substring(o3.raw.length);
          let p3 = t3.at(-1);
          o3.type === "text" && p3?.type === "text" ? (p3.raw += o3.raw, p3.text += o3.text) : t3.push(o3);
          continue;
        }
        if (o3 = this.tokenizer.emStrong(e3, n2, a3)) {
          e3 = e3.substring(o3.raw.length), t3.push(o3);
          continue;
        }
        if (o3 = this.tokenizer.codespan(e3)) {
          e3 = e3.substring(o3.raw.length), t3.push(o3);
          continue;
        }
        if (o3 = this.tokenizer.br(e3)) {
          e3 = e3.substring(o3.raw.length), t3.push(o3);
          continue;
        }
        if (o3 = this.tokenizer.del(e3, n2, a3)) {
          e3 = e3.substring(o3.raw.length), t3.push(o3);
          continue;
        }
        if (o3 = this.tokenizer.autolink(e3)) {
          e3 = e3.substring(o3.raw.length), t3.push(o3);
          continue;
        }
        if (!this.state.inLink && (o3 = this.tokenizer.url(e3))) {
          e3 = e3.substring(o3.raw.length), t3.push(o3);
          continue;
        }
        let l3 = e3;
        if (this.options.extensions?.startInline) {
          let p3 = 1 / 0, c3 = e3.slice(1), d3;
          this.options.extensions.startInline.forEach((h3) => {
            d3 = h3.call({ lexer: this }, c3), typeof d3 == "number" && d3 >= 0 && (p3 = Math.min(p3, d3));
          }), p3 < 1 / 0 && p3 >= 0 && (l3 = e3.substring(0, p3 + 1));
        }
        if (o3 = this.tokenizer.inlineText(l3)) {
          e3 = e3.substring(o3.raw.length), o3.raw.slice(-1) !== "_" && (a3 = o3.raw.slice(-1)), s3 = true;
          let p3 = t3.at(-1);
          p3?.type === "text" ? (p3.raw += o3.raw, p3.text += o3.text) : t3.push(o3);
          continue;
        }
        if (e3) {
          let p3 = "Infinite loop on byte: " + e3.charCodeAt(0);
          if (this.options.silent) {
            console.error(p3);
            break;
          } else
            throw new Error(p3);
        }
      }
      return t3;
    }
  };
  var y3 = class {
    options;
    parser;
    constructor(e3) {
      this.options = e3 || T3;
    }
    space(e3) {
      return "";
    }
    code({ text: e3, lang: t3, escaped: n2 }) {
      let r3 = (t3 || "").match(m3.notSpaceStart)?.[0], i3 = e3.replace(m3.endingNewline, "") + `
`;
      return r3 ? '<pre><code class="language-' + O2(r3) + '">' + (n2 ? i3 : O2(i3, true)) + `</code></pre>
` : "<pre><code>" + (n2 ? i3 : O2(i3, true)) + `</code></pre>
`;
    }
    blockquote({ tokens: e3 }) {
      return `<blockquote>
${this.parser.parse(e3)}</blockquote>
`;
    }
    html({ text: e3 }) {
      return e3;
    }
    def(e3) {
      return "";
    }
    heading({ tokens: e3, depth: t3 }) {
      return `<h${t3}>${this.parser.parseInline(e3)}</h${t3}>
`;
    }
    hr(e3) {
      return `<hr>
`;
    }
    list(e3) {
      let { ordered: t3, start: n2 } = e3, r3 = "";
      for (let a3 = 0;a3 < e3.items.length; a3++) {
        let o3 = e3.items[a3];
        r3 += this.listitem(o3);
      }
      let i3 = t3 ? "ol" : "ul", s3 = t3 && n2 !== 1 ? ' start="' + n2 + '"' : "";
      return "<" + i3 + s3 + `>
` + r3 + "</" + i3 + `>
`;
    }
    listitem(e3) {
      return `<li>${this.parser.parse(e3.tokens)}</li>
`;
    }
    checkbox({ checked: e3 }) {
      return "<input " + (e3 ? 'checked="" ' : "") + 'disabled="" type="checkbox"> ';
    }
    paragraph({ tokens: e3 }) {
      return `<p>${this.parser.parseInline(e3)}</p>
`;
    }
    table(e3) {
      let t3 = "", n2 = "";
      for (let i3 = 0;i3 < e3.header.length; i3++)
        n2 += this.tablecell(e3.header[i3]);
      t3 += this.tablerow({ text: n2 });
      let r3 = "";
      for (let i3 = 0;i3 < e3.rows.length; i3++) {
        let s3 = e3.rows[i3];
        n2 = "";
        for (let a3 = 0;a3 < s3.length; a3++)
          n2 += this.tablecell(s3[a3]);
        r3 += this.tablerow({ text: n2 });
      }
      return r3 && (r3 = `<tbody>${r3}</tbody>`), `<table>
<thead>
` + t3 + `</thead>
` + r3 + `</table>
`;
    }
    tablerow({ text: e3 }) {
      return `<tr>
${e3}</tr>
`;
    }
    tablecell(e3) {
      let t3 = this.parser.parseInline(e3.tokens), n2 = e3.header ? "th" : "td";
      return (e3.align ? `<${n2} align="${e3.align}">` : `<${n2}>`) + t3 + `</${n2}>
`;
    }
    strong({ tokens: e3 }) {
      return `<strong>${this.parser.parseInline(e3)}</strong>`;
    }
    em({ tokens: e3 }) {
      return `<em>${this.parser.parseInline(e3)}</em>`;
    }
    codespan({ text: e3 }) {
      return `<code>${O2(e3, true)}</code>`;
    }
    br(e3) {
      return "<br>";
    }
    del({ tokens: e3 }) {
      return `<del>${this.parser.parseInline(e3)}</del>`;
    }
    link({ href: e3, title: t3, tokens: n2 }) {
      let r3 = this.parser.parseInline(n2), i3 = X(e3);
      if (i3 === null)
        return r3;
      e3 = i3;
      let s3 = '<a href="' + e3 + '"';
      return t3 && (s3 += ' title="' + O2(t3) + '"'), s3 += ">" + r3 + "</a>", s3;
    }
    image({ href: e3, title: t3, text: n2, tokens: r3 }) {
      r3 && (n2 = this.parser.parseInline(r3, this.parser.textRenderer));
      let i3 = X(e3);
      if (i3 === null)
        return O2(n2);
      e3 = i3;
      let s3 = `<img src="${e3}" alt="${O2(n2)}"`;
      return t3 && (s3 += ` title="${O2(t3)}"`), s3 += ">", s3;
    }
    text(e3) {
      return "tokens" in e3 && e3.tokens ? this.parser.parseInline(e3.tokens) : ("escaped" in e3) && e3.escaped ? e3.text : O2(e3.text);
    }
  };
  var $2 = class {
    strong({ text: e3 }) {
      return e3;
    }
    em({ text: e3 }) {
      return e3;
    }
    codespan({ text: e3 }) {
      return e3;
    }
    del({ text: e3 }) {
      return e3;
    }
    html({ text: e3 }) {
      return e3;
    }
    text({ text: e3 }) {
      return e3;
    }
    link({ text: e3 }) {
      return "" + e3;
    }
    image({ text: e3 }) {
      return "" + e3;
    }
    br() {
      return "";
    }
    checkbox({ raw: e3 }) {
      return e3;
    }
  };
  var b = class u4 {
    options;
    renderer;
    textRenderer;
    constructor(e3) {
      this.options = e3 || T3, this.options.renderer = this.options.renderer || new y3, this.renderer = this.options.renderer, this.renderer.options = this.options, this.renderer.parser = this, this.textRenderer = new $2;
    }
    static parse(e3, t3) {
      return new u4(t3).parse(e3);
    }
    static parseInline(e3, t3) {
      return new u4(t3).parseInline(e3);
    }
    parse(e3) {
      let t3 = "";
      for (let n2 = 0;n2 < e3.length; n2++) {
        let r3 = e3[n2];
        if (this.options.extensions?.renderers?.[r3.type]) {
          let s3 = r3, a3 = this.options.extensions.renderers[s3.type].call({ parser: this }, s3);
          if (a3 !== false || !["space", "hr", "heading", "code", "table", "blockquote", "list", "html", "def", "paragraph", "text"].includes(s3.type)) {
            t3 += a3 || "";
            continue;
          }
        }
        let i3 = r3;
        switch (i3.type) {
          case "space": {
            t3 += this.renderer.space(i3);
            break;
          }
          case "hr": {
            t3 += this.renderer.hr(i3);
            break;
          }
          case "heading": {
            t3 += this.renderer.heading(i3);
            break;
          }
          case "code": {
            t3 += this.renderer.code(i3);
            break;
          }
          case "table": {
            t3 += this.renderer.table(i3);
            break;
          }
          case "blockquote": {
            t3 += this.renderer.blockquote(i3);
            break;
          }
          case "list": {
            t3 += this.renderer.list(i3);
            break;
          }
          case "checkbox": {
            t3 += this.renderer.checkbox(i3);
            break;
          }
          case "html": {
            t3 += this.renderer.html(i3);
            break;
          }
          case "def": {
            t3 += this.renderer.def(i3);
            break;
          }
          case "paragraph": {
            t3 += this.renderer.paragraph(i3);
            break;
          }
          case "text": {
            t3 += this.renderer.text(i3);
            break;
          }
          default: {
            let s3 = 'Token with "' + i3.type + '" type was not found.';
            if (this.options.silent)
              return console.error(s3), "";
            throw new Error(s3);
          }
        }
      }
      return t3;
    }
    parseInline(e3, t3 = this.renderer) {
      let n2 = "";
      for (let r3 = 0;r3 < e3.length; r3++) {
        let i3 = e3[r3];
        if (this.options.extensions?.renderers?.[i3.type]) {
          let a3 = this.options.extensions.renderers[i3.type].call({ parser: this }, i3);
          if (a3 !== false || !["escape", "html", "link", "image", "strong", "em", "codespan", "br", "del", "text"].includes(i3.type)) {
            n2 += a3 || "";
            continue;
          }
        }
        let s3 = i3;
        switch (s3.type) {
          case "escape": {
            n2 += t3.text(s3);
            break;
          }
          case "html": {
            n2 += t3.html(s3);
            break;
          }
          case "link": {
            n2 += t3.link(s3);
            break;
          }
          case "image": {
            n2 += t3.image(s3);
            break;
          }
          case "checkbox": {
            n2 += t3.checkbox(s3);
            break;
          }
          case "strong": {
            n2 += t3.strong(s3);
            break;
          }
          case "em": {
            n2 += t3.em(s3);
            break;
          }
          case "codespan": {
            n2 += t3.codespan(s3);
            break;
          }
          case "br": {
            n2 += t3.br(s3);
            break;
          }
          case "del": {
            n2 += t3.del(s3);
            break;
          }
          case "text": {
            n2 += t3.text(s3);
            break;
          }
          default: {
            let a3 = 'Token with "' + s3.type + '" type was not found.';
            if (this.options.silent)
              return console.error(a3), "";
            throw new Error(a3);
          }
        }
      }
      return n2;
    }
  };
  var P2 = class {
    options;
    block;
    constructor(e3) {
      this.options = e3 || T3;
    }
    static passThroughHooks = new Set(["preprocess", "postprocess", "processAllTokens", "emStrongMask"]);
    static passThroughHooksRespectAsync = new Set(["preprocess", "postprocess", "processAllTokens"]);
    preprocess(e3) {
      return e3;
    }
    postprocess(e3) {
      return e3;
    }
    processAllTokens(e3) {
      return e3;
    }
    emStrongMask(e3) {
      return e3;
    }
    provideLexer() {
      return this.block ? x2.lex : x2.lexInline;
    }
    provideParser() {
      return this.block ? b.parse : b.parseInline;
    }
  };
  var B3 = class {
    defaults = M2();
    options = this.setOptions;
    parse = this.parseMarkdown(true);
    parseInline = this.parseMarkdown(false);
    Parser = b;
    Renderer = y3;
    TextRenderer = $2;
    Lexer = x2;
    Tokenizer = w3;
    Hooks = P2;
    constructor(...e3) {
      this.use(...e3);
    }
    walkTokens(e3, t3) {
      let n2 = [];
      for (let r3 of e3)
        switch (n2 = n2.concat(t3.call(this, r3)), r3.type) {
          case "table": {
            let i3 = r3;
            for (let s3 of i3.header)
              n2 = n2.concat(this.walkTokens(s3.tokens, t3));
            for (let s3 of i3.rows)
              for (let a3 of s3)
                n2 = n2.concat(this.walkTokens(a3.tokens, t3));
            break;
          }
          case "list": {
            let i3 = r3;
            n2 = n2.concat(this.walkTokens(i3.items, t3));
            break;
          }
          default: {
            let i3 = r3;
            this.defaults.extensions?.childTokens?.[i3.type] ? this.defaults.extensions.childTokens[i3.type].forEach((s3) => {
              let a3 = i3[s3].flat(1 / 0);
              n2 = n2.concat(this.walkTokens(a3, t3));
            }) : i3.tokens && (n2 = n2.concat(this.walkTokens(i3.tokens, t3)));
          }
        }
      return n2;
    }
    use(...e3) {
      let t3 = this.defaults.extensions || { renderers: {}, childTokens: {} };
      return e3.forEach((n2) => {
        let r3 = { ...n2 };
        if (r3.async = this.defaults.async || r3.async || false, n2.extensions && (n2.extensions.forEach((i3) => {
          if (!i3.name)
            throw new Error("extension name required");
          if ("renderer" in i3) {
            let s3 = t3.renderers[i3.name];
            s3 ? t3.renderers[i3.name] = function(...a3) {
              let o3 = i3.renderer.apply(this, a3);
              return o3 === false && (o3 = s3.apply(this, a3)), o3;
            } : t3.renderers[i3.name] = i3.renderer;
          }
          if ("tokenizer" in i3) {
            if (!i3.level || i3.level !== "block" && i3.level !== "inline")
              throw new Error("extension level must be 'block' or 'inline'");
            let s3 = t3[i3.level];
            s3 ? s3.unshift(i3.tokenizer) : t3[i3.level] = [i3.tokenizer], i3.start && (i3.level === "block" ? t3.startBlock ? t3.startBlock.push(i3.start) : t3.startBlock = [i3.start] : i3.level === "inline" && (t3.startInline ? t3.startInline.push(i3.start) : t3.startInline = [i3.start]));
          }
          "childTokens" in i3 && i3.childTokens && (t3.childTokens[i3.name] = i3.childTokens);
        }), r3.extensions = t3), n2.renderer) {
          let i3 = this.defaults.renderer || new y3(this.defaults);
          for (let s3 in n2.renderer) {
            if (!(s3 in i3))
              throw new Error(`renderer '${s3}' does not exist`);
            if (["options", "parser"].includes(s3))
              continue;
            let a3 = s3, o3 = n2.renderer[a3], l3 = i3[a3];
            i3[a3] = (...p3) => {
              let c3 = o3.apply(i3, p3);
              return c3 === false && (c3 = l3.apply(i3, p3)), c3 || "";
            };
          }
          r3.renderer = i3;
        }
        if (n2.tokenizer) {
          let i3 = this.defaults.tokenizer || new w3(this.defaults);
          for (let s3 in n2.tokenizer) {
            if (!(s3 in i3))
              throw new Error(`tokenizer '${s3}' does not exist`);
            if (["options", "rules", "lexer"].includes(s3))
              continue;
            let a3 = s3, o3 = n2.tokenizer[a3], l3 = i3[a3];
            i3[a3] = (...p3) => {
              let c3 = o3.apply(i3, p3);
              return c3 === false && (c3 = l3.apply(i3, p3)), c3;
            };
          }
          r3.tokenizer = i3;
        }
        if (n2.hooks) {
          let i3 = this.defaults.hooks || new P2;
          for (let s3 in n2.hooks) {
            if (!(s3 in i3))
              throw new Error(`hook '${s3}' does not exist`);
            if (["options", "block"].includes(s3))
              continue;
            let a3 = s3, o3 = n2.hooks[a3], l3 = i3[a3];
            P2.passThroughHooks.has(s3) ? i3[a3] = (p3) => {
              if (this.defaults.async && P2.passThroughHooksRespectAsync.has(s3))
                return (async () => {
                  let d3 = await o3.call(i3, p3);
                  return l3.call(i3, d3);
                })();
              let c3 = o3.call(i3, p3);
              return l3.call(i3, c3);
            } : i3[a3] = (...p3) => {
              if (this.defaults.async)
                return (async () => {
                  let d3 = await o3.apply(i3, p3);
                  return d3 === false && (d3 = await l3.apply(i3, p3)), d3;
                })();
              let c3 = o3.apply(i3, p3);
              return c3 === false && (c3 = l3.apply(i3, p3)), c3;
            };
          }
          r3.hooks = i3;
        }
        if (n2.walkTokens) {
          let i3 = this.defaults.walkTokens, s3 = n2.walkTokens;
          r3.walkTokens = function(a3) {
            let o3 = [];
            return o3.push(s3.call(this, a3)), i3 && (o3 = o3.concat(i3.call(this, a3))), o3;
          };
        }
        this.defaults = { ...this.defaults, ...r3 };
      }), this;
    }
    setOptions(e3) {
      return this.defaults = { ...this.defaults, ...e3 }, this;
    }
    lexer(e3, t3) {
      return x2.lex(e3, t3 ?? this.defaults);
    }
    parser(e3, t3) {
      return b.parse(e3, t3 ?? this.defaults);
    }
    parseMarkdown(e3) {
      return (n2, r3) => {
        let i3 = { ...r3 }, s3 = { ...this.defaults, ...i3 }, a3 = this.onError(!!s3.silent, !!s3.async);
        if (this.defaults.async === true && i3.async === false)
          return a3(new Error("marked(): The async option was set to true by an extension. Remove async: false from the parse options object to return a Promise."));
        if (typeof n2 > "u" || n2 === null)
          return a3(new Error("marked(): input parameter is undefined or null"));
        if (typeof n2 != "string")
          return a3(new Error("marked(): input parameter is of type " + Object.prototype.toString.call(n2) + ", string expected"));
        if (s3.hooks && (s3.hooks.options = s3, s3.hooks.block = e3), s3.async)
          return (async () => {
            let o3 = s3.hooks ? await s3.hooks.preprocess(n2) : n2, p3 = await (s3.hooks ? await s3.hooks.provideLexer() : e3 ? x2.lex : x2.lexInline)(o3, s3), c3 = s3.hooks ? await s3.hooks.processAllTokens(p3) : p3;
            s3.walkTokens && await Promise.all(this.walkTokens(c3, s3.walkTokens));
            let h3 = await (s3.hooks ? await s3.hooks.provideParser() : e3 ? b.parse : b.parseInline)(c3, s3);
            return s3.hooks ? await s3.hooks.postprocess(h3) : h3;
          })().catch(a3);
        try {
          s3.hooks && (n2 = s3.hooks.preprocess(n2));
          let l3 = (s3.hooks ? s3.hooks.provideLexer() : e3 ? x2.lex : x2.lexInline)(n2, s3);
          s3.hooks && (l3 = s3.hooks.processAllTokens(l3)), s3.walkTokens && this.walkTokens(l3, s3.walkTokens);
          let c3 = (s3.hooks ? s3.hooks.provideParser() : e3 ? b.parse : b.parseInline)(l3, s3);
          return s3.hooks && (c3 = s3.hooks.postprocess(c3)), c3;
        } catch (o3) {
          return a3(o3);
        }
      };
    }
    onError(e3, t3) {
      return (n2) => {
        if (n2.message += `
Please report this to https://github.com/markedjs/marked.`, e3) {
          let r3 = "<p>An error occurred:</p><pre>" + O2(n2.message + "", true) + "</pre>";
          return t3 ? Promise.resolve(r3) : r3;
        }
        if (t3)
          return Promise.reject(n2);
        throw n2;
      };
    }
  };
  var L = new B3;
  function g2(u5, e3) {
    return L.parse(u5, e3);
  }
  g2.options = g2.setOptions = function(u5) {
    return L.setOptions(u5), g2.defaults = L.defaults, G2(g2.defaults), g2;
  };
  g2.getDefaults = M2;
  g2.defaults = T3;
  g2.use = function(...u5) {
    return L.use(...u5), g2.defaults = L.defaults, G2(g2.defaults), g2;
  };
  g2.walkTokens = function(u5, e3) {
    return L.walkTokens(u5, e3);
  };
  g2.parseInline = L.parseInline;
  g2.Parser = b;
  g2.parser = b.parse;
  g2.Renderer = y3;
  g2.TextRenderer = $2;
  g2.Lexer = x2;
  g2.lexer = x2.lex;
  g2.Tokenizer = w3;
  g2.Hooks = P2;
  g2.parse = g2;
  var Ut = g2.options;
  var Kt = g2.setOptions;
  var Wt = g2.use;
  var Xt = g2.walkTokens;
  var Jt = g2.parseInline;
  var Yt = b.parse;
  var en = x2.lex;

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
        let m4;
        const afterMatch = match.input.substring(afterMatchIndex);
        if (m4 = afterMatch.match(/^\s*=/)) {
          response.ignoreMatch();
          return;
        }
        if (m4 = afterMatch.match(/^\s+extends\s+/)) {
          if (m4.index === 0) {
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
      ].map((x3) => `${x3}\\s*\\(`)), IDENT_RE$1, regex.lookahead(/\s*\(/)),
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
        let m4;
        const afterMatch = match.input.substring(afterMatchIndex);
        if (m4 = afterMatch.match(/^\s*=/)) {
          response.ignoreMatch();
          return;
        }
        if (m4 = afterMatch.match(/^\s+extends\s+/)) {
          if (m4.index === 0) {
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
      ].map((x3) => `${x3}\\s*\\(`)), IDENT_RE$1, regex.lookahead(/\s*\(/)),
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
      const indx = mode.contains.findIndex((m4) => m4.label === label);
      if (indx === -1) {
        throw new Error("can not find mode to replace");
      }
      mode.contains.splice(indx, 1, replacement);
    };
    Object.assign(tsLanguage.keywords, KEYWORDS$1);
    tsLanguage.exports.PARAMS_CONTAINS.push(DECORATOR);
    const ATTRIBUTE_HIGHLIGHT = tsLanguage.contains.find((c3) => c3.scope === "attr");
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
    const functionDeclaration = tsLanguage.contains.find((m4) => m4.label === "func.def");
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
        keyword: reduceRelevancy(KEYWORDS3, { when: (x3) => x3.length < 3 }),
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
    ].forEach((m4) => {
      m4.contains = m4.contains.concat(CONTAINABLE);
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

  // src/markdown.ts
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
  g2.setOptions({
    highlight: function(code, lang) {
      if (lang && core_default.getLanguage(lang)) {
        return core_default.highlight(code, { language: lang }).value;
      }
      return core_default.highlightAuto(code).value;
    },
    breaks: true,
    gfm: true
  });
  function renderMarkdown(text) {
    return g2.parse(text);
  }
  // node_modules/preact/jsx-runtime/dist/jsxRuntime.module.js
  var f3 = 0;
  function u5(e3, t3, n2, o3, i3, u6) {
    t3 || (t3 = {});
    var a3, c3, p3 = t3;
    if ("ref" in p3)
      for (c3 in p3 = {}, t3)
        c3 == "ref" ? a3 = t3[c3] : p3[c3] = t3[c3];
    var l3 = { type: e3, props: p3, key: n2, ref: a3, __k: null, __: null, __b: 0, __e: null, __c: null, constructor: undefined, __v: --f3, __i: -1, __u: 0, __source: i3, __self: u6 };
    if (typeof e3 == "function" && (a3 = e3.defaultProps))
      for (c3 in a3)
        p3[c3] === undefined && (p3[c3] = a3[c3]);
    return l.vnode && l.vnode(l3), l3;
  }

  // src/components/plan/slide-approve.tsx
  function SlideApprove({ label, cmd, warn }) {
    const trackRef = A2(null);
    const thumbRef = A2(null);
    const labelRef = A2(null);
    y2(() => {
      const track = trackRef.current;
      const thumb = thumbRef.current;
      if (!track || !thumb)
        return;
      const ac = new AbortController;
      const signal = ac.signal;
      let dragging = false;
      let startX = 0;
      let thumbStartLeft = 0;
      function getMaxLeft() {
        return track.offsetWidth - thumb.offsetWidth - 6;
      }
      function markApproved() {
        track.classList.add("approved");
        if (labelRef.current)
          labelRef.current.textContent = "Approved!";
        thumb.classList.add("hidden");
      }
      function onStart(e3) {
        if (track.classList.contains("approved"))
          return;
        dragging = true;
        thumb.classList.add("dragging");
        const clientX = "touches" in e3 ? e3.touches[0].clientX : e3.clientX;
        startX = clientX;
        thumbStartLeft = thumb.offsetLeft - 3;
        e3.preventDefault();
      }
      function onMove(e3) {
        if (!dragging)
          return;
        const clientX = "touches" in e3 ? e3.touches[0].clientX : e3.clientX;
        const dx = clientX - startX;
        const newLeft = Math.max(0, Math.min(thumbStartLeft + dx, getMaxLeft()));
        thumb.style.left = newLeft + 3 + "px";
        e3.preventDefault();
      }
      function onEnd() {
        if (!dragging)
          return;
        dragging = false;
        thumb.classList.remove("dragging");
        const currentLeft = thumb.offsetLeft - 3;
        const maxLeft = getMaxLeft();
        if (currentLeft >= maxLeft * 0.8) {
          markApproved();
          sendCommand(cmd);
        } else {
          thumb.style.left = "3px";
        }
      }
      thumb.addEventListener("touchstart", onStart, {
        passive: false,
        signal
      });
      thumb.addEventListener("mousedown", onStart, { signal });
      document.addEventListener("touchmove", onMove, {
        passive: false,
        signal
      });
      document.addEventListener("mousemove", onMove, { signal });
      document.addEventListener("touchend", onEnd, { signal });
      document.addEventListener("mouseup", onEnd, { signal });
      return () => ac.abort();
    }, [cmd]);
    const borderStyle = warn ? { borderColor: "var(--warn, #ff9800)" } : undefined;
    const thumbStyle = warn ? { background: "var(--warn, #ff9800)" } : undefined;
    return /* @__PURE__ */ u5("div", {
      class: "slide-approve-wrap",
      children: /* @__PURE__ */ u5("div", {
        class: "slide-approve-track glass glass-interactive",
        ref: trackRef,
        style: borderStyle,
        children: [
          /* @__PURE__ */ u5("div", {
            class: "slide-approve-thumb",
            ref: thumbRef,
            style: thumbStyle,
            children: /* @__PURE__ */ u5("svg", {
              viewBox: "0 0 24 24",
              children: /* @__PURE__ */ u5("path", {
                d: "M5 12h14m-6-6 6 6-6 6",
                stroke: "currentColor",
                "stroke-width": "2.5",
                "stroke-linecap": "round",
                "stroke-linejoin": "round",
                fill: "none"
              }, undefined, false, undefined, this)
            }, undefined, false, undefined, this)
          }, undefined, false, undefined, this),
          /* @__PURE__ */ u5("div", {
            class: "slide-approve-label",
            ref: labelRef,
            children: label
          }, undefined, false, undefined, this)
        ]
      }, undefined, true, undefined, this)
    }, undefined, false, undefined, this);
  }

  // src/components/plan/orch-canvas.tsx
  var BOB = [0, -1, -2, -1];
  var FRAME_MS = {
    idle: 450,
    waiting: 650,
    toolcall: 90,
    talking: 280,
    entering: 220,
    exiting: 220
  };
  var WALK = 55;
  var conductor;
  var secretary;
  var heartbeat;
  var subagents;
  var slots = {};
  var freeSlots = [];
  var inited = false;
  var orchWs = null;
  var orchReconnectTimer = null;
  var lastTs = null;
  function makeChar(id, emoji, home) {
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
  function initChars() {
    const MAP = window.MAP_POSITIONS;
    conductor = makeChar("conductor", "\uD83D\uDC51", MAP.conductor);
    secretary = makeChar("secretary", "\uD83D\uDC69‍\uD83D\uDCBC", MAP.secretary);
    heartbeat = makeChar("heartbeat", "\uD83D\uDD4A️", MAP.heartbeat || { x: 230, y: 58 });
    conductor.alive = true;
    secretary.alive = false;
    heartbeat.alive = true;
    conductor.statusText = null;
    heartbeat.facing = 1;
    heartbeat.flipTimer = 0;
    const ps = [
      { id: "s0", emoji: "\uD83D\uDD0D" },
      { id: "s1", emoji: "\uD83D\uDCCA" },
      { id: "s2", emoji: "\uD83D\uDCBB" },
      { id: "s3", emoji: "\uD83D\uDD27" },
      { id: "s4", emoji: "\uD83C\uDFAF" }
    ];
    subagents = ps.map((p3, i3) => {
      const c3 = makeChar(p3.id, p3.emoji, MAP.stations[i3]);
      c3.x = MAP.door.x;
      c3.y = MAP.door.y;
      return c3;
    });
    slots = {};
    freeSlots = subagents.slice();
  }
  function allChars() {
    return [conductor, secretary, heartbeat, ...subagents];
  }
  function syncBadge(id, state, alive) {
    const el = document.getElementById("orch-badge-" + id);
    if (!el)
      return;
    el.className = "orch-badge" + (alive ? " alive" : "") + (state === "talking" ? " talking" : "") + (state === "toolcall" ? " toolcall" : "") + (state === "waiting" ? " waiting" : "");
  }
  function setState(c3, state) {
    c3.state = state;
    syncBadge(c3.id, state, c3.alive);
    if (c3 === conductor) {
      if (state === "waiting")
        c3.statusText = "\uD83E\uDD14";
      else if (state === "toolcall")
        c3.statusText = "⌨";
      else if (state === "user_waiting")
        c3.statusText = "⏳";
      else if (state === "plan_interviewing")
        c3.statusText = "\uD83D\uDCCB";
      else if (state === "plan_review")
        c3.statusText = "\uD83D\uDD0D";
      else if (state === "plan_executing")
        c3.statusText = "▶️";
      else if (state === "plan_completed")
        c3.statusText = "✅";
      else
        c3.statusText = null;
      const inPlan = state.indexOf("plan_") === 0;
      if (secretary.alive !== inPlan) {
        secretary.alive = inPlan;
        syncBadge("secretary", secretary.state, secretary.alive);
      }
    }
  }
  function moveTo(c3, pos, cb) {
    c3.target = pos;
    c3._onArrive = cb || null;
  }
  function say(c3, text, ttl = 2200) {
    c3.bubble = { text, ttl };
  }
  function charForId(id) {
    if (id === "heartbeat")
      return heartbeat;
    if (slots[id])
      return slots[id];
    return conductor;
  }
  function spawn(id) {
    if (/^subagent-/.test(id)) {
      const c3 = freeSlots.shift();
      if (!c3)
        return;
      slots[id] = c3;
      c3.alive = true;
      c3.x = window.MAP_POSITIONS.door.x;
      c3.y = window.MAP_POSITIONS.door.y;
      setState(c3, "entering");
      moveTo(c3, c3.home, () => setState(c3, "idle"));
    } else {
      const ch = charForId(id);
      ch.alive = true;
      setState(ch, "waiting");
    }
  }
  function gc(id) {
    if (/^subagent-/.test(id)) {
      const c3 = slots[id];
      if (!c3)
        return;
      delete slots[id];
      freeSlots.push(c3);
      setState(c3, "exiting");
      moveTo(c3, window.MAP_POSITIONS.door, () => {
        c3.alive = false;
        setState(c3, "idle");
      });
    } else {
      const ch = charForId(id);
      if (ch === heartbeat) {
        setState(ch, "idle");
      } else if (ch === conductor) {
        setState(ch, "user_waiting");
      } else {
        ch.alive = false;
        setState(ch, "idle");
      }
    }
  }
  function converse(fromId, toId, text) {
    const from = charForId(fromId);
    const to = charForId(toId);
    if (!from || !to || from === to)
      return;
    const label = (text || "").slice(0, 18);
    const mid = { x: (from.x + to.x) / 2, y: (from.y + to.y) / 2 };
    setState(from, "talking");
    setState(to, "talking");
    moveTo(from, { x: mid.x - 18, y: mid.y }, () => say(from, label, 2400));
    moveTo(to, { x: mid.x + 18, y: mid.y }, () => {
      setTimeout(() => {
        moveTo(from, from.home, () => setState(from, "idle"));
        moveTo(to, to.home, () => setState(to, "idle"));
      }, 2600);
    });
  }
  function update(dt) {
    allChars().forEach((c3) => {
      if (!c3.alive && c3.state !== "entering")
        return;
      if (c3 === heartbeat) {
        if (c3.state === "idle") {
          c3.frame = 0;
        } else {
          c3.frameTimer += dt;
          const pDur = c3.state === "toolcall" ? 130 : 380;
          if (c3.frameTimer >= pDur) {
            c3.frame = (c3.frame + 1) % 4;
            c3.frameTimer -= pDur;
          }
        }
      } else {
        c3.frameTimer += dt;
        const dur = FRAME_MS[c3.state] || 450;
        if (c3.frameTimer >= dur) {
          c3.frame = (c3.frame + 1) % 4;
          c3.frameTimer -= dur;
        }
      }
      if (c3.target) {
        const dx = c3.target.x - c3.x;
        const dy = c3.target.y - c3.y;
        const dist = Math.sqrt(dx * dx + dy * dy);
        if (dist > 1.5) {
          const spd = WALK * dt / 1000;
          c3.x += dx / dist * spd;
          c3.y += dy / dist * spd;
        } else {
          c3.x = c3.target.x;
          c3.y = c3.target.y;
          c3.target = null;
          if (c3._onArrive) {
            c3._onArrive();
            c3._onArrive = null;
          }
        }
      }
      if (c3.bubble) {
        c3.bubble.ttl -= dt;
        if (c3.bubble.ttl <= 0)
          c3.bubble = null;
      }
      if (c3 === heartbeat) {
        if (c3.target) {
          const pdx = c3.target.x - c3.x;
          if (Math.abs(pdx) > 1)
            c3.facing = pdx > 0 ? 1 : -1;
        } else {
          const flipRate = c3.state === "toolcall" ? 280 : c3.state === "waiting" ? 600 : 2800;
          c3.flipTimer = (c3.flipTimer || 0) + dt;
          if (c3.flipTimer >= flipRate) {
            c3.flipTimer -= flipRate;
            c3.facing = -(c3.facing || 1);
          }
        }
      }
    });
  }
  function drawStatus(ctx, c3) {
    if (!c3.statusText)
      return;
    const yOff = BOB[c3.frame];
    const cx = Math.floor(c3.x);
    const cy = Math.floor(c3.y + yOff) - 20;
    ctx.font = "11px serif";
    ctx.textAlign = "center";
    ctx.textBaseline = "middle";
    ctx.fillText(c3.statusText, cx, cy);
  }
  function drawBubble(ctx, c3) {
    if (!c3.bubble)
      return;
    const yOff = BOB[c3.frame];
    const bx = c3.x;
    const by = c3.y + yOff - 18;
    ctx.font = "7px Silkscreen,monospace";
    const tw = ctx.measureText(c3.bubble.text).width;
    const pw = tw + 8;
    const ph = 12;
    const lx = Math.max(4, Math.min(316 - pw, bx - pw / 2));
    ctx.fillStyle = "#facc15";
    ctx.fillRect(Math.floor(lx), Math.floor(by - ph), Math.ceil(pw), Math.ceil(ph));
    ctx.fillRect(Math.floor(bx) - 1, Math.floor(by), 3, 3);
    ctx.fillStyle = "#0a0a00";
    ctx.textAlign = "left";
    ctx.textBaseline = "middle";
    ctx.fillText(c3.bubble.text, Math.floor(lx + 4), Math.floor(by - ph / 2));
  }
  function drawChar(ctx, c3) {
    if (!c3.alive && c3.state !== "entering" && c3.state !== "exiting")
      return;
    const yOff = BOB[c3.frame];
    const cx = Math.floor(c3.x);
    const cy = Math.floor(c3.y + yOff);
    if (c3.state === "toolcall") {
      ctx.fillStyle = "rgba(251,146,60,0.35)";
      ctx.beginPath();
      ctx.arc(cx, cy, 13, 0, Math.PI * 2);
      ctx.fill();
    } else if (c3.state === "waiting") {
      ctx.fillStyle = "rgba(96,165,250,0.25)";
      ctx.beginPath();
      ctx.arc(cx, cy, 11, 0, Math.PI * 2);
      ctx.fill();
    } else if (c3.state === "user_waiting" || c3.state === "plan_review") {
      ctx.fillStyle = "rgba(167,139,250,0.18)";
      ctx.beginPath();
      ctx.arc(cx, cy, 10, 0, Math.PI * 2);
      ctx.fill();
    } else if (c3.state === "plan_executing") {
      ctx.fillStyle = "rgba(74,222,128,0.18)";
      ctx.beginPath();
      ctx.arc(cx, cy, 10, 0, Math.PI * 2);
      ctx.fill();
    }
    ctx.font = "18px serif";
    ctx.textAlign = "center";
    ctx.textBaseline = "middle";
    if (c3.facing === -1) {
      ctx.save();
      ctx.translate(cx, cy);
      ctx.scale(-1, 1);
      ctx.fillText(c3.emoji, 0, 0);
      ctx.restore();
    } else {
      ctx.fillText(c3.emoji, cx, cy);
    }
    ctx.font = "6px Silkscreen,monospace";
    ctx.textAlign = "center";
    ctx.textBaseline = "top";
    ctx.fillStyle = c3.state === "talking" ? "#facc15" : "#3a4a7a";
    ctx.fillText(c3.id.toUpperCase(), cx, cy + 11);
    drawStatus(ctx, c3);
    drawBubble(ctx, c3);
  }
  function OrchCanvas({ active }) {
    const canvasRef = A2(null);
    const animRef = A2(0);
    y2(() => {
      if (!active) {
        if (orchReconnectTimer) {
          clearTimeout(orchReconnectTimer);
          orchReconnectTimer = null;
        }
        if (orchWs) {
          orchWs.close();
          orchWs = null;
        }
        if (animRef.current) {
          cancelAnimationFrame(animRef.current);
          animRef.current = 0;
        }
        return;
      }
      const canvas = canvasRef.current;
      if (!canvas)
        return;
      const ctx = canvas.getContext("2d");
      if (!ctx)
        return;
      if (!inited) {
        inited = true;
        initChars();
      }
      lastTs = null;
      function renderLoop(ts) {
        if (lastTs === null)
          lastTs = ts;
        const dt = Math.min(ts - lastTs, 80);
        lastTs = ts;
        update(dt);
        ctx.imageSmoothingEnabled = false;
        window.drawMap(ctx);
        allChars().forEach((c3) => drawChar(ctx, c3));
        animRef.current = requestAnimationFrame(renderLoop);
      }
      window.loadMapAsset(() => {
        lastTs = null;
        animRef.current = requestAnimationFrame(renderLoop);
      });
      connectOrchWs();
      return () => {
        if (animRef.current) {
          cancelAnimationFrame(animRef.current);
          animRef.current = 0;
        }
      };
    }, [active]);
    return /* @__PURE__ */ u5("div", {
      class: "card glass",
      style: { marginTop: "16px", padding: "0" },
      children: [
        /* @__PURE__ */ u5("div", {
          class: "orch-room-row",
          children: [
            /* @__PURE__ */ u5("div", {
              class: "orch-side",
              id: "orch-panel-left",
              children: [
                /* @__PURE__ */ u5("div", {
                  class: "orch-badge alive",
                  id: "orch-badge-conductor",
                  children: [
                    /* @__PURE__ */ u5("div", {
                      class: "orch-badge-emoji",
                      children: "\uD83D\uDC51"
                    }, undefined, false, undefined, this),
                    /* @__PURE__ */ u5("div", {
                      class: "orch-badge-label",
                      children: "CNDR"
                    }, undefined, false, undefined, this),
                    /* @__PURE__ */ u5("div", {
                      class: "orch-badge-dot"
                    }, undefined, false, undefined, this)
                  ]
                }, undefined, true, undefined, this),
                /* @__PURE__ */ u5("div", {
                  class: "orch-badge",
                  id: "orch-badge-secretary",
                  children: [
                    /* @__PURE__ */ u5("div", {
                      class: "orch-badge-emoji",
                      children: "\uD83D\uDC69‍\uD83D\uDCBC"
                    }, undefined, false, undefined, this),
                    /* @__PURE__ */ u5("div", {
                      class: "orch-badge-label",
                      children: "SEC"
                    }, undefined, false, undefined, this),
                    /* @__PURE__ */ u5("div", {
                      class: "orch-badge-dot"
                    }, undefined, false, undefined, this)
                  ]
                }, undefined, true, undefined, this),
                /* @__PURE__ */ u5("div", {
                  class: "orch-badge alive",
                  id: "orch-badge-heartbeat",
                  children: [
                    /* @__PURE__ */ u5("div", {
                      class: "orch-badge-emoji",
                      children: "\uD83D\uDD4A️"
                    }, undefined, false, undefined, this),
                    /* @__PURE__ */ u5("div", {
                      class: "orch-badge-label",
                      children: "HB"
                    }, undefined, false, undefined, this),
                    /* @__PURE__ */ u5("div", {
                      class: "orch-badge-dot"
                    }, undefined, false, undefined, this)
                  ]
                }, undefined, true, undefined, this)
              ]
            }, undefined, true, undefined, this),
            /* @__PURE__ */ u5("div", {
              class: "orch-canvas-wrap",
              children: /* @__PURE__ */ u5("canvas", {
                ref: canvasRef,
                id: "orch-canvas",
                width: "320",
                height: "320"
              }, undefined, false, undefined, this)
            }, undefined, false, undefined, this),
            /* @__PURE__ */ u5("div", {
              class: "orch-side",
              id: "orch-panel-right",
              children: [
                { id: "s0", emoji: "\uD83D\uDD0D", label: "SCOUT" },
                { id: "s1", emoji: "\uD83D\uDCCA", label: "ANLY" },
                { id: "s2", emoji: "\uD83D\uDCBB", label: "CODE" },
                { id: "s3", emoji: "\uD83D\uDD27", label: "WRKR" },
                { id: "s4", emoji: "\uD83C\uDFAF", label: "CORD" }
              ].map((s3) => /* @__PURE__ */ u5("div", {
                class: "orch-badge",
                id: `orch-badge-${s3.id}`,
                children: [
                  /* @__PURE__ */ u5("div", {
                    class: "orch-badge-emoji",
                    children: s3.emoji
                  }, undefined, false, undefined, this),
                  /* @__PURE__ */ u5("div", {
                    class: "orch-badge-label",
                    children: s3.label
                  }, undefined, false, undefined, this),
                  /* @__PURE__ */ u5("div", {
                    class: "orch-badge-dot"
                  }, undefined, false, undefined, this)
                ]
              }, s3.id, true, undefined, this))
            }, undefined, false, undefined, this)
          ]
        }, undefined, true, undefined, this),
        /* @__PURE__ */ u5("div", {
          class: "orch-status",
          children: [
            /* @__PURE__ */ u5("span", {
              class: "orch-dot",
              id: "orch-status-dot"
            }, undefined, false, undefined, this),
            /* @__PURE__ */ u5("span", {
              id: "orch-status-text",
              children: "Connecting..."
            }, undefined, false, undefined, this)
          ]
        }, undefined, true, undefined, this)
      ]
    }, undefined, true, undefined, this);
  }
  function connectOrchWs() {
    if (orchWs && orchWs.readyState <= 1)
      return;
    const initData = window.Telegram?.WebApp?.initData || "";
    const proto = location.protocol === "https:" ? "wss:" : "ws:";
    const url = proto + "//" + location.host + "/miniapp/api/orchestration/ws?initData=" + encodeURIComponent(initData);
    orchWs = new WebSocket(url);
    orchWs.onopen = () => {
      const dot = document.getElementById("orch-status-dot");
      const txt = document.getElementById("orch-status-text");
      if (dot)
        dot.classList.add("on");
      if (txt)
        txt.textContent = "Live";
    };
    orchWs.onmessage = (e3) => {
      let msg;
      try {
        msg = JSON.parse(e3.data);
      } catch {
        return;
      }
      if (msg.type === "init") {
        (msg.agents || []).forEach((info) => {
          spawn(info.id);
          if (info.state && info.state !== "idle") {
            const c3 = charForId(info.id);
            if (c3)
              setState(c3, info.state);
          }
        });
      } else if (msg.type === "event") {
        const ev = msg.event || {};
        if (ev.type === "agent_spawn")
          spawn(ev.id);
        if (ev.type === "agent_state") {
          const c3 = charForId(ev.id);
          if (c3)
            setState(c3, ev.state);
        }
        if (ev.type === "agent_gc")
          gc(ev.id);
        if (ev.type === "conversation")
          converse(ev.from, ev.to, ev.text);
      }
    };
    orchWs.onclose = () => {
      const dot = document.getElementById("orch-status-dot");
      const txt = document.getElementById("orch-status-text");
      if (dot)
        dot.classList.remove("on");
      if (txt)
        txt.textContent = "Disconnected";
      orchWs = null;
    };
    orchWs.onerror = () => {};
  }

  // src/components/plan/plan-tab.tsx
  function PlanTab({ active, sse }) {
    const [plan, setPlan] = d2(null);
    const [loading, setLoading] = d2(true);
    const [error, setError] = d2(false);
    const loadPlan = q2(async () => {
      setLoading(true);
      setError(false);
      try {
        const data = await apiFetch("/miniapp/api/plan");
        setPlan(data);
      } catch {
        setError(true);
      } finally {
        setLoading(false);
      }
    }, []);
    y2(() => {
      if (sse.plan) {
        setPlan(sse.plan);
        setLoading(false);
      }
    }, [sse.plan]);
    y2(() => {
      if (active && !isFresh(sse.lastUpdate, "plan")) {
        loadPlan();
      }
    }, [active]);
    y2(() => {
      loadPlan();
    }, []);
    if (loading && !plan)
      return /* @__PURE__ */ u5("div", {
        class: "loading",
        children: "Loading plan..."
      }, undefined, false, undefined, this);
    if (error && !plan)
      return /* @__PURE__ */ u5("div", {
        class: "loading",
        children: "Failed to load plan."
      }, undefined, false, undefined, this);
    if (!plan)
      return null;
    return /* @__PURE__ */ u5(k, {
      children: [
        /* @__PURE__ */ u5(PlanContent, {
          plan
        }, undefined, false, undefined, this),
        window.ORCH_ENABLED && /* @__PURE__ */ u5(OrchCanvas, {
          active
        }, undefined, false, undefined, this)
      ]
    }, undefined, true, undefined, this);
  }
  function PlanContent({ plan }) {
    if (!plan.has_plan) {
      return /* @__PURE__ */ u5(NoPlan, {}, undefined, false, undefined, this);
    }
    const isInterviewOrReview = plan.status === "interviewing" || plan.status === "review";
    return /* @__PURE__ */ u5(k, {
      children: [
        /* @__PURE__ */ u5("div", {
          class: "card glass",
          children: [
            /* @__PURE__ */ u5("div", {
              class: "card-title",
              children: "Status"
            }, undefined, false, undefined, this),
            /* @__PURE__ */ u5("div", {
              class: "card-value",
              children: plan.status
            }, undefined, false, undefined, this),
            /* @__PURE__ */ u5("div", {
              style: { color: "var(--hint)", marginTop: "4px" },
              children: [
                "Phase ",
                plan.current_phase,
                " / ",
                plan.total_phases
              ]
            }, undefined, true, undefined, this)
          ]
        }, undefined, true, undefined, this),
        isInterviewOrReview && plan.memory && /* @__PURE__ */ u5("div", {
          class: "memory-view glass",
          dangerouslySetInnerHTML: { __html: renderMarkdown(plan.memory) }
        }, undefined, false, undefined, this),
        plan.status === "review" && /* @__PURE__ */ u5(k, {
          children: [
            /* @__PURE__ */ u5(SlideApprove, {
              label: "Slide to Approve",
              cmd: "/plan start"
            }, undefined, false, undefined, this),
            /* @__PURE__ */ u5(SlideApprove, {
              label: "Approve & Clear History",
              cmd: "/plan start clear",
              warn: true
            }, undefined, false, undefined, this)
          ]
        }, undefined, true, undefined, this),
        !isInterviewOrReview && plan.phases && plan.phases.length > 0 && /* @__PURE__ */ u5(Phases, {
          phases: plan.phases,
          currentPhase: plan.current_phase
        }, undefined, false, undefined, this)
      ]
    }, undefined, true, undefined, this);
  }
  function NoPlan() {
    const [task, setTask] = d2("");
    const handleStart = async () => {
      const t3 = task.trim();
      if (!t3)
        return;
      const ok = await sendCommand("/plan " + t3);
      if (ok)
        setTask("");
    };
    return /* @__PURE__ */ u5(k, {
      children: [
        /* @__PURE__ */ u5("div", {
          class: "empty-state",
          children: "No active plan."
        }, undefined, false, undefined, this),
        /* @__PURE__ */ u5("div", {
          class: "card glass",
          style: { marginTop: "16px" },
          children: [
            /* @__PURE__ */ u5("div", {
              class: "card-title",
              children: "Start a Plan"
            }, undefined, false, undefined, this),
            /* @__PURE__ */ u5("div", {
              style: { display: "flex", gap: "8px", marginTop: "8px" },
              children: [
                /* @__PURE__ */ u5("input", {
                  class: "send-input glass glass-interactive",
                  placeholder: "Describe your task...",
                  value: task,
                  onInput: (e3) => setTask(e3.target.value),
                  onKeyDown: (e3) => e3.key === "Enter" && handleStart()
                }, undefined, false, undefined, this),
                /* @__PURE__ */ u5("button", {
                  class: "send-btn",
                  onClick: handleStart,
                  children: "Start"
                }, undefined, false, undefined, this)
              ]
            }, undefined, true, undefined, this)
          ]
        }, undefined, true, undefined, this)
      ]
    }, undefined, true, undefined, this);
  }
  function Phases({
    phases,
    currentPhase
  }) {
    const onStepClick = (phaseNum, stepIdx, done) => {
      if (done)
        return;
      sendCommand("/plan done " + stepIdx);
    };
    return /* @__PURE__ */ u5(k, {
      children: phases.map((phase) => {
        const doneCount = phase.steps.filter((s3) => s3.done).length;
        const total = phase.steps.length;
        let indicatorClass;
        let indicator;
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
        return /* @__PURE__ */ u5("div", {
          class: "phase",
          children: [
            /* @__PURE__ */ u5("div", {
              class: "phase-header",
              children: [
                /* @__PURE__ */ u5("div", {
                  class: `phase-indicator ${indicatorClass}`,
                  children: indicator
                }, undefined, false, undefined, this),
                /* @__PURE__ */ u5("span", {
                  class: "phase-title",
                  children: phase.title || "Phase " + phase.number
                }, undefined, false, undefined, this),
                total > 0 && /* @__PURE__ */ u5("span", {
                  class: "phase-progress",
                  children: [
                    doneCount,
                    "/",
                    total
                  ]
                }, undefined, true, undefined, this)
              ]
            }, undefined, true, undefined, this),
            phase.steps.map((step) => /* @__PURE__ */ u5("div", {
              class: `step${step.done ? " step-done" : ""}`,
              onClick: () => onStepClick(phase.number, step.index, step.done),
              children: [
                /* @__PURE__ */ u5("div", {
                  class: `step-check${step.done ? " done" : ""}`
                }, undefined, false, undefined, this),
                /* @__PURE__ */ u5("div", {
                  class: `step-text${step.done ? " done" : ""}`,
                  children: step.description
                }, undefined, false, undefined, this)
              ]
            }, step.index, true, undefined, this))
          ]
        }, phase.number, true, undefined, this);
      })
    }, undefined, false, undefined, this);
  }

  // src/components/work/git-section.tsx
  function GitSection({ repos, worktrees, onReload }) {
    const [detailRepo, setDetailRepo] = d2(null);
    const [detailName, setDetailName] = d2(null);
    const [loadingDetail, setLoadingDetail] = d2(false);
    const loadDetail = q2(async (name) => {
      setDetailName(name);
      setLoadingDetail(true);
      try {
        const data = await apiFetch("/miniapp/api/git?repo=" + encodeURIComponent(name));
        setDetailRepo(data);
      } catch {}
      setLoadingDetail(false);
    }, []);
    const goBack = q2(() => {
      setDetailRepo(null);
      setDetailName(null);
      onReload();
    }, [onReload]);
    if (detailName) {
      return /* @__PURE__ */ u5(RepoDetail, {
        repo: detailRepo,
        name: detailName,
        loading: loadingDetail,
        onBack: goBack
      }, undefined, false, undefined, this);
    }
    return /* @__PURE__ */ u5(k, {
      children: [
        /* @__PURE__ */ u5(Worktrees, {
          items: worktrees,
          onReload
        }, undefined, false, undefined, this),
        /* @__PURE__ */ u5(RepoList, {
          repos,
          onSelect: loadDetail
        }, undefined, false, undefined, this)
      ]
    }, undefined, true, undefined, this);
  }
  function Worktrees({
    items,
    onReload
  }) {
    const [busy, setBusy] = d2(null);
    const handleAction = async (action, name, isDirty) => {
      let force = false;
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
      setBusy(name + ":" + action);
      try {
        await apiPost("/miniapp/api/worktrees", { action, name, force });
        onReload();
      } catch (err) {
        alert(err.message || "Action failed");
      }
      setBusy(null);
    };
    return /* @__PURE__ */ u5("div", {
      class: "card glass",
      children: [
        /* @__PURE__ */ u5("div", {
          class: "card-title",
          children: "Worktrees"
        }, undefined, false, undefined, this),
        items.length === 0 ? /* @__PURE__ */ u5("div", {
          class: "empty-state",
          style: { padding: "12px 0 4px" },
          children: "No active worktrees."
        }, undefined, false, undefined, this) : /* @__PURE__ */ u5("div", {
          class: "worktree-list",
          children: items.map((wt) => {
            let last = "(no commits)";
            if (wt.last_commit_hash) {
              last = wt.last_commit_hash + " " + (wt.last_commit_subject || "");
              if (wt.last_commit_age)
                last += " (" + wt.last_commit_age + ")";
            }
            return /* @__PURE__ */ u5("div", {
              class: `worktree-item${wt.has_uncommitted ? " dirty" : ""}`,
              children: [
                /* @__PURE__ */ u5("div", {
                  class: "worktree-main",
                  children: [
                    /* @__PURE__ */ u5("div", {
                      class: "worktree-name-row",
                      children: [
                        /* @__PURE__ */ u5("span", {
                          class: "worktree-name",
                          children: wt.name
                        }, undefined, false, undefined, this),
                        wt.has_uncommitted ? /* @__PURE__ */ u5("span", {
                          class: "worktree-dirty",
                          children: "DIRTY"
                        }, undefined, false, undefined, this) : /* @__PURE__ */ u5("span", {
                          class: "worktree-clean",
                          children: "CLEAN"
                        }, undefined, false, undefined, this)
                      ]
                    }, undefined, true, undefined, this),
                    /* @__PURE__ */ u5("div", {
                      class: "worktree-branch",
                      children: wt.branch || "?"
                    }, undefined, false, undefined, this),
                    /* @__PURE__ */ u5("div", {
                      class: "worktree-last",
                      children: last
                    }, undefined, false, undefined, this)
                  ]
                }, undefined, true, undefined, this),
                /* @__PURE__ */ u5("div", {
                  class: "worktree-actions",
                  children: [
                    /* @__PURE__ */ u5("button", {
                      class: "worktree-btn merge",
                      disabled: busy === wt.name + ":merge",
                      onClick: () => handleAction("merge", wt.name, wt.has_uncommitted),
                      children: busy === wt.name + ":merge" ? "Merging..." : "Merge"
                    }, undefined, false, undefined, this),
                    /* @__PURE__ */ u5("button", {
                      class: "worktree-btn dispose",
                      disabled: busy === wt.name + ":dispose",
                      onClick: () => handleAction("dispose", wt.name, wt.has_uncommitted),
                      children: busy === wt.name + ":dispose" ? "Disposing..." : "Dispose"
                    }, undefined, false, undefined, this)
                  ]
                }, undefined, true, undefined, this)
              ]
            }, wt.name, true, undefined, this);
          })
        }, undefined, false, undefined, this)
      ]
    }, undefined, true, undefined, this);
  }
  function RepoList({
    repos,
    onSelect
  }) {
    if (!repos || repos.length === 0) {
      return /* @__PURE__ */ u5("div", {
        class: "empty-state",
        style: { marginTop: "12px" },
        children: "No git repositories found."
      }, undefined, false, undefined, this);
    }
    return /* @__PURE__ */ u5(k, {
      children: [
        /* @__PURE__ */ u5("div", {
          style: {
            padding: "10px 4px 8px",
            fontSize: "12px",
            color: "var(--hint)"
          },
          children: "Repositories"
        }, undefined, false, undefined, this),
        repos.map((r3) => /* @__PURE__ */ u5("div", {
          class: "git-repo-item glass glass-interactive",
          onClick: () => onSelect(r3.name),
          children: [
            /* @__PURE__ */ u5("div", {
              class: "git-repo-body",
              children: [
                /* @__PURE__ */ u5("div", {
                  class: "git-repo-name",
                  children: r3.name
                }, undefined, false, undefined, this),
                /* @__PURE__ */ u5("div", {
                  class: "git-repo-branch",
                  children: r3.branch || "?"
                }, undefined, false, undefined, this)
              ]
            }, undefined, true, undefined, this),
            /* @__PURE__ */ u5("span", {
              class: "git-repo-arrow",
              children: "›"
            }, undefined, false, undefined, this)
          ]
        }, r3.name, true, undefined, this))
      ]
    }, undefined, true, undefined, this);
  }
  function RepoDetail({
    repo,
    name,
    loading,
    onBack
  }) {
    if (loading || !repo) {
      return /* @__PURE__ */ u5(k, {
        children: [
          /* @__PURE__ */ u5("button", {
            class: "git-back-btn",
            onClick: onBack,
            children: [
              "←",
              " ",
              name
            ]
          }, undefined, true, undefined, this),
          /* @__PURE__ */ u5("div", {
            class: "loading",
            children: [
              "Loading ",
              name,
              "..."
            ]
          }, undefined, true, undefined, this)
        ]
      }, undefined, true, undefined, this);
    }
    return /* @__PURE__ */ u5(k, {
      children: [
        /* @__PURE__ */ u5("button", {
          class: "git-back-btn",
          onClick: onBack,
          children: [
            "←",
            " ",
            repo.name || name
          ]
        }, undefined, true, undefined, this),
        /* @__PURE__ */ u5("div", {
          class: "card glass",
          children: [
            /* @__PURE__ */ u5("div", {
              class: "card-title",
              children: [
                repo.name,
                " — ",
                repo.branch || "?"
              ]
            }, undefined, true, undefined, this),
            repo.modified && repo.modified.length > 0 && /* @__PURE__ */ u5(k, {
              children: [
                /* @__PURE__ */ u5("div", {
                  style: {
                    padding: "4px 12px 8px",
                    fontSize: "12px",
                    color: "var(--hint)"
                  },
                  children: [
                    "Changes (",
                    repo.modified.length,
                    ")"
                  ]
                }, undefined, true, undefined, this),
                repo.modified.map((f4, i3) => /* @__PURE__ */ u5("div", {
                  class: "git-commit",
                  children: [
                    /* @__PURE__ */ u5("span", {
                      class: `git-status git-status-${f4.status === "??" ? "u" : f4.status.toLowerCase()}`,
                      children: f4.status
                    }, undefined, false, undefined, this),
                    /* @__PURE__ */ u5("span", {
                      class: "git-subject",
                      children: f4.path
                    }, undefined, false, undefined, this)
                  ]
                }, i3, true, undefined, this))
              ]
            }, undefined, true, undefined, this),
            repo.commits && repo.commits.length > 0 ? /* @__PURE__ */ u5(k, {
              children: [
                /* @__PURE__ */ u5("div", {
                  style: {
                    padding: "4px 12px 8px",
                    fontSize: "12px",
                    color: "var(--hint)"
                  },
                  children: "Commits"
                }, undefined, false, undefined, this),
                repo.commits.map((c3, i3) => /* @__PURE__ */ u5("div", {
                  class: "git-commit",
                  children: [
                    /* @__PURE__ */ u5("span", {
                      class: "git-hash",
                      children: c3.hash
                    }, undefined, false, undefined, this),
                    /* @__PURE__ */ u5("span", {
                      class: "git-subject",
                      children: c3.subject
                    }, undefined, false, undefined, this),
                    /* @__PURE__ */ u5("span", {
                      class: "git-meta",
                      children: c3.date
                    }, undefined, false, undefined, this)
                  ]
                }, i3, true, undefined, this))
              ]
            }, undefined, true, undefined, this) : /* @__PURE__ */ u5("div", {
              style: { padding: "12px", color: "var(--hint)" },
              children: "No commits found."
            }, undefined, false, undefined, this)
          ]
        }, undefined, true, undefined, this)
      ]
    }, undefined, true, undefined, this);
  }

  // src/components/work/session-section.tsx
  function SessionSection({
    sessions,
    stats,
    graph,
    context
  }) {
    return /* @__PURE__ */ u5(k, {
      children: [
        /* @__PURE__ */ u5(ActiveSessions, {
          sessions
        }, undefined, false, undefined, this),
        /* @__PURE__ */ u5(StatsCards, {
          stats
        }, undefined, false, undefined, this),
        context && /* @__PURE__ */ u5(ContextCard, {
          context
        }, undefined, false, undefined, this),
        graph && /* @__PURE__ */ u5(SessionGraph, {
          graph
        }, undefined, false, undefined, this)
      ]
    }, undefined, true, undefined, this);
  }
  function ActiveSessions({ sessions }) {
    if (!sessions || sessions.length === 0) {
      return /* @__PURE__ */ u5("div", {
        class: "card glass",
        children: [
          /* @__PURE__ */ u5("div", {
            class: "card-title",
            children: "Active Sessions"
          }, undefined, false, undefined, this),
          /* @__PURE__ */ u5("div", {
            style: { color: "var(--hint)", fontSize: "13px" },
            children: "No active sessions"
          }, undefined, false, undefined, this)
        ]
      }, undefined, true, undefined, this);
    }
    return /* @__PURE__ */ u5("div", {
      class: "card glass",
      children: [
        /* @__PURE__ */ u5("div", {
          class: "card-title",
          children: "Active Sessions"
        }, undefined, false, undefined, this),
        sessions.map((s3) => {
          const label = formatSessionLabel(s3.session_key);
          const isHeartbeat = s3.session_key.startsWith("heartbeat:");
          const latestMsg = s3.latest_message || null;
          return /* @__PURE__ */ u5("div", {
            style: {
              padding: "8px 0",
              borderBottom: "1px solid var(--secondary-bg)"
            },
            children: [
              /* @__PURE__ */ u5("div", {
                style: {
                  display: "flex",
                  alignItems: "center",
                  gap: "6px"
                },
                children: [
                  /* @__PURE__ */ u5("span", {
                    style: {
                      color: isHeartbeat ? "var(--link)" : "var(--done)",
                      fontSize: "10px"
                    },
                    children: isHeartbeat ? "\uD83E\uDD16" : "●"
                  }, undefined, false, undefined, this),
                  /* @__PURE__ */ u5("span", {
                    style: { fontWeight: 600, fontSize: "14px", flex: 1 },
                    children: label
                  }, undefined, false, undefined, this),
                  /* @__PURE__ */ u5("span", {
                    style: {
                      color: "var(--hint)",
                      fontSize: "12px",
                      flexShrink: 0
                    },
                    children: [
                      s3.turn_count || 0,
                      " turns"
                    ]
                  }, undefined, true, undefined, this),
                  /* @__PURE__ */ u5("span", {
                    style: {
                      marginLeft: "4px",
                      color: "var(--hint)",
                      fontSize: "12px"
                    },
                    children: formatAge(s3.age_sec)
                  }, undefined, false, undefined, this)
                ]
              }, undefined, true, undefined, this),
              latestMsg && /* @__PURE__ */ u5("div", {
                style: {
                  color: "var(--hint)",
                  fontSize: "12px",
                  paddingLeft: "22px",
                  marginTop: "2px",
                  overflow: "hidden",
                  textOverflow: "ellipsis",
                  whiteSpace: "nowrap"
                },
                children: latestMsg
              }, undefined, false, undefined, this)
            ]
          }, s3.session_key, true, undefined, this);
        })
      ]
    }, undefined, true, undefined, this);
  }
  function StatsCards({ stats }) {
    if (!stats || stats.status === "stats not enabled") {
      return /* @__PURE__ */ u5("div", {
        class: "empty-state",
        children: [
          "Stats tracking not enabled.",
          /* @__PURE__ */ u5("br", {}, undefined, false, undefined, this),
          "Start gateway with --stats flag."
        ]
      }, undefined, true, undefined, this);
    }
    const since = stats.since ? new Date(stats.since).toLocaleDateString() : "N/A";
    const today = stats.today || {};
    return /* @__PURE__ */ u5(k, {
      children: [
        /* @__PURE__ */ u5("div", {
          class: "card glass",
          children: [
            /* @__PURE__ */ u5("div", {
              class: "card-title",
              children: "Today"
            }, undefined, false, undefined, this),
            /* @__PURE__ */ u5("div", {
              class: "stat-row",
              children: [
                /* @__PURE__ */ u5("span", {
                  class: "stat-label",
                  children: "Prompts"
                }, undefined, false, undefined, this),
                /* @__PURE__ */ u5("span", {
                  class: "stat-value",
                  children: today.prompts || 0
                }, undefined, false, undefined, this)
              ]
            }, undefined, true, undefined, this),
            /* @__PURE__ */ u5("div", {
              class: "stat-row",
              children: [
                /* @__PURE__ */ u5("span", {
                  class: "stat-label",
                  children: "Requests"
                }, undefined, false, undefined, this),
                /* @__PURE__ */ u5("span", {
                  class: "stat-value",
                  children: today.requests || 0
                }, undefined, false, undefined, this)
              ]
            }, undefined, true, undefined, this),
            /* @__PURE__ */ u5("div", {
              class: "stat-row",
              children: [
                /* @__PURE__ */ u5("span", {
                  class: "stat-label",
                  children: "Tokens"
                }, undefined, false, undefined, this),
                /* @__PURE__ */ u5("span", {
                  class: "stat-value",
                  children: formatTokens(today.total_tokens || 0)
                }, undefined, false, undefined, this)
              ]
            }, undefined, true, undefined, this)
          ]
        }, undefined, true, undefined, this),
        /* @__PURE__ */ u5("div", {
          class: "card glass",
          children: [
            /* @__PURE__ */ u5("div", {
              class: "card-title",
              children: [
                "All Time (since ",
                since,
                ")"
              ]
            }, undefined, true, undefined, this),
            /* @__PURE__ */ u5("div", {
              class: "stat-row",
              children: [
                /* @__PURE__ */ u5("span", {
                  class: "stat-label",
                  children: "Prompts"
                }, undefined, false, undefined, this),
                /* @__PURE__ */ u5("span", {
                  class: "stat-value",
                  children: stats.total_prompts || 0
                }, undefined, false, undefined, this)
              ]
            }, undefined, true, undefined, this),
            /* @__PURE__ */ u5("div", {
              class: "stat-row",
              children: [
                /* @__PURE__ */ u5("span", {
                  class: "stat-label",
                  children: "Requests"
                }, undefined, false, undefined, this),
                /* @__PURE__ */ u5("span", {
                  class: "stat-value",
                  children: stats.total_requests || 0
                }, undefined, false, undefined, this)
              ]
            }, undefined, true, undefined, this),
            /* @__PURE__ */ u5("div", {
              class: "stat-row",
              children: [
                /* @__PURE__ */ u5("span", {
                  class: "stat-label",
                  children: "Total Tokens"
                }, undefined, false, undefined, this),
                /* @__PURE__ */ u5("span", {
                  class: "stat-value",
                  children: formatTokens(stats.total_tokens || 0)
                }, undefined, false, undefined, this)
              ]
            }, undefined, true, undefined, this),
            /* @__PURE__ */ u5("div", {
              class: "stat-row",
              children: [
                /* @__PURE__ */ u5("span", {
                  class: "stat-label",
                  children: "Prompt Tokens"
                }, undefined, false, undefined, this),
                /* @__PURE__ */ u5("span", {
                  class: "stat-value",
                  children: formatTokens(stats.total_prompt_tokens || 0)
                }, undefined, false, undefined, this)
              ]
            }, undefined, true, undefined, this),
            /* @__PURE__ */ u5("div", {
              class: "stat-row",
              children: [
                /* @__PURE__ */ u5("span", {
                  class: "stat-label",
                  children: "Completion Tokens"
                }, undefined, false, undefined, this),
                /* @__PURE__ */ u5("span", {
                  class: "stat-value",
                  children: formatTokens(stats.total_completion_tokens || 0)
                }, undefined, false, undefined, this)
              ]
            }, undefined, true, undefined, this)
          ]
        }, undefined, true, undefined, this)
      ]
    }, undefined, true, undefined, this);
  }
  function ContextCard({ context }) {
    const [showPrompt, setShowPrompt] = d2(false);
    const [promptText, setPromptText] = d2(null);
    const [promptLoading, setPromptLoading] = d2(false);
    const togglePrompt = async () => {
      if (showPrompt) {
        setShowPrompt(false);
        return;
      }
      setPromptLoading(true);
      try {
        const data = await apiFetch("/miniapp/api/prompt");
        setPromptText(data.prompt || "(empty)");
        setShowPrompt(true);
      } catch {}
      setPromptLoading(false);
    };
    const wd = context.work_dir || "—";
    const pwd = context.plan_work_dir || "—";
    const ws = context.workspace || "—";
    return /* @__PURE__ */ u5("div", {
      class: "card glass",
      children: [
        /* @__PURE__ */ u5("div", {
          class: "card-title",
          children: "Context"
        }, undefined, false, undefined, this),
        /* @__PURE__ */ u5("div", {
          style: { fontSize: "12px" },
          children: [
            /* @__PURE__ */ u5("div", {
              class: "stat-row",
              children: [
                /* @__PURE__ */ u5("span", {
                  class: "stat-label",
                  children: "workDir"
                }, undefined, false, undefined, this),
                /* @__PURE__ */ u5("span", {
                  class: "stat-value",
                  style: {
                    fontSize: "12px",
                    overflow: "hidden",
                    textOverflow: "ellipsis"
                  },
                  title: wd,
                  children: wd
                }, undefined, false, undefined, this)
              ]
            }, undefined, true, undefined, this),
            /* @__PURE__ */ u5("div", {
              class: "stat-row",
              children: [
                /* @__PURE__ */ u5("span", {
                  class: "stat-label",
                  children: "planWorkDir"
                }, undefined, false, undefined, this),
                /* @__PURE__ */ u5("span", {
                  class: "stat-value",
                  style: {
                    fontSize: "12px",
                    overflow: "hidden",
                    textOverflow: "ellipsis"
                  },
                  title: pwd,
                  children: pwd
                }, undefined, false, undefined, this)
              ]
            }, undefined, true, undefined, this),
            /* @__PURE__ */ u5("div", {
              class: "stat-row",
              children: [
                /* @__PURE__ */ u5("span", {
                  class: "stat-label",
                  children: "workspace"
                }, undefined, false, undefined, this),
                /* @__PURE__ */ u5("span", {
                  class: "stat-value",
                  style: {
                    fontSize: "12px",
                    overflow: "hidden",
                    textOverflow: "ellipsis"
                  },
                  title: ws,
                  children: ws
                }, undefined, false, undefined, this)
              ]
            }, undefined, true, undefined, this)
          ]
        }, undefined, true, undefined, this),
        context.bootstrap && context.bootstrap.length > 0 && /* @__PURE__ */ u5("div", {
          style: { marginTop: "8px" },
          children: context.bootstrap.map((b2, i3) => {
            const path = b2.path || "—";
            const scope = b2.scope === "global" ? "global" : "project";
            return /* @__PURE__ */ u5("div", {
              style: {
                display: "flex",
                gap: "8px",
                padding: "2px 0",
                fontSize: "12px"
              },
              children: [
                /* @__PURE__ */ u5("span", {
                  style: {
                    minWidth: "90px",
                    fontWeight: 600,
                    color: b2.path ? "var(--text)" : "var(--hint)"
                  },
                  children: b2.name
                }, undefined, false, undefined, this),
                /* @__PURE__ */ u5("span", {
                  style: {
                    color: "var(--hint)",
                    flex: 1,
                    overflow: "hidden",
                    textOverflow: "ellipsis",
                    whiteSpace: "nowrap"
                  },
                  title: path,
                  children: path
                }, undefined, false, undefined, this),
                /* @__PURE__ */ u5("span", {
                  style: { color: "var(--hint)", fontSize: "11px" },
                  children: scope
                }, undefined, false, undefined, this)
              ]
            }, i3, true, undefined, this);
          })
        }, undefined, false, undefined, this),
        /* @__PURE__ */ u5("div", {
          style: { marginTop: "8px", textAlign: "center" },
          children: /* @__PURE__ */ u5("button", {
            onClick: togglePrompt,
            style: {
              background: "var(--secondary-bg)",
              color: "var(--text)",
              border: "none",
              padding: "6px 12px",
              borderRadius: "8px",
              fontSize: "12px",
              cursor: "pointer"
            },
            children: promptLoading ? "Loading..." : showPrompt ? "Hide System Prompt" : "Show System Prompt"
          }, undefined, false, undefined, this)
        }, undefined, false, undefined, this),
        showPrompt && promptText && /* @__PURE__ */ u5("pre", {
          style: {
            marginTop: "8px",
            fontSize: "11px",
            maxHeight: "400px",
            overflow: "auto",
            background: "var(--secondary-bg)",
            padding: "8px",
            borderRadius: "6px",
            whiteSpace: "pre-wrap",
            wordBreak: "break-word"
          },
          children: promptText
        }, undefined, false, undefined, this)
      ]
    }, undefined, true, undefined, this);
  }
  function SessionGraph({ graph }) {
    if (!graph || !graph.nodes || graph.nodes.length === 0)
      return null;
    const childrenMap = {};
    const nodeMap = {};
    const roots = [];
    graph.nodes.forEach((n2) => {
      childrenMap[n2.key] = [];
      nodeMap[n2.key] = n2;
    });
    graph.edges.forEach((e3) => {
      if (childrenMap[e3.from])
        childrenMap[e3.from].push(e3.to);
    });
    graph.nodes.forEach((n2) => {
      const isChild = graph.edges.some((e3) => e3.to === n2.key);
      if (!isChild)
        roots.push(n2.key);
    });
    function renderNode(key) {
      const n2 = nodeMap[key];
      if (!n2)
        return null;
      const icon = n2.status === "completed" ? "✓" : "●";
      const iconClass = n2.status === "completed" ? "completed" : "active";
      const label = n2.label || n2.short_key || n2.key;
      const kids = childrenMap[key] || [];
      return /* @__PURE__ */ u5("li", {
        class: "session-tree-node",
        children: [
          /* @__PURE__ */ u5("span", {
            class: `session-tree-icon ${iconClass}`,
            children: icon
          }, undefined, false, undefined, this),
          /* @__PURE__ */ u5("span", {
            class: "session-tree-label",
            children: label
          }, undefined, false, undefined, this),
          /* @__PURE__ */ u5("span", {
            class: "session-tree-meta",
            children: [
              "turns=",
              n2.turn_count
            ]
          }, undefined, true, undefined, this),
          kids.length > 0 && /* @__PURE__ */ u5("ul", {
            class: "session-tree-children",
            children: kids.map(renderNode)
          }, undefined, false, undefined, this)
        ]
      }, key, true, undefined, this);
    }
    return /* @__PURE__ */ u5("div", {
      class: "card glass",
      children: [
        /* @__PURE__ */ u5("div", {
          class: "card-title",
          children: "Session Graph"
        }, undefined, false, undefined, this),
        /* @__PURE__ */ u5("ul", {
          class: "session-tree",
          children: roots.map(renderNode)
        }, undefined, false, undefined, this)
      ]
    }, undefined, true, undefined, this);
  }

  // src/components/work/work-tab.tsx
  function WorkTab({ active, sse }) {
    const [gitRepos, setGitRepos] = d2(null);
    const [worktrees, setWorktrees] = d2([]);
    const [sessions, setSessions] = d2(null);
    const [stats, setStats] = d2(null);
    const [graph, setGraph] = d2(null);
    const [context, setContext] = d2(null);
    const [loading, setLoading] = d2(true);
    const loadAll = q2(async () => {
      setLoading(true);
      try {
        const [gitData, wtData, sessionData, sessionsData, ctxData, graphData] = await Promise.all([
          apiFetch("/miniapp/api/git"),
          apiFetch("/miniapp/api/worktrees").catch(() => []),
          apiFetch("/miniapp/api/session"),
          apiFetch("/miniapp/api/sessions").catch(() => []),
          apiFetch("/miniapp/api/context").catch(() => null),
          apiFetch("/miniapp/api/sessions/graph").catch(() => null)
        ]);
        setGitRepos(gitData);
        setWorktrees(wtData);
        setStats(sessionData);
        setSessions(sessionsData);
        setContext(ctxData);
        setGraph(graphData);
      } catch {}
      setLoading(false);
    }, []);
    y2(() => {
      if (sse.session) {
        setSessions(sse.session.sessions || []);
        setStats(sse.session.stats || null);
        if (sse.session.graph)
          setGraph(sse.session.graph);
      }
    }, [sse.session]);
    y2(() => {
      if (sse.context)
        setContext(sse.context);
    }, [sse.context]);
    y2(() => {
      if (active)
        loadAll();
    }, [active]);
    if (loading && !gitRepos && !sessions)
      return /* @__PURE__ */ u5("div", {
        class: "loading",
        children: "Loading..."
      }, undefined, false, undefined, this);
    return /* @__PURE__ */ u5(k, {
      children: [
        /* @__PURE__ */ u5(GitSection, {
          repos: gitRepos,
          worktrees,
          onReload: loadAll
        }, undefined, false, undefined, this),
        /* @__PURE__ */ u5(SessionSection, {
          sessions,
          stats,
          graph,
          context
        }, undefined, false, undefined, this)
      ]
    }, undefined, true, undefined, this);
  }

  // src/components/tools/skills-section.tsx
  function SkillsSection({ active, sse }) {
    const [skills, setSkills] = d2(null);
    const [loading, setLoading] = d2(true);
    const [selected, setSelected] = d2(null);
    const [msg, setMsg] = d2("");
    const loadSkills = async () => {
      setLoading(true);
      try {
        const data = await apiFetch("/miniapp/api/skills");
        setSkills(data);
      } catch {}
      setLoading(false);
    };
    y2(() => {
      if (sse.skills) {
        setSkills(sse.skills);
        setLoading(false);
      }
    }, [sse.skills]);
    y2(() => {
      if (active && !isFresh(sse.lastUpdate, "skills"))
        loadSkills();
    }, [active]);
    const handleSend = async () => {
      if (!selected)
        return;
      const m4 = msg.trim();
      const cmd = m4 ? "/skill " + selected + " " + m4 : "/skill " + selected;
      const ok = await sendCommand(cmd);
      if (ok)
        setMsg("");
    };
    return /* @__PURE__ */ u5("div", {
      class: "card glass",
      children: [
        /* @__PURE__ */ u5("div", {
          class: "card-title",
          children: "Skills"
        }, undefined, false, undefined, this),
        loading && !skills ? /* @__PURE__ */ u5("div", {
          class: "loading",
          style: { padding: "12px" },
          children: "Loading skills..."
        }, undefined, false, undefined, this) : !skills || skills.length === 0 ? /* @__PURE__ */ u5("div", {
          class: "empty-state",
          style: { padding: "12px 0" },
          children: "No skills installed."
        }, undefined, false, undefined, this) : /* @__PURE__ */ u5(k, {
          children: [
            skills.map((s3) => /* @__PURE__ */ u5("div", {
              class: `skill-item glass glass-interactive${selected === s3.name ? " selected" : ""}`,
              onClick: () => {
                setSelected(selected === s3.name ? null : s3.name);
              },
              children: [
                /* @__PURE__ */ u5("div", {
                  class: "skill-body",
                  children: [
                    /* @__PURE__ */ u5("div", {
                      class: "skill-name",
                      children: s3.name
                    }, undefined, false, undefined, this),
                    /* @__PURE__ */ u5("div", {
                      class: "skill-desc",
                      children: s3.description || "No description"
                    }, undefined, false, undefined, this),
                    /* @__PURE__ */ u5("span", {
                      class: "skill-source",
                      children: s3.source
                    }, undefined, false, undefined, this)
                  ]
                }, undefined, true, undefined, this),
                /* @__PURE__ */ u5("span", {
                  class: "skill-arrow",
                  children: "›"
                }, undefined, false, undefined, this)
              ]
            }, s3.name, true, undefined, this)),
            selected && /* @__PURE__ */ u5("div", {
              style: {
                display: "flex",
                gap: "8px",
                marginTop: "10px"
              },
              children: [
                /* @__PURE__ */ u5("input", {
                  class: "send-input glass glass-interactive",
                  placeholder: `Message for /${selected}...`,
                  value: msg,
                  onInput: (e3) => setMsg(e3.target.value),
                  onKeyDown: (e3) => e3.key === "Enter" && handleSend(),
                  style: { flex: 1 }
                }, undefined, false, undefined, this),
                /* @__PURE__ */ u5("button", {
                  class: "send-btn",
                  onClick: handleSend,
                  children: "Send"
                }, undefined, false, undefined, this)
              ]
            }, undefined, true, undefined, this)
          ]
        }, undefined, true, undefined, this)
      ]
    }, undefined, true, undefined, this);
  }

  // src/components/tools/commands-section.tsx
  var QUICK_CMDS = ["/session", "/skills", "/plan clear"];
  function CommandsSection() {
    const [customCmd, setCustomCmd] = d2("");
    const btnRef = A2(null);
    const handleQuick = async (cmd, e3) => {
      const ok = await sendCommand(cmd);
      if (ok)
        flashSent(e3.currentTarget);
    };
    const handleCustom = async () => {
      const cmd = customCmd.trim();
      if (!cmd || !cmd.startsWith("/"))
        return;
      const ok = await sendCommand(cmd);
      if (ok) {
        setCustomCmd("");
        if (btnRef.current)
          flashSent(btnRef.current);
      }
    };
    return /* @__PURE__ */ u5(k, {
      children: [
        /* @__PURE__ */ u5("div", {
          class: "card glass",
          children: [
            /* @__PURE__ */ u5("div", {
              class: "card-title",
              children: "Quick Commands"
            }, undefined, false, undefined, this),
            /* @__PURE__ */ u5("div", {
              class: "cmd-tiles",
              children: QUICK_CMDS.map((cmd) => /* @__PURE__ */ u5("button", {
                class: "cmd-tile glass glass-interactive",
                onClick: (e3) => handleQuick(cmd, e3),
                children: cmd
              }, cmd, false, undefined, this))
            }, undefined, false, undefined, this)
          ]
        }, undefined, true, undefined, this),
        /* @__PURE__ */ u5("div", {
          class: "card glass",
          children: [
            /* @__PURE__ */ u5("div", {
              class: "card-title",
              children: "Custom Command"
            }, undefined, false, undefined, this),
            /* @__PURE__ */ u5("div", {
              style: { display: "flex", gap: "8px", marginTop: "8px" },
              children: [
                /* @__PURE__ */ u5("input", {
                  class: "send-input glass glass-interactive",
                  placeholder: "/command args...",
                  value: customCmd,
                  onInput: (e3) => setCustomCmd(e3.target.value),
                  onKeyDown: (e3) => e3.key === "Enter" && handleCustom(),
                  style: { flex: 1 }
                }, undefined, false, undefined, this),
                /* @__PURE__ */ u5("button", {
                  class: "send-btn",
                  ref: btnRef,
                  onClick: handleCustom,
                  children: "Send"
                }, undefined, false, undefined, this)
              ]
            }, undefined, true, undefined, this)
          ]
        }, undefined, true, undefined, this)
      ]
    }, undefined, true, undefined, this);
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
  function escapeHtml3(value) {
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
    return ` <span class="log-fields">{${escapeHtml3(parts.join(", "))}}</span>`;
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
      const componentHTML = entry?.component ? `<span class="log-comp">${escapeHtml3(entry.component)}</span>` : "";
      const fieldsHTML = renderFields(entry?.fields);
      const message = escapeHtml3(entry?.message || "");
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

  // src/components/tools/logs-section.tsx
  var LOGS_PAGE_SIZE = 60;
  function LogsSection({ active }) {
    const [entries, setEntries] = d2([]);
    const [component, setComponent] = d2("");
    const [page, setPage] = d2(1);
    const [connected, setConnected] = d2(false);
    const wsRef = A2(null);
    const reconnectRef = A2(null);
    const containerRef = A2(null);
    const connect = q2(() => {
      if (wsRef.current && wsRef.current.readyState <= 1)
        return;
      const initData = window.Telegram?.WebApp?.initData || "";
      const proto = location.protocol === "https:" ? "wss:" : "ws:";
      let url = proto + "//" + location.host + "/miniapp/api/logs/ws?initData=" + encodeURIComponent(initData);
      if (component)
        url += "&component=" + encodeURIComponent(component);
      const ws = new WebSocket(url);
      wsRef.current = ws;
      ws.onopen = () => setConnected(true);
      ws.onmessage = (e3) => {
        const msg = JSON.parse(e3.data);
        if (msg.type === "init") {
          setEntries(msg.entries || []);
          setPage(1);
        } else if (msg.type === "entry") {
          setEntries((prev) => {
            const next = [...prev, msg.entry];
            return next.length > 200 ? next.slice(1) : next;
          });
        }
      };
      ws.onclose = () => {
        setConnected(false);
        wsRef.current = null;
        if (active) {
          reconnectRef.current = setTimeout(connect, 3000);
        }
      };
      ws.onerror = () => {};
    }, [component, active]);
    const disconnect = q2(() => {
      if (reconnectRef.current) {
        clearTimeout(reconnectRef.current);
        reconnectRef.current = null;
      }
      if (wsRef.current) {
        wsRef.current.close();
        wsRef.current = null;
      }
      setConnected(false);
    }, []);
    y2(() => {
      if (active) {
        connect();
      } else {
        disconnect();
      }
      return disconnect;
    }, [active, component]);
    const handleFilterChange = (comp) => {
      setComponent(comp);
      setPage(1);
      setEntries([]);
      disconnect();
    };
    const view = renderLogs(entries, {
      component,
      page,
      pageSize: LOGS_PAGE_SIZE
    });
    const handleSaveSnapshot = async () => {
      try {
        const initData = window.Telegram?.WebApp?.initData || "";
        const res = await fetch(location.origin + "/miniapp/api/logs/snapshot?initData=" + encodeURIComponent(initData), { method: "POST" });
        if (!res.ok)
          return;
        const data = await res.json();
        if (data.download_url) {
          const a3 = document.createElement("a");
          a3.href = location.origin + data.download_url + "?initData=" + encodeURIComponent(initData);
          a3.download = "";
          document.body.appendChild(a3);
          a3.click();
          document.body.removeChild(a3);
        }
      } catch {}
    };
    return /* @__PURE__ */ u5("div", {
      class: "card glass",
      children: [
        /* @__PURE__ */ u5("div", {
          style: {
            display: "flex",
            alignItems: "center",
            gap: "8px",
            marginBottom: "8px"
          },
          children: [
            /* @__PURE__ */ u5("span", {
              class: "card-title",
              style: { margin: 0 },
              children: "Logs"
            }, undefined, false, undefined, this),
            /* @__PURE__ */ u5("span", {
              class: `dev-target-dot${connected ? " on" : ""}`
            }, undefined, false, undefined, this)
          ]
        }, undefined, true, undefined, this),
        /* @__PURE__ */ u5("div", {
          class: "log-filter-chips",
          children: [
            { label: "All", comp: "" },
            { label: "Telego", comp: "telego" },
            { label: "Console", comp: "dev-console" }
          ].map((f4) => /* @__PURE__ */ u5("button", {
            class: `log-filter-chip${component === f4.comp ? " active" : ""}`,
            onClick: () => handleFilterChange(f4.comp),
            children: f4.label
          }, f4.comp, false, undefined, this))
        }, undefined, false, undefined, this),
        /* @__PURE__ */ u5("div", {
          ref: containerRef,
          id: "logs-content",
          dangerouslySetInnerHTML: { __html: view.html || '<div class="empty-state">No logs.</div>' }
        }, undefined, false, undefined, this),
        /* @__PURE__ */ u5("div", {
          class: "log-pagination",
          children: [
            /* @__PURE__ */ u5("button", {
              class: "log-page-btn",
              disabled: view.currentPage <= 1,
              onClick: () => setPage((p3) => Math.max(1, p3 - 1)),
              children: "Newer"
            }, undefined, false, undefined, this),
            /* @__PURE__ */ u5("span", {
              children: [
                view.currentPage,
                "/",
                view.totalPages,
                " (",
                view.totalItems,
                ")"
              ]
            }, undefined, true, undefined, this),
            /* @__PURE__ */ u5("button", {
              class: "log-page-btn",
              disabled: view.currentPage >= view.totalPages,
              onClick: () => setPage((p3) => p3 + 1),
              children: "Older"
            }, undefined, false, undefined, this)
          ]
        }, undefined, true, undefined, this),
        /* @__PURE__ */ u5("div", {
          class: "log-actions",
          children: /* @__PURE__ */ u5("button", {
            class: "log-snap-btn",
            onClick: handleSaveSnapshot,
            children: "Save Snapshot"
          }, undefined, false, undefined, this)
        }, undefined, false, undefined, this)
      ]
    }, undefined, true, undefined, this);
  }

  // src/components/tools/research-section.tsx
  var STATUS_COLORS = {
    pending: { bg: "rgba(234,179,8,0.15)", text: "#ca8a04" },
    active: { bg: "rgba(59,130,246,0.15)", text: "#2563eb" },
    completed: { bg: "rgba(34,197,94,0.15)", text: "#16a34a" },
    failed: { bg: "rgba(239,68,68,0.15)", text: "#dc2626" },
    canceled: { bg: "rgba(107,114,128,0.15)", text: "#6b7280" }
  };
  function ResearchSection({ active }) {
    const [tasks, setTasks] = d2(null);
    const [loading, setLoading] = d2(false);
    const [showForm, setShowForm] = d2(false);
    const [detailId, setDetailId] = d2(null);
    const loadTasks = q2(async () => {
      setLoading(true);
      try {
        const data = await apiFetch("/miniapp/api/research");
        setTasks(data);
      } catch {
        setTasks(null);
      }
      setLoading(false);
    }, []);
    y2(() => {
      if (active)
        loadTasks();
    }, [active]);
    if (detailId) {
      return /* @__PURE__ */ u5(TaskDetail, {
        taskId: detailId,
        onBack: () => {
          setDetailId(null);
          loadTasks();
        }
      }, undefined, false, undefined, this);
    }
    return /* @__PURE__ */ u5("div", {
      class: "card glass",
      children: [
        /* @__PURE__ */ u5("div", {
          style: {
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
            marginBottom: "12px"
          },
          children: [
            /* @__PURE__ */ u5("span", {
              class: "card-title",
              style: { margin: 0 },
              children: "Research Tasks"
            }, undefined, false, undefined, this),
            /* @__PURE__ */ u5("button", {
              class: "send-btn",
              style: { padding: "6px 14px", fontSize: "13px" },
              onClick: () => setShowForm(true),
              children: "+ New"
            }, undefined, false, undefined, this)
          ]
        }, undefined, true, undefined, this),
        showForm && /* @__PURE__ */ u5(NewTaskForm, {
          onCreated: () => {
            setShowForm(false);
            loadTasks();
          },
          onCancel: () => setShowForm(false)
        }, undefined, false, undefined, this),
        loading && !tasks ? /* @__PURE__ */ u5("div", {
          class: "loading",
          style: { padding: "12px" },
          children: "Loading tasks..."
        }, undefined, false, undefined, this) : !tasks || tasks.length === 0 ? /* @__PURE__ */ u5("div", {
          class: "empty-state",
          style: { padding: "24px" },
          children: "No research tasks yet."
        }, undefined, false, undefined, this) : tasks.map((t3) => {
          const sc = STATUS_COLORS[t3.status] || STATUS_COLORS.pending;
          return /* @__PURE__ */ u5("div", {
            class: "card glass glass-interactive",
            style: {
              cursor: "pointer",
              padding: "14px",
              ...t3.focused ? { borderLeft: "3px solid #a855f7" } : {}
            },
            onClick: () => setDetailId(t3.id),
            children: [
              /* @__PURE__ */ u5("div", {
                style: {
                  display: "flex",
                  alignItems: "center",
                  justifyContent: "space-between",
                  gap: "8px"
                },
                children: [
                  /* @__PURE__ */ u5("span", {
                    style: { fontWeight: 600, fontSize: "15px" },
                    children: t3.title
                  }, undefined, false, undefined, this),
                  /* @__PURE__ */ u5("div", {
                    style: {
                      display: "flex",
                      gap: "4px",
                      alignItems: "center"
                    },
                    children: [
                      t3.focused && /* @__PURE__ */ u5("span", {
                        style: {
                          fontSize: "10px",
                          fontWeight: 600,
                          padding: "2px 6px",
                          borderRadius: "8px",
                          background: "rgba(168,85,247,0.2)",
                          color: "#a855f7"
                        },
                        children: "focused"
                      }, undefined, false, undefined, this),
                      /* @__PURE__ */ u5("span", {
                        style: {
                          fontSize: "11px",
                          fontWeight: 600,
                          padding: "2px 8px",
                          borderRadius: "10px",
                          background: sc.bg,
                          color: sc.text
                        },
                        children: t3.status
                      }, undefined, false, undefined, this)
                    ]
                  }, undefined, true, undefined, this)
                ]
              }, undefined, true, undefined, this),
              t3.description && /* @__PURE__ */ u5("div", {
                style: {
                  color: "var(--hint)",
                  fontSize: "13px",
                  marginTop: "4px",
                  lineHeight: 1.4
                },
                children: t3.description.substring(0, 120)
              }, undefined, false, undefined, this),
              /* @__PURE__ */ u5("div", {
                style: {
                  color: "var(--hint)",
                  fontSize: "11px",
                  marginTop: "6px"
                },
                children: [
                  t3.document_count,
                  " docs",
                  " · ⏱ ",
                  (t3.interval === "24h" ? "1d" : t3.interval) || "1d",
                  t3.last_researched_at && " · last: " + new Date(t3.last_researched_at).toLocaleDateString()
                ]
              }, undefined, true, undefined, this)
            ]
          }, t3.id, true, undefined, this);
        })
      ]
    }, undefined, true, undefined, this);
  }
  function NewTaskForm({
    onCreated,
    onCancel
  }) {
    const [title, setTitle] = d2("");
    const [desc, setDesc] = d2("");
    const handleCreate = async () => {
      const t3 = title.trim();
      if (!t3)
        return;
      try {
        await apiPost("/miniapp/api/research", {
          title: t3,
          description: desc.trim()
        });
        onCreated();
      } catch {}
    };
    return /* @__PURE__ */ u5("div", {
      class: "card glass",
      style: { padding: "12px", marginBottom: "12px" },
      children: [
        /* @__PURE__ */ u5("input", {
          class: "send-input glass glass-interactive",
          placeholder: "Task title...",
          value: title,
          onInput: (e3) => setTitle(e3.target.value),
          style: { width: "100%", marginBottom: "8px" }
        }, undefined, false, undefined, this),
        /* @__PURE__ */ u5("textarea", {
          class: "send-input glass glass-interactive",
          placeholder: "Description (optional)...",
          value: desc,
          onInput: (e3) => setDesc(e3.target.value),
          style: {
            width: "100%",
            minHeight: "60px",
            resize: "vertical",
            marginBottom: "8px",
            borderRadius: "12px",
            padding: "10px 16px"
          }
        }, undefined, false, undefined, this),
        /* @__PURE__ */ u5("div", {
          style: { display: "flex", gap: "8px", justifyContent: "flex-end" },
          children: [
            /* @__PURE__ */ u5("button", {
              class: "send-btn",
              style: {
                padding: "6px 14px",
                fontSize: "13px",
                background: "var(--hint)"
              },
              onClick: onCancel,
              children: "Cancel"
            }, undefined, false, undefined, this),
            /* @__PURE__ */ u5("button", {
              class: "send-btn",
              style: { padding: "6px 14px", fontSize: "13px" },
              onClick: handleCreate,
              children: "Create"
            }, undefined, false, undefined, this)
          ]
        }, undefined, true, undefined, this)
      ]
    }, undefined, true, undefined, this);
  }
  function TaskDetail({
    taskId,
    onBack
  }) {
    const [task, setTask] = d2(null);
    const [loading, setLoading] = d2(true);
    const load = q2(async () => {
      setLoading(true);
      try {
        const data = await apiFetch("/miniapp/api/research/" + taskId);
        setTask(data);
      } catch {}
      setLoading(false);
    }, [taskId]);
    y2(() => {
      load();
    }, [taskId]);
    const handleAction = async (action) => {
      try {
        await apiPost("/miniapp/api/research/" + taskId, { action });
        load();
      } catch {}
    };
    const handleFocus = async (recall) => {
      try {
        await apiPost("/miniapp/api/research/focus", {
          action: recall ? "recall" : "forget",
          task_id: taskId
        });
        load();
      } catch {}
    };
    const handleInterval = async (interval) => {
      try {
        await apiPost("/miniapp/api/research/" + taskId, {
          action: "set_interval",
          interval
        });
        load();
      } catch {}
    };
    if (loading || !task) {
      return /* @__PURE__ */ u5("div", {
        class: "card glass",
        children: [
          /* @__PURE__ */ u5("button", {
            class: "git-back-btn",
            onClick: onBack,
            children: [
              "‹",
              " Back"
            ]
          }, undefined, true, undefined, this),
          /* @__PURE__ */ u5("div", {
            class: "loading",
            children: "Loading..."
          }, undefined, false, undefined, this)
        ]
      }, undefined, true, undefined, this);
    }
    const sc = STATUS_COLORS[task.status] || STATUS_COLORS.pending;
    const canCancel = task.status === "pending" || task.status === "active";
    const canReopen = task.status === "completed" || task.status === "failed";
    const curInterval = (task.interval === "24h" ? "1d" : task.interval) || "1d";
    return /* @__PURE__ */ u5(k, {
      children: [
        /* @__PURE__ */ u5("button", {
          class: "git-back-btn",
          onClick: onBack,
          children: [
            "‹",
            " Back"
          ]
        }, undefined, true, undefined, this),
        /* @__PURE__ */ u5("div", {
          class: "card glass",
          children: [
            /* @__PURE__ */ u5("div", {
              style: {
                display: "flex",
                alignItems: "center",
                justifyContent: "space-between",
                gap: "8px",
                marginBottom: "8px"
              },
              children: [
                /* @__PURE__ */ u5("span", {
                  style: { fontWeight: 700, fontSize: "17px" },
                  children: task.title
                }, undefined, false, undefined, this),
                /* @__PURE__ */ u5("span", {
                  style: {
                    fontSize: "11px",
                    fontWeight: 600,
                    padding: "2px 8px",
                    borderRadius: "10px",
                    background: sc.bg,
                    color: sc.text
                  },
                  children: task.status
                }, undefined, false, undefined, this)
              ]
            }, undefined, true, undefined, this),
            task.description && /* @__PURE__ */ u5("div", {
              style: {
                color: "var(--hint)",
                fontSize: "13px",
                lineHeight: 1.5,
                marginBottom: "8px",
                whiteSpace: "pre-wrap"
              },
              children: task.description
            }, undefined, false, undefined, this),
            /* @__PURE__ */ u5("div", {
              style: {
                color: "var(--hint)",
                fontSize: "11px",
                display: "flex",
                alignItems: "center",
                gap: "6px"
              },
              children: [
                /* @__PURE__ */ u5("span", {
                  children: "Interval:"
                }, undefined, false, undefined, this),
                /* @__PURE__ */ u5("select", {
                  value: curInterval,
                  onChange: (e3) => handleInterval(e3.target.value),
                  style: {
                    fontSize: "11px",
                    padding: "1px 4px",
                    borderRadius: "6px",
                    background: "var(--tab-track-bg)",
                    color: "var(--text)",
                    border: "1px solid var(--glass-divider)",
                    outline: "none"
                  },
                  children: ["30m", "1h", "6h", "12h", "1d", "3d", "7d"].map((v4) => /* @__PURE__ */ u5("option", {
                    value: v4,
                    children: v4
                  }, v4, false, undefined, this))
                }, undefined, false, undefined, this),
                /* @__PURE__ */ u5("span", {
                  children: task.last_researched_at ? "· Last: " + new Date(task.last_researched_at).toLocaleString() : "· Not yet researched"
                }, undefined, false, undefined, this)
              ]
            }, undefined, true, undefined, this),
            /* @__PURE__ */ u5("div", {
              style: {
                color: "var(--hint)",
                fontSize: "11px",
                marginTop: "2px"
              },
              children: [
                "Created: ",
                new Date(task.created_at).toLocaleString(),
                task.completed_at && " · Completed: " + new Date(task.completed_at).toLocaleString()
              ]
            }, undefined, true, undefined, this),
            /* @__PURE__ */ u5("div", {
              style: { marginTop: "10px", display: "flex", gap: "8px" },
              children: [
                task.focused ? /* @__PURE__ */ u5("button", {
                  class: "worktree-btn dispose",
                  style: {
                    background: "rgba(168,85,247,0.15)",
                    color: "#a855f7",
                    borderColor: "#a855f7"
                  },
                  onClick: () => handleFocus(false),
                  children: "Forget"
                }, undefined, false, undefined, this) : /* @__PURE__ */ u5("button", {
                  class: "worktree-btn merge",
                  style: {
                    background: "rgba(168,85,247,0.15)",
                    color: "#a855f7",
                    borderColor: "#a855f7"
                  },
                  onClick: () => handleFocus(true),
                  children: "Recall"
                }, undefined, false, undefined, this),
                task.status === "pending" && /* @__PURE__ */ u5("button", {
                  class: "worktree-btn merge",
                  onClick: () => handleAction("activate"),
                  children: "Activate"
                }, undefined, false, undefined, this),
                task.status === "active" && /* @__PURE__ */ u5("button", {
                  class: "worktree-btn merge",
                  style: {
                    background: "rgba(34,197,94,0.15)",
                    color: "#22c55e",
                    borderColor: "#22c55e"
                  },
                  onClick: () => handleAction("complete"),
                  children: "Complete"
                }, undefined, false, undefined, this),
                canCancel && /* @__PURE__ */ u5("button", {
                  class: "worktree-btn dispose",
                  onClick: () => handleAction("cancel"),
                  children: "Cancel"
                }, undefined, false, undefined, this),
                canReopen && /* @__PURE__ */ u5("button", {
                  class: "worktree-btn merge",
                  onClick: () => handleAction("reopen"),
                  children: "Reopen"
                }, undefined, false, undefined, this)
              ]
            }, undefined, true, undefined, this)
          ]
        }, undefined, true, undefined, this),
        /* @__PURE__ */ u5("div", {
          class: "card-title",
          style: { marginTop: "12px" },
          children: [
            "Documents (",
            task.documents.length,
            ")"
          ]
        }, undefined, true, undefined, this),
        task.documents.length === 0 ? /* @__PURE__ */ u5("div", {
          class: "empty-state",
          style: { padding: "24px" },
          children: "No documents yet."
        }, undefined, false, undefined, this) : task.documents.map((d3) => /* @__PURE__ */ u5(DocCard, {
          doc: d3,
          taskId
        }, d3.id, false, undefined, this))
      ]
    }, undefined, true, undefined, this);
  }
  function DocCard({ doc, taskId }) {
    const [expanded, setExpanded] = d2(false);
    const [content, setContent] = d2(null);
    const [loading, setLoading] = d2(false);
    const toggle = async () => {
      if (expanded) {
        setExpanded(false);
        return;
      }
      setExpanded(true);
      if (content !== null)
        return;
      setLoading(true);
      try {
        const data = await apiFetch("/miniapp/api/research/" + taskId + "/doc/" + doc.id);
        setContent(data.content);
      } catch {
        setContent("Failed to load document.");
      }
      setLoading(false);
    };
    return /* @__PURE__ */ u5("div", {
      class: "card glass",
      style: { padding: "12px", cursor: "pointer" },
      onClick: toggle,
      children: [
        /* @__PURE__ */ u5("div", {
          style: { display: "flex", alignItems: "center", gap: "8px" },
          children: [
            /* @__PURE__ */ u5("span", {
              style: {
                color: "var(--hint)",
                fontFamily: "monospace",
                fontSize: "12px"
              },
              children: [
                "#",
                doc.seq
              ]
            }, undefined, true, undefined, this),
            /* @__PURE__ */ u5("span", {
              style: { fontWeight: 600, fontSize: "14px", flex: 1 },
              children: doc.title
            }, undefined, false, undefined, this),
            /* @__PURE__ */ u5("span", {
              style: {
                fontSize: "10px",
                padding: "2px 6px",
                borderRadius: "8px",
                background: "var(--tab-track-bg)",
                color: "var(--hint)"
              },
              children: doc.doc_type
            }, undefined, false, undefined, this)
          ]
        }, undefined, true, undefined, this),
        doc.summary && /* @__PURE__ */ u5("div", {
          style: {
            color: "var(--hint)",
            fontSize: "12px",
            marginTop: "4px"
          },
          children: doc.summary
        }, undefined, false, undefined, this),
        expanded && /* @__PURE__ */ u5("div", {
          style: {
            marginTop: "8px",
            borderTop: "1px solid var(--glass-divider)",
            paddingTop: "8px"
          },
          children: loading ? /* @__PURE__ */ u5("div", {
            class: "loading",
            style: { padding: "12px" },
            children: "Loading..."
          }, undefined, false, undefined, this) : content ? /* @__PURE__ */ u5("div", {
            class: "md-rendered",
            style: { maxHeight: "50vh", overflow: "auto", padding: "8px 0" },
            dangerouslySetInnerHTML: { __html: renderMarkdown(content) }
          }, undefined, false, undefined, this) : null
        }, undefined, false, undefined, this)
      ]
    }, undefined, true, undefined, this);
  }

  // src/components/tools/tools-tab.tsx
  function ToolsTab({ active, sse }) {
    return /* @__PURE__ */ u5(k, {
      children: [
        /* @__PURE__ */ u5(SkillsSection, {
          active,
          sse
        }, undefined, false, undefined, this),
        /* @__PURE__ */ u5(CommandsSection, {}, undefined, false, undefined, this),
        /* @__PURE__ */ u5(LogsSection, {
          active
        }, undefined, false, undefined, this),
        /* @__PURE__ */ u5(ResearchSection, {
          active
        }, undefined, false, undefined, this)
      ]
    }, undefined, true, undefined, this);
  }

  // src/components/dev/dev-tab.tsx
  function DevTab({ active, sse }) {
    const [data, setData] = d2(null);
    const loadDev = q2(async () => {
      try {
        const d3 = await apiFetch("/miniapp/api/dev");
        setData(d3);
      } catch {}
    }, []);
    y2(() => {
      if (sse.dev)
        setData(sse.dev);
    }, [sse.dev]);
    y2(() => {
      if (active && !isFresh(sse.lastUpdate, "dev"))
        loadDev();
    }, [active]);
    const targets = data?.targets || [];
    const activeId = data?.active_id || "";
    const isActive = !!data?.active;
    const handleToggle = async (id) => {
      const action = id === activeId ? "deactivate" : "activate";
      const body = action === "activate" ? { action: "activate", id } : { action: "deactivate" };
      try {
        const d3 = await apiPost("/miniapp/api/dev", body);
        if (!d3.error)
          setData(d3);
      } catch {}
    };
    const handleDelete = async (id, name) => {
      if (!confirm('Remove "' + name + '"?'))
        return;
      try {
        const d3 = await apiPost("/miniapp/api/dev", {
          action: "unregister",
          id
        });
        if (!d3.error)
          setData(d3);
      } catch {}
    };
    const iframeSrc = isActive ? location.origin + "/miniapp/dev/" : "";
    const targetDisplay = data?.target ? data.target.replace(/^https?:\/\//, "") : "";
    return /* @__PURE__ */ u5(k, {
      children: [
        /* @__PURE__ */ u5("div", {
          class: "dev-header",
          children: [
            /* @__PURE__ */ u5("span", {
              class: `dev-target-dot${isActive ? " on" : ""}`
            }, undefined, false, undefined, this),
            /* @__PURE__ */ u5("span", {
              class: "dev-header-title",
              children: "Dev Preview"
            }, undefined, false, undefined, this),
            /* @__PURE__ */ u5("span", {
              class: "dev-header-target",
              children: targetDisplay
            }, undefined, false, undefined, this)
          ]
        }, undefined, true, undefined, this),
        targets.length === 0 ? /* @__PURE__ */ u5("div", {
          class: "empty-state",
          children: [
            "No targets registered.",
            /* @__PURE__ */ u5("br", {}, undefined, false, undefined, this),
            "Ask the agent to start a dev server."
          ]
        }, undefined, true, undefined, this) : targets.map((t3) => {
          const isTargetActive = t3.id === activeId;
          const displayUrl = t3.target.replace(/^https?:\/\//, "");
          return /* @__PURE__ */ u5("div", {
            class: `dev-target-item glass glass-interactive${isTargetActive ? " active" : ""}`,
            onClick: () => handleToggle(t3.id),
            children: [
              /* @__PURE__ */ u5("span", {
                class: `dev-target-dot${isTargetActive ? " on" : ""}`
              }, undefined, false, undefined, this),
              /* @__PURE__ */ u5("span", {
                class: "dev-target-name",
                children: t3.name
              }, undefined, false, undefined, this),
              /* @__PURE__ */ u5("span", {
                class: "dev-target-url",
                children: displayUrl
              }, undefined, false, undefined, this),
              /* @__PURE__ */ u5("span", {
                class: "dev-target-delete",
                onClick: (e3) => {
                  e3.stopPropagation();
                  handleDelete(t3.id, t3.name);
                },
                children: "×"
              }, undefined, false, undefined, this)
            ]
          }, t3.id, true, undefined, this);
        }),
        isActive && /* @__PURE__ */ u5("div", {
          style: { marginTop: "8px" },
          children: /* @__PURE__ */ u5("div", {
            class: "card glass",
            style: { padding: 0, overflow: "hidden" },
            children: /* @__PURE__ */ u5("iframe", {
              src: iframeSrc,
              style: {
                width: "100%",
                height: "70vh",
                border: "none",
                borderRadius: "16px"
              }
            }, undefined, false, undefined, this)
          }, undefined, false, undefined, this)
        }, undefined, false, undefined, this)
      ]
    }, undefined, true, undefined, this);
  }

  // src/app.tsx
  var TABS = [
    { id: "plan", label: "Plan" },
    { id: "work", label: "Work" },
    { id: "tools", label: "Tools" },
    { id: "dev", label: "Dev" }
  ];
  function App() {
    const [activeTab, setActiveTab] = d2("plan");
    const indicatorRef = A2(null);
    const sse = useSSE();
    const switchTab = q2((id, index) => {
      setActiveTab(id);
      if (indicatorRef.current) {
        indicatorRef.current.style.transform = `translateX(${index * 100}%)`;
      }
    }, []);
    return /* @__PURE__ */ u5(k, {
      children: [
        /* @__PURE__ */ u5("div", {
          class: "tabs",
          children: /* @__PURE__ */ u5("div", {
            class: "tabs-inner",
            children: [
              /* @__PURE__ */ u5("div", {
                class: "tab-indicator",
                ref: indicatorRef
              }, undefined, false, undefined, this),
              TABS.map((tab, i3) => /* @__PURE__ */ u5("button", {
                class: `tab${activeTab === tab.id ? " active" : ""}`,
                onClick: () => switchTab(tab.id, i3),
                children: tab.label
              }, tab.id, false, undefined, this))
            ]
          }, undefined, true, undefined, this)
        }, undefined, false, undefined, this),
        /* @__PURE__ */ u5("div", {
          class: `panel${activeTab === "plan" ? " active" : ""}`,
          children: /* @__PURE__ */ u5(PlanTab, {
            active: activeTab === "plan",
            sse
          }, undefined, false, undefined, this)
        }, undefined, false, undefined, this),
        /* @__PURE__ */ u5("div", {
          class: `panel${activeTab === "work" ? " active" : ""}`,
          children: /* @__PURE__ */ u5(WorkTab, {
            active: activeTab === "work",
            sse
          }, undefined, false, undefined, this)
        }, undefined, false, undefined, this),
        /* @__PURE__ */ u5("div", {
          class: `panel${activeTab === "tools" ? " active" : ""}`,
          children: /* @__PURE__ */ u5(ToolsTab, {
            active: activeTab === "tools",
            sse
          }, undefined, false, undefined, this)
        }, undefined, false, undefined, this),
        /* @__PURE__ */ u5("div", {
          class: `panel${activeTab === "dev" ? " active" : ""}`,
          children: /* @__PURE__ */ u5(DevTab, {
            active: activeTab === "dev",
            sse
          }, undefined, false, undefined, this)
        }, undefined, false, undefined, this)
      ]
    }, undefined, true, undefined, this);
  }

  // src/index.tsx
  var tg = window.Telegram.WebApp;
  tg.ready();
  var root = document.getElementById("app");
  if (root) {
    J(/* @__PURE__ */ u5(App, {}, undefined, false, undefined, this), root);
  }
})();
