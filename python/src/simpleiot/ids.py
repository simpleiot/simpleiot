"""Node IDs, as ``data/id.go``."""

from __future__ import annotations

import base64
import os
import re

__all__ = ["new_id", "valid_id"]

# lowercase Crockford base32, the alphabet data.NewID uses
_STD = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"
_CROCKFORD = "0123456789abcdefghjkmnpqrstvwxyz"
_TO_CROCKFORD = str.maketrans(_STD, _CROCKFORD)

_VALID = re.compile(r"[A-Za-z0-9][A-Za-z0-9-]{0,63}")


def new_id() -> str:
    """Returns a new random node ID: 128 random bits as 26 characters."""
    b = base64.b32encode(os.urandom(16)).decode().rstrip("=")
    return b.translate(_TO_CROCKFORD)


def valid_id(id: str) -> bool:
    """Reports whether an ID chosen by hand is safe to use.

    An ID is 1 to 64 ASCII letters, digits, and ``-``, not starting with
    ``-``. Other characters would split or corrupt a NATS subject.
    """
    return _VALID.fullmatch(id) is not None
