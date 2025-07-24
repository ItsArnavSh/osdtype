osdtype

A fully featured evolution of the OSDHack Mini-event at type.osdc.dev, osdtype is a next-generation typing test platform with a robust backend and advanced, real-time anti-cheating systems.
🚧 UNDER PROGRESS 🚧
Overview

osdtype is designed for speed, security, and fairness:

    Modern UI for engaging typing tests

    Secure GitHub OAuth authentication

    Real-time anti-cheat technologies

    Robust, scalable backend powered by PostgreSQL (managed via Docker)

Roadmap

    In Progress:

        WebSocket-Based Anti-Cheat:
        User actions and keystrokes are streamed in real-time for active analysis and cheat detection during tests.

        Efficient Keystroke Compression:
        Each keystroke’s timing is delta-encoded: time between strokes is quantized, and most events are stored in just one byte. Long pauses are handled by a compact multi-byte encoding, ensuring low storage and accurate replay for all sessions.

Compression Details

    Keystrokes are stored as the delta (time difference) from the previous stroke, measured in 20ms intervals.

    Most user activity is captured at 1 byte per press; only unusually long pauses use extra space.

    This design produces audit-friendly, replayable logs while minimizing data size and protecting user privacy.



Note:  Once the prototype is complete, osdtype will become part of OSDC.
