#!/usr/bin/env python3
"""QualiGuard Python analyzer — outputs FileAnalysis JSON to stdout."""

from __future__ import annotations

import ast
import io
import json
import sys
import tokenize
from dataclasses import dataclass, field
from typing import Any


@dataclass
class Scope:
    kind: str
    name: str
    assigns: dict[str, int] = field(default_factory=dict)
    uses: set[str] = field(default_factory=set)


class Analyzer(ast.NodeVisitor):
    def __init__(self, source: str, path: str) -> None:
        self.source = source
        self.path = path
        self.lines = source.splitlines()
        self.imports: list[dict[str, Any]] = []
        self.functions: list[dict[str, Any]] = []
        self.assignments: list[dict[str, Any]] = []
        self.except_blocks: list[dict[str, Any]] = []
        self.calls: list[dict[str, Any]] = []
        self.strings: list[dict[str, Any]] = []
        self.secrets: list[dict[str, Any]] = []
        self.scopes: list[Scope] = [Scope("module", "<module>")]
        self.function_nodes: list[ast.FunctionDef | ast.AsyncFunctionDef] = []

    @property
    def current(self) -> Scope:
        return self.scopes[-1]

    def visit_Import(self, node: ast.Import) -> None:
        for alias in node.names:
            name = alias.asname or alias.name.split(".")[0]
            self.imports.append(
                {"name": alias.name, "alias": alias.asname or "", "line": node.lineno, "used": False}
            )
            self.current.uses.add(name)
        self.generic_visit(node)

    def visit_ImportFrom(self, node: ast.ImportFrom) -> None:
        module = node.module or ""
        for alias in node.names:
            if alias.name == "*":
                continue
            name = alias.asname or alias.name
            full = f"{module}.{alias.name}" if module else alias.name
            self.imports.append(
                {"name": full, "alias": alias.asname or "", "line": node.lineno, "used": False}
            )
            self.current.uses.add(name)
        self.generic_visit(node)

    def visit_FunctionDef(self, node: ast.FunctionDef | ast.AsyncFunctionDef) -> None:
        self.function_nodes.append(node)
        end_line = getattr(node, "end_lineno", node.lineno)
        params = [arg.arg for arg in node.args.args if arg.arg not in {"self", "cls"}]
        complexity = complexity_for(node)
        self.functions.append(
            {
                "name": node.name,
                "line": node.lineno,
                "end_line": end_line,
                "complexity": complexity,
                "param_count": len(params),
            }
        )

        self.scopes.append(Scope("function", node.name))
        for arg in node.args.args:
            self.current.assigns[arg.arg] = node.lineno
        self.generic_visit(node)
        self._flush_function_assigns(node.name)
        self.scopes.pop()

    visit_AsyncFunctionDef = visit_FunctionDef

    def visit_Assign(self, node: ast.Assign) -> None:
        for target in node.targets:
            for name in extract_names(target):
                self.current.assigns[name] = node.lineno
                if looks_like_secret_name(name) and isinstance(node.value, ast.Constant) and isinstance(node.value.value, str):
                    self.secrets.append({"name": name, "line": node.lineno})
        self.generic_visit(node)

    def visit_AnnAssign(self, node: ast.AnnAssign) -> None:
        for name in extract_names(node.target):
            self.current.assigns[name] = node.lineno
            if looks_like_secret_name(name) and isinstance(node.value, ast.Constant) and isinstance(node.value.value, str):
                self.secrets.append({"name": name, "line": node.lineno})
        self.generic_visit(node)

    def visit_For(self, node: ast.For) -> None:
        for name in extract_names(node.target):
            self.current.assigns[name] = node.lineno
        self.generic_visit(node)

    def visit_Name(self, node: ast.Name) -> None:
        if isinstance(node.ctx, ast.Load):
            self.current.uses.add(node.id)
        self.generic_visit(node)

    def visit_ExceptHandler(self, node: ast.ExceptHandler) -> None:
        bare = node.type is None
        empty = len(node.body) == 0 or (
            len(node.body) == 1 and isinstance(node.body[0], ast.Pass)
        )
        self.except_blocks.append({"line": node.lineno, "bare": bare, "empty": empty})
        self.generic_visit(node)

    def visit_Call(self, node: ast.Call) -> None:
        func_name = call_name(node.func)
        has_user_input = expression_has_user_input(node)
        is_fstring = False
        dynamic_sql = False
        variable_arg = False
        if node.args:
            is_fstring = contains_fstring(node.args[0])
            dynamic_sql = is_dynamic_sql(node.args[0])
            variable_arg = contains_variable(node.args[0])
        self.calls.append(
            {
                "func": func_name,
                "line": node.lineno,
                "has_user_input": has_user_input,
                "is_fstring": is_fstring,
                "dynamic_sql": dynamic_sql,
                "variable_arg": variable_arg,
            }
        )
        self.generic_visit(node)

    def visit_Constant(self, node: ast.Constant) -> None:
        if isinstance(node.value, str):
            self.strings.append(
                {"value": node.value, "line": node.lineno, "kind": "constant"}
            )
        self.generic_visit(node)

    def _flush_function_assigns(self, scope_name: str) -> None:
        scope = self.current
        for name, line in scope.assigns.items():
            if name.startswith("_"):
                continue
            self.assignments.append(
                {
                    "name": name,
                    "line": line,
                    "used": name in scope.uses,
                    "scope": scope_name,
                }
            )


def looks_like_secret_name(name: str) -> bool:
    lower = name.lower()
    hints = ("password", "passwd", "pwd", "secret", "token", "api_key", "apikey", "api")
    return any(h in lower for h in hints)


def extract_names(node: ast.AST) -> list[str]:
    names: list[str] = []
    if isinstance(node, ast.Name):
        names.append(node.id)
    elif isinstance(node, ast.Tuple):
        for elt in node.elts:
            names.extend(extract_names(elt))
    return names


def call_name(node: ast.AST) -> str:
    if isinstance(node, ast.Name):
        return node.id
    if isinstance(node, ast.Attribute):
        base = call_name(node.value)
        if base:
            return f"{base}.{node.attr}"
        return node.attr
    return ""


def contains_fstring(node: ast.AST) -> bool:
    if isinstance(node, ast.JoinedStr):
        return True
    if isinstance(node, ast.BinOp) and isinstance(node.op, ast.Add):
        return contains_fstring(node.left) or contains_fstring(node.right)
    return False


def is_dynamic_sql(node: ast.AST) -> bool:
    if isinstance(node, ast.JoinedStr):
        return True
    if isinstance(node, ast.BinOp) and isinstance(node.op, ast.Add):
        return True
    if isinstance(node, ast.Name):
        return True
    if isinstance(node, ast.Call):
        return True
    return False


def contains_variable(node: ast.AST) -> bool:
    found = False

    class Visitor(ast.NodeVisitor):
        def visit_Name(self, n: ast.Name) -> None:
            nonlocal found
            if isinstance(n.ctx, ast.Load):
                found = True

    Visitor().visit(node)
    return found


def is_shell_command(func_name: str) -> bool:
    name = func_name.lower()
    return name in {"os.system", "os.popen", "subprocess.call", "subprocess.run", "subprocess.popen"}


USER_INPUT_HINTS = {
    "input",
    "request",
    "request.args",
    "request.form",
    "request.json",
    "request.values",
    "request.get_json",
    "sys.argv",
    "argv",
}


def expression_has_user_input(node: ast.AST) -> bool:
    found = False

    class Visitor(ast.NodeVisitor):
        def visit_Call(self, n: ast.Call) -> None:
            nonlocal found
            name = call_name(n.func)
            if name in USER_INPUT_HINTS or name.endswith(".get"):
                found = True
            self.generic_visit(n)

        def visit_Name(self, n: ast.Name) -> None:
            nonlocal found
            if n.id in {"request", "argv"}:
                found = True
            self.generic_visit(n)

    Visitor().visit(node)
    return found


def complexity_for(node: ast.AST) -> int:
    total = 1

    class Visitor(ast.NodeVisitor):
        def visit_If(self, n: ast.If) -> None:
            nonlocal total
            total += 1
            self.generic_visit(n)

        def visit_For(self, n: ast.For) -> None:
            nonlocal total
            total += 1
            self.generic_visit(n)

        def visit_While(self, n: ast.While) -> None:
            nonlocal total
            total += 1
            self.generic_visit(n)

        def visit_ExceptHandler(self, n: ast.ExceptHandler) -> None:
            nonlocal total
            total += 1
            self.generic_visit(n)

        def visit_BoolOp(self, n: ast.BoolOp) -> None:
            nonlocal total
            total += max(0, len(n.values) - 1)
            self.generic_visit(n)

        def visit_comprehension(self, n: ast.comprehension) -> None:
            nonlocal total
            total += 1
            self.generic_visit(n)

        def visit_ListComp(self, n: ast.ListComp) -> None:
            for gen in n.generators:
                self.visit_comprehension(gen)
            self.generic_visit(n)

        def visit_SetComp(self, n: ast.SetComp) -> None:
            for gen in n.generators:
                self.visit_comprehension(gen)
            self.generic_visit(n)

        def visit_DictComp(self, n: ast.DictComp) -> None:
            for gen in n.generators:
                self.visit_comprehension(gen)
            self.generic_visit(n)

        def visit_GeneratorExp(self, n: ast.GeneratorExp) -> None:
            for gen in n.generators:
                self.visit_comprehension(gen)
            self.generic_visit(n)

    Visitor().visit(node)
    return total


def count_ncloc(source: str) -> int:
    count = 0
    try:
        tokens = tokenize.generate_tokens(io.StringIO(source).readline)
        prev = tokenize.INDENT
        for tok in tokens:
            if tok.type == tokenize.COMMENT:
                continue
            if tok.type in {tokenize.NL, tokenize.NEWLINE}:
                if prev not in {tokenize.NL, tokenize.NEWLINE}:
                    count += 1
            prev = tok.type
    except tokenize.TokenError:
        return max(1, len([line for line in source.splitlines() if line.strip()]))
    return count


def mark_used_imports(tree: ast.AST, imports: list[dict[str, Any]]) -> None:
    used_names: set[str] = set()

    class Visitor(ast.NodeVisitor):
        def visit_Name(self, node: ast.Name) -> None:
            used_names.add(node.id)

    Visitor().visit(tree)

    for item in imports:
        alias = item.get("alias") or item["name"].split(".")[-1]
        base = item["name"].split(".")[0]
        item["used"] = alias in used_names or base in used_names


def apply_tree_analysis(
    result: dict[str, Any], source: str, path: str, tree: ast.AST, line_offset: int = 0
) -> None:
    analyzer = Analyzer(source, path)
    analyzer.visit(tree)
    mark_used_imports(tree, analyzer.imports)

    def shift(items: list[dict[str, Any]]) -> list[dict[str, Any]]:
        if line_offset == 0:
            return items
        shifted: list[dict[str, Any]] = []
        for item in items:
            copy = dict(item)
            if "line" in copy:
                copy["line"] = int(copy["line"]) + line_offset
            if "end_line" in copy:
                copy["end_line"] = int(copy["end_line"]) + line_offset
            shifted.append(copy)
        return shifted

    result["imports"].extend(shift(analyzer.imports))
    result["functions"].extend(shift(analyzer.functions))
    result["assignments"].extend(shift(analyzer.assignments))
    result["except_blocks"].extend(shift(analyzer.except_blocks))
    result["calls"].extend(shift(analyzer.calls))
    result["strings"].extend(shift(analyzer.strings))
    result["secrets"].extend(shift(analyzer.secrets))


def try_parse_chunk(chunk: str, path: str) -> ast.AST | None:
    if not chunk.strip():
        return None
    try:
        return ast.parse(chunk + "\n", filename=path)
    except SyntaxError:
        pass

    stripped = chunk.rstrip()
    if stripped.endswith(":"):
        indent = "    "
        if stripped.startswith("class ") or stripped.startswith("def ") or stripped.startswith("async "):
            try:
                return ast.parse(chunk + "\n" + indent + "pass\n", filename=path)
            except SyntaxError:
                pass
    return None


def partial_analyze(source: str, path: str, result: dict[str, Any]) -> None:
    lines = source.splitlines()
    offset = 0

    while offset < len(lines):
        best_tree: ast.AST | None = None
        best_end = offset

        for end in range(offset + 1, len(lines) + 1):
            chunk = "\n".join(lines[offset:end])
            if not chunk.strip():
                best_end = end
                continue
            tree = try_parse_chunk(chunk, path)
            if tree is not None:
                best_tree = tree
                best_end = end
            elif best_tree is not None:
                break

        if best_tree is None:
            offset += 1
            continue

        apply_tree_analysis(result, source, path, best_tree, line_offset=offset)
        if best_end <= offset:
            offset += 1
        else:
            offset = best_end


def analyze_file(path: str) -> dict[str, Any]:
    with open(path, encoding="utf-8") as fh:
        source = fh.read()

    result: dict[str, Any] = {
        "file": path,
        "ncloc": count_ncloc(source),
        "imports": [],
        "functions": [],
        "assignments": [],
        "except_blocks": [],
        "calls": [],
        "strings": [],
        "secrets": [],
    }

    try:
        tree = ast.parse(source, filename=path)
    except SyntaxError as exc:
        result["parse_error"] = {
            "message": exc.msg or str(exc),
            "line": exc.lineno or 1,
            "column": exc.offset or 0,
        }
        partial_analyze(source, path, result)
        return result

    apply_tree_analysis(result, source, path, tree)
    return result


def main() -> None:
    if len(sys.argv) != 2:
        print("usage: analyzer.py <file.py>", file=sys.stderr)
        sys.exit(2)
    try:
        print(json.dumps(analyze_file(sys.argv[1])))
    except Exception as exc:  # noqa: BLE001
        print(json.dumps({"error": str(exc)}), file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
