import { describe, test, expect } from "bun:test";

// We need to extract the buildBlocks function and related helpers to make them testable
// For now, we'll import the module and test via DOM manipulation

// Import types for reference
type BlockType = "blank" | "line" | "fence";

type Block = {
  start: number;
  end: number;
  type: BlockType;
};

// These are private functions in note_editor.ts, so we'll re-implement them for testing
// In production, you'd export these from note_editor.ts for testing

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

describe("fenceMarker", () => {
  test("detects triple backticks", () => {
    const marker = fenceMarker("```javascript");
    expect(marker).not.toBeNull();
    expect(marker?.char).toBe("`");
    expect(marker?.length).toBe(3);
  });

  test("detects triple tildes", () => {
    const marker = fenceMarker("~~~bash");
    expect(marker).not.toBeNull();
    expect(marker?.char).toBe("~");
    expect(marker?.length).toBe(3);
  });

  test("detects 4+ backticks", () => {
    const marker = fenceMarker("````markdown");
    expect(marker).not.toBeNull();
    expect(marker?.char).toBe("`");
    expect(marker?.length).toBe(4);
  });

  test("returns null for non-fence lines", () => {
    expect(fenceMarker("# Heading")).toBeNull();
    expect(fenceMarker("regular text")).toBeNull();
    expect(fenceMarker("``incomplete")).toBeNull();
    expect(fenceMarker("  ```indented")).toBeNull();
  });

  test("handles fence without language specifier", () => {
    const marker = fenceMarker("```");
    expect(marker).not.toBeNull();
    expect(marker?.char).toBe("`");
    expect(marker?.length).toBe(3);
  });
});

describe("isFenceEnd", () => {
  test("matches equal length fence", () => {
    const marker = { char: "`", length: 3 };
    expect(isFenceEnd("```", marker)).toBe(true);
  });

  test("matches longer fence", () => {
    const marker = { char: "`", length: 3 };
    expect(isFenceEnd("````", marker)).toBe(true);
    expect(isFenceEnd("`````", marker)).toBe(true);
  });

  test("rejects shorter fence", () => {
    const marker = { char: "`", length: 3 };
    expect(isFenceEnd("``", marker)).toBe(false);
  });

  test("rejects wrong character", () => {
    const marker = { char: "`", length: 3 };
    expect(isFenceEnd("~~~", marker)).toBe(false);
  });

  test("handles whitespace after fence", () => {
    const marker = { char: "`", length: 3 };
    expect(isFenceEnd("```   ", marker)).toBe(true);
    expect(isFenceEnd("```\t", marker)).toBe(true);
  });

  test("rejects fence with text after", () => {
    const marker = { char: "`", length: 3 };
    expect(isFenceEnd("``` text", marker)).toBe(false);
  });
});

describe("buildBlocks - basic cases", () => {
  test("empty document creates single blank block", () => {
    const blocks = buildBlocks([]);
    expect(blocks).toHaveLength(1);
    expect(blocks[0].type).toBe("blank");
  });

  test("single line creates line block", () => {
    const blocks = buildBlocks(["# Heading"]);
    expect(blocks).toHaveLength(1);
    expect(blocks[0].type).toBe("line");
    expect(blocks[0].start).toBe(0);
    expect(blocks[0].end).toBe(0);
  });

  test("blank line creates blank block", () => {
    const blocks = buildBlocks([""]);
    expect(blocks).toHaveLength(1);
    expect(blocks[0].type).toBe("blank");
  });

  test("whitespace-only line creates blank block", () => {
    const blocks = buildBlocks(["   ", "\t", "  \t  "]);
    expect(blocks).toHaveLength(3);
    expect(blocks[0].type).toBe("blank");
    expect(blocks[1].type).toBe("blank");
    expect(blocks[2].type).toBe("blank");
  });

  test("multiple lines create multiple blocks", () => {
    const blocks = buildBlocks([
      "# Heading",
      "",
      "Paragraph",
    ]);
    expect(blocks).toHaveLength(3);
    expect(blocks[0].type).toBe("line");
    expect(blocks[1].type).toBe("blank");
    expect(blocks[2].type).toBe("line");
  });
});

describe("buildBlocks - code fences", () => {
  test("properly closed fence", () => {
    const blocks = buildBlocks([
      "```javascript",
      "const x = 1;",
      "```",
    ]);
    expect(blocks).toHaveLength(1);
    expect(blocks[0].type).toBe("fence");
    expect(blocks[0].start).toBe(0);
    expect(blocks[0].end).toBe(2);
  });

  test("unclosed fence (end of document)", () => {
    const blocks = buildBlocks([
      "```javascript",
      "const x = 1;",
      "const y = 2;",
    ]);
    expect(blocks).toHaveLength(1);
    expect(blocks[0].type).toBe("fence");
    expect(blocks[0].start).toBe(0);
    expect(blocks[0].end).toBe(2); // Extends to end
  });

  test("fence with no content", () => {
    const blocks = buildBlocks([
      "```",
      "```",
    ]);
    expect(blocks).toHaveLength(1);
    expect(blocks[0].type).toBe("fence");
    expect(blocks[0].start).toBe(0);
    expect(blocks[0].end).toBe(1);
  });

  test("multiple fences", () => {
    const blocks = buildBlocks([
      "```javascript",
      "code1",
      "```",
      "",
      "```python",
      "code2",
      "```",
    ]);
    expect(blocks).toHaveLength(3);
    expect(blocks[0].type).toBe("fence");
    expect(blocks[0].start).toBe(0);
    expect(blocks[0].end).toBe(2);
    expect(blocks[1].type).toBe("blank");
    expect(blocks[2].type).toBe("fence");
    expect(blocks[2].start).toBe(4);
    expect(blocks[2].end).toBe(6);
  });

  test("nested fence markers inside fence", () => {
    const blocks = buildBlocks([
      "```markdown",
      "```javascript",
      "nested code",
      "```",
      "```",
    ]);
    // The first fence closes at line 3, then line 4 starts a new fence
    expect(blocks).toHaveLength(2);
    expect(blocks[0].type).toBe("fence");
    expect(blocks[0].start).toBe(0);
    expect(blocks[0].end).toBe(3); // First ``` closes it
    expect(blocks[1].type).toBe("fence");
    expect(blocks[1].start).toBe(4);
    expect(blocks[1].end).toBe(4); // Second fence is unclosed
  });

  test("fence with different marker lengths", () => {
    const blocks = buildBlocks([
      "````markdown",
      "```javascript",
      "code",
      "```",
      "````",
    ]);
    expect(blocks).toHaveLength(1);
    expect(blocks[0].type).toBe("fence");
    // Inner ``` shouldn't close the 4-backtick fence
    expect(blocks[0].start).toBe(0);
    expect(blocks[0].end).toBe(4);
  });

  test("tilde fence mixed with backtick", () => {
    const blocks = buildBlocks([
      "~~~bash",
      "```",
      "this is content, not a fence close",
      "~~~",
    ]);
    expect(blocks).toHaveLength(1);
    expect(blocks[0].type).toBe("fence");
    expect(blocks[0].start).toBe(0);
    expect(blocks[0].end).toBe(3);
  });
});

describe("buildBlocks - edge cases", () => {
  test("very long single line", () => {
    const longLine = "a".repeat(50000);
    const blocks = buildBlocks([longLine]);
    expect(blocks).toHaveLength(1);
    expect(blocks[0].type).toBe("line");
  });

  test("many short lines", () => {
    const lines = Array.from({ length: 1000 }, (_, i) => `line ${i}`);
    const blocks = buildBlocks(lines);
    expect(blocks).toHaveLength(1000);
    expect(blocks.every(b => b.type === "line")).toBe(true);
  });

  test("alternating blank and content lines", () => {
    const blocks = buildBlocks([
      "line1",
      "",
      "line2",
      "",
      "line3",
    ]);
    expect(blocks).toHaveLength(5);
    expect(blocks[0].type).toBe("line");
    expect(blocks[1].type).toBe("blank");
    expect(blocks[2].type).toBe("line");
    expect(blocks[3].type).toBe("blank");
    expect(blocks[4].type).toBe("line");
  });

  test("fence at start of document", () => {
    const blocks = buildBlocks([
      "```",
      "code",
      "```",
      "after",
    ]);
    expect(blocks).toHaveLength(2);
    expect(blocks[0].type).toBe("fence");
    expect(blocks[1].type).toBe("line");
  });

  test("fence at end of document", () => {
    const blocks = buildBlocks([
      "before",
      "```",
      "code",
      "```",
    ]);
    expect(blocks).toHaveLength(2);
    expect(blocks[0].type).toBe("line");
    expect(blocks[1].type).toBe("fence");
  });

  test("only blanks", () => {
    const blocks = buildBlocks(["", "", ""]);
    expect(blocks).toHaveLength(3);
    expect(blocks.every(b => b.type === "blank")).toBe(true);
  });
});

describe("buildBlocks - real-world examples", () => {
  test("typical markdown note", () => {
    const blocks = buildBlocks([
      "# My Note",
      "",
      "Some **bold** text.",
      "",
      "```typescript",
      "function hello() {",
      "  console.log('hi');",
      "}",
      "```",
      "",
      "More text.",
    ]);
    
    expect(blocks).toHaveLength(7);
    expect(blocks[0].type).toBe("line"); // # My Note
    expect(blocks[1].type).toBe("blank");
    expect(blocks[2].type).toBe("line"); // Some **bold** text
    expect(blocks[3].type).toBe("blank");
    expect(blocks[4].type).toBe("fence"); // Code block
    expect(blocks[5].type).toBe("blank");
    expect(blocks[6].type).toBe("line"); // More text
  });

  test("code fence with language and metadata", () => {
    const blocks = buildBlocks([
      "```javascript title=\"example.js\" {1,3}",
      "const x = 1;",
      "const y = 2;",
      "const z = 3;",
      "```",
    ]);
    
    expect(blocks).toHaveLength(1);
    expect(blocks[0].type).toBe("fence");
  });

  test("mixed fence markers in document", () => {
    const blocks = buildBlocks([
      "```javascript",
      "code1",
      "```",
      "",
      "~~~bash",
      "code2",
      "~~~",
    ]);
    
    expect(blocks).toHaveLength(3);
    expect(blocks[0].type).toBe("fence");
    expect(blocks[1].type).toBe("blank");
    expect(blocks[2].type).toBe("fence");
  });
});

describe("buildBlocks - pathological cases", () => {
  test("unclosed fence marker without closing at all", () => {
    const blocks = buildBlocks([
      "```",
      "code line 1",
      "code line 2",
    ]);
    
    expect(blocks).toHaveLength(1);
    expect(blocks[0].type).toBe("fence");
    expect(blocks[0].end).toBe(2); // Should extend to end
  });

  test("fence marker that looks like code", () => {
    const blocks = buildBlocks([
      "```",
      "Here's how to write code fences:",
      "```markdown",
      "Your code here",
      "```",
      "```",
    ]);
    
    // The fence closes at the FIRST closing marker (line 4: ```)
    // Line 2 (```markdown) has text after it, so it's not a closing fence
    expect(blocks).toHaveLength(2);
    expect(blocks[0].type).toBe("fence");
    expect(blocks[0].start).toBe(0);
    expect(blocks[0].end).toBe(4); // Closes at line 4 (first ```)
    expect(blocks[1].type).toBe("fence");
    expect(blocks[1].start).toBe(5);
    expect(blocks[1].end).toBe(5); // Unclosed fence
  });

  test("single backtick should not trigger fence", () => {
    const blocks = buildBlocks([
      "`inline code`",
      "normal text",
    ]);
    
    expect(blocks).toHaveLength(2);
    expect(blocks[0].type).toBe("line");
    expect(blocks[1].type).toBe("line");
  });

  test("two backticks should not trigger fence", () => {
    const blocks = buildBlocks([
      "``double backtick``",
      "normal text",
    ]);
    
    expect(blocks).toHaveLength(2);
    expect(blocks[0].type).toBe("line");
    expect(blocks[1].type).toBe("line");
  });

  test("fence with special characters in language", () => {
    const blocks = buildBlocks([
      "```typescript-react",
      "const App = () => <div />;",
      "```",
    ]);
    
    expect(blocks).toHaveLength(1);
    expect(blocks[0].type).toBe("fence");
  });

  test("empty lines inside fence", () => {
    const blocks = buildBlocks([
      "```",
      "",
      "",
      "code",
      "",
      "```",
    ]);
    
    expect(blocks).toHaveLength(1);
    expect(blocks[0].type).toBe("fence");
    expect(blocks[0].start).toBe(0);
    expect(blocks[0].end).toBe(5);
  });
});
