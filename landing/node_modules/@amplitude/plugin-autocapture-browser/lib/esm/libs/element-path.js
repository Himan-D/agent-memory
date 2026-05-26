/* istanbul ignore file */
import { __values } from "tslib";
// Code is adapted from The Chromium Authors.
// Source: https://github.com/ChromeDevTools/devtools-frontend/blob/main/front_end/panels/elements/DOMPath.ts#L14
// License: BSD-style license
//
// Copyright 2014 The Chromium Authors
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
//    * Redistributions of source code must retain the above copyright
// notice, this list of conditions and the following disclaimer.
//    * Redistributions in binary form must reproduce the above
// copyright notice, this list of conditions and the following disclaimer
// in the documentation and/or other materials provided with the
// distribution.
//    * Neither the name of Google Inc. nor the names of its
// contributors may be used to endorse or promote products derived from
// this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
var Step = /** @class */ (function () {
    function Step(value, optimized) {
        this.value = value;
        this.optimized = optimized;
    }
    Step.prototype.toString = function () {
        return this.value;
    };
    return Step;
}());
export var cssPath = function (node, optimized) {
    // `node` is already an Element; this check is defensive.
    if (node.nodeType !== Node.ELEMENT_NODE) {
        return '';
    }
    var steps = [];
    var contextNode = node;
    while (contextNode) {
        var step = cssPathStep(contextNode, Boolean(optimized), contextNode === node);
        if (!step) {
            break;
        } // bail out early
        steps.push(step);
        if (step.optimized) {
            break;
        }
        contextNode = contextNode.parentElement;
    }
    steps.reverse();
    return steps.join(' > ');
};
var cssPathStep = function (node, optimized, isTargetNode) {
    var e_1, _a;
    if (node.nodeType !== Node.ELEMENT_NODE) {
        return null;
    }
    var id = node.getAttribute('id');
    if (optimized) {
        if (id) {
            return new Step(idSelector(id), true);
        }
        var nodeNameLower = node.tagName.toLowerCase();
        if (nodeNameLower === 'body' || nodeNameLower === 'head' || nodeNameLower === 'html') {
            return new Step(nodeNameLower, true);
        }
    }
    var nodeName = node.tagName.toLowerCase();
    if (id) {
        return new Step(nodeName + idSelector(id), true);
    }
    var parent = node.parentNode;
    if (!parent || parent.nodeType === Node.DOCUMENT_NODE) {
        return new Step(nodeName, true);
    }
    function prefixedElementClassNames(el) {
        var classAttribute = el.getAttribute('class');
        if (!classAttribute) {
            return [];
        }
        return classAttribute
            .split(/\s+/g)
            .filter(Boolean)
            .map(function (name) {
            // The prefix is required to store "__proto__" in a object-based map.
            return '$' + name;
        });
    }
    function idSelector(id) {
        return '#' + CSS.escape(id);
    }
    var prefixedOwnClassNamesArray = prefixedElementClassNames(node);
    var needsClassNames = false;
    var needsNthChild = false;
    var ownIndex = -1;
    var elementIndex = -1;
    var siblings = parent.children;
    for (var i = 0; siblings && (ownIndex === -1 || !needsNthChild) && i < siblings.length; ++i) {
        var sibling = siblings[i];
        if (sibling.nodeType !== Node.ELEMENT_NODE) {
            continue;
        }
        elementIndex += 1;
        if (sibling === node) {
            ownIndex = elementIndex;
            continue;
        }
        if (needsNthChild) {
            continue;
        }
        if (sibling.tagName.toLowerCase() !== nodeName) {
            continue;
        }
        needsClassNames = true;
        var ownClassNames = new Set(prefixedOwnClassNamesArray);
        if (!ownClassNames.size) {
            needsNthChild = true;
            continue;
        }
        var siblingClassNamesArray = prefixedElementClassNames(sibling);
        for (var j = 0; j < siblingClassNamesArray.length; ++j) {
            var siblingClass = siblingClassNamesArray[j];
            if (!ownClassNames.has(siblingClass)) {
                continue;
            }
            ownClassNames.delete(siblingClass);
            if (!ownClassNames.size) {
                needsNthChild = true;
                break;
            }
        }
    }
    var result = nodeName;
    if (isTargetNode &&
        nodeName.toLowerCase() === 'input' &&
        node.getAttribute('type') &&
        !node.getAttribute('id') &&
        !node.getAttribute('class')) {
        result += '[type=' + CSS.escape(node.getAttribute('type') || '') + ']';
    }
    if (needsNthChild) {
        result += ':nth-child(' + String(ownIndex + 1) + ')';
    }
    else if (needsClassNames) {
        try {
            for (var prefixedOwnClassNamesArray_1 = __values(prefixedOwnClassNamesArray), prefixedOwnClassNamesArray_1_1 = prefixedOwnClassNamesArray_1.next(); !prefixedOwnClassNamesArray_1_1.done; prefixedOwnClassNamesArray_1_1 = prefixedOwnClassNamesArray_1.next()) {
                var prefixedName = prefixedOwnClassNamesArray_1_1.value;
                result += '.' + CSS.escape(prefixedName.slice(1));
            }
        }
        catch (e_1_1) { e_1 = { error: e_1_1 }; }
        finally {
            try {
                if (prefixedOwnClassNamesArray_1_1 && !prefixedOwnClassNamesArray_1_1.done && (_a = prefixedOwnClassNamesArray_1.return)) _a.call(prefixedOwnClassNamesArray_1);
            }
            finally { if (e_1) throw e_1.error; }
        }
    }
    return new Step(result, false);
};
//# sourceMappingURL=element-path.js.map