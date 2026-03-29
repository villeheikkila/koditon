#!/usr/bin/env python3
"""
generate_public_schema.py

Generates an idempotent public schema snapshot from the current database.
Only includes structural elements (domains, enums, types, tables, indexes, constraints).
Skips functions, views, and triggers.

NOTE: This is a legacy helper and does not represent the full squashed baseline migration.

Usage:
    python3 scripts/schema/generate_public_schema.py > public_schema_snapshot.sql
    python3 scripts/schema/generate_public_schema.py --db-url "postgresql://..." > snapshot.sql
"""

import argparse
import re
import subprocess
import sys
from datetime import datetime, timezone

DEFAULT_DB_URL = "postgresql://postgres:postgres@localhost:5433/koditon"


def get_schema_dump(db_url: str) -> str:
    """Dump the public schema from the database (structure only, no functions)."""
    result = subprocess.run(
        [
            "pg_dump",
            db_url,
            "--schema=public",
            "--schema-only",
            "--no-owner",
            "--no-privileges",
            "--no-tablespaces",
            "--no-security-labels",
            "--no-comments",
        ],
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        print(f"Error running pg_dump: {result.stderr}", file=sys.stderr)
        sys.exit(1)
    return result.stdout


def extract_statements(dump: str) -> list[tuple[str, str]]:
    """
    Extract SQL statements from the dump.
    Returns list of (statement_type, statement_sql) tuples.
    """
    statements = []

    # Remove \restrict line and SET statements
    lines = []
    for line in dump.split("\n"):
        if (
            line.startswith("\\")
            or line.startswith("SET ")
            or line.startswith("SELECT pg_catalog")
        ):
            continue
        lines.append(line)

    content = "\n".join(lines)

    # Extract different statement types using regex
    # Domains
    for match in re.finditer(r"CREATE DOMAIN public\.(\w+)[^;]+;", content, re.DOTALL):
        statements.append(("domain", match.group(0).strip()))

    # Enums
    for match in re.finditer(
        r"CREATE TYPE public\.(\w+)\s+AS\s+ENUM\s*\([^)]+\);", content, re.DOTALL
    ):
        statements.append(("enum", match.group(0).strip()))

    # Composite types
    for match in re.finditer(
        r"CREATE TYPE public\.(\w+)\s+AS\s*\([^;]+\);", content, re.DOTALL
    ):
        if "AS ENUM" not in match.group(0):
            statements.append(("composite_type", match.group(0).strip()))

    # Tables (need to handle multi-line with proper matching)
    # Find CREATE TABLE statements by looking for balanced parentheses
    table_pattern = r"CREATE TABLE public\.(\w+)\s*\("
    for match in re.finditer(table_pattern, content):
        start = match.start()
        table_name = match.group(1)
        # Find the matching closing paren and semicolon
        paren_count = 0
        i = match.end() - 1  # Start at the opening paren
        while i < len(content):
            if content[i] == "(":
                paren_count += 1
            elif content[i] == ")":
                paren_count -= 1
                if paren_count == 0:
                    # Find the semicolon
                    end = content.find(";", i)
                    if end != -1:
                        statements.append(("table", content[start : end + 1].strip()))
                    break
            i += 1

    # Sequences
    for match in re.finditer(
        r"CREATE SEQUENCE public\.(\w+)[^;]*;", content, re.DOTALL
    ):
        statements.append(("sequence", match.group(0).strip()))

    # ALTER SEQUENCE ... OWNED BY
    for match in re.finditer(
        r"ALTER SEQUENCE public\.\w+\s+OWNED BY\s+public\.\w+\.\w+;",
        content,
        re.IGNORECASE,
    ):
        statements.append(("alter_sequence", match.group(0).strip()))

    # Indexes (CREATE INDEX and CREATE UNIQUE INDEX)
    for match in re.finditer(
        r"CREATE (?:UNIQUE )?INDEX\s+\w+\s+ON\s+public\.\w+[^;]+;",
        content,
        re.IGNORECASE,
    ):
        statements.append(("index", match.group(0).strip()))

    # Primary key constraints
    for match in re.finditer(
        r"ALTER TABLE ONLY public\.(\w+)\s+ADD CONSTRAINT\s+(\w+)\s+PRIMARY KEY\s*\([^)]+\);",
        content,
        re.IGNORECASE | re.DOTALL,
    ):
        statements.append(("pk_constraint", match.group(0).strip()))

    # Unique constraints
    for match in re.finditer(
        r"ALTER TABLE ONLY public\.(\w+)\s+ADD CONSTRAINT\s+(\w+)\s+UNIQUE\s*\([^)]+\);",
        content,
        re.IGNORECASE | re.DOTALL,
    ):
        statements.append(("unique_constraint", match.group(0).strip()))

    # Foreign key constraints
    for match in re.finditer(
        r"ALTER TABLE ONLY public\.(\w+)\s+ADD CONSTRAINT\s+(\w+)\s+FOREIGN KEY\s*\([^)]+\)\s+REFERENCES\s+[^;]+;",
        content,
        re.IGNORECASE | re.DOTALL,
    ):
        statements.append(("fk_constraint", match.group(0).strip()))

    # Check constraints
    for match in re.finditer(
        r"ALTER TABLE ONLY public\.(\w+)\s+ADD CONSTRAINT\s+(\w+)\s+CHECK\s*\([^;]+;",
        content,
        re.IGNORECASE | re.DOTALL,
    ):
        statements.append(("check_constraint", match.group(0).strip()))

    return statements


def make_domain_idempotent(sql: str) -> str:
    """Wrap domain creation in DO block for idempotency."""
    match = re.search(r"CREATE DOMAIN public\.(\w+)", sql)
    if not match:
        return sql
    name = match.group(1)
    # Escape single quotes in SQL
    escaped_sql = sql.replace("'", "''")
    return f"""DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = '{name}') THEN
        EXECUTE '{escaped_sql}';
    END IF;
END
$$;"""


def make_enum_idempotent(sql: str) -> str:
    """Wrap enum/type creation in DO block for idempotency."""
    match = re.search(r"CREATE TYPE public\.(\w+)", sql)
    if not match:
        return sql
    name = match.group(1)
    escaped_sql = sql.replace("'", "''")
    return f"""DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = '{name}' AND typnamespace = (SELECT oid FROM pg_namespace WHERE nspname = 'public')) THEN
        EXECUTE '{escaped_sql}';
    END IF;
END
$$;"""


def make_table_idempotent(sql: str) -> str:
    """Add IF NOT EXISTS to CREATE TABLE."""
    return sql.replace("CREATE TABLE public.", "CREATE TABLE IF NOT EXISTS public.")


def make_sequence_idempotent(sql: str) -> str:
    """Add IF NOT EXISTS to CREATE SEQUENCE."""
    return sql.replace(
        "CREATE SEQUENCE public.", "CREATE SEQUENCE IF NOT EXISTS public."
    )


def make_index_idempotent(sql: str) -> str:
    """Add IF NOT EXISTS to CREATE INDEX."""
    sql = re.sub(r"CREATE INDEX ", "CREATE INDEX IF NOT EXISTS ", sql)
    sql = re.sub(r"CREATE UNIQUE INDEX ", "CREATE UNIQUE INDEX IF NOT EXISTS ", sql)
    return sql


def make_constraint_idempotent(sql: str) -> str:
    """Wrap constraint in conditional check."""
    match = re.search(r"ADD CONSTRAINT\s+(\w+)", sql)
    if not match:
        return sql
    constraint_name = match.group(1)

    # Extract table name for schema-qualified check
    table_match = re.search(r"ALTER TABLE ONLY public\.(\w+)", sql)
    table_name = table_match.group(1) if table_match else None

    escaped_sql = sql.replace("'", "''")

    # Use schema-qualified check to avoid conflicts with auth.roles etc.
    if table_name:
        return f"""DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint c
        JOIN pg_class t ON c.conrelid = t.oid
        JOIN pg_namespace n ON t.relnamespace = n.oid
        WHERE c.conname = '{constraint_name}'
          AND t.relname = '{table_name}'
          AND n.nspname = 'public'
    ) THEN
        EXECUTE '{escaped_sql}';
    END IF;
END
$$;"""
    else:
        return f"""DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = '{constraint_name}') THEN
        EXECUTE '{escaped_sql}';
    END IF;
END
$$;"""


def generate_migration(statements: list[tuple[str, str]]) -> str:
    """Generate the full migration SQL."""
    timestamp = datetime.now(timezone.utc).strftime("%Y-%m-%d %H:%M:%S UTC")

    output = [
        f"""-- ============================================================
-- Public Schema Snapshot (Legacy Helper)
-- Generated: {timestamp}
-- ============================================================
--
-- This output contains public schema structural objects only.
--
-- Behavior:
-- - Produces idempotent structural SQL (tables/indexes/constraints)
-- - Excludes functions/views/triggers
--
-- NOTE: Use backend/db/migrations/001_initial.sql for the full baseline migration.
--
-- ============================================================
"""
    ]

    # Group statements by type
    by_type: dict[str, list[str]] = {}
    for stmt_type, sql in statements:
        if stmt_type not in by_type:
            by_type[stmt_type] = []
        by_type[stmt_type].append(sql)

    # Domains
    if "domain" in by_type:
        output.append(
            "\n-- ============================================================"
        )
        output.append("-- PHASE 1: DOMAINS")
        output.append(
            "-- ============================================================\n"
        )
        for sql in by_type["domain"]:
            output.append(make_domain_idempotent(sql))
            output.append("")

    # Enums
    if "enum" in by_type:
        output.append(
            "\n-- ============================================================"
        )
        output.append("-- PHASE 2: ENUMS")
        output.append(
            "-- ============================================================\n"
        )
        for sql in by_type["enum"]:
            output.append(make_enum_idempotent(sql))
            output.append("")

    # Composite types
    if "composite_type" in by_type:
        output.append(
            "\n-- ============================================================"
        )
        output.append("-- PHASE 3: COMPOSITE TYPES")
        output.append(
            "-- ============================================================\n"
        )
        for sql in by_type["composite_type"]:
            output.append(make_enum_idempotent(sql))
            output.append("")

    # Sequences
    if "sequence" in by_type:
        output.append(
            "\n-- ============================================================"
        )
        output.append("-- PHASE 4: SEQUENCES")
        output.append(
            "-- ============================================================\n"
        )
        for sql in by_type["sequence"]:
            output.append(make_sequence_idempotent(sql))
            output.append("")

    # Tables
    if "table" in by_type:
        output.append(
            "\n-- ============================================================"
        )
        output.append("-- PHASE 5: TABLES")
        output.append(
            "-- ============================================================\n"
        )
        for sql in by_type["table"]:
            output.append(make_table_idempotent(sql))
            output.append("")

    # Alter sequences (owned by)
    if "alter_sequence" in by_type:
        output.append(
            "\n-- ============================================================"
        )
        output.append("-- PHASE 6: SEQUENCE OWNERSHIP")
        output.append(
            "-- ============================================================\n"
        )
        for sql in by_type["alter_sequence"]:
            output.append(sql)
            output.append("")

    # Indexes
    if "index" in by_type:
        output.append(
            "\n-- ============================================================"
        )
        output.append("-- PHASE 7: INDEXES")
        output.append(
            "-- ============================================================\n"
        )
        for sql in by_type["index"]:
            output.append(make_index_idempotent(sql))
            output.append("")

    # Primary key constraints
    if "pk_constraint" in by_type:
        output.append(
            "\n-- ============================================================"
        )
        output.append("-- PHASE 8: PRIMARY KEY CONSTRAINTS")
        output.append(
            "-- ============================================================\n"
        )
        for sql in by_type["pk_constraint"]:
            output.append(make_constraint_idempotent(sql))
            output.append("")

    # Unique constraints
    if "unique_constraint" in by_type:
        output.append(
            "\n-- ============================================================"
        )
        output.append("-- PHASE 9: UNIQUE CONSTRAINTS")
        output.append(
            "-- ============================================================\n"
        )
        for sql in by_type["unique_constraint"]:
            output.append(make_constraint_idempotent(sql))
            output.append("")

    # Foreign key constraints
    if "fk_constraint" in by_type:
        output.append(
            "\n-- ============================================================"
        )
        output.append("-- PHASE 10: FOREIGN KEY CONSTRAINTS")
        output.append(
            "-- ============================================================\n"
        )
        for sql in by_type["fk_constraint"]:
            output.append(make_constraint_idempotent(sql))
            output.append("")

    # Check constraints
    if "check_constraint" in by_type:
        output.append(
            "\n-- ============================================================"
        )
        output.append("-- PHASE 11: CHECK CONSTRAINTS")
        output.append(
            "-- ============================================================\n"
        )
        for sql in by_type["check_constraint"]:
            output.append(make_constraint_idempotent(sql))
            output.append("")

    output.append("""
-- ============================================================
-- END OF PUBLIC SCHEMA SNAPSHOT
-- ============================================================
--
-- Notes:
-- 1. Functions are NOT included
-- 2. Views are NOT included
-- 3. Triggers are NOT included
-- 4. Use backend/db/migrations/001_initial.sql for a full baseline migration
--
-- ============================================================
""")

    return "\n".join(output)


def main():
    parser = argparse.ArgumentParser(
        description="Generate idempotent public schema snapshot from current database"
    )
    parser.add_argument(
        "--db-url",
        default=DEFAULT_DB_URL,
        help=f"Database connection URL (default: {DEFAULT_DB_URL})",
    )
    parser.add_argument(
        "--stats-only",
        action="store_true",
        help="Only print statistics, don't generate migration",
    )
    args = parser.parse_args()

    # Get schema dump
    print(
        f"Connecting to: {args.db_url.split('@')[1] if '@' in args.db_url else args.db_url}",
        file=sys.stderr,
    )
    dump = get_schema_dump(args.db_url)

    # Extract statements
    statements = extract_statements(dump)

    # Count by type
    counts: dict[str, int] = {}
    for stmt_type, _ in statements:
        counts[stmt_type] = counts.get(stmt_type, 0) + 1

    print(f"Extracted: {sum(counts.values())} statements", file=sys.stderr)
    for stmt_type, count in sorted(counts.items()):
        print(f"  - {stmt_type}: {count}", file=sys.stderr)

    if args.stats_only:
        return

    # Generate migration
    migration = generate_migration(statements)
    print(migration)


if __name__ == "__main__":
    main()
