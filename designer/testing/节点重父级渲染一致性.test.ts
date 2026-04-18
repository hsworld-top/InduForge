import { mount } from "@vue/test-utils";
import { computed, defineComponent, nextTick } from "vue";
import { describe, expect, it } from "vitest";
import { DocumentModel } from "../src/editor-core/document/DocumentModel";
import { createComponentNode, createEmptySchema, createPageNode } from "../src/editor-core/document/types";

const SnapshotTreeRenderer = defineComponent({
  name: "SnapshotTreeRenderer",
  props: {
    doc: {
      type: Object,
      required: true,
    },
    nodeId: {
      type: String,
      required: true,
    },
    docVersion: {
      type: Number,
      required: true,
    },
  },
  setup(props) {
    const node = computed(() => {
      void props.docVersion;
      return (props.doc as DocumentModel).getNode(props.nodeId);
    });

    const childNodeIds = computed<string[]>(() => {
      void props.docVersion;
      const children = Array.isArray(node.value?.children) ? node.value.children : [];
      return [...children];
    });

    return {
      node,
      childNodeIds,
    };
  },
  template: `
    <div v-if="node" class="snapshot-node" :data-node-id="node.id">
      <SnapshotTreeRenderer
        v-for="childId in childNodeIds"
        :key="childId"
        :doc="doc"
        :node-id="childId"
        :doc-version="docVersion"
      />
    </div>
  `,
});

function createNode(id: string, type: string) {
  return createComponentNode(type, {
    id,
    label: id,
    props: {},
    style: {},
    children: [],
  });
}

function createDocumentForReparentFlow(): DocumentModel {
  const schema = createEmptySchema({ projectId: "proj-reparent" });
  const root = createNode("root", "AbsoluteLayout");
  const layout = createNode("layout", "HorizontalLayout");
  const buttonA = createNode("button-a", "Button");
  const buttonB = createNode("button-b", "Button");

  root.children = ["layout", "button-b"];
  layout.children = ["button-a"];

  schema.nodesById[root.id] = root;
  schema.nodesById[layout.id] = layout;
  schema.nodesById[buttonA.id] = buttonA;
  schema.nodesById[buttonB.id] = buttonB;

  const page = createPageNode({
    id: "page-reparent",
    name: "重父级回归页",
    path: "/page-reparent",
    logicalId: "logic-reparent",
    rootNodeId: root.id,
  });
  schema.pagesById[page.id] = page;
  schema.entry.homePageId = page.id;

  return new DocumentModel(schema);
}

function createDocumentForInsertionFlow(): DocumentModel {
  const schema = createEmptySchema({ projectId: "proj-insert" });
  const root = createNode("root", "AbsoluteLayout");
  const layout = createNode("layout", "HorizontalLayout");
  const buttonA = createNode("button-a", "Button");
  const buttonB = createNode("button-b", "Button");

  root.children = ["layout"];
  layout.children = ["button-a", "button-b"];

  schema.nodesById[root.id] = root;
  schema.nodesById[layout.id] = layout;
  schema.nodesById[buttonA.id] = buttonA;
  schema.nodesById[buttonB.id] = buttonB;

  const page = createPageNode({
    id: "page-insert",
    name: "插入回归页",
    path: "/page-insert",
    logicalId: "logic-insert",
    rootNodeId: root.id,
  });
  schema.pagesById[page.id] = page;
  schema.entry.homePageId = page.id;

  return new DocumentModel(schema);
}

describe("节点重父级渲染一致性回归", () => {
  it("C87/C90/C92/C94/C96：拖出-重叠层级调整-再放回后，同一节点渲染实例始终唯一", async () => {
    const doc = createDocumentForReparentFlow();
    let docVersion = 0;

    const wrapper = mount(SnapshotTreeRenderer, {
      props: {
        doc,
        nodeId: "root",
        docVersion,
      },
    });

    const expectSingleRenderedNode = (nodeId: string) => {
      expect(wrapper.findAll(`[data-node-id="${nodeId}"]`)).toHaveLength(1);
    };

    expectSingleRenderedNode("button-a");

    const layoutBeforeMove = doc.getNode("layout");
    const rootBeforeMove = doc.getNode("root");
    const layoutChildrenBefore = layoutBeforeMove?.children;
    const rootChildrenBefore = rootBeforeMove?.children;

    doc._moveNode("button-a", "root", 1);
    docVersion += 1;
    await wrapper.setProps({ docVersion });
    await nextTick();

    expectSingleRenderedNode("button-a");
    expect(doc.getNode("layout")?.children).toEqual([]);
    expect(doc.getNode("root")?.children).toEqual(["layout", "button-a", "button-b"]);
    expect(doc.getNode("layout")?.children).not.toBe(layoutChildrenBefore);
    expect(doc.getNode("root")?.children).not.toBe(rootChildrenBefore);

    doc._moveNode("button-a", "root", 0);
    docVersion += 1;
    await wrapper.setProps({ docVersion });
    await nextTick();
    expectSingleRenderedNode("button-a");
    expect(doc.getNode("root")?.children).toEqual(["button-a", "layout", "button-b"]);

    doc._moveNode("button-a", "root", 999);
    docVersion += 1;
    await wrapper.setProps({ docVersion });
    await nextTick();
    expectSingleRenderedNode("button-a");
    expect(doc.getNode("root")?.children).toEqual(["layout", "button-b", "button-a"]);

    doc._moveNode("button-a", "layout", 0);
    docVersion += 1;
    await wrapper.setProps({ docVersion });
    await nextTick();

    expectSingleRenderedNode("button-a");
    expect(doc.getNode("root")?.children).toEqual(["layout", "button-b"]);
    expect(doc.getNode("layout")?.children).toEqual(["button-a"]);
  });

  it("C43-C46：前插/中插/后插顺序保持正确", () => {
    const doc = createDocumentForInsertionFlow();

    const buttonFront = createNode("button-front", "Button");
    const buttonMiddle = createNode("button-middle", "Button");
    const buttonBack = createNode("button-back", "Button");

    const layoutBeforeInsert = doc.getNode("layout");
    const layoutChildrenBeforeInsert = layoutBeforeInsert?.children;

    doc._insertNode("layout", 0, buttonFront);
    expect(doc.getNode("layout")?.children).toEqual(["button-front", "button-a", "button-b"]);
    expect(doc.getNode("layout")?.children).not.toBe(layoutChildrenBeforeInsert);

    const layoutChildrenAfterFront = doc.getNode("layout")?.children;
    doc._insertNode("layout", 2, buttonMiddle);
    expect(doc.getNode("layout")?.children).toEqual([
      "button-front",
      "button-a",
      "button-middle",
      "button-b",
    ]);
    expect(doc.getNode("layout")?.children).not.toBe(layoutChildrenAfterFront);

    const layoutChildrenAfterMiddle = doc.getNode("layout")?.children;
    doc._insertNode("layout", 999, buttonBack);
    expect(doc.getNode("layout")?.children).toEqual([
      "button-front",
      "button-a",
      "button-middle",
      "button-b",
      "button-back",
    ]);
    expect(doc.getNode("layout")?.children).not.toBe(layoutChildrenAfterMiddle);
  });
});
