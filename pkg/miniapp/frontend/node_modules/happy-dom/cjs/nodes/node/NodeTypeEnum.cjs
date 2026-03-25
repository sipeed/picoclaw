"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
var NodeTypeEnum;
(function (NodeTypeEnum) {
    NodeTypeEnum[NodeTypeEnum["elementNode"] = 1] = "elementNode";
    NodeTypeEnum[NodeTypeEnum["attributeNode"] = 2] = "attributeNode";
    NodeTypeEnum[NodeTypeEnum["textNode"] = 3] = "textNode";
    NodeTypeEnum[NodeTypeEnum["cdataSectionNode"] = 4] = "cdataSectionNode";
    NodeTypeEnum[NodeTypeEnum["commentNode"] = 8] = "commentNode";
    NodeTypeEnum[NodeTypeEnum["documentNode"] = 9] = "documentNode";
    NodeTypeEnum[NodeTypeEnum["documentTypeNode"] = 10] = "documentTypeNode";
    NodeTypeEnum[NodeTypeEnum["documentFragmentNode"] = 11] = "documentFragmentNode";
    NodeTypeEnum[NodeTypeEnum["processingInstructionNode"] = 7] = "processingInstructionNode";
})(NodeTypeEnum || (NodeTypeEnum = {}));
exports.default = NodeTypeEnum;
//# sourceMappingURL=NodeTypeEnum.cjs.map