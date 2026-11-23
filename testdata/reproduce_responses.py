"""Independently check SPEC 7.1 response framing and byte-budget fixtures."""

import json
from pathlib import Path


def reject_constant(value):
    """JSON has no NaN or Infinity literals."""
    raise ValueError(f"non-JSON constant: {value}")


def accepts(method, body, limit):
    """Check the byte budget, one JSON document, and its operation's shape."""
    if method not in ("jobs", "fees"):
        raise ValueError(f"unknown response method: {method}")
    if limit < 0 or (limit > 0 and len(body.encode("utf-8")) > limit):
        return False
    try:
        value = json.loads(body, parse_constant=reject_constant)
    except ValueError:
        return False
    # The reference client's Go JSON decoder also accepts null.
    if value is None:
        return True
    if method == "jobs":
        return isinstance(value, list) and all(
            item is None or isinstance(item, dict) for item in value
        )
    return isinstance(value, dict)


def main():
    path = Path(__file__).with_name("response_vectors.json")
    cases = json.loads(path.read_text(encoding="utf-8"))
    for case in cases:
        if accepts(case["method"], case["body"], case["limit"]) != case["accept"]:
            raise SystemExit(f"response conformance failed: {case['name']}")
    print(f"All {len(cases)} response framing, shape, and budget vectors match SPEC 7.1")


if __name__ == "__main__":
    main()
