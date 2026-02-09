import { describe, test, expect } from "bun:test";

// Task line parsing and toggling logic extracted from note_editor.ts
// Tests the regex patterns and state transformations

function parseTaskLine(line: string): { indent: number; checked: boolean; content: string } | null {
  const taskMatch = line.match(/^\s*[-*+]\s+\[( |x|X)\]\s+(.*)$/);
  if (!taskMatch) {
    return null;
  }
  
  const indentMatch = line.match(/^\s*/);
  const indent = indentMatch ? indentMatch[0].length : 0;
  const checked = taskMatch[1].toLowerCase() === "x";
  const content = taskMatch[2];
  
  return { indent, checked, content };
}

function toggleTaskLine(line: string): string | null {
  const match = line.match(/^(\s*[-*+]\s+)\[( |x|X)\](\s+.*)$/);
  if (!match) {
    return null;
  }
  const checked = match[2].toLowerCase() === "x";
  return `${match[1]}[${checked ? " " : "x"}]${match[3]}`;
}

describe("parseTaskLine", () => {
  test("parses unchecked task with dash", () => {
    const result = parseTaskLine("- [ ] Buy milk");
    expect(result).not.toBeNull();
    expect(result?.checked).toBe(false);
    expect(result?.content).toBe("Buy milk");
    expect(result?.indent).toBe(0);
  });

  test("parses checked task with dash", () => {
    const result = parseTaskLine("- [x] Complete project");
    expect(result).not.toBeNull();
    expect(result?.checked).toBe(true);
    expect(result?.content).toBe("Complete project");
  });

  test("parses checked task with capital X", () => {
    const result = parseTaskLine("- [X] Done task");
    expect(result).not.toBeNull();
    expect(result?.checked).toBe(true);
    expect(result?.content).toBe("Done task");
  });

  test("parses task with asterisk", () => {
    const result = parseTaskLine("* [ ] Task with asterisk");
    expect(result).not.toBeNull();
    expect(result?.checked).toBe(false);
    expect(result?.content).toBe("Task with asterisk");
  });

  test("parses task with plus", () => {
    const result = parseTaskLine("+ [x] Task with plus");
    expect(result).not.toBeNull();
    expect(result?.checked).toBe(true);
    expect(result?.content).toBe("Task with plus");
  });

  test("parses indented task", () => {
    const result = parseTaskLine("  - [ ] Indented task");
    expect(result).not.toBeNull();
    expect(result?.indent).toBe(2);
    expect(result?.content).toBe("Indented task");
  });

  test("parses deeply indented task", () => {
    const result = parseTaskLine("      - [ ] Deeply indented");
    expect(result).not.toBeNull();
    expect(result?.indent).toBe(6);
  });

  test("parses tab-indented task", () => {
    const result = parseTaskLine("\t- [ ] Tab indented");
    expect(result).not.toBeNull();
    expect(result?.indent).toBe(1); // Tab is 1 character
  });

  test("returns null for non-task line", () => {
    expect(parseTaskLine("Just regular text")).toBeNull();
    expect(parseTaskLine("- Regular list item")).toBeNull();
    expect(parseTaskLine("[ ] Not a task")).toBeNull();
  });

  test("returns null for malformed checkbox", () => {
    expect(parseTaskLine("- [y] Invalid state")).toBeNull();
    expect(parseTaskLine("- [] Missing space")).toBeNull();
    expect(parseTaskLine("- [  ] Double space")).toBeNull();
  });

  test("handles task with special characters", () => {
    const result = parseTaskLine("- [ ] Buy & sell items");
    expect(result).not.toBeNull();
    expect(result?.content).toBe("Buy & sell items");
  });

  test("handles task with markdown formatting", () => {
    const result = parseTaskLine("- [ ] Read **important** document");
    expect(result).not.toBeNull();
    expect(result?.content).toBe("Read **important** document");
  });

  test("handles task with code backticks", () => {
    const result = parseTaskLine("- [ ] Update `config.json` file");
    expect(result).not.toBeNull();
    expect(result?.content).toBe("Update `config.json` file");
  });

  test("handles task with link", () => {
    const result = parseTaskLine("- [ ] Check [docs](https://example.com)");
    expect(result).not.toBeNull();
    expect(result?.content).toBe("Check [docs](https://example.com)");
  });

  test("handles empty task content", () => {
    const result = parseTaskLine("- [ ] ");
    expect(result).not.toBeNull();
    expect(result?.content).toBe("");
  });

  test("handles task with trailing spaces", () => {
    const result = parseTaskLine("- [ ] Task with spaces   ");
    expect(result).not.toBeNull();
    expect(result?.content).toBe("Task with spaces   ");
  });
});

describe("toggleTaskLine", () => {
  test("toggles unchecked to checked", () => {
    const result = toggleTaskLine("- [ ] Buy milk");
    expect(result).toBe("- [x] Buy milk");
  });

  test("toggles checked to unchecked", () => {
    const result = toggleTaskLine("- [x] Complete project");
    expect(result).toBe("- [ ] Complete project");
  });

  test("toggles capital X to unchecked", () => {
    const result = toggleTaskLine("- [X] Done task");
    expect(result).toBe("- [ ] Done task");
  });

  test("preserves indent when toggling", () => {
    const result = toggleTaskLine("  - [ ] Indented task");
    expect(result).toBe("  - [x] Indented task");
  });

  test("preserves asterisk marker", () => {
    const result = toggleTaskLine("* [ ] Task");
    expect(result).toBe("* [x] Task");
  });

  test("preserves plus marker", () => {
    const result = toggleTaskLine("+ [x] Task");
    expect(result).toBe("+ [ ] Task");
  });

  test("preserves special characters in content", () => {
    const result = toggleTaskLine("- [ ] Buy & sell items");
    expect(result).toBe("- [x] Buy & sell items");
  });

  test("preserves markdown formatting", () => {
    const result = toggleTaskLine("- [ ] Read **important** document");
    expect(result).toBe("- [x] Read **important** document");
  });

  test("preserves multiple spaces after checkbox", () => {
    const result = toggleTaskLine("- [ ]   Task with spaces");
    expect(result).toBe("- [x]   Task with spaces");
  });

  test("returns null for non-task line", () => {
    expect(toggleTaskLine("Regular text")).toBeNull();
    expect(toggleTaskLine("- Regular list")).toBeNull();
  });

  test("handles empty task content", () => {
    const result = toggleTaskLine("- [ ] ");
    expect(result).toBe("- [x] ");
  });

  test("toggle twice returns to original state", () => {
    const original = "- [ ] Task";
    const toggled = toggleTaskLine(original);
    expect(toggled).toBe("- [x] Task");
    const toggledBack = toggleTaskLine(toggled!);
    expect(toggledBack).toBe(original);
  });
});

describe("task line edge cases", () => {
  test("task with no space before checkbox", () => {
    // This is malformed according to markdown spec
    expect(parseTaskLine("-[ ] Invalid")).toBeNull();
  });

  test("task with extra space in checkbox", () => {
    // [ x] with space before x is technically wrong
    const result = parseTaskLine("- [ x] Invalid spacing");
    expect(result).toBeNull();
  });

  test("task with lowercase x surrounded by spaces", () => {
    // Valid according to our regex
    const result = parseTaskLine("- [x] Valid task");
    expect(result).not.toBeNull();
    expect(result?.checked).toBe(true);
  });

  test("very long task content", () => {
    const longContent = "A".repeat(1000);
    const line = `- [ ] ${longContent}`;
    const result = parseTaskLine(line);
    expect(result).not.toBeNull();
    expect(result?.content).toBe(longContent);
  });

  test("task with emoji", () => {
    const result = parseTaskLine("- [ ] 🎉 Celebrate!");
    expect(result).not.toBeNull();
    expect(result?.content).toBe("🎉 Celebrate!");
  });

  test("task with unicode", () => {
    const result = parseTaskLine("- [ ] 日本語のタスク");
    expect(result).not.toBeNull();
    expect(result?.content).toBe("日本語のタスク");
  });

  test("nested task-like content", () => {
    const result = parseTaskLine("- [ ] Review PR: - [ ] Check tests");
    expect(result).not.toBeNull();
    expect(result?.content).toBe("Review PR: - [ ] Check tests");
  });

  test("task with HTML-like content", () => {
    const result = parseTaskLine("- [ ] Fix <Component> rendering");
    expect(result).not.toBeNull();
    expect(result?.content).toBe("Fix <Component> rendering");
  });

  test("task with script tag (XSS concern)", () => {
    const result = parseTaskLine("- [ ] <script>alert('xss')</script>");
    expect(result).not.toBeNull();
    // Content is parsed but should be sanitized when rendered
    expect(result?.content).toBe("<script>alert('xss')</script>");
  });
});

describe("task list stress tests", () => {
  test("100 rapid toggles maintain consistency", () => {
    let line = "- [ ] Task";
    for (let i = 0; i < 100; i++) {
      const toggled = toggleTaskLine(line);
      expect(toggled).not.toBeNull();
      line = toggled!;
    }
    // After 100 toggles (even number), should be back to unchecked
    expect(line).toBe("- [ ] Task");
  });

  test("deeply nested indent levels", () => {
    const indent = " ".repeat(40);
    const line = `${indent}- [ ] Deep task`;
    const result = parseTaskLine(line);
    expect(result).not.toBeNull();
    expect(result?.indent).toBe(40);
  });

  test("all valid checkbox states", () => {
    const states = [" ", "x", "X"];
    for (const state of states) {
      const line = `- [${state}] Task`;
      const result = parseTaskLine(line);
      expect(result).not.toBeNull();
    }
  });

  test("all list markers work", () => {
    const markers = ["-", "*", "+"];
    for (const marker of markers) {
      const line = `${marker} [ ] Task`;
      const result = parseTaskLine(line);
      expect(result).not.toBeNull();
      expect(result?.checked).toBe(false);
    }
  });
});
