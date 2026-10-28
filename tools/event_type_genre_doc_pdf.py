#!/usr/bin/env python3
"""Build a PDF overview of kulturbytes event types and genres."""

import argparse
from datetime import datetime, timezone
from html import escape
from pathlib import Path

import requests
from reportlab.lib import colors
from reportlab.lib.enums import TA_LEFT
from reportlab.lib.pagesizes import A4
from reportlab.lib.styles import getSampleStyleSheet, ParagraphStyle
from reportlab.lib.units import mm
from reportlab.platypus import (
    SimpleDocTemplate,
    Paragraph,
    Spacer,
    Table,
    TableStyle,
    PageBreak,
    KeepTogether,
)

API_URL = "https://api.kulturbytes.de/api/event/type-genre-lookup"

BLUE = colors.HexColor("#1a71e4")
LIGHT_BLUE = colors.HexColor("#eef4ff")
GREY = colors.HexColor("#64748b")
BORDER = colors.HexColor("#d8dee8")


def fetch_lookup(lang: str) -> tuple[dict, dict]:
    """Fetch the lookup and return the localized types and API metadata."""
    response = requests.get(
        API_URL,
        params={"lang": lang},
        timeout=30,
        headers={"Accept": "application/json"},
    )
    response.raise_for_status()

    payload = response.json()

    if payload.get("status", 200) != 200:
        raise RuntimeError(f"API returned status {payload.get('status')}")

    data = payload.get("data", payload)

    # Support either a language-specific response or a response containing
    # all languages under data[lang].
    if lang in data and isinstance(data[lang], dict):
        data = data[lang]

    types = data.get("types")
    if not isinstance(types, dict):
        raise ValueError(
            "Could not find the event types in the API response. "
            f"Response keys: {list(data.keys())}"
        )

    return types, payload.get("metadata", {})


def make_styles():
    styles = getSampleStyleSheet()

    styles.add(ParagraphStyle(
        name="CoverTitle",
        parent=styles["Title"],
        fontName="Helvetica-Bold",
        fontSize=27,
        leading=32,
        textColor=BLUE,
        alignment=TA_LEFT,
        spaceAfter=8,
    ))

    styles.add(ParagraphStyle(
        name="Subtitle",
        parent=styles["Normal"],
        fontSize=11,
        leading=16,
        textColor=GREY,
        spaceAfter=8,
    ))

    styles.add(ParagraphStyle(
        name="SectionTitle",
        parent=styles["Heading1"],
        fontSize=18,
        leading=22,
        textColor=BLUE,
        spaceBefore=5,
        spaceAfter=8,
    ))

    styles.add(ParagraphStyle(
        name="TypeTitle",
        parent=styles["Heading2"],
        fontSize=12,
        leading=15,
        textColor=BLUE,
        spaceBefore=6,
        spaceAfter=3,
    ))

    styles.add(ParagraphStyle(
        name="Genre",
        parent=styles["BodyText"],
        fontSize=8.5,
        leading=11,
        spaceAfter=1,
    ))

    styles.add(ParagraphStyle(
        name="SmallText",
        parent=styles["Normal"],
        fontSize=8,
        leading=11,
        textColor=GREY,
    ))

    return styles


def add_footer(canvas, doc):
    canvas.saveState()
    canvas.setStrokeColor(BORDER)
    canvas.line(18 * mm, 14 * mm, 192 * mm, 14 * mm)

    canvas.setFont("Helvetica", 8)
    canvas.setFillColor(GREY)
    canvas.drawString(
        18 * mm, 9 * mm, "kulturbytes | Event types and genres"
    )
    canvas.drawRightString(
        192 * mm, 9 * mm, f"Page {doc.page}"
    )
    canvas.restoreState()


def build_pdf(lang: str, output: Path):
    types, metadata = fetch_lookup(lang)
    styles = make_styles()

    total_genres = sum(
        len(item.get("genres", {}))
        for item in types.values()
    )
    types_with_genres = sum(
        bool(item.get("genres"))
        for item in types.values()
    )

    doc = SimpleDocTemplate(
        str(output),
        pagesize=A4,
        rightMargin=18 * mm,
        leftMargin=18 * mm,
        topMargin=17 * mm,
        bottomMargin=20 * mm,
        title="kulturbytes - Event Types and Genres",
        author="kulturbytes API",
    )

    story = []

    # Cover and introduction
    story.append(Spacer(1, 15 * mm))
    story.append(Paragraph(
        "Event Types<br/>and Genres",
        styles["CoverTitle"],
    ))
    story.append(Paragraph(
        f"kulturbytes API structure overview | Language: {lang}",
        styles["Subtitle"],
    ))
    story.append(Spacer(1, 4 * mm))

    story.append(Paragraph(
        "This document visualizes the event classification used by "
        "kulturbytes. Each event type represents a top-level category. "
        "Where available, genres provide more specific classifications "
        "within that category.",
        styles["BodyText"],
    ))
    story.append(Spacer(1, 7 * mm))

    # Summary statistics
    summary = Table(
        [
            [
                Paragraph("<b>Event types</b>", styles["SmallText"]),
                Paragraph("<b>Types with genres</b>", styles["SmallText"]),
                Paragraph("<b>Genre entries</b>", styles["SmallText"]),
            ],
            [
                str(len(types)),
                str(types_with_genres),
                str(total_genres),
            ],
        ],
        colWidths=[52 * mm, 52 * mm, 52 * mm],
    )
    summary.setStyle(TableStyle([
        ("BACKGROUND", (0, 0), (-1, 0), LIGHT_BLUE),
        ("BOX", (0, 0), (-1, -1), 0.5, BORDER),
        ("INNERGRID", (0, 0), (-1, -1), 0.4, BORDER),
        ("ALIGN", (0, 0), (-1, -1), "CENTER"),
        ("VALIGN", (0, 0), (-1, -1), "MIDDLE"),
        ("TOPPADDING", (0, 0), (-1, -1), 7),
        ("BOTTOMPADDING", (0, 0), (-1, -1), 7),
    ]))
    story.append(summary)
    story.append(Spacer(1, 8 * mm))

    story.append(Paragraph(
        "How to read the hierarchy",
        styles["SectionTitle"],
    ))
    story.append(Paragraph(
        "<b>Event type</b> → <b>Genre</b>. "
        "The IDs shown in the detailed listing are taken directly from "
        "the API response. Event types without genres are listed as "
        "categories with no associated genres.",
        styles["BodyText"],
    ))

    story.append(PageBreak())
    story.append(Paragraph(
        "Complete reference",
        styles["SectionTitle"],
    ))
    story.append(Paragraph(
        "Event types and genre names are displayed in the selected API "
        "language. Numeric identifiers are preserved exactly as returned.",
        styles["SmallText"],
    ))
    story.append(Spacer(1, 3 * mm))

    # Detailed listing. Sort numeric IDs numerically.
    for type_id, item in sorted(
        types.items(), key=lambda entry: int(entry[0])
    ):
        type_name = escape(str(item.get("name", "Unnamed type")))
        genres = item.get("genres") or {}

        block = [
            Paragraph(
                f"{escape(str(type_id))} · {type_name}",
                styles["TypeTitle"],
            )
        ]

        if not genres:
            block.append(Paragraph(
                "No genres associated with this event type.",
                styles["SmallText"],
            ))
        else:
            genre_items = sorted(
                genres.items(),
                key=lambda entry: int(entry[0]),
            )

            # Three-column layout, retaining the original genre IDs.
            rows = []
            for i in range(0, len(genre_items), 3):
                row = []

                for genre_id, genre_name in genre_items[i:i + 3]:
                    row.append(Paragraph(
                        f'<font color="#64748b">'
                        f'{escape(str(genre_id))}</font> · '
                        f'{escape(str(genre_name))}',
                        styles["Genre"],
                    ))

                while len(row) < 3:
                    row.append("")

                rows.append(row)

            genre_table = Table(
                rows,
                colWidths=[56 * mm, 56 * mm, 56 * mm],
                hAlign="LEFT",
            )
            genre_table.setStyle(TableStyle([
                ("VALIGN", (0, 0), (-1, -1), "TOP"),
                ("LEFTPADDING", (0, 0), (-1, -1), 3),
                ("RIGHTPADDING", (0, 0), (-1, -1), 3),
                ("TOPPADDING", (0, 0), (-1, -1), 1),
                ("BOTTOMPADDING", (0, 0), (-1, -1), 1),
            ]))
            block.append(genre_table)

        block.append(Spacer(1, 2 * mm))
        story.append(KeepTogether(block))

    # Source and generation details
    story.append(Spacer(1, 5 * mm))
    story.append(Paragraph(
        "Source: "
        "https://api.kulturbytes.de/api/event/type-genre-lookup"
        f"?lang={lang}",
        styles["SmallText"],
    ))
    story.append(Paragraph(
        "Generated on "
        + datetime.now(timezone.utc).strftime("%Y-%m-%d %H:%M UTC")
        + ". This document reflects the API response at generation time.",
        styles["SmallText"],
    ))

    doc.build(
        story,
        onFirstPage=add_footer,
        onLaterPages=add_footer,
    )

    print(f"PDF created: {output.resolve()}")
    print(f"Language: {lang}")
    print(f"Event types: {len(types)}")
    print(f"Types with genres: {types_with_genres}")
    print(f"Genre entries: {total_genres}")


def main():
    parser = argparse.ArgumentParser(
        description="Create a PDF of kulturbytes event types and genres."
    )
    parser.add_argument(
        "--lang",
        choices=["de", "da", "en"],
        default="en",
        help="Language used for event type and genre names (default: en)",
    )
    parser.add_argument(
        "--output",
        help="Output PDF path (default: event_types_<lang>.pdf)",
    )
    args = parser.parse_args()

    output = Path(
        args.output or f"event_types_{args.lang}.pdf"
    )

    build_pdf(args.lang, output)


if __name__ == "__main__":
    main()
