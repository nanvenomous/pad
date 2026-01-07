import MarkdownIt from "markdown-it";

type BlockType = "blank" | "line" | "fence";

type Block = {
  start: number;
  end: number;
  type: BlockType;
};

type ActiveEdit = {
  blockIndex: number;
  textarea: HTMLTextAreaElement;
};

type EditorState = {
  editor: HTMLElement;
  textarea: HTMLTextAreaElement;
  render: HTMLElement;
  lines: string[];
  blocks: Block[];
  activeEdit: ActiveEdit | null;
};

const md = new MarkdownIt({
  html: false,
  breaks: true,
  linkify: true,
});

const editorStates = new WeakMap<HTMLElement, EditorState>();

export function initNoteEditors(root: ParentNode = document): void {
  const editors = root.querySelectorAll<HTMLElement>("[data-note-editor]");
  editors.forEach((editor) => {
    if (editorStates.has(editor)) {
      return;
    }
    const textarea = editor.querySelector<HTMLTextAreaElement>("[data-note-body]");
    const render = editor.querySelector<HTMLElement>("[data-note-render]");
    if (!textarea || !render) {
      return;
    }

    const state: EditorState = {
      editor,
      textarea,
      render,
      lines: [],
      blocks: [],
      activeEdit: null,
    };

    editorStates.set(editor, state);

    render.classList.remove("hidden");
    textarea.classList.add("hidden");

    refreshFromTextarea(state);
  });
}

function refreshFromTextarea(state: EditorState): void {
  state.lines = state.textarea.value.split("\n");
  state.blocks = buildBlocks(state.lines);
  renderBlocks(state);
}

function buildBlocks(lines: string[]): Block[] {
  const blocks: Block[] = [];
  let i = 0;
  while (i < lines.length) {
    const line = lines[i];
    const fence = fenceMarker(line);
    if (fence) {
      let j = i + 1;
      while (j < lines.length && !isFenceEnd(lines[j], fence)) {
        j += 1;
      }
      if (j < lines.length) {
        j += 1;
      }
      blocks.push({ start: i, end: j - 1, type: "fence" });
      i = j;
      continue;
    }
    if (line.trim() === "") {
      blocks.push({ start: i, end: i, type: "blank" });
      i += 1;
      continue;
    }
    blocks.push({ start: i, end: i, type: "line" });
    i += 1;
  }
  if (blocks.length === 0) {
    blocks.push({ start: 0, end: 0, type: "blank" });
  }
  return blocks;
}

function fenceMarker(line: string): { char: string; length: number } | null {
  const match = line.match(/^(```+|~~~+)/);
  if (!match) {
    return null;
  }
  return { char: match[1][0], length: match[1].length };
}

function isFenceEnd(line: string, marker: { char: string; length: number }): boolean {
  const regex = new RegExp(`^${marker.char}{${marker.length},}\\s*$`);
  return regex.test(line.trim());
}

function renderBlocks(state: EditorState): void {
  const scrollTop = state.render.scrollTop;
  state.render.innerHTML = "";
  state.blocks.forEach((block, index) => {
    const blockEl = document.createElement("div");
    blockEl.className = "note-block";
    blockEl.dataset.blockIndex = String(index);
    blockEl.dataset.lineStart = String(block.start);
    blockEl.dataset.lineEnd = String(block.end);
    blockEl.dataset.blockType = block.type;
    blockEl.addEventListener("click", (event) => {
      const checkbox = findInPath<HTMLInputElement>(
        event,
        "input[type=checkbox][data-line]",
      );
      if (checkbox) {
        event.preventDefault();
        event.stopPropagation();
        toggleTask(state, checkbox);
        return;
      }
      startEditBlock(state, index, blockEl);
    });

    if (block.type === "blank") {
      blockEl.classList.add("note-blank");
      const spacer = document.createElement("div");
      spacer.className = "min-h-6";
      blockEl.appendChild(spacer);
    } else if (block.type === "fence") {
      const text = state.lines.slice(block.start, block.end + 1).join("\n");
      blockEl.classList.add("note-fence");
      blockEl.innerHTML = md.render(text);
    } else {
      const text = state.lines[block.start] ?? "";
      renderLineBlock(blockEl, text, block.start, index);
    }

  state.render.appendChild(blockEl);
  });
  state.render.scrollTop = scrollTop;
}

function renderLineBlock(
  blockEl: HTMLElement,
  line: string,
  lineIndex: number,
  blockIndex: number,
): void {
  const taskMatch = line.match(/^\s*[-*+]\s+\[( |x|X)\]\s+(.*)$/);
  if (taskMatch) {
    const indentMatch = line.match(/^\s*/);
    const indent = indentMatch ? indentMatch[0].length : 0;
    const checked = taskMatch[1].toLowerCase() === "x";
    const content = taskMatch[2];
    const wrapper = document.createElement("div");
    wrapper.className = "flex items-start gap-2";
    if (indent > 0) {
      wrapper.style.marginLeft = `${indent * 0.5}rem`;
    }
    const checkbox = document.createElement("input");
    checkbox.type = "checkbox";
    checkbox.className = "checkbox checkbox-xs mt-1";
    checkbox.checked = checked;
    checkbox.dataset.line = String(lineIndex);
    checkbox.dataset.blockIndex = String(blockIndex);
    checkbox.setAttribute("aria-label", "Toggle task");
    const text = document.createElement("div");
    text.className = "note-task-text";
    text.innerHTML = md.renderInline(content);
    wrapper.appendChild(checkbox);
    wrapper.appendChild(text);
    blockEl.appendChild(wrapper);
    return;
  }

  if (line.trim() === "") {
    const spacer = document.createElement("div");
    spacer.className = "min-h-6";
    blockEl.appendChild(spacer);
    return;
  }

  blockEl.innerHTML = md.render(line);
}

function startEditBlock(state: EditorState, blockIndex: number, blockEl: HTMLElement): void {
  if (state.activeEdit) {
    state.activeEdit.textarea.blur();
  }
  const block = state.blocks[blockIndex];
  if (!block) {
    return;
  }
  const text = state.lines.slice(block.start, block.end + 1).join("\n");
  const editor = document.createElement("textarea");
  editor.className =
    "textarea textarea-bordered w-full rounded-2xl bg-base-100/70 p-3 text-sm leading-6 focus:border-accent focus:outline-none";
  editor.value = text;
  editor.rows = Math.max(1, text.split("\n").length);
  syncTextareaHeight(editor);
  editor.addEventListener("blur", () => {
    commitEdit(state, blockIndex, editor.value);
  });
  editor.addEventListener("input", () => {
    syncTextareaHeight(editor);
  });
  editor.addEventListener("keydown", (event) => {
    if (event.key === "Escape") {
      event.preventDefault();
      state.activeEdit = null;
      renderBlocks(state);
      return;
    }
    if ((event.metaKey || event.ctrlKey) && event.key === "Enter") {
      event.preventDefault();
      editor.blur();
    }
  });

  blockEl.dataset.editing = "true";
  blockEl.innerHTML = "";
  blockEl.appendChild(editor);
  state.activeEdit = { blockIndex, textarea: editor };
  editor.focus();
  editor.setSelectionRange(text.length, text.length);
  requestAnimationFrame(() => {
    syncTextareaHeight(editor);
  });
}

function commitEdit(state: EditorState, blockIndex: number, value: string): void {
  const block = state.blocks[blockIndex];
  if (!block) {
    return;
  }
  if (value.trim() === "") {
    state.lines.splice(block.start, block.end - block.start + 1);
    if (state.lines.length === 0) {
      state.lines.push("");
    }
  } else {
    const newLines = value.split("\n");
    state.lines.splice(block.start, block.end - block.start + 1, ...newLines);
  }
  state.activeEdit = null;
  syncTextarea(state);
  state.blocks = buildBlocks(state.lines);
  renderBlocks(state);
}

function toggleTask(state: EditorState, checkbox: HTMLInputElement): void {
  const lineIndex = Number(checkbox.dataset.line);
  if (Number.isNaN(lineIndex) || lineIndex < 0 || lineIndex >= state.lines.length) {
    return;
  }
  const line = state.lines[lineIndex];
  const match = line.match(/^(\s*[-*+]\s+)\[( |x|X)\](\s+.*)$/);
  if (!match) {
    return;
  }
  const checked = match[2].toLowerCase() === "x";
  const replacement = `${match[1]}[${checked ? " " : "x"}]${match[3]}`;
  state.lines[lineIndex] = replacement;
  syncTextarea(state);
  state.blocks = buildBlocks(state.lines);
  renderBlocks(state);
}

function syncTextarea(state: EditorState): void {
  state.textarea.value = state.lines.join("\n");
  state.textarea.dispatchEvent(new Event("input", { bubbles: true }));
}

function syncTextareaHeight(textarea: HTMLTextAreaElement, minHeight?: number): void {
  const baseHeight = minHeight ?? 0;
  textarea.style.height = "auto";
  textarea.style.minHeight = "0px";
  const styles = window.getComputedStyle(textarea);
  const lineHeight = Number.parseFloat(styles.lineHeight) || 20;
  const paddingTop = Number.parseFloat(styles.paddingTop) || 0;
  const paddingBottom = Number.parseFloat(styles.paddingBottom) || 0;
  const borderTop = Number.parseFloat(styles.borderTopWidth) || 0;
  const borderBottom = Number.parseFloat(styles.borderBottomWidth) || 0;
  const singleLine = lineHeight + paddingTop + paddingBottom + borderTop + borderBottom;
  const next = Math.max(textarea.scrollHeight, baseHeight, singleLine);
  textarea.style.height = `${next}px`;
}

function findInPath<T extends Element>(event: Event, selector: string): T | null {
  if (typeof event.composedPath !== "function") {
    return null;
  }
  const path = event.composedPath();
  for (const entry of path) {
    if (entry instanceof Element && entry.matches(selector)) {
      return entry as T;
    }
  }
  return null;
}
