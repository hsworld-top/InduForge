/**
 * 图形操作命令
 * 包含 Canvas 图形的插入、删除、更新、移动命令
 */

import { Command } from "./Command.js";

/**
 * @typedef {import('../document/DocumentModel.js').DocumentModel} DocumentModel
 * @typedef {import('../document/types.js').GraphicNode} GraphicNode
 * @typedef {import('../document/types.js').GraphicProps} GraphicProps
 */

/**
 * 插入图形命令
 */
export class InsertGraphicCommand extends Command {
  get type() {
    return "InsertGraphic";
  }

  /**
   * 创建插入图形命令
   * @param {string} pageId - 页面 ID
   * @param {GraphicNode} graphic - 图形节点
   */
  constructor(pageId, graphic) {
    super();
    /** @type {string} */
    this._pageId = pageId;
    /** @type {GraphicNode} */
    this._graphic = graphic;
  }

  /**
   * @param {DocumentModel} doc
   */
  execute(doc) {
    doc._insertGraphic(this._pageId, this._graphic);
  }

  /**
   * @param {DocumentModel} doc
   */
  undo(doc) {
    doc._removeGraphic(this._graphic.id);
  }

  getDescription() {
    return `插入图形: ${this._graphic.type}`;
  }
}

/**
 * 删除图形命令
 */
export class RemoveGraphicCommand extends Command {
  get type() {
    return "RemoveGraphic";
  }

  /**
   * 创建删除图形命令
   * @param {string} graphicId - 图形 ID
   */
  constructor(graphicId) {
    super();
    /** @type {string} */
    this._graphicId = graphicId;
    /** @type {GraphicNode | null} */
    this._removedGraphic = null;
    /** @type {string | null} */
    this._pageId = null;
  }

  /**
   * @param {DocumentModel} doc
   */
  execute(doc) {
    // 保存删除前的状态
    this._pageId = doc.getGraphicPageId(this._graphicId);
    const graphic = doc.getGraphic(this._graphicId);
    if (graphic) {
      this._removedGraphic = JSON.parse(JSON.stringify(graphic));
    }

    // 执行删除
    doc._removeGraphic(this._graphicId);
  }

  /**
   * @param {DocumentModel} doc
   */
  undo(doc) {
    if (this._removedGraphic && this._pageId) {
      doc._insertGraphic(this._pageId, this._removedGraphic);
    }
  }

  getDescription() {
    return `删除图形`;
  }
}

/**
 * 更新图形命令
 */
export class UpdateGraphicCommand extends Command {
  get type() {
    return "UpdateGraphic";
  }

  /**
   * 创建更新图形命令
   * @param {string} graphicId - 图形 ID
   * @param {Partial<GraphicNode>} patch - 更新内容
   */
  constructor(graphicId, patch) {
    super();
    /** @type {string} */
    this._graphicId = graphicId;
    /** @type {Partial<GraphicNode>} */
    this._patch = patch;
    /** @type {Partial<GraphicNode> | null} */
    this._oldValues = null;
  }

  /**
   * @param {DocumentModel} doc
   */
  execute(doc) {
    const graphic = doc.getGraphic(this._graphicId);
    if (graphic) {
      // 保存旧值
      this._oldValues = {};
      for (const key of Object.keys(this._patch)) {
        this._oldValues[key] = JSON.parse(JSON.stringify(graphic[key] ?? null));
      }

      // 执行更新
      doc._updateGraphic(this._graphicId, this._patch);
    }
  }

  /**
   * @param {DocumentModel} doc
   */
  undo(doc) {
    if (this._oldValues) {
      doc._updateGraphic(this._graphicId, this._oldValues);
    }
  }

  /**
   * @param {Command} other
   * @returns {boolean}
   */
  canMerge(other) {
    return (
      other instanceof UpdateGraphicCommand &&
      other._graphicId === this._graphicId
    );
  }

  /**
   * @param {UpdateGraphicCommand} other
   * @returns {UpdateGraphicCommand}
   */
  merge(other) {
    const mergedPatch = { ...this._patch, ...other._patch };
    const cmd = new UpdateGraphicCommand(this._graphicId, mergedPatch);
    cmd._oldValues = this._oldValues;
    return cmd;
  }

  getDescription() {
    return `更新图形属性`;
  }
}

/**
 * 移动图形命令（位置偏移）
 */
export class MoveGraphicCommand extends Command {
  get type() {
    return "MoveGraphic";
  }

  /**
   * 创建移动图形命令
   * @param {string} graphicId - 图形 ID
   * @param {number} deltaX - X 方向偏移
   * @param {number} deltaY - Y 方向偏移
   */
  constructor(graphicId, deltaX, deltaY) {
    super();
    /** @type {string} */
    this._graphicId = graphicId;
    /** @type {number} */
    this._deltaX = deltaX;
    /** @type {number} */
    this._deltaY = deltaY;
    /** @type {GraphicProps | null} */
    this._oldProps = null;
  }

  /**
   * @param {DocumentModel} doc
   */
  execute(doc) {
    const graphic = doc.getGraphic(this._graphicId);
    if (graphic) {
      this._oldProps = JSON.parse(JSON.stringify(graphic.props));
      const newProps = this._applyDelta(
        graphic.props,
        this._deltaX,
        this._deltaY,
      );
      doc._updateGraphic(this._graphicId, { props: newProps });
    }
  }

  /**
   * @param {DocumentModel} doc
   */
  undo(doc) {
    if (this._oldProps) {
      doc._updateGraphic(this._graphicId, { props: this._oldProps });
    }
  }

  /**
   * @param {Command} other
   * @returns {boolean}
   */
  canMerge(other) {
    return (
      other instanceof MoveGraphicCommand &&
      other._graphicId === this._graphicId
    );
  }

  /**
   * @param {MoveGraphicCommand} other
   * @returns {MoveGraphicCommand}
   */
  merge(other) {
    const cmd = new MoveGraphicCommand(
      this._graphicId,
      this._deltaX + other._deltaX,
      this._deltaY + other._deltaY,
    );
    cmd._oldProps = this._oldProps;
    return cmd;
  }

  /**
   * 应用位置偏移
   * @param {GraphicProps} props
   * @param {number} dx
   * @param {number} dy
   * @returns {GraphicProps}
   * @private
   */
  _applyDelta(props, dx, dy) {
    const newProps = JSON.parse(JSON.stringify(props));

    // 处理不同图形类型的位置属性
    if (typeof newProps.x === "number") newProps.x += dx;
    if (typeof newProps.y === "number") newProps.y += dy;
    if (typeof newProps.cx === "number") newProps.cx += dx;
    if (typeof newProps.cy === "number") newProps.cy += dy;

    // 处理 points 数组（线段、多边形、管道）
    if (Array.isArray(newProps.points)) {
      newProps.points = newProps.points.map(([x, y]) => [x + dx, y + dy]);
    }

    return newProps;
  }

  getDescription() {
    return `移动图形`;
  }
}

/**
 * 调整图形层级命令
 */
export class ReorderGraphicCommand extends Command {
  get type() {
    return "ReorderGraphic";
  }

  /**
   * 创建调整图形层级命令
   * @param {string} graphicId - 图形 ID
   * @param {number} newZ - 新层级
   */
  constructor(graphicId, newZ) {
    super();
    /** @type {string} */
    this._graphicId = graphicId;
    /** @type {number} */
    this._newZ = newZ;
    /** @type {number} */
    this._oldZ = 0;
  }

  /**
   * @param {DocumentModel} doc
   */
  execute(doc) {
    const graphic = doc.getGraphic(this._graphicId);
    if (graphic) {
      this._oldZ = graphic.z;
      doc._updateGraphic(this._graphicId, { z: this._newZ });
    }
  }

  /**
   * @param {DocumentModel} doc
   */
  undo(doc) {
    doc._updateGraphic(this._graphicId, { z: this._oldZ });
  }

  getDescription() {
    return `调整图形层级`;
  }
}

/**
 * 图形编组命令
 */
export class GroupGraphicsCommand extends Command {
  get type() {
    return "GroupGraphics";
  }

  /**
   * 创建图形编组命令
   * @param {string} pageId - 页面 ID
   * @param {string[]} graphicIds - 要编组的图形 ID 列表
   */
  constructor(pageId, graphicIds) {
    super();
    /** @type {string} */
    this._pageId = pageId;
    /** @type {string[]} */
    this._graphicIds = graphicIds;
    /** @type {string} */
    this._groupId =
      "gfx_group_" + crypto.randomUUID().replace(/-/g, "").substring(0, 8);
  }

  /**
   * @param {DocumentModel} doc
   */
  execute(doc) {
    // 获取最大 z 值
    const maxZ = Math.max(
      ...this._graphicIds.map((id) => doc.getGraphic(id)?.z ?? 0),
    );

    // 创建编组图形
    /** @type {GraphicNode} */
    const group = {
      id: this._groupId,
      type: "Canvas.Group",
      props: { children: [...this._graphicIds] },
      bindings: {},
      events: {},
      animations: [],
      z: maxZ,
    };

    doc._insertGraphic(this._pageId, group);
  }

  /**
   * @param {DocumentModel} doc
   */
  undo(doc) {
    doc._removeGraphic(this._groupId);
  }

  /**
   * 获取创建的编组 ID
   * @returns {string}
   */
  getGroupId() {
    return this._groupId;
  }

  getDescription() {
    return `编组图形 (${this._graphicIds.length} 个)`;
  }
}

/**
 * 取消图形编组命令
 */
export class UngroupGraphicsCommand extends Command {
  get type() {
    return "UngroupGraphics";
  }

  /**
   * 创建取消编组命令
   * @param {string} groupId - 编组图形 ID
   */
  constructor(groupId) {
    super();
    /** @type {string} */
    this._groupId = groupId;
    /** @type {GraphicNode | null} */
    this._removedGroup = null;
    /** @type {string | null} */
    this._pageId = null;
  }

  /**
   * @param {DocumentModel} doc
   */
  execute(doc) {
    const group = doc.getGraphic(this._groupId);
    if (!group || group.type !== "Canvas.Group") return;

    // 保存编组信息
    this._removedGroup = JSON.parse(JSON.stringify(group));
    this._pageId = doc.getGraphicPageId(this._groupId);

    // 删除编组
    doc._removeGraphic(this._groupId);
  }

  /**
   * @param {DocumentModel} doc
   */
  undo(doc) {
    if (this._removedGroup && this._pageId) {
      doc._insertGraphic(this._pageId, this._removedGroup);
    }
  }

  getDescription() {
    return `取消编组`;
  }
}

/**
 * 设置图形绑定命令
 */
export class SetGraphicBindingCommand extends Command {
  get type() {
    return "SetGraphicBinding";
  }

  /**
   * 创建设置图形绑定命令
   * @param {string} graphicId - 图形 ID
   * @param {string} propKey - 属性键
   * @param {import('../document/types.js').Binding | null} binding - 绑定配置
   */
  constructor(graphicId, propKey, binding) {
    super();
    /** @type {string} */
    this._graphicId = graphicId;
    /** @type {string} */
    this._propKey = propKey;
    /** @type {import('../document/types.js').Binding | null} */
    this._binding = binding;
    /** @type {import('../document/types.js').Binding | null} */
    this._oldBinding = null;
  }

  /**
   * @param {DocumentModel} doc
   */
  execute(doc) {
    const graphic = doc.getGraphic(this._graphicId);
    if (graphic) {
      this._oldBinding = graphic.bindings?.[this._propKey] ?? null;

      const newBindings = { ...graphic.bindings };
      if (this._binding) {
        newBindings[this._propKey] = this._binding;
      } else {
        delete newBindings[this._propKey];
      }

      doc._updateGraphic(this._graphicId, { bindings: newBindings });
    }
  }

  /**
   * @param {DocumentModel} doc
   */
  undo(doc) {
    const graphic = doc.getGraphic(this._graphicId);
    if (graphic) {
      const newBindings = { ...graphic.bindings };
      if (this._oldBinding) {
        newBindings[this._propKey] = this._oldBinding;
      } else {
        delete newBindings[this._propKey];
      }

      doc._updateGraphic(this._graphicId, { bindings: newBindings });
    }
  }

  getDescription() {
    return this._binding
      ? `设置图形绑定: ${this._propKey}`
      : `移除图形绑定: ${this._propKey}`;
  }
}

export default {
  InsertGraphicCommand,
  RemoveGraphicCommand,
  UpdateGraphicCommand,
  MoveGraphicCommand,
  ReorderGraphicCommand,
  GroupGraphicsCommand,
  UngroupGraphicsCommand,
  SetGraphicBindingCommand,
};
