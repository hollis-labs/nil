import { ReactRenderer } from "@tiptap/react";
import Mention from "@tiptap/extension-mention";
import { mergeAttributes } from "@tiptap/core";
import tippy from "tippy.js";
import type { Instance as TippyInstance } from "tippy.js";
import WikilinkSuggestion from "../components/WikilinkSuggestion";
import * as Backend from "../../wailsjs/go/main/App";

export const WikilinkExtension = Mention.extend({
  name: "wikilink",

  addAttributes() {
    return {
      id: {
        default: null,
        parseHTML: (element: HTMLElement) => element.getAttribute("data-id"),
        renderHTML: (attrs: Record<string, any>) => ({ "data-id": attrs['id'] }),
      },
      label: {
        default: null,
        parseHTML: (element: HTMLElement) => element.getAttribute("data-label"),
        renderHTML: (attrs: Record<string, any>) => ({ "data-label": attrs['label'] }),
      },
      refType: {
        default: "todo",
        parseHTML: (element: HTMLElement) => element.getAttribute("data-ref-type") || "todo",
        renderHTML: (attrs: Record<string, any>) => ({ "data-ref-type": attrs['refType'] }),
      },
    };
  },

  parseHTML() {
    return [{ tag: 'span[data-type="wikilink"]' }];
  },

  renderHTML({ node, HTMLAttributes }: { node: any; HTMLAttributes: Record<string, any> }) {
    return [
      "span",
      mergeAttributes(HTMLAttributes, {
        "data-type": "wikilink",
        "data-id": node.attrs.id,
        "data-label": node.attrs.label,
        "data-ref-type": node.attrs.refType,
        class: `ref-chip ref-chip--${node.attrs.refType}`,
        style: [
          "display:inline-flex",
          "align-items:center",
          "padding:1px 6px",
          "border-radius:4px",
          "font-size:0.9em",
          "font-weight:500",
          "cursor:pointer",
          "user-select:none",
          node.attrs.refType === "note"
            ? "background:rgba(96,165,250,0.15);color:var(--term-info);border:1px solid var(--term-info)"
            : "background:rgba(74,222,128,0.15);color:var(--term-success);border:1px solid var(--term-success)",
        ].join(";"),
      }),
      `@${node.attrs.label}`,
    ];
  },
}).configure({
  suggestion: {
    char: "@",
    allowSpaces: false,

    items: async ({ query }: { query: string }) => {
      if (!query) return [];
      try {
        const results = await (Backend.Search as any)({
          query,
          type: "all",
          page: 0,
          page_size: 8,
          sort_by: "updated_at",
          sort_dir: "desc",
          statuses: [],
          projects: [],
          contexts: [],
          tags: [],
          priorities: [],
        });
        return (results || []).map((t: any) => ({
          id: t.id,
          title: t.title,
          type: t.type || "todo",
        }));
      } catch {
        return [];
      }
    },

    render: () => {
      let component: ReactRenderer;
      let popup: TippyInstance[];

      return {
        onStart(props: any) {
          component = new ReactRenderer(WikilinkSuggestion, {
            props,
            editor: props.editor,
          });
          if (!props.clientRect) return;
          popup = tippy("body", {
            getReferenceClientRect: props.clientRect,
            appendTo: () => document.body,
            content: component.element,
            showOnCreate: true,
            interactive: true,
            trigger: "manual",
            placement: "bottom-start",
          }) as unknown as TippyInstance[];
        },

        onUpdate(props: any) {
          component?.updateProps(props);
          popup?.[0]?.setProps({ getReferenceClientRect: props.clientRect });
        },

        onKeyDown(props: any) {
          if (props.event.key === "Escape") {
            popup?.[0]?.hide();
            return true;
          }
          return (component?.ref as any)?.onKeyDown(props) ?? false;
        },

        onExit() {
          popup?.[0]?.destroy();
          component?.destroy();
        },
      };
    },
  },
});
