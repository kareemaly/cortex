#!/usr/bin/env python3
"""
Migrate tickets from cortex0 (markdown) to cortex (JSON) format.

Usage:
    python ephemeral/migrate_tickets.py <source_project_path>

Example:
    python ephemeral/migrate_tickets.py ~/projects/nau/pcrf-e2e

The script reads from: <source>/tickets/{backlog,progress,review,done}/*.md
And writes to: <source>/.cortex/tickets/{status}/*.json

Note: This script does NOT delete the original tickets folder.
"""

import json
import os
import re
import sys
import uuid
from datetime import datetime
from pathlib import Path
from typing import Optional


def parse_frontmatter(content: str) -> tuple[dict, str]:
    """Extract YAML frontmatter and remaining content."""
    if not content.startswith("---"):
        return {}, content

    # Find the closing ---
    end_match = re.search(r"\n---\n", content[3:])
    if not end_match:
        return {}, content

    frontmatter_text = content[3:end_match.start() + 3]
    remaining = content[end_match.end() + 3 + 1:]  # +1 for the \n after ---

    # Simple YAML parsing for our use case
    frontmatter = {}
    for line in frontmatter_text.strip().split("\n"):
        if ":" in line:
            key, _, value = line.partition(":")
            frontmatter[key.strip()] = value.strip()

    return frontmatter, remaining


def extract_title_and_body(content: str) -> tuple[str, str]:
    """Extract title (first # line) and body (everything after)."""
    lines = content.strip().split("\n")

    title = ""
    body_start = 0

    for i, line in enumerate(lines):
        if line.startswith("# "):
            title = line[2:].strip()
            body_start = i + 1
            break

    # Skip leading empty lines in body
    while body_start < len(lines) and not lines[body_start].strip():
        body_start += 1

    body = "\n".join(lines[body_start:]).strip()
    return title, body


def generate_slug(title: str, max_length: int = 20) -> str:
    """Generate a URL-friendly slug from a title (matching Go implementation)."""
    # Lowercase and replace spaces/underscores with hyphens
    slug = title.lower()
    slug = slug.replace(" ", "-")
    slug = slug.replace("_", "-")

    # Remove non-alphanumeric except hyphens
    slug = re.sub(r"[^a-z0-9-]", "", slug)

    # Collapse multiple hyphens
    slug = re.sub(r"-+", "-", slug)

    # Trim hyphens from ends
    slug = slug.strip("-")

    # Truncate at word boundary
    if len(slug) > max_length:
        truncated = slug[:max_length]
        last_hyphen = truncated.rfind("-")
        if last_hyphen > 0:
            slug = truncated[:last_hyphen]
        else:
            slug = truncated

    return slug or "ticket"


def parse_date(date_str: Optional[str]) -> Optional[datetime]:
    """Parse a date string from frontmatter."""
    if not date_str:
        return None

    # Try common formats
    for fmt in ["%Y-%m-%d", "%Y-%m-%dT%H:%M:%SZ", "%Y-%m-%dT%H:%M:%S"]:
        try:
            return datetime.strptime(date_str, fmt)
        except ValueError:
            continue
    return None


def migrate_ticket(source_path: Path, status: str, dest_dir: Path) -> dict:
    """Migrate a single ticket file."""
    content = source_path.read_text()

    # Parse frontmatter
    frontmatter, remaining = parse_frontmatter(content)

    # Extract title and body
    title, body = extract_title_and_body(remaining)

    if not title:
        # Fallback: use filename
        title = source_path.stem.replace("-", " ").title()

    # Generate ID and slug
    ticket_id = str(uuid.uuid4())
    slug = generate_slug(title)
    short_id = ticket_id[:8]

    # Determine dates
    file_mtime = datetime.fromtimestamp(source_path.stat().st_mtime)

    created_at = parse_date(frontmatter.get("created_at"))
    if not created_at:
        created_at = file_mtime

    updated_at = parse_date(frontmatter.get("updated_at"))
    if not updated_at:
        updated_at = file_mtime

    # Build ticket JSON
    dates = {
        "created": created_at.strftime("%Y-%m-%dT%H:%M:%SZ"),
        "updated": updated_at.strftime("%Y-%m-%dT%H:%M:%SZ"),
    }

    # Set status date based on current status
    if status == "progress":
        dates["progress"] = updated_at.strftime("%Y-%m-%dT%H:%M:%SZ")
    elif status == "review":
        dates["reviewed"] = updated_at.strftime("%Y-%m-%dT%H:%M:%SZ")
    elif status == "done":
        dates["done"] = updated_at.strftime("%Y-%m-%dT%H:%M:%SZ")

    ticket = {
        "id": ticket_id,
        "title": title,
        "body": body,
        "dates": dates,
        "comments": [],
        "session": None,
    }

    # Write to destination
    dest_status_dir = dest_dir / status
    dest_status_dir.mkdir(parents=True, exist_ok=True)

    dest_file = dest_status_dir / f"{slug}-{short_id}.json"
    dest_file.write_text(json.dumps(ticket, indent=2) + "\n")

    return {
        "source": str(source_path),
        "dest": str(dest_file),
        "title": title,
        "status": status,
    }


def migrate_project(project_path: str) -> None:
    """Migrate all tickets from a project."""
    project = Path(project_path).expanduser().resolve()

    source_dir = project / "tickets"
    dest_dir = project / ".cortex" / "tickets"

    if not source_dir.exists():
        print(f"Error: Source tickets directory not found: {source_dir}")
        sys.exit(1)

    print(f"Migrating tickets from: {source_dir}")
    print(f"                    to: {dest_dir}")
    print()

    # Status folders to process
    statuses = ["backlog", "progress", "review", "done"]

    migrated = []
    errors = []

    for status in statuses:
        status_dir = source_dir / status
        if not status_dir.exists():
            continue

        md_files = list(status_dir.glob("*.md"))
        if not md_files:
            continue

        print(f"Processing {status}/ ({len(md_files)} tickets)...")

        for md_file in sorted(md_files):
            try:
                result = migrate_ticket(md_file, status, dest_dir)
                migrated.append(result)
                print(f"  - {md_file.name} -> {Path(result['dest']).name}")
            except Exception as e:
                errors.append({"file": str(md_file), "error": str(e)})
                print(f"  ! {md_file.name}: ERROR - {e}")

    print()
    print("=" * 60)
    print(f"Migration complete: {len(migrated)} tickets migrated")
    if errors:
        print(f"Errors: {len(errors)}")
        for err in errors:
            print(f"  - {err['file']}: {err['error']}")

    print()
    print("Note: Original tickets folder was NOT deleted.")
    print("Please verify the migration and delete it manually if satisfied.")


if __name__ == "__main__":
    if len(sys.argv) != 2:
        print(__doc__)
        sys.exit(1)

    migrate_project(sys.argv[1])
