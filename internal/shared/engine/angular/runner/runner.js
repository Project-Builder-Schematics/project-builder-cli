// runner.js — minimal walking-skeleton runner for S-000.
// Emits a single {"type":"done","seq":1} NDJSON line and exits.
// Full NDJSON protocol (all 12 event types) is implemented in S-003/S-005.
'use strict';
process.stdout.write(JSON.stringify({ type: 'done', seq: 1 }) + '\n');
