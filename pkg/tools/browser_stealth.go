//go:build cdp

package tools

// generateStealthJS returns a JavaScript snippet that patches common
// automation detection vectors. When injected via Page.addScriptToEvaluateOnNewDocument,
// it runs before any page script and makes CDP-controlled browsers harder to detect.
//
// Borrowed from opencli's stealth module with adaptations for Go embedding.
func generateStealthJS() string {
	return `(function() {
  // Guard: prevent double injection
  var _gProto = EventTarget.prototype;
  var _gKey = '__pcw_stealth';
  if (_gProto[_gKey]) return 'skipped';
  Object.defineProperty(_gProto, _gKey, { value: true, enumerable: false, configurable: true });

  // --- Shared toString disguise infrastructure ---
  var _origToString = Function.prototype.toString;
  var _disguised = new WeakMap();

  Object.defineProperty(Function.prototype, 'toString', {
    value: function() {
      var override = _disguised.get(this);
      return override !== undefined ? override : _origToString.call(this);
    },
    writable: true,
    configurable: true
  });

  function _disguise(fn, name) {
    _disguised.set(fn, 'function ' + name + '() { [native code] }');
    try { Object.defineProperty(fn, 'name', { value: name, configurable: true }); } catch(e) {}
    return fn;
  }

  // 1. navigator.webdriver → false
  Object.defineProperty(navigator, 'webdriver', {
    get: function() { return false; },
    configurable: true
  });

  // 2. window.chrome stub
  if (!window.chrome) {
    window.chrome = {
      runtime: {
        onConnect: { addListener: function(){}, removeListener: function(){} },
        onMessage: { addListener: function(){}, removeListener: function(){} }
      },
      loadTimes: function() { return {}; },
      csi: function() { return {}; }
    };
  }

  // 3. navigator.plugins population
  if (!navigator.plugins || navigator.plugins.length === 0) {
    var fakePlugins = [
      { name: 'PDF Viewer', filename: 'internal-pdf-viewer', description: 'Portable Document Format' },
      { name: 'Chrome PDF Viewer', filename: 'internal-pdf-viewer', description: '' },
      { name: 'Chromium PDF Viewer', filename: 'internal-pdf-viewer', description: '' },
      { name: 'Microsoft Edge PDF Viewer', filename: 'internal-pdf-viewer', description: '' },
      { name: 'WebKit built-in PDF', filename: 'internal-pdf-viewer', description: '' }
    ];
    fakePlugins.item = function(i) { return fakePlugins[i] || null; };
    fakePlugins.namedItem = function(n) { return fakePlugins.find(function(p) { return p.name === n; }) || null; };
    fakePlugins.refresh = function() {};
    Object.defineProperty(navigator, 'plugins', {
      get: function() { return fakePlugins; },
      configurable: true
    });
  }

  // 4. navigator.languages guarantee
  if (!navigator.languages || navigator.languages.length === 0) {
    Object.defineProperty(navigator, 'languages', {
      get: function() { return ['en-US', 'en']; },
      configurable: true
    });
  }

  // 5. Permissions.query normalization
  var origQuery = window.Permissions && window.Permissions.prototype && window.Permissions.prototype.query;
  if (origQuery) {
    window.Permissions.prototype.query = function(parameters) {
      if (parameters && parameters.name === 'notifications') {
        return Promise.resolve({ state: Notification.permission, onchange: null });
      }
      return origQuery.call(this, parameters);
    };
  }

  // 6. Clean automation artifacts
  try { delete window.__playwright; } catch(e) {}
  try { delete window.__puppeteer; } catch(e) {}
  var propNames = Object.getOwnPropertyNames(window);
  for (var i = 0; i < propNames.length; i++) {
    if (propNames[i].indexOf('cdc_') === 0 || propNames[i].indexOf('__cdc_') === 0) {
      try { delete window[propNames[i]]; } catch(e) {}
    }
  }

  // 7. CDP stack trace cleanup
  var _origStackDesc = Object.getOwnPropertyDescriptor(Error.prototype, 'stack');
  var _cdpPatterns = ['puppeteer_evaluation_script', 'pptr:', 'debugger://', '__playwright', '__puppeteer'];
  if (_origStackDesc && _origStackDesc.get) {
    Object.defineProperty(Error.prototype, 'stack', {
      get: function() {
        var raw = _origStackDesc.get.call(this);
        if (typeof raw !== 'string') return raw;
        return raw.split('\n').filter(function(line) {
          for (var j = 0; j < _cdpPatterns.length; j++) {
            if (line.indexOf(_cdpPatterns[j]) !== -1) return false;
          }
          return true;
        }).join('\n');
      },
      configurable: true
    });
  }

  // 8. Anti-debugger statement trap
  var _OrigFunction = window.Function;
  var _origEval = window.eval;
  var _debuggerRe = /(?:^|(?<=[;{}\n\r]))\s*debugger\s*;?/g;
  var _cleanDebugger = function(src) {
    return typeof src === 'string' ? src.replace(_debuggerRe, '') : src;
  };

  var _PatchedFunction = function() {
    var args = Array.prototype.slice.call(arguments);
    if (args.length > 0) {
      args[args.length - 1] = _cleanDebugger(args[args.length - 1]);
    }
    if (this instanceof _PatchedFunction) {
      return new (Function.prototype.bind.apply(_OrigFunction, [null].concat(args)))();
    }
    return _OrigFunction.apply(this, args);
  };
  _PatchedFunction.prototype = _OrigFunction.prototype;
  _disguise(_PatchedFunction, 'Function');
  window.Function = _PatchedFunction;

  var _patchedEval = function(code) {
    return _origEval.call(this, _cleanDebugger(code));
  };
  _disguise(_patchedEval, 'eval');
  window.eval = _patchedEval;

  // 9. Console method fingerprinting defense
  var _consoleMethods = ['log', 'warn', 'error', 'info', 'debug', 'table',
    'trace', 'dir', 'group', 'groupEnd', 'groupCollapsed', 'clear', 'count',
    'assert', 'profile', 'profileEnd', 'time', 'timeEnd', 'timeStamp'];

  for (var ci = 0; ci < _consoleMethods.length; ci++) {
    var _m = _consoleMethods[ci];
    if (typeof console[_m] !== 'function') continue;
    var _origMethod = console[_m];
    var _wrapper = (function(orig) {
      return function() { return orig.apply(console, arguments); };
    })(_origMethod);
    Object.defineProperty(_wrapper, 'length', { value: _origMethod.length || 0, configurable: true });
    _disguise(_wrapper, _m);
    console[_m] = _wrapper;
  }

  // 10. window.outerWidth/outerHeight defense
  var _normalWidthDelta = window.outerWidth - window.innerWidth;
  var _normalHeightDelta = window.outerHeight - window.innerHeight;
  if (_normalWidthDelta > 100 || _normalHeightDelta > 200) {
    Object.defineProperty(window, 'outerWidth', {
      get: function() { return window.innerWidth; },
      configurable: true
    });
    var _heightOffset = Math.max(40, Math.min(120, _normalHeightDelta));
    Object.defineProperty(window, 'outerHeight', {
      get: function() { return window.innerHeight + _heightOffset; },
      configurable: true
    });
  }

  // 11. Performance API cleanup
  var _origGetEntries = Performance.prototype.getEntries;
  var _origGetByType = Performance.prototype.getEntriesByType;
  var _origGetByName = Performance.prototype.getEntriesByName;
  var _suspiciousPatterns = ['debugger', 'devtools', '__puppeteer', '__playwright', 'pptr:'];
  var _filterEntries = function(entries) {
    if (!Array.isArray(entries)) return entries;
    return entries.filter(function(e) {
      var name = e.name || '';
      for (var si = 0; si < _suspiciousPatterns.length; si++) {
        if (name.indexOf(_suspiciousPatterns[si]) !== -1) return false;
      }
      return true;
    });
  };
  Performance.prototype.getEntries = function() { return _filterEntries(_origGetEntries.call(this)); };
  Performance.prototype.getEntriesByType = function(type) { return _filterEntries(_origGetByType.call(this, type)); };
  Performance.prototype.getEntriesByName = function(name, type) { return _filterEntries(_origGetByName.call(this, name, type)); };

  // 12. WebDriver document property defense
  var docProps = Object.getOwnPropertyNames(document);
  for (var di = 0; di < docProps.length; di++) {
    if (docProps[di].indexOf('$cdc_') === 0 || docProps[di].indexOf('$chrome_') === 0) {
      try { delete document[docProps[di]]; } catch(e) {}
    }
  }

  // 13. Iframe contentWindow.chrome consistency
  var _origHTMLIFrame = HTMLIFrameElement.prototype;
  var _origContentWindow = Object.getOwnPropertyDescriptor(_origHTMLIFrame, 'contentWindow');
  if (_origContentWindow && _origContentWindow.get) {
    Object.defineProperty(_origHTMLIFrame, 'contentWindow', {
      get: function() {
        var _w = _origContentWindow.get.call(this);
        if (_w) {
          try {
            if (!_w.chrome) {
              Object.defineProperty(_w, 'chrome', {
                value: window.chrome,
                writable: true,
                configurable: true
              });
            }
          } catch(e) {}
        }
        return _w;
      },
      configurable: true
    });
  }

  return 'ok';
})();`
}
