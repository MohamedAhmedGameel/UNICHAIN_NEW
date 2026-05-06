"""
services/gpa.py — Client-side GPA conversion (mirrors Ethereum Python layer).
"""
from __future__ import annotations

GPA_TABLE: list[tuple[int, float]] = [
    (97, 4.0), (93, 4.0), (90, 3.7),
    (87, 3.3), (83, 3.0), (80, 2.7),
    (77, 2.3), (73, 2.0), (70, 1.7),
    (67, 1.3), (63, 1.0), (60, 0.7),
    (0,  0.0),
]

LETTER_TABLE: list[tuple[int, str]] = [
    (97, "A+"), (93, "A"), (90, "A-"),
    (87, "B+"), (83, "B"), (80, "B-"),
    (77, "C+"), (73, "C"), (70, "C-"),
    (67, "D+"), (63, "D"), (60, "D-"),
    (0,  "F"),
]


def mark_to_gpa(mark: int) -> float:
    if mark <= 0:
        return 0.0
    for threshold, pts in GPA_TABLE:
        if mark >= threshold:
            return pts
    return 0.0


def letter_grade(mark: int) -> str:
    if mark <= 0:
        return "F"
    for threshold, grade in LETTER_TABLE:
        if mark >= threshold:
            return grade
    return "F"


def compute_gpa(records: list[dict]) -> float | None:
    graded = [r for r in records if r.get("active") and r.get("mark", 0) > 0]
    if not graded:
        return None
    return round(sum(mark_to_gpa(r["mark"]) for r in graded) / len(graded), 2)


def build_transcript(student_id: str, records: list[dict]) -> list[dict]:
    """Group enrollments by semester and compute per-semester GPA."""
    semesters: dict[str, list[dict]] = {}
    for r in records:
        sem = r.get("semester", "")
        semesters.setdefault(sem, []).append(r)

    result = []
    for sem, enrollments in semesters.items():
        graded = [e for e in enrollments if e.get("active") and e.get("mark", 0) > 0]
        gpa = round(sum(mark_to_gpa(e["mark"]) for e in graded) / len(graded), 2) if graded else None
        for e in enrollments:
            if e.get("mark", 0) > 0:
                e["grade"] = letter_grade(e["mark"])
                e["gpa_points"] = mark_to_gpa(e["mark"])
        result.append({
            "semester": sem,
            "enrollments": enrollments,
            "gpa": gpa,
            "total_courses": sum(1 for e in enrollments if e.get("active")),
            "graded_count": len(graded),
        })
    return result
